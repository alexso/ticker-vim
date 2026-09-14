package uiconfig

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestLoadDefaultsAndOverrides(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	if err := fs.MkdirAll("/config/ticker", 0o755); err != nil {
		t.Fatal(err)
	}
	contents := []byte("keybindings:\n  select-down: ctrl+n\n  add-stock: A\nhighlight:\n  background: '#123456'\n")
	if err := afero.WriteFile(fs, "/config/ticker/ticker-vim.yaml", contents, 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := Load(fs, "/config/ticker/.ticker.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if config.Keybindings.SelectDown != "ctrl+n" || config.Keybindings.SelectUp != "k" {
		t.Fatalf("unexpected merged keybindings: %#v", config.Keybindings)
	}
	if config.Keybindings.AddStock != "A" || config.Highlight.Background != "#123456" || config.Display.FirstLine != "name" {
		t.Fatalf("unexpected UI config: %#v", config)
	}
	if config.Keybindings.PageUp != "u" || config.Keybindings.PageDown != "d" || config.Keybindings.DeleteStock != "x" {
		t.Fatalf("unexpected navigation defaults: %#v", config.Keybindings)
	}
	if config.Keybindings.ToggleFirstLine != "t" || config.Keybindings.EditName != "e" {
		t.Fatalf("unexpected display key defaults: %#v", config.Keybindings)
	}
	if !config.Display.QuoteTime {
		t.Fatal("quote timestamps should be enabled by default")
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	t.Parallel()
	config, err := Load(afero.NewMemMapFs(), "/config/ticker/.ticker.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if config.Keybindings.SelectDown != "j" || config.Keybindings.PageUp != "u" || config.Keybindings.DeleteStock != "x" || config.Highlight.Background != "#142350" {
		t.Fatalf("unexpected defaults: %#v", config)
	}
}

func TestSortForGroupAndSave(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	if err := fs.MkdirAll("/config/ticker", 0o755); err != nil {
		t.Fatal(err)
	}
	path := "/config/ticker/ticker-vim.yaml"
	input := "# interface preferences\nsorting:\n  default: alpha # valid: alpha, change, value, user\n  groups: {} # populated automatically\ndisplay:\n  first-line: name # name or symbol\n  quote-time: true # keep timestamps\n"
	if err := afero.WriteFile(fs, path, []byte(input), 0o640); err != nil {
		t.Fatal(err)
	}
	config, err := Load(fs, "/config/ticker/.ticker.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.SortForGroup("Stockholm"); got != "alpha" {
		t.Fatalf("default sort = %q, want alpha", got)
	}
	if err := SaveGroupSort(fs, "/config/ticker/.ticker.yaml", config, "Stockholm", "value"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(fs, "/config/ticker/.ticker.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.SortForGroup("Stockholm"); got != "value" {
		t.Fatalf("remembered sort = %q, want value", got)
	}
	output, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{"# interface preferences", "# valid: alpha, change, value, user", "# populated automatically", "# name or symbol", "# keep timestamps"} {
		if !strings.Contains(string(output), comment) {
			t.Fatalf("comment %q was not preserved after sorting update:\n%s", comment, output)
		}
	}
	if err := SaveFirstLine(fs, "/config/ticker/.ticker.yaml", reloaded, "symbol"); err != nil {
		t.Fatal(err)
	}
	reloaded, err = Load(fs, "/config/ticker/.ticker.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Display.FirstLine != "symbol" {
		t.Fatalf("remembered first line = %q, want symbol", reloaded.Display.FirstLine)
	}
	output, err = afero.ReadFile(fs, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{"# interface preferences", "# valid: alpha, change, value, user", "# populated automatically", "# name or symbol", "# keep timestamps"} {
		if !strings.Contains(string(output), comment) {
			t.Fatalf("comment %q was not preserved after display update:\n%s", comment, output)
		}
	}
	if info, err := fs.Stat(path); err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("config mode was not preserved: info=%v err=%v", info, err)
	}
}
