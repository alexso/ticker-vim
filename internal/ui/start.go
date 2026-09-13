package ui

import (
	c "github.com/alexso/ticker-vim/v5/internal/common"
	mon "github.com/alexso/ticker-vim/v5/internal/monitor"
	"github.com/alexso/ticker-vim/v5/internal/uiconfig"
	tea "github.com/charmbracelet/bubbletea"
)

// Start launches the command line interface and starts capturing input
func Start(dep *c.Dependencies, ctx *c.Context, version string) func() error {
	return func() error {
		uiConfig, err := uiconfig.Load(dep.Fs, ctx.ConfigPath)
		if err != nil {
			return err
		}

		monitors, _ := mon.NewMonitor(mon.ConfigMonitor{
			RefreshInterval: ctx.Config.RefreshInterval,
			TargetCurrency:  ctx.Config.Currency,
			Logger:          ctx.Logger,
			Cache:           ctx.Cache,
			ConfigMonitorsYahoo: mon.ConfigMonitorsYahoo{
				BaseURL:           dep.MonitorYahooBaseURL,
				SessionRootURL:    dep.MonitorYahooSessionRootURL,
				SessionCrumbURL:   dep.MonitorYahooSessionCrumbURL,
				SessionConsentURL: dep.MonitorYahooSessionConsentURL,
			},
			ConfigMonitorPriceCoinbase: mon.ConfigMonitorPriceCoinbase{
				BaseURL:      dep.MonitorPriceCoinbaseBaseURL,
				StreamingURL: dep.MonitorPriceCoinbaseStreamingURL,
			},
		})

		p := tea.NewProgram(
			NewModel(*dep, *ctx, monitors, version, uiConfig),
			tea.WithMouseCellMotion(),
			tea.WithAltScreen(),
		)

		err = monitors.SetOnUpdate(mon.ConfigUpdateFns{
			OnUpdateAssetQuote: func(symbol string, assetQuote c.AssetQuote, versionVector int) {
				p.Send(SetAssetQuoteMsg{
					symbol:        symbol,
					assetQuote:    assetQuote,
					versionVector: versionVector,
				})
			},
			OnUpdateAssetGroupQuote: func(assetGroupQuote c.AssetGroupQuote, versionVector int) {
				p.Send(SetAssetGroupQuoteMsg{
					assetGroupQuote: assetGroupQuote,
					versionVector:   versionVector,
				})
			},
		})

		if err != nil {

			return err
		}

		_, err = p.Run()

		return err
	}

}
