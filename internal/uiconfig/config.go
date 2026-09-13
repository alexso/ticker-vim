package uiconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

const FileName = "ticker-vim.yaml"

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`) //nolint:gochecknoglobals

type Config struct {
	Keybindings Keybindings `yaml:"keybindings"`
	Highlight   Highlight   `yaml:"highlight"`
	Sorting     Sorting     `yaml:"sorting"`
	Display     Display     `yaml:"display"`
}

type Keybindings struct {
	SelectUp        string   `yaml:"select-up"`
	SelectDown      string   `yaml:"select-down"`
	SelectFirst     string   `yaml:"select-first"`
	SelectLast      string   `yaml:"select-last"`
	PageUp          string   `yaml:"page-up"`
	PageDown        string   `yaml:"page-down"`
	PreviousGroup   string   `yaml:"previous-group"`
	NextGroup       string   `yaml:"next-group"`
	Groups          []string `yaml:"groups"`
	Filter          string   `yaml:"filter"`
	AddStock        string   `yaml:"add-stock"`
	DeleteStock     string   `yaml:"delete-stock"`
	ToggleFirstLine string   `yaml:"toggle-first-line"`
	EditName        string   `yaml:"edit-name"`
}

type Highlight struct {
	Background string `yaml:"background"`
}

type Sorting struct {
	Default string            `yaml:"default"`
	Groups  map[string]string `yaml:"groups,omitempty"`
}

type Display struct {
	FirstLine string `yaml:"first-line"`
	QuoteTime bool   `yaml:"quote-time"`
}

func Default() Config {
	return Config{
		Keybindings: Keybindings{
			SelectUp:        "k",
			SelectDown:      "j",
			SelectFirst:     "g",
			SelectLast:      "G",
			PageUp:          "u",
			PageDown:        "d",
			PreviousGroup:   "h",
			NextGroup:       "l",
			Groups:          []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
			Filter:          "/",
			AddStock:        "a",
			DeleteStock:     "x",
			ToggleFirstLine: "t",
			EditName:        "e",
		},
		Highlight: Highlight{Background: "#142350"},
		Sorting:   Sorting{Default: "alpha", Groups: make(map[string]string)},
		Display:   Display{FirstLine: "name", QuoteTime: true},
	}
}

func Path(stockConfigPath string) string {
	return filepath.Join(filepath.Dir(stockConfigPath), FileName)
}

func Load(fs afero.Fs, stockConfigPath string) (Config, error) {
	config := Default()
	path := Path(stockConfigPath)
	data, err := afero.ReadFile(fs, path)
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read UI config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("invalid UI config %s: %w", path, err)
	}
	applyDefaults(&config)
	if !colorPattern.MatchString(config.Highlight.Background) {
		return Config{}, fmt.Errorf("invalid highlight.background %q: expected #RRGGBB", config.Highlight.Background) //nolint:goerr113
	}
	if config.Display.FirstLine != "name" && config.Display.FirstLine != "symbol" {
		return Config{}, fmt.Errorf("invalid display.first-line %q: expected name or symbol", config.Display.FirstLine) //nolint:goerr113
	}

	return config, nil
}

func applyDefaults(config *Config) {
	defaults := Default()
	if config.Keybindings.SelectUp == "" {
		config.Keybindings.SelectUp = defaults.Keybindings.SelectUp
	}
	if config.Keybindings.SelectDown == "" {
		config.Keybindings.SelectDown = defaults.Keybindings.SelectDown
	}
	if config.Keybindings.SelectFirst == "" {
		config.Keybindings.SelectFirst = defaults.Keybindings.SelectFirst
	}
	if config.Keybindings.SelectLast == "" {
		config.Keybindings.SelectLast = defaults.Keybindings.SelectLast
	}
	if config.Keybindings.PageUp == "" {
		config.Keybindings.PageUp = defaults.Keybindings.PageUp
	}
	if config.Keybindings.PageDown == "" {
		config.Keybindings.PageDown = defaults.Keybindings.PageDown
	}
	if config.Keybindings.PreviousGroup == "" {
		config.Keybindings.PreviousGroup = defaults.Keybindings.PreviousGroup
	}
	if config.Keybindings.NextGroup == "" {
		config.Keybindings.NextGroup = defaults.Keybindings.NextGroup
	}
	if len(config.Keybindings.Groups) == 0 {
		config.Keybindings.Groups = defaults.Keybindings.Groups
	}
	if config.Keybindings.Filter == "" {
		config.Keybindings.Filter = defaults.Keybindings.Filter
	}
	if config.Keybindings.AddStock == "" {
		config.Keybindings.AddStock = defaults.Keybindings.AddStock
	}
	if config.Keybindings.DeleteStock == "" {
		config.Keybindings.DeleteStock = defaults.Keybindings.DeleteStock
	}
	if config.Keybindings.ToggleFirstLine == "" {
		config.Keybindings.ToggleFirstLine = defaults.Keybindings.ToggleFirstLine
	}
	if config.Keybindings.EditName == "" {
		config.Keybindings.EditName = defaults.Keybindings.EditName
	}
	if config.Highlight.Background == "" {
		config.Highlight.Background = defaults.Highlight.Background
	}
	if config.Sorting.Default == "" {
		config.Sorting.Default = defaults.Sorting.Default
	}
	if config.Sorting.Groups == nil {
		config.Sorting.Groups = make(map[string]string)
	}
	if config.Display.FirstLine == "" {
		config.Display.FirstLine = defaults.Display.FirstLine
	}
}

// SortForGroup returns a group's remembered sort or the UI default.
func (config Config) SortForGroup(groupName string) string {
	if sort, ok := config.Sorting.Groups[groupName]; ok {
		return decodeSort(sort)
	}

	return decodeSort(config.Sorting.Default)
}

// SaveGroupSort remembers a sort in ticker-vim.yaml, not the stock configuration.
func SaveGroupSort(fs afero.Fs, stockConfigPath string, config Config, groupName string, sort string) error {
	if config.Sorting.Groups == nil {
		config.Sorting.Groups = make(map[string]string)
	}
	config.Sorting.Groups[groupName] = encodeSort(sort)

	return save(fs, stockConfigPath, config)
}

// SaveFirstLine remembers whether names or symbols are displayed first.
func SaveFirstLine(fs afero.Fs, stockConfigPath string, config Config, firstLine string) error {
	config.Display.FirstLine = firstLine

	return save(fs, stockConfigPath, config)
}

func save(fs afero.Fs, stockConfigPath string, config Config) error {
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("encode UI config: %w", err)
	}
	path := Path(stockConfigPath)
	mode := os.FileMode(0o644)
	if info, err := fs.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	temporary, err := afero.TempFile(fs, filepath.Dir(path), ".ticker-vim-ui-*.yaml")
	if err != nil {
		return fmt.Errorf("create temporary UI config: %w", err)
	}
	temporaryPath := temporary.Name()
	defer fs.Remove(temporaryPath) //nolint:errcheck
	if _, err := temporary.Write(output.Bytes()); err != nil {
		temporary.Close() //nolint:errcheck

		return fmt.Errorf("write temporary UI config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary UI config: %w", err)
	}
	if err := fs.Chmod(temporaryPath, mode); err != nil {
		return fmt.Errorf("preserve UI config permissions: %w", err)
	}
	if err := fs.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace UI config: %w", err)
	}

	return nil
}

func encodeSort(sort string) string {
	if sort == "" {
		return "change"
	}

	return sort
}

func decodeSort(sort string) string {
	if sort == "change" {
		return ""
	}

	return sort
}
