package monitor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	c "github.com/alexso/ticker-vim/v5/internal/common"
	monitorPriceCoinbase "github.com/alexso/ticker-vim/v5/internal/monitor/coinbase/monitor-price"
	monitorCurrencyRate "github.com/alexso/ticker-vim/v5/internal/monitor/yahoo/monitor-currency-rates"
	monitorPriceYahoo "github.com/alexso/ticker-vim/v5/internal/monitor/yahoo/monitor-price"
	unaryClientYahoo "github.com/alexso/ticker-vim/v5/internal/monitor/yahoo/unary"
)

// Monitor represents an overall monitor which manages API specific monitors
type Monitor struct {
	monitors                map[c.QuoteSource]c.Monitor
	monitorCurrencyRate     c.MonitorCurrencyRate
	chanError               chan error
	chanUpdateAssetQuote    chan c.MessageUpdate[c.AssetQuote]
	chanUpdateCurrencyRates chan c.CurrencyRates
	onUpdateAssetQuote      func(symbol string, assetQuote c.AssetQuote, versionVector int)
	onUpdateAssetGroupQuote func(assetGroupQuote c.AssetGroupQuote, versionVector int)
	assetGroupVersionVector int
	assetGroup              c.AssetGroup
	mu                      sync.RWMutex
	muAssetGroup            sync.Mutex
	assetGroupRequest       uint64
	assetGroupLoaded        bool
	logger                  *log.Logger
	ctx                     context.Context
	cancel                  context.CancelFunc
}

// ConfigMonitor represents the configuration for the main monitor
type ConfigMonitor struct {
	RefreshInterval int
	TargetCurrency  string
	Logger          *log.Logger
	Cache           c.Cache
	ConfigMonitorPriceCoinbase
	ConfigMonitorsYahoo
}

// ConfigMonitorPriceCoinbase represents the configuration for the Coinbase monitor
type ConfigMonitorPriceCoinbase struct {
	BaseURL      string
	StreamingURL string
}

// ConfigMonitorsYahoo represents the configuration for the Yahoo monitors (price and currency rate)
type ConfigMonitorsYahoo struct {
	BaseURL           string
	SessionRootURL    string
	SessionCrumbURL   string
	SessionConsentURL string
}

// ConfigUpdateFns represents the callback functions for when asset quotes are updated
type ConfigUpdateFns struct {
	OnUpdateAssetQuote      func(symbol string, assetQuote c.AssetQuote, versionVector int)
	OnUpdateAssetGroupQuote func(assetGroupQuote c.AssetGroupQuote, versionVector int)
}

// New creates a new instance of the Coinbase monitor
func NewMonitor(configMonitor ConfigMonitor) (*Monitor, error) {

	chanError := make(chan error, 5)
	chanUpdateAssetQuote := make(chan c.MessageUpdate[c.AssetQuote], 10)
	chanUpdateCurrencyRate := make(chan c.CurrencyRates, 10)
	chanRequestCurrencyRate := make(chan []string, 10)

	ctx, cancel := context.WithCancel(context.Background())

	coinbase := monitorPriceCoinbase.NewMonitorPriceCoinbase(
		monitorPriceCoinbase.Config{
			Ctx:                      ctx,
			UnaryURL:                 configMonitor.ConfigMonitorPriceCoinbase.BaseURL,
			ChanError:                chanError,
			ChanUpdateAssetQuote:     chanUpdateAssetQuote,
			ChanRequestCurrencyRates: chanRequestCurrencyRate,
			Cache:                    configMonitor.Cache,
		},
		monitorPriceCoinbase.WithStreamingURL(configMonitor.ConfigMonitorPriceCoinbase.StreamingURL),
		monitorPriceCoinbase.WithRefreshInterval(time.Duration(configMonitor.RefreshInterval)*time.Second),
	)

	// Create and configure the API client for the Yahoo API shared between monitors
	unaryAPI := unaryClientYahoo.NewUnaryAPI(unaryClientYahoo.Config{
		BaseURL:           configMonitor.ConfigMonitorsYahoo.BaseURL,
		SessionRootURL:    configMonitor.ConfigMonitorsYahoo.SessionRootURL,
		SessionCrumbURL:   configMonitor.ConfigMonitorsYahoo.SessionCrumbURL,
		SessionConsentURL: configMonitor.ConfigMonitorsYahoo.SessionConsentURL,
		Cache:             configMonitor.Cache,
	})

	yahoo := monitorPriceYahoo.NewMonitorPriceYahoo(
		monitorPriceYahoo.Config{
			Ctx:                      ctx,
			UnaryAPI:                 unaryAPI,
			ChanError:                chanError,
			ChanUpdateAssetQuote:     chanUpdateAssetQuote,
			ChanRequestCurrencyRates: chanRequestCurrencyRate,
			Cache:                    configMonitor.Cache,
		},
		monitorPriceYahoo.WithRefreshInterval(time.Duration(configMonitor.RefreshInterval)*time.Second),
	)

	yahooCurrencyRate := monitorCurrencyRate.NewMonitorCurrencyRateYahoo(
		monitorCurrencyRate.Config{
			Ctx:                      ctx,
			UnaryAPI:                 unaryAPI,
			ChanUpdateCurrencyRates:  chanUpdateCurrencyRate,
			ChanRequestCurrencyRates: chanRequestCurrencyRate,
			ChanError:                chanError,
			Cache:                    configMonitor.Cache,
		},
	)

	yahooCurrencyRate.SetTargetCurrency(configMonitor.TargetCurrency)

	m := &Monitor{
		monitors: map[c.QuoteSource]c.Monitor{
			c.QuoteSourceCoinbase: coinbase,
			c.QuoteSourceYahoo:    yahoo,
		},
		monitorCurrencyRate:     yahooCurrencyRate,
		chanUpdateAssetQuote:    chanUpdateAssetQuote,
		chanUpdateCurrencyRates: chanUpdateCurrencyRate,
		chanError:               chanError,
		onUpdateAssetGroupQuote: func(assetGroupQuote c.AssetGroupQuote, versionVector int) {},
		onUpdateAssetQuote:      func(symbol string, assetQuote c.AssetQuote, versionVector int) {},
		logger:                  configMonitor.Logger,
		ctx:                     ctx,
		cancel:                  cancel,
	}

	return m, nil
}

// SetAssetGroup sets the asset group for the monitor
func (m *Monitor) SetAssetGroup(assetGroup c.AssetGroup, versionVector int) error {
	m.mu.Lock()
	m.assetGroupRequest++
	request := m.assetGroupRequest
	m.mu.Unlock()

	// A timeout limits how long the caller waits, not the lifetime of the load.
	// Publish successful late results so startup can recover without a group change.
	done := make(chan error, 1)
	go func() {
		// Serialize loads so an older request cannot overwrite a newer source cache.
		m.muAssetGroup.Lock()
		defer m.muAssetGroup.Unlock()

		m.mu.RLock()
		current := request == m.assetGroupRequest
		m.mu.RUnlock()
		if !current {
			done <- nil

			return
		}

		var wg sync.WaitGroup
		chanError := make(chan error, len(assetGroup.SymbolsBySource))
		for _, symbolBySource := range assetGroup.SymbolsBySource {
			if monitor, exists := m.monitors[symbolBySource.Source]; exists {
				wg.Add(1)
				go func(mon c.Monitor, symbols []string) {
					defer wg.Done()
					if err := mon.SetSymbols(symbols, versionVector); err != nil {
						chanError <- err
					}
				}(monitor, symbolBySource.Symbols)
			}
		}
		wg.Wait()
		close(chanError)
		var loadErrors []error
		for err := range chanError {
			loadErrors = append(loadErrors, err)
		}
		if len(loadErrors) > 0 {
			done <- fmt.Errorf("errors setting symbols on monitor(s): %v", loadErrors)

			return
		}

		m.mu.Lock()
		if request != m.assetGroupRequest || m.ctx.Err() != nil {
			m.mu.Unlock()
			done <- nil

			return
		}
		m.assetGroupVersionVector = versionVector
		m.assetGroup = assetGroup
		m.assetGroupLoaded = true
		assetGroupQuote := m.GetAssetGroupQuote()
		go m.onUpdateAssetGroupQuote(assetGroupQuote, versionVector)
		m.mu.Unlock()
		done <- nil
	}()

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		return errors.New("timeout waiting for monitor(s) to set symbols; loading continues in background")
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

// SetOnUpdate sets the callback functions for when asset quotes are updated
func (m *Monitor) SetOnUpdate(config ConfigUpdateFns) error {

	if config.OnUpdateAssetQuote == nil || config.OnUpdateAssetGroupQuote == nil {
		return errors.New("onUpdateAssetQuote and onUpdateAssetGroupQuote must be set")
	}

	m.onUpdateAssetQuote = config.OnUpdateAssetQuote
	m.onUpdateAssetGroupQuote = config.OnUpdateAssetGroupQuote

	return nil
}

// Start starts all monitors
func (m *Monitor) Start() {

	m.monitorCurrencyRate.Start() //nolint:errcheck

	for _, monitor := range m.monitors {
		monitor.Start() //nolint:errcheck
	}

	go m.handleUpdates()
}

// GetAssetGroupQuote synchronously gets price quotes a group of assets across all sources
func (m *Monitor) GetAssetGroupQuote(ignoreCache ...bool) c.AssetGroupQuote {

	assetQuotesFromAllSources := make([]c.AssetQuote, 0)

	for _, symbolBySource := range m.assetGroup.SymbolsBySource {

		assetQuotes, _ := m.monitors[symbolBySource.Source].GetAssetQuotes(ignoreCache...)
		assetQuotesFromAllSources = append(assetQuotesFromAllSources, assetQuotes...)

	}

	return c.AssetGroupQuote{
		AssetQuotes: assetQuotesFromAllSources,
		AssetGroup:  m.assetGroup,
	}
}

// handleUpdates listens for asset quote updates and errors from monitors
func (m *Monitor) handleUpdates() {
	for {
		select {
		case <-m.ctx.Done():

			return
		case update := <-m.chanUpdateAssetQuote:

			m.mu.RLock()

			// Skip updates from previous asset groups
			if update.VersionVector != m.assetGroupVersionVector {
				m.mu.RUnlock()

				continue
			}
			m.mu.RUnlock()

			// Call the callback function for individual asset quote updates
			go m.onUpdateAssetQuote(update.Data.Symbol, update.Data, update.VersionVector)

		case err := <-m.chanError:
			// Log errors using the configured logger if one is set
			if m.logger != nil {
				m.logger.Printf("%v", err)
			}

		case currencyRates := <-m.chanUpdateCurrencyRates:
			// Set currency rates on each each monitor
			for _, monitor := range m.monitors {
				err := monitor.SetCurrencyRates(currencyRates)
				if err != nil {
					m.chanError <- err
				}
			}

			// Get asset quotes for all sources with new currency rates
			m.mu.RLock()
			if !m.assetGroupLoaded {
				m.mu.RUnlock()

				continue
			}
			assetGroupQuote := m.GetAssetGroupQuote()
			versionVector := m.assetGroupVersionVector
			m.mu.RUnlock()

			// Callback with new asset quotes which include the new currency rates
			go m.onUpdateAssetGroupQuote(assetGroupQuote, versionVector)
		}
	}
}

// Stop stops all monitors and cancels the context
func (m *Monitor) Stop() {

	for _, monitor := range m.monitors {
		monitor.Stop() //nolint:errcheck
	}

	m.cancel()

}
