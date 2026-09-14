package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	grid "github.com/achannarasappa/term-grid"
	"github.com/alexso/ticker-vim/v5/internal/asset"
	"github.com/alexso/ticker-vim/v5/internal/cli"
	c "github.com/alexso/ticker-vim/v5/internal/common"
	mon "github.com/alexso/ticker-vim/v5/internal/monitor"
	"github.com/alexso/ticker-vim/v5/internal/stocksearch"
	"github.com/alexso/ticker-vim/v5/internal/ui/component/summary"
	"github.com/alexso/ticker-vim/v5/internal/ui/component/watchlist"
	"github.com/alexso/ticker-vim/v5/internal/ui/component/watchlist/row"
	"github.com/alexso/ticker-vim/v5/internal/uiconfig"
	"github.com/alexso/ticker-vim/v5/internal/updater"

	util "github.com/alexso/ticker-vim/v5/internal/ui/util"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/afero"
)

//nolint:gochecknoglobals
var (
	styleLogo          = util.NewStyle("#ffffd7", "#ff8700", true)
	styleGroup         = util.NewStyle("#8a8a8a", "#303030", false)
	styleGroupSelected = util.NewStyle("#ffffd7", "#5f5f5f", true)
	styleHelp          = util.NewStyle("#4e4e4e", "", true)
)

const (
	footerHeight = 1
	keyEscape    = "esc"
	keyEnter     = "enter"
	keyInterrupt = "ctrl+c"
)

type addState int

const (
	addClosed addState = iota
	addInput
	addSearching
	addResults
	addConfirm
	addError
)

// Model for UI
type Model struct {
	ctx                c.Context
	ready              bool
	headerHeight       int
	versionVector      int
	requestInterval    int
	assets             []c.Asset
	assetQuotes        []c.AssetQuote
	assetQuotesLookup  map[string]int
	positionSummary    asset.PositionSummary
	viewport           viewport.Model
	watchlist          *watchlist.Model
	summary            *summary.Model
	lastUpdateTime     string
	groupSelectedIndex int
	groupMaxIndex      int
	currentSort        string
	filterActive       bool
	filterQuery        string
	uiConfig           uiconfig.Config
	addState           addState
	addInput           textinput.Model
	addCandidates      []stocksearch.Candidate
	addSelectedIndex   int
	addError           string
	addRequestID       int
	deleteConfirm      bool
	deleteAsset        c.Asset
	deleteError        string
	editNameActive     bool
	editNameInput      textinput.Model
	editNameAsset      c.Asset
	editNameError      string
	stockFinder        stocksearch.Finder
	monitors           *mon.Monitor
	mu                 sync.RWMutex
	version            string
	latestVersion      string
	releasesURL        string
	fs                 afero.Fs
}

type tickMsg struct {
	versionVector int
}

type updateCheckMsg string

type updateCheckTickMsg struct{}

type stockSearchMsg struct {
	candidates []stocksearch.Candidate
	err        error
	requestID  int
}

type stockPreviewMsg struct {
	candidate stocksearch.Candidate
	err       error
	requestID int
}

type SetAssetQuoteMsg struct {
	symbol        string
	assetQuote    c.AssetQuote
	versionVector int
}

type SetAssetGroupQuoteMsg struct {
	assetGroupQuote c.AssetGroupQuote
	versionVector   int
}

// NewModel is the constructor for UI model
func NewModel(dep c.Dependencies, ctx c.Context, monitors *mon.Monitor, version string, configs ...uiconfig.Config) *Model {

	groupMaxIndex := len(ctx.Groups) - 1
	uiConfig := uiconfig.Default()
	if len(configs) > 0 {
		uiConfig = configs[0]
	}
	stockInput := textinput.New()
	stockInput.Prompt = "> "
	stockInput.Placeholder = "AAPL or Apple"
	stockInput.CharLimit = 80
	nameInput := textinput.New()
	nameInput.Prompt = "> "
	nameInput.CharLimit = 120
	initialSort := uiConfig.SortForGroup(ctx.Groups[0].Name)

	return &Model{
		ctx:               ctx,
		headerHeight:      getVerticalMargin(ctx.Config),
		ready:             false,
		requestInterval:   ctx.Config.RefreshInterval,
		versionVector:     0,
		assets:            make([]c.Asset, 0),
		assetQuotes:       make([]c.AssetQuote, 0),
		assetQuotesLookup: make(map[string]int),
		positionSummary:   asset.PositionSummary{},
		watchlist: watchlist.NewModel(watchlist.Config{
			Sort:                  initialSort,
			Separate:              ctx.Config.Separate,
			ShowPositions:         ctx.Config.ShowPositions,
			ExtraInfoExchange:     ctx.Config.ExtraInfoExchange,
			ExtraInfoFundamentals: ctx.Config.ExtraInfoFundamentals,
			Styles:                ctx.Reference.Styles,
			HighlightBackground:   uiConfig.Highlight.Background,
			FirstLine:             uiConfig.Display.FirstLine,
			ShowQuoteTime:         uiConfig.Display.QuoteTime,
		}),
		summary:            summary.NewModel(ctx),
		groupMaxIndex:      groupMaxIndex,
		groupSelectedIndex: 0,
		currentSort:        initialSort,
		uiConfig:           uiConfig,
		addInput:           stockInput,
		editNameInput:      nameInput,
		stockFinder:        stocksearch.NewYahooFinder(dep.MonitorYahooBaseURL),
		monitors:           monitors,
		version:            version,
		releasesURL:        dep.GitHubReleasesURL,
		fs:                 dep.Fs,
	}
}

// Init is the initialization hook for bubbletea
func (m *Model) Init() tea.Cmd {
	(*m.monitors).Start()

	// Start renderer and set symbols in parallel
	return tea.Batch(
		tick(0),
		updateCheckTick(),
		func() tea.Msg {
			err := (*m.monitors).SetAssetGroup(m.ctx.Groups[m.groupSelectedIndex], m.versionVector)

			if m.ctx.Config.Debug && err != nil {
				m.ctx.Logger.Println(err)
			}

			return nil
		},
		func() tea.Msg {
			return updateCheckMsg(updater.Check(m.version, m.releasesURL, updater.CacheFilePath(), m.fs))
		},
	)
}

// Update hook for bubbletea
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:maintidx,gocyclo
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.editNameActive {
			return m.handleEditNameKey(msg)
		}
		if m.deleteConfirm {
			return m.handleDeleteKey(msg)
		}
		if m.addState != addClosed {
			return m.handleAddKey(msg)
		}

		if m.filterActive {
			switch msg.String() {
			case keyInterrupt:
				return m, tea.Quit
			case keyEscape:
				m.filterActive = false
				m.filterQuery = ""

				return m.applyFilter()
			case keyEnter:
				m.filterActive = false

				return m, nil
			case "backspace", "ctrl+h":
				query := []rune(m.filterQuery)
				if len(query) > 0 {
					m.filterQuery = string(query[:len(query)-1])
				}

				return m.applyFilter()
			default:
				if len(msg.Runes) > 0 {
					m.filterQuery += string(msg.Runes)

					return m.applyFilter()
				}

				return m, nil
			}
		}

		key := msg.String()
		keys := m.uiConfig.Keybindings
		switch {
		case key == "tab" || key == keys.NextGroup:
			return m.changeGroup(1)
		case key == "shift+tab" || key == keys.PreviousGroup:
			return m.changeGroup(-1)
		case key == "up" || key == keys.SelectUp:
			return m.moveSelection(-1)
		case key == "down" || key == keys.SelectDown:
			return m.moveSelection(1)
		case key == keys.SelectFirst:
			m.watchlist, _ = m.watchlist.Update(watchlist.SetSelectionMsg(0))
			m.viewport.GotoTop()

			return m, nil
		case key == keys.SelectLast:
			m.watchlist, _ = m.watchlist.Update(watchlist.SetSelectionMsg(int(^uint(0) >> 1)))
			m.viewport.GotoBottom()

			return m, nil
		case key == keys.PageUp:
			return m.moveSelectionPage(-1)
		case key == keys.PageDown:
			return m.moveSelectionPage(1)
		case key == keys.Filter:
			m.filterActive = true

			return m, nil
		case key == keys.AddStock:
			m.addState = addInput
			m.addError = ""
			m.addCandidates = nil
			m.addInput.SetValue("")

			return m, m.addInput.Focus()
		case key == keys.DeleteStock:
			return m.openDeleteDialog()
		case key == keys.ToggleFirstLine:
			return m.toggleFirstLine()
		case key == keys.EditName:
			return m.openEditNameDialog()
		case groupIndexForKey(keys.Groups, key) >= 0:
			groupIndex := groupIndexForKey(keys.Groups, key)
			if groupIndex <= m.groupMaxIndex {
				return m.changeGroupTo(groupIndex)
			}

			return m, nil
		}

		switch key {
		case keyInterrupt:
			fallthrough
		case "q":
			return m, tea.Quit
		case keyEscape:
			if m.filterQuery != "" {
				m.filterQuery = ""

				return m.applyFilter()
			}

			return m, tea.Quit
		case "pgup":
			m.viewport.PageUp()

			return m, nil
		case "pgdown":
			m.viewport.PageDown()

			return m, nil
		case "s":
			m.mu.Lock()

			// Cycle through sort options: default -> alpha -> value -> user -> default
			sortOptions := []string{"", "alpha", "value", "user"}
			currentIndex := -1
			for i, sortOpt := range sortOptions {
				if m.currentSort == sortOpt {
					currentIndex = i

					break
				}
			}

			// Move to next sort option
			nextIndex := (currentIndex + 1) % len(sortOptions)
			m.currentSort = sortOptions[nextIndex]
			groupName := m.ctx.Groups[m.groupSelectedIndex].Name
			if m.currentSort == "" {
				m.uiConfig.Sorting.Groups[groupName] = "change"
			} else {
				m.uiConfig.Sorting.Groups[groupName] = m.currentSort
			}

			m.mu.Unlock()
			if err := uiconfig.SaveGroupSort(m.fs, m.ctx.ConfigPath, m.uiConfig, groupName, m.currentSort); err != nil && m.ctx.Config.Debug {
				m.ctx.Logger.Println(err)
			}

			// Update watchlist component with new sort
			m.watchlist, cmd = m.watchlist.Update(watchlist.ChangeSortMsg(m.currentSort))

			return m, cmd
		}

	case stockSearchMsg:
		if msg.requestID != m.addRequestID || m.addState != addSearching {
			return m, nil
		}
		if msg.err != nil {
			m.addState = addError
			m.addError = msg.err.Error()

			return m, nil
		}
		if len(msg.candidates) == 0 {
			m.addState = addError
			m.addError = "No matching Yahoo Finance symbols found"

			return m, nil
		}
		m.addCandidates = prioritizeCandidates(msg.candidates, m.ctx.Groups[m.groupSelectedIndex].Name)
		m.addSelectedIndex = 0
		m.addState = addResults

		return m, nil

	case stockPreviewMsg:
		if msg.requestID != m.addRequestID || m.addState != addSearching {
			return m, nil
		}
		if msg.err != nil {
			m.addState = addError
			m.addError = msg.err.Error()

			return m, nil
		}
		m.addCandidates[m.addSelectedIndex] = msg.candidate
		m.addState = addConfirm

		return m, nil

	case tea.WindowSizeMsg:

		var cmd tea.Cmd

		m.mu.Lock()
		defer m.mu.Unlock()

		viewportHeight := msg.Height - m.headerHeight - footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight
		}

		// Forward window size message to watchlist and summary component
		m.watchlist, cmd = m.watchlist.Update(msg)
		m.summary, _ = m.summary.Update(msg)

		return m, cmd

	// Trigger component re-render if data has changed
	case tickMsg:

		var cmd tea.Cmd
		cmds := make([]tea.Cmd, 0)

		m.mu.Lock()
		defer m.mu.Unlock()

		// Do not re-render if versionVector has changed and do not start a new timer with this versionVector
		if msg.versionVector != m.versionVector {
			return m, nil
		}

		// Update watchlist and summary components
		m.watchlist, cmd = m.watchlist.Update(watchlist.SetAssetsMsg(m.assets))
		m.summary, _ = m.summary.Update(summary.SetSummaryMsg(m.positionSummary))

		cmds = append(cmds, cmd)

		// Set the current tick time
		m.lastUpdateTime = getTime()

		// Update the viewport
		if m.ready {
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

		cmds = append(cmds, tick(msg.versionVector))

		return m, tea.Batch(cmds...)

	case SetAssetGroupQuoteMsg:

		m.mu.Lock()
		defer m.mu.Unlock()

		// Do not update the assets and position summary if the versionVector has changed
		if msg.versionVector != m.versionVector {
			return m, nil
		}

		assets, positionSummary := asset.GetAssets(m.ctx, msg.assetGroupQuote)

		m.assets = assets
		m.positionSummary = positionSummary

		m.assetQuotes = msg.assetGroupQuote.AssetQuotes
		for i, assetQuote := range m.assetQuotes {
			m.assetQuotesLookup[assetQuote.Symbol] = i
		}

		return m, nil

	case SetAssetQuoteMsg:

		var i int
		var ok bool

		m.mu.Lock()
		defer m.mu.Unlock()

		if msg.versionVector != m.versionVector {
			return m, nil
		}

		// Check if this symbol is in the lookup
		if i, ok = m.assetQuotesLookup[msg.symbol]; !ok {
			return m, nil
		}

		// Check if the index is out of bounds
		if i >= len(m.assetQuotes) {
			return m, nil
		}

		// Check if the symbol is the same
		if m.assetQuotes[i].Symbol != msg.symbol {
			return m, nil
		}

		// Update the asset quote and generate a new position summary
		m.assetQuotes[i] = msg.assetQuote

		assetGroupQuote := c.AssetGroupQuote{
			AssetQuotes: m.assetQuotes,
			AssetGroup:  m.ctx.Groups[m.groupSelectedIndex],
		}

		assets, positionSummary := asset.GetAssets(m.ctx, assetGroupQuote)

		m.assets = assets
		m.positionSummary = positionSummary

		return m, nil

	case row.FrameMsg:
		var cmd tea.Cmd
		m.watchlist, cmd = m.watchlist.Update(msg)

		return m, cmd

	case updateCheckMsg:
		m.latestVersion = string(msg)

		return m, nil

	case updateCheckTickMsg:
		return m, tea.Batch(
			updateCheckTick(),
			func() tea.Msg {
				return updateCheckMsg(updater.Check(m.version, m.releasesURL, updater.CacheFilePath(), m.fs))
			},
		)
	}

	return m, nil
}

func groupIndexForKey(keys []string, key string) int {
	for index, candidate := range keys {
		if candidate == key {
			return index
		}
	}

	return -1
}

func (m *Model) handleAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == keyInterrupt {
		return m, tea.Quit
	}

	switch m.addState {
	case addInput:
		if key == keyEscape {
			m.closeAddDialog()

			return m, nil
		}
		if key == keyEnter && strings.TrimSpace(m.addInput.Value()) != "" {
			m.addInput.Blur()
			m.addState = addSearching
			m.addRequestID++

			return m, searchStock(m.stockFinder, m.addInput.Value(), m.addRequestID)
		}
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)

		return m, cmd

	case addResults:
		switch key {
		case keyEscape:
			m.closeAddDialog()
		case "up", m.uiConfig.Keybindings.SelectUp:
			if m.addSelectedIndex > 0 {
				m.addSelectedIndex--
			}
		case "down", m.uiConfig.Keybindings.SelectDown:
			if m.addSelectedIndex+1 < len(m.addCandidates) {
				m.addSelectedIndex++
			}
		case keyEnter:
			m.addState = addSearching
			m.addRequestID++

			return m, previewStock(m.stockFinder, m.addCandidates[m.addSelectedIndex], m.addRequestID)
		}

		return m, nil

	case addConfirm:
		switch key {
		case "y", keyEnter:
			return m.addSelectedStock()
		case "n":
			m.addState = addResults
		case keyEscape:
			m.closeAddDialog()
		}

		return m, nil

	case addError:
		switch key {
		case keyEnter:
			m.addState = addInput
			m.addError = ""

			return m, m.addInput.Focus()
		case keyEscape:
			m.closeAddDialog()
		}

	case addSearching:
		if key == keyEscape {
			m.closeAddDialog()
		}

		return m, nil

	case addClosed:
		return m, nil
	}

	return m, nil
}

func searchStock(finder stocksearch.Finder, query string, requestID int) tea.Cmd {
	return func() tea.Msg {
		candidates, err := finder.Search(query)

		return stockSearchMsg{candidates: candidates, err: err, requestID: requestID}
	}
}

func previewStock(finder stocksearch.Finder, candidate stocksearch.Candidate, requestID int) tea.Cmd {
	return func() tea.Msg {
		candidate, err := finder.Preview(candidate)

		return stockPreviewMsg{candidate: candidate, err: err, requestID: requestID}
	}
}

func (m *Model) addSelectedStock() (tea.Model, tea.Cmd) {
	if m.addSelectedIndex >= len(m.addCandidates) {
		return m, nil
	}
	candidate := m.addCandidates[m.addSelectedIndex]
	group := &m.ctx.Groups[m.groupSelectedIndex]
	if err := cli.AddSymbolToConfig(m.fs, m.ctx.ConfigPath, group.Name, candidate.Symbol, candidate.Name); err != nil {
		m.addState = addError
		m.addError = err.Error()

		return m, nil
	}

	group.Watchlist = append(group.Watchlist, candidate.Symbol)
	if group.DisplayNames == nil {
		group.DisplayNames = make(map[string]string)
	}
	group.DisplayNames[strings.ToLower(candidate.Symbol)] = candidate.Name
	yahooGroupIndex := -1
	for index := range group.SymbolsBySource {
		if group.SymbolsBySource[index].Source == c.QuoteSourceYahoo {
			yahooGroupIndex = index

			break
		}
	}
	if yahooGroupIndex < 0 {
		group.SymbolsBySource = append(group.SymbolsBySource, c.AssetGroupSymbolsBySource{
			Source:  c.QuoteSourceYahoo,
			Symbols: []string{candidate.Symbol},
		})
	} else {
		group.SymbolsBySource[yahooGroupIndex].Symbols = append(group.SymbolsBySource[yahooGroupIndex].Symbols, candidate.Symbol)
	}

	m.versionVector++
	versionVector := m.versionVector
	if err := m.monitors.SetAssetGroup(*group, versionVector); err != nil {
		m.addState = addError
		m.addError = "Stock was saved, but refreshing failed: " + err.Error()

		return m, nil
	}
	m.closeAddDialog()

	return m, tickImmediate(versionVector)
}

func (m *Model) closeAddDialog() {
	m.addRequestID++
	m.addInput.Blur()
	m.addState = addClosed
	m.addError = ""
	m.addCandidates = nil
	m.addSelectedIndex = 0
}

func (m *Model) toggleFirstLine() (tea.Model, tea.Cmd) {
	firstLine := "name"
	if m.uiConfig.Display.FirstLine == "name" {
		firstLine = "symbol"
	}
	m.uiConfig.Display.FirstLine = firstLine
	m.watchlist, _ = m.watchlist.Update(watchlist.ChangeFirstLineMsg(firstLine))
	m.ensureSelectionVisible()
	if err := uiconfig.SaveFirstLine(m.fs, m.ctx.ConfigPath, m.uiConfig, firstLine); err != nil && m.ctx.Config.Debug {
		m.ctx.Logger.Println(err)
	}

	return m, nil
}

func (m *Model) openEditNameDialog() (tea.Model, tea.Cmd) {
	selected, ok := m.watchlist.SelectedAsset()
	if !ok {
		return m, nil
	}
	if _, ok := configuredWatchlistSymbol(m.ctx.Groups[m.groupSelectedIndex], selected.Symbol); !ok {
		return m, nil
	}
	m.editNameAsset = selected
	m.editNameError = ""
	m.editNameActive = true
	m.editNameInput.SetValue(selected.Name)
	m.editNameInput.CursorEnd()

	return m, m.editNameInput.Focus()
}

func (m *Model) handleEditNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyInterrupt:
		return m, tea.Quit
	case keyEscape:
		m.closeEditNameDialog()

		return m, nil
	case keyEnter:
		return m.saveEditedName()
	}
	var cmd tea.Cmd
	m.editNameInput, cmd = m.editNameInput.Update(msg)

	return m, cmd
}

func (m *Model) saveEditedName() (tea.Model, tea.Cmd) {
	group := &m.ctx.Groups[m.groupSelectedIndex]
	symbol, ok := configuredWatchlistSymbol(*group, m.editNameAsset.Symbol)
	if !ok {
		m.editNameError = "This item comes from a holding/lot and has no watchlist comment to edit."

		return m, nil
	}
	name := strings.TrimSpace(strings.Join(strings.Fields(m.editNameInput.Value()), " "))
	if err := cli.UpdateSymbolComment(m.fs, m.ctx.ConfigPath, group.Name, symbol, name); err != nil {
		m.editNameError = err.Error()

		return m, nil
	}
	if name == "" {
		delete(group.DisplayNames, strings.ToLower(symbol))
		name = m.providerName(symbol)
	} else {
		if group.DisplayNames == nil {
			group.DisplayNames = make(map[string]string)
		}
		group.DisplayNames[strings.ToLower(symbol)] = name
	}
	for index := range m.assets {
		if strings.EqualFold(m.assets[index].Symbol, symbol) {
			m.assets[index].Name = name
		}
	}
	m.watchlist, _ = m.watchlist.Update(watchlist.SetAssetsMsg(m.assets))
	m.closeEditNameDialog()

	return m, nil
}

func configuredWatchlistSymbol(group c.AssetGroup, symbol string) (string, bool) {
	for _, configuredSymbol := range group.Watchlist {
		if strings.EqualFold(configuredSymbol, symbol) {
			return configuredSymbol, true
		}
	}

	return "", false
}

func (m *Model) providerName(symbol string) string {
	for _, quote := range m.assetQuotes {
		if strings.EqualFold(quote.Symbol, symbol) {
			return quote.Name
		}
	}

	return symbol
}

func (m *Model) closeEditNameDialog() {
	m.editNameInput.Blur()
	m.editNameActive = false
	m.editNameError = ""
}

func (m *Model) editNameDialogView() string {
	body := fmt.Sprintf("Symbol: %s\n\nDisplay name:\n%s\n\nEnter: save   Esc: cancel", m.editNameAsset.Symbol, m.editNameInput.View())
	if m.editNameError != "" {
		body += "\n\nCould not save: " + m.editNameError
	}
	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5977c9")).
		Padding(1, 2).
		Width(min(100, max(56, m.viewport.Width-8))).
		Render(styleGroupSelected(" Edit display name ") + "\n\n" + body)

	return lipgloss.Place(m.viewport.Width, m.viewport.Height, lipgloss.Center, lipgloss.Center, dialog)
}

func (m *Model) openDeleteDialog() (tea.Model, tea.Cmd) {
	selected, ok := m.watchlist.SelectedAsset()
	if !ok {
		return m, nil
	}
	m.deleteAsset = selected
	m.deleteError = ""
	m.deleteConfirm = true

	return m, nil
}

func (m *Model) handleDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyInterrupt:
		return m, tea.Quit
	case keyEscape, "n":
		m.deleteConfirm = false
		m.deleteError = ""

		return m, nil
	case "y", keyEnter:
		if m.deleteError != "" {
			return m, nil
		}

		return m.deleteSelectedStock()
	}

	return m, nil
}

func (m *Model) deleteSelectedStock() (tea.Model, tea.Cmd) {
	group := &m.ctx.Groups[m.groupSelectedIndex]
	symbol := m.deleteAsset.Symbol
	watchlistIndex := -1
	for index, configuredSymbol := range group.Watchlist {
		if strings.EqualFold(configuredSymbol, symbol) {
			watchlistIndex = index
			symbol = configuredSymbol

			break
		}
	}
	if watchlistIndex < 0 {
		m.deleteError = "This stock comes from a holding/lot, not the watchlist. Remove its lot from .ticker.yaml instead."

		return m, nil
	}
	if err := cli.RemoveSymbolFromConfig(m.fs, m.ctx.ConfigPath, group.Name, symbol); err != nil {
		m.deleteError = err.Error()

		return m, nil
	}

	group.Watchlist = append(group.Watchlist[:watchlistIndex], group.Watchlist[watchlistIndex+1:]...)
	delete(group.DisplayNames, strings.ToLower(symbol))
	if !groupHasSymbolInLots(*group, symbol) {
		for sourceIndex := range group.SymbolsBySource {
			symbols := group.SymbolsBySource[sourceIndex].Symbols
			for symbolIndex, monitoredSymbol := range symbols {
				if strings.EqualFold(monitoredSymbol, symbol) {
					symbols = append(symbols[:symbolIndex], symbols[symbolIndex+1:]...)
					group.SymbolsBySource[sourceIndex].Symbols = symbols

					break
				}
			}
		}
	}

	m.deleteConfirm = false
	m.deleteError = ""
	m.versionVector++
	versionVector := m.versionVector
	if err := m.monitors.SetAssetGroup(*group, versionVector); err != nil {
		m.deleteConfirm = true
		m.deleteError = "Stock was removed, but refreshing failed: " + err.Error()

		return m, nil
	}

	return m, tickImmediate(versionVector)
}

func groupHasSymbolInLots(group c.AssetGroup, symbol string) bool {
	for _, lot := range group.Lots {
		if strings.EqualFold(lot.Symbol, symbol) {
			return true
		}
	}

	return false
}

func (m *Model) deleteDialogView() string {
	groupName := m.ctx.Groups[m.groupSelectedIndex].Name
	body := fmt.Sprintf("%s\n%s\nPrice: %.2f %s\n\nRemove this stock from %s?\n\ny/Enter: delete   n/Esc: cancel", m.deleteAsset.Symbol, m.deleteAsset.Name, m.deleteAsset.QuotePrice.Price, m.deleteAsset.Currency.FromCurrencyCode, groupName)
	if m.deleteError != "" {
		body = "Could not delete stock:\n\n" + m.deleteError + "\n\nn/Esc: close"
	}
	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5977c9")).
		Padding(1, 2).
		Width(min(100, max(56, m.viewport.Width-8))).
		Render(styleGroupSelected(" Delete stock ") + "\n\n" + body)

	return lipgloss.Place(m.viewport.Width, m.viewport.Height, lipgloss.Center, lipgloss.Center, dialog)
}

func (m *Model) addDialogView() string {
	groupName := m.ctx.Groups[m.groupSelectedIndex].Name
	title := "Add stock to " + groupName
	body := ""

	switch m.addState {
	case addClosed:
		return ""
	case addInput:
		body = "Search by symbol or company name\n\n" + m.addInput.View() + "\n\nEnter: search   Esc: cancel"
	case addSearching:
		if len(m.addCandidates) == 0 {
			body = "Searching Yahoo Finance…"
		} else {
			body = "Loading current quote…"
		}
	case addResults:
		var results strings.Builder
		results.WriteString("Select the correct result:\n\n")
		for index, candidate := range m.addCandidates {
			cursor := "  "
			if index == m.addSelectedIndex {
				cursor = "› "
			}
			_, _ = fmt.Fprintf(&results, "%s%-12s %-30s [%s]\n", cursor, candidate.Symbol, truncate(candidate.Name, 30), truncate(candidate.Exchange, 15))
		}
		_, _ = fmt.Fprintf(&results, "\n%s/%s: select   Enter: preview   Esc: cancel", m.uiConfig.Keybindings.SelectDown, m.uiConfig.Keybindings.SelectUp)
		body = results.String()
	case addConfirm:
		candidate := m.addCandidates[m.addSelectedIndex]
		body = fmt.Sprintf("%s\n%s\nExchange: %s\nPrice: %.2f %s\n\nAdd this stock to %s?  y/Enter: yes   n: back   Esc: cancel", candidate.Symbol, candidate.Name, candidate.Exchange, candidate.Price, candidate.Currency, groupName)
	case addError:
		body = "Could not add stock:\n\n" + m.addError + "\n\nEnter: try again   Esc: cancel"
	}

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5977c9")).
		Padding(1, 2).
		Width(min(100, max(56, m.viewport.Width-8))).
		Render(styleGroupSelected(" "+title+" ") + "\n\n" + body)

	return lipgloss.Place(m.viewport.Width, m.viewport.Height, lipgloss.Center, lipgloss.Center, dialog)
}

func truncate(value string, maxLength int) string {
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}

	return string(runes[:maxLength-1]) + "…"
}

func prioritizeCandidates(candidates []stocksearch.Candidate, groupName string) []stocksearch.Candidate {
	groupName = strings.ToLower(groupName)
	prioritized := append([]stocksearch.Candidate(nil), candidates...)
	for index, candidate := range prioritized {
		if strings.Contains(strings.ToLower(candidate.Exchange), groupName) {
			copy(prioritized[1:index+1], prioritized[0:index])
			prioritized[0] = candidate

			break
		}
	}

	return prioritized
}

// View rendering hook for bubbletea
func (m *Model) View() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return "\n  Initializing..."
	}
	if m.addState != addClosed {
		return m.addDialogView() + "\n" +
			footer(m.viewport.Width, m.lastUpdateTime, m.ctx.Groups, m.groupSelectedIndex, m.currentSort, m.latestVersion, m.filterQuery, m.filterActive, m.uiConfig.Keybindings)
	}
	if m.editNameActive {
		return m.editNameDialogView() + "\n" +
			footer(m.viewport.Width, m.lastUpdateTime, m.ctx.Groups, m.groupSelectedIndex, m.currentSort, m.latestVersion, m.filterQuery, m.filterActive, m.uiConfig.Keybindings)
	}
	if m.deleteConfirm {
		return m.deleteDialogView() + "\n" +
			footer(m.viewport.Width, m.lastUpdateTime, m.ctx.Groups, m.groupSelectedIndex, m.currentSort, m.latestVersion, m.filterQuery, m.filterActive, m.uiConfig.Keybindings)
	}

	m.viewport.SetContent(m.watchlist.View())

	viewSummary := ""

	if m.ctx.Config.ShowSummary {
		viewSummary += m.summary.View() + "\n"
	}

	return viewSummary +
		m.viewport.View() + "\n" +
		footer(m.viewport.Width, m.lastUpdateTime, m.ctx.Groups, m.groupSelectedIndex, m.currentSort, m.latestVersion, m.filterQuery, m.filterActive, m.uiConfig.Keybindings)

}

func footer(width int, time string, groups []c.AssetGroup, groupSelectedIndex int, currentSort string, latestVersion string, filterQuery string, filterActive bool, keys uiconfig.Keybindings) string {

	if width < 80 {
		return styleLogo(" ticker-vim ")
	}

	// Get display name for current sort
	sortDisplayName := "change"
	switch currentSort {
	case "alpha":
		sortDisplayName = "alpha"
	case "value":
		sortDisplayName = "value"
	case "user":
		sortDisplayName = "user"
	}

	baseHelpText := fmt.Sprintf(" q:exit %s/%s:select %s/%s:jump %s/%s:first/last %s:filter %s:add %s:delete %s:edit %s:toggle", keys.SelectDown, keys.SelectUp, keys.PageDown, keys.PageUp, keys.SelectFirst, keys.SelectLast, keys.Filter, keys.AddStock, keys.DeleteStock, keys.EditName, keys.ToggleFirstLine)
	sortHelpText := " s: change sort (" + sortDisplayName + ")"
	filterText := ""
	if filterActive || filterQuery != "" {
		filterText = " /" + filterQuery
		if filterActive {
			filterText += "▏"
		}
	}

	rightText := "↻  " + time
	if latestVersion != "" {
		rightText = "↑ " + latestVersion + " available"
	}

	// Calculate minimum width for sort help text to appear
	// Longest sort text is "s: change sort (change)" = 24 characters
	// Minimum width needed includes the logo, group tabs, help, sort, and update time.
	const sortHelpMinWidth = 130
	groupText, groupWidth := renderGroupTabs(groups, groupSelectedIndex)

	return grid.Render(grid.Grid{
		Rows: []grid.Row{
			{
				Width: width,
				Cells: []grid.Cell{
					{Text: styleLogo(" ticker-vim "), Width: 12},
					{Text: groupText, Width: groupWidth, VisibleMinWidth: 80},
					{Text: styleHelp(filterText), Width: len(filterText), VisibleMinWidth: 80},
					{Text: styleHelp(baseHelpText), Width: len(baseHelpText), VisibleMinWidth: 105},
					{Text: styleHelp(sortHelpText), Width: len(sortHelpText), VisibleMinWidth: sortHelpMinWidth},
					{Text: styleHelp(rightText), Align: grid.Right},
				},
			},
		},
	})

}

func renderGroupTabs(groups []c.AssetGroup, selectedIndex int) (string, int) {
	var tabs strings.Builder
	width := 0
	for i, group := range groups {
		name := group.Name
		if name == "" {
			name = "default"
		}
		text := fmt.Sprintf(" %d %s ", i+1, name)
		if i == selectedIndex {
			tabs.WriteString(styleGroupSelected(text))
		} else {
			tabs.WriteString(styleGroup(text))
		}
		width += len(name) + 4
	}

	return tabs.String(), width
}

func (m *Model) applyFilter() (*Model, tea.Cmd) {
	var cmd tea.Cmd
	m.watchlist, cmd = m.watchlist.Update(watchlist.ChangeFilterMsg(m.filterQuery))
	m.watchlist, _ = m.watchlist.Update(watchlist.SetSelectionMsg(0))
	m.viewport.GotoTop()

	return m, cmd
}

func (m *Model) changeGroup(cursor int) (*Model, tea.Cmd) {
	groupIndex := (m.groupSelectedIndex + cursor + m.groupMaxIndex + 1) % (m.groupMaxIndex + 1)

	return m.changeGroupTo(groupIndex)
}

func (m *Model) changeGroupTo(groupIndex int) (*Model, tea.Cmd) {
	m.mu.Lock()
	m.groupSelectedIndex = groupIndex
	m.versionVector++
	m.filterActive = false
	m.filterQuery = ""
	m.watchlist, _ = m.watchlist.Update(watchlist.ChangeFilterMsg(""))
	m.watchlist, _ = m.watchlist.Update(watchlist.SetSelectionMsg(0))
	m.viewport.GotoTop()
	m.currentSort = m.uiConfig.SortForGroup(m.ctx.Groups[groupIndex].Name)
	m.watchlist, _ = m.watchlist.Update(watchlist.ChangeSortMsg(m.currentSort))
	versionVector := m.versionVector
	group := m.ctx.Groups[m.groupSelectedIndex]
	m.mu.Unlock()

	// Set the new symbols and request a refresh. The resulting quotes are versioned,
	// so late responses from the previous group are safely ignored.
	m.monitors.SetAssetGroup(group, versionVector) //nolint:errcheck

	return m, tickImmediate(versionVector)
}

func (m *Model) moveSelection(offset int) (*Model, tea.Cmd) {
	m.watchlist, _ = m.watchlist.Update(watchlist.MoveSelectionMsg(offset))
	m.ensureSelectionVisible()

	return m, nil
}

func (m *Model) moveSelectionPage(direction int) (*Model, tea.Cmd) {
	m.watchlist, _ = m.watchlist.Update(watchlist.MoveSelectionPageMsg{Direction: direction, Height: max(1, m.viewport.Height/2)})
	m.ensureSelectionVisible()

	return m, nil
}

func (m *Model) ensureSelectionVisible() {
	start, end, ok := m.watchlist.SelectedLineRange()
	if !ok {
		return
	}

	if start < m.viewport.YOffset {
		m.viewport.SetYOffset(start)
	} else if end >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(end - m.viewport.Height + 1)
	}
}

func getVerticalMargin(config c.Config) int {
	if config.ShowSummary {
		return 2
	}

	return 0
}

func updateCheckTick() tea.Cmd {
	return tea.Tick(3*time.Hour, func(time.Time) tea.Msg {
		return updateCheckTickMsg{}
	})
}

// Send a new tick message with the versionVector 200ms from now
func tick(versionVector int) tea.Cmd {
	return tea.Tick(time.Second/5, func(time.Time) tea.Msg {
		return tickMsg{
			versionVector: versionVector,
		}
	})
}

// Send a new tick message immediately
func tickImmediate(versionVector int) tea.Cmd {

	return func() tea.Msg {
		return tickMsg{
			versionVector: versionVector,
		}
	}
}

func getTime() string {
	t := time.Now()

	return fmt.Sprintf("%s %02d:%02d:%02d", t.Weekday().String(), t.Hour(), t.Minute(), t.Second())
}
