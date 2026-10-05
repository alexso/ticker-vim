package monitor

import (
	"context"
	"errors"
	"testing"
	"time"

	c "github.com/alexso/ticker-vim/v5/internal/common"
)

type startupMonitor struct {
	started chan struct{}
	release chan struct{}
	err     error
	symbols []string
}

func (m *startupMonitor) Start() error                           { return nil }
func (m *startupMonitor) Stop() error                            { return nil }
func (m *startupMonitor) SetCurrencyRates(c.CurrencyRates) error { return nil }
func (m *startupMonitor) SetSymbols(symbols []string, _ int) error {
	if m.started != nil {
		close(m.started)
		m.started = nil
		<-m.release
	}
	m.symbols = symbols
	return m.err
}
func (m *startupMonitor) GetAssetQuotes(...bool) ([]c.AssetQuote, error) {
	quotes := make([]c.AssetQuote, len(m.symbols))
	for i, symbol := range m.symbols {
		quotes[i].Symbol = symbol
	}
	return quotes, nil
}

func TestStartupPublishesQuotesAfterTimeout(t *testing.T) {
	source := &startupMonitor{started: make(chan struct{}), release: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan c.AssetGroupQuote, 1)
	m := &Monitor{
		ctx:                     ctx,
		monitors:                map[c.QuoteSource]c.Monitor{c.QuoteSourceYahoo: source},
		onUpdateAssetGroupQuote: func(quote c.AssetGroupQuote, _ int) { updates <- quote },
	}
	group := c.AssetGroup{SymbolsBySource: []c.AssetGroupSymbolsBySource{{Source: c.QuoteSourceYahoo, Symbols: []string{"AAPL"}}}}
	err := m.SetAssetGroup(group, 0)
	if err == nil {
		t.Fatal("expected caller to time out")
	}
	close(source.release)
	select {
	case quote := <-updates:
		if len(quote.AssetQuotes) != 1 || quote.AssetQuotes[0].Symbol != "AAPL" {
			t.Fatalf("late startup quotes = %+v", quote.AssetQuotes)
		}
	case <-time.After(time.Second):
		t.Fatal("successful late load did not publish initial quotes")
	}
}

func TestSetAssetGroupReportsCompletedSourceError(t *testing.T) {
	source := &startupMonitor{err: errors.New("fetch failed")}
	m := &Monitor{
		ctx:                     context.Background(),
		monitors:                map[c.QuoteSource]c.Monitor{c.QuoteSourceYahoo: source},
		onUpdateAssetGroupQuote: func(c.AssetGroupQuote, int) { t.Error("failed load published quotes") },
	}
	group := c.AssetGroup{SymbolsBySource: []c.AssetGroupSymbolsBySource{{Source: c.QuoteSourceYahoo, Symbols: []string{"AAPL"}}}}
	for range 100 {
		if err := m.SetAssetGroup(group, 0); err == nil {
			t.Fatal("source error was lost")
		}
	}
}

func TestLateStartupDoesNotReplaceNewerGroup(t *testing.T) {
	source := &startupMonitor{started: make(chan struct{}), release: make(chan struct{})}
	started := source.started
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan int, 2)
	m := &Monitor{
		ctx:                     ctx,
		monitors:                map[c.QuoteSource]c.Monitor{c.QuoteSourceYahoo: source},
		onUpdateAssetGroupQuote: func(_ c.AssetGroupQuote, version int) { updates <- version },
	}
	group := func(symbol string) c.AssetGroup {
		return c.AssetGroup{SymbolsBySource: []c.AssetGroupSymbolsBySource{{Source: c.QuoteSourceYahoo, Symbols: []string{symbol}}}}
	}
	first := make(chan error, 1)
	go func() { first <- m.SetAssetGroup(group("AAPL"), 0) }()
	<-started
	// Register the new group before allowing the old startup request to complete.
	second := make(chan error, 1)
	go func() { second <- m.SetAssetGroup(group("MSFT"), 1) }()
	deadline := time.Now().Add(time.Second)
	for {
		m.mu.RLock()
		registered := m.assetGroupRequest == 2
		m.mu.RUnlock()
		if registered {
			break
		}
		if time.Now().After(deadline) {
			close(source.release)
			t.Fatal("new group request was not registered")
		}
		time.Sleep(time.Millisecond)
	}
	close(source.release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	select {
	case version := <-updates:
		if version != 1 {
			t.Fatalf("stale group published with version %d", version)
		}
	case <-time.After(time.Second):
		t.Fatal("new group did not publish")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if quotes := m.GetAssetGroupQuote().AssetQuotes; len(quotes) != 1 || quotes[0].Symbol != "MSFT" {
		t.Fatalf("active group quotes = %+v", quotes)
	}
}
