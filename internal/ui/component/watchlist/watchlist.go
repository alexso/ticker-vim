package watchlist

import (
	"fmt"
	"regexp"
	"strings"

	c "github.com/alexso/ticker-vim/v5/internal/common"
	s "github.com/alexso/ticker-vim/v5/internal/sorter"
	row "github.com/alexso/ticker-vim/v5/internal/ui/component/watchlist/row"
	u "github.com/alexso/ticker-vim/v5/internal/ui/util"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ansiStylePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`) //nolint:gochecknoglobals

// Config represents the configuration for the watchlist component
type Config struct {
	Separate              bool
	ShowPositions         bool
	ExtraInfoExchange     bool
	ExtraInfoFundamentals bool
	Sort                  string
	Styles                c.Styles
	HighlightBackground   string
	FirstLine             string
	ShowQuoteTime         bool
}

// Model for watchlist section
type Model struct {
	width              int
	sourceAssets       []*c.Asset
	assets             []*c.Asset
	assetsBySymbol     map[string]*c.Asset
	sorter             s.Sorter
	config             Config
	cellWidths         row.CellWidthsContainer
	rows               []*row.Model
	rowsBySymbol       map[string]*row.Model
	filter             string
	selectedIndex      int
	selectedBackground string
}

// Messages for replacing assets
type SetAssetsMsg []c.Asset

// Messages for updating assets
type UpdateAssetsMsg []c.Asset

// Messages for changing sort
type ChangeSortMsg string

// ChangeFilterMsg updates the case-insensitive symbol/name filter.
type ChangeFilterMsg string

// ChangeFirstLineMsg changes whether the name or symbol is rendered first.
type ChangeFirstLineMsg string

// MoveSelectionMsg moves the highlighted row by the supplied offset.
type MoveSelectionMsg int

// MoveSelectionPageMsg moves the selection approximately one viewport page.
type MoveSelectionPageMsg struct {
	Direction int
	Height    int
}

// SetSelectionMsg selects an exact visible row index.
type SetSelectionMsg int

// NewModel returns a model with default values
func NewModel(config Config) *Model {
	selectedBackground := backgroundSequence(config.HighlightBackground)

	return &Model{
		width:              80,
		sourceAssets:       make([]*c.Asset, 0),
		config:             config,
		assets:             make([]*c.Asset, 0),
		assetsBySymbol:     make(map[string]*c.Asset),
		sorter:             s.NewSorter(config.Sort),
		rowsBySymbol:       make(map[string]*row.Model),
		selectedBackground: selectedBackground,
	}
}

// Init initializes the watchlist
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the watchlist
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SetAssetsMsg:
		// Convert []c.Asset to []*c.Asset.
		assets := make([]*c.Asset, len(msg))

		for i := range msg {
			assets[i] = &msg[i]
		}

		m.sourceAssets = assets
		assets = m.filteredAssets()

		return m.setVisibleAssets(assets)

	case ChangeFilterMsg:
		m.filter = string(msg)

		return m.setVisibleAssets(m.filteredAssets())

	case ChangeFirstLineMsg:
		m.config.FirstLine = string(msg)
		for index, currentRow := range m.rows {
			m.rows[index], _ = currentRow.Update(row.SetFirstLineMsg(msg))
		}

		return m, nil

	case MoveSelectionMsg:
		m.setSelectedIndex(m.selectedIndex + int(msg))

		return m, nil

	case MoveSelectionPageMsg:
		m.moveSelectionPage(msg.Direction, msg.Height)

		return m, nil

	case SetSelectionMsg:
		m.setSelectedIndex(int(msg))

		return m, nil

	case tea.WindowSizeMsg:

		m.width = msg.Width
		m.cellWidths = getCellWidths(m.assets)
		for i, r := range m.rows {
			m.rows[i], _ = r.Update(row.SetCellWidthsMsg{
				Width:      m.width,
				CellWidths: m.cellWidths,
			})
		}

		return m, nil

	case row.FrameMsg:

		var cmd tea.Cmd
		cmds := make([]tea.Cmd, 0)

		// TODO: send message to a specific row rather than all rows
		for i, r := range m.rows {
			m.rows[i], cmd = r.Update(msg)
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)

	case ChangeSortMsg:
		// Update the sorter with the new sort option
		m.config.Sort = string(msg)
		m.sorter = s.NewSorter(m.config.Sort)

		return m.setVisibleAssets(m.filteredAssets())

	}

	return m, nil
}

func (m *Model) filteredAssets() []*c.Asset {
	assets := make([]*c.Asset, 0, len(m.sourceAssets))
	for _, asset := range m.sourceAssets {
		if fuzzyMatch(asset.Symbol+" "+asset.Name, m.filter) {
			assets = append(assets, asset)
		}
	}

	return m.sorter(assets)
}

func (m *Model) setVisibleAssets(assets []*c.Asset) (*Model, tea.Cmd) {
	var cmd tea.Cmd
	cmds := make([]tea.Cmd, 0)
	assetsBySymbol := make(map[string]*c.Asset, len(assets))
	rowsBySymbol := make(map[string]*row.Model, len(assets))

	for i, asset := range assets {
		assetsBySymbol[asset.Symbol] = asset
		if i < len(m.rows) {
			m.rows[i], cmd = m.rows[i].Update(row.UpdateAssetMsg(asset))
			cmds = append(cmds, cmd)
		} else {
			m.rows = append(m.rows, row.New(row.Config{
				Separate:              m.config.Separate,
				ExtraInfoExchange:     m.config.ExtraInfoExchange,
				ExtraInfoFundamentals: m.config.ExtraInfoFundamentals,
				ShowPositions:         m.config.ShowPositions,
				Styles:                m.config.Styles,
				Asset:                 asset,
				FirstLine:             m.config.FirstLine,
				ShowQuoteTime:         m.config.ShowQuoteTime,
			}))
		}
		rowsBySymbol[asset.Symbol] = m.rows[i]
	}

	if len(assets) < len(m.rows) {
		m.rows = m.rows[:len(assets)]
	}

	m.assets = assets
	m.assetsBySymbol = assetsBySymbol
	m.rowsBySymbol = rowsBySymbol
	m.setSelectedIndex(m.selectedIndex)
	m.cellWidths = getCellWidths(m.assets)
	for i, r := range m.rows {
		m.rows[i], _ = r.Update(row.SetCellWidthsMsg{
			Width:      m.width,
			CellWidths: m.cellWidths,
		})
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) setSelectedIndex(index int) {
	if len(m.rows) == 0 {
		m.selectedIndex = 0

		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(m.rows) {
		index = len(m.rows) - 1
	}
	m.selectedIndex = index
}

// SelectedLineRange returns the first and last rendered lines of the selected row.
func (m *Model) SelectedLineRange() (int, int, bool) {
	if len(m.rows) == 0 {
		return 0, 0, false
	}

	start := 0
	for i := range m.selectedIndex {
		start += strings.Count(m.rows[i].View(), "\n") + 1
	}
	height := strings.Count(m.rows[m.selectedIndex].View(), "\n") + 1

	return start, start + height - 1, true
}

// SelectedAsset returns the currently highlighted visible asset.
func (m *Model) SelectedAsset() (c.Asset, bool) {
	if len(m.assets) == 0 || m.selectedIndex >= len(m.assets) {
		return c.Asset{}, false
	}

	return *m.assets[m.selectedIndex], true
}

func (m *Model) moveSelectionPage(direction int, height int) {
	if len(m.rows) == 0 || direction == 0 {
		return
	}
	if height < 1 {
		height = 1
	}
	start, _, _ := m.SelectedLineRange()
	target := start + direction*height
	bestIndex := 0
	bestDistance := int(^uint(0) >> 1)
	line := 0
	for index, row := range m.rows {
		distance := line - target
		if distance < 0 {
			distance = -distance
		}
		if distance < bestDistance {
			bestIndex = index
			bestDistance = distance
		}
		line += strings.Count(row.View(), "\n") + 1
	}
	m.setSelectedIndex(bestIndex)
}

// fuzzyMatch performs a case-insensitive subsequence match. Empty queries match all assets.
func fuzzyMatch(value string, query string) bool {
	valueRunes := []rune(strings.ToLower(value))
	queryRunes := []rune(strings.ToLower(strings.TrimSpace(query)))
	if len(queryRunes) == 0 {
		return true
	}

	queryIndex := 0
	for _, valueRune := range valueRunes {
		if valueRune == queryRunes[queryIndex] {
			queryIndex++
			if queryIndex == len(queryRunes) {
				return true
			}
		}
	}

	return false
}

// View rendering hook for bubbletea
func (m *Model) View() string {

	if m.width < 80 {
		return fmt.Sprintf("Terminal window too narrow to render content\nResize to fix (%d/80)", m.width)
	}
	if len(m.rows) == 0 && m.filter != "" {
		return "No symbols match /" + m.filter
	}

	rows := make([]string, 0)
	for i, row := range m.rows {
		view := row.View()
		if i == m.selectedIndex {
			view = highlightRow(view, m.width, m.selectedBackground)
		}
		rows = append(rows, view)
	}

	return strings.Join(rows, "\n")

}

func highlightRow(view string, width int, background string) string {
	padded := lipgloss.NewStyle().Width(width).Render(view)
	// Text, tags, and animated prices contain their own ANSI resets and backgrounds.
	// Reapply the selection background after every style sequence so the entire row
	// remains one continuous highlighted surface.
	highlighted := ansiStylePattern.ReplaceAllStringFunc(padded, func(style string) string {
		return style + background
	})

	return background + highlighted + "\x1b[0m"
}

func backgroundSequence(color string) string {
	color = strings.TrimPrefix(color, "#")
	var red, green, blue uint8
	_, _ = fmt.Sscanf(color, "%02x%02x%02x", &red, &green, &blue)

	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", red, green, blue)
}
func getCellWidths(assets []*c.Asset) row.CellWidthsContainer {

	cellMaxWidths := row.CellWidthsContainer{}

	for _, asset := range assets {
		var quoteLength int

		volumeMarketCapLength := len(u.ConvertFloatToString(asset.QuoteExtended.MarketCap, true))

		if asset.QuoteExtended.FiftyTwoWeekHigh == 0.0 {
			quoteLength = len(u.ConvertFloatToString(asset.QuotePrice.Price, asset.Meta.IsVariablePrecision))
		}

		if asset.QuoteExtended.FiftyTwoWeekHigh != 0.0 {
			quoteLength = len(u.ConvertFloatToString(asset.QuoteExtended.FiftyTwoWeekHigh, asset.Meta.IsVariablePrecision))
		}

		if volumeMarketCapLength > cellMaxWidths.WidthVolumeMarketCap {
			cellMaxWidths.WidthVolumeMarketCap = volumeMarketCapLength
		}

		if quoteLength > cellMaxWidths.QuoteLength {
			cellMaxWidths.QuoteLength = quoteLength
			cellMaxWidths.WidthQuote = quoteLength + row.WidthChangeStatic
			cellMaxWidths.WidthQuoteExtended = quoteLength
			cellMaxWidths.WidthQuoteRange = row.WidthRangeStatic + (quoteLength * 2)
		}

		if asset.Position != (c.Position{}) {
			positionLength := len(u.ConvertFloatToString(asset.Position.Value, asset.Meta.IsVariablePrecision))
			positionQuantityLength := len(u.ConvertFloatToString(asset.Position.Quantity, asset.Meta.IsVariablePrecision))
			positionUnitCostLength := len(u.ConvertFloatToString(asset.Position.UnitCost, asset.Meta.IsVariablePrecision))

			if positionLength > cellMaxWidths.PositionLength {
				cellMaxWidths.PositionLength = positionLength
				cellMaxWidths.WidthPosition = positionLength + row.WidthChangeStatic + row.WidthPositionGutter
			}

			if positionLength > cellMaxWidths.WidthPositionExtended {
				cellMaxWidths.WidthPositionExtended = positionLength
			}

			if positionQuantityLength > cellMaxWidths.WidthPositionExtended {
				cellMaxWidths.WidthPositionExtended = positionQuantityLength
			}

			if positionUnitCostLength > cellMaxWidths.WidthPositionExtended {
				cellMaxWidths.WidthPositionExtended = positionUnitCostLength
			}

		}

	}

	return cellMaxWidths

}
