package cli

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

func TestAddSymbolToConfig(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	path := "/ticker/.ticker.yaml"
	if err := fs.MkdirAll("/ticker", 0o755); err != nil {
		t.Fatal(err)
	}
	input := "# portfolio\ngroups:\n  - name: Stockholm\n    watchlist:\n      - ERIC-B.ST # Ericsson\n\n  - name: USA\n    watchlist:\n      - AMD\n"
	if err := afero.WriteFile(fs, path, []byte(input), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := AddSymbolToConfig(fs, path, "Stockholm", "BOL.ST", "Boliden AB (publ)"); err != nil {
		t.Fatal(err)
	}
	output, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "# portfolio") || !strings.Contains(string(output), "# Ericsson") {
		t.Fatalf("comments were not preserved:\n%s", output)
	}
	if !strings.Contains(string(output), "      - ERIC-B.ST # Ericsson\n      - BOL.ST # Boliden AB (publ)\n\n  - name: USA") {
		t.Fatalf("surrounding formatting was not preserved:\n%s", output)
	}

	var decoded struct {
		Groups []struct {
			Name      string   `yaml:"name"`
			Watchlist []string `yaml:"watchlist"`
		} `yaml:"groups"`
	}
	if err := yaml.Unmarshal(output, &decoded); err != nil {
		t.Fatal(err)
	}
	if got := decoded.Groups[0].Watchlist; len(got) != 2 || got[1] != "BOL.ST" {
		t.Fatalf("unexpected Stockholm watchlist: %#v", got)
	}
	if got := decoded.Groups[1].Watchlist; len(got) != 1 || got[0] != "AMD" {
		t.Fatalf("USA group changed unexpectedly: %#v", got)
	}
	info, err := fs.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
}

func TestRemoveSymbolFromConfig(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	path := "/ticker/.ticker.yaml"
	if err := fs.MkdirAll("/ticker", 0o755); err != nil {
		t.Fatal(err)
	}
	input := "# portfolio\ngroups:\n  - name: Stockholm\n    watchlist:\n      - ERIC-B.ST # keep this\n      - BOL.ST # remove this\n\n  - name: USA\n    watchlist:\n      - AMD\n"
	if err := afero.WriteFile(fs, path, []byte(input), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := RemoveSymbolFromConfig(fs, path, "Stockholm", "bol.st"); err != nil {
		t.Fatal(err)
	}
	output, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(output), "BOL.ST") {
		t.Fatalf("deleted symbol remains:\n%s", output)
	}
	if !strings.Contains(string(output), "      - ERIC-B.ST # keep this\n\n  - name: USA") || !strings.Contains(string(output), "      - AMD") {
		t.Fatalf("surrounding formatting changed:\n%s", output)
	}
	info, err := fs.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
	if err := RemoveSymbolFromConfig(fs, path, "Stockholm", "BOL.ST"); err == nil {
		t.Fatal("expected an error when deleting a missing symbol")
	}
}

func TestUpdateSymbolComment(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	path := "/ticker/.ticker.yaml"
	if err := fs.MkdirAll("/ticker", 0o755); err != nil {
		t.Fatal(err)
	}
	input := "groups:\n  - name: funds\n    watchlist:\n      - FUND.ST # Old name\n\n  - name: USA\n    watchlist:\n      - AMD # Advanced Micro Devices, Inc.\n"
	if err := afero.WriteFile(fs, path, []byte(input), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := UpdateSymbolComment(fs, path, "funds", "FUND.ST", "Useful Fund Name"); err != nil {
		t.Fatal(err)
	}
	output, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "      - FUND.ST # Useful Fund Name\n\n  - name: USA") {
		t.Fatalf("comment was not updated cleanly:\n%s", output)
	}
	if !strings.Contains(string(output), "AMD # Advanced Micro Devices, Inc.") {
		t.Fatalf("another group changed:\n%s", output)
	}
}

func TestAddSymbolToEmptyWatchlistPreservesComments(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	path := "/ticker/.ticker.yaml"
	if err := fs.MkdirAll("/ticker", 0o755); err != nil {
		t.Fatal(err)
	}
	input := "# portfolio\nshow-tags: true # keep tags\ngroups:\n  - name: Stockholm # local market\n    watchlist: [] # starts empty\n"
	if err := afero.WriteFile(fs, path, []byte(input), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := AddSymbolToConfig(fs, path, "Stockholm", "BOL.ST", "Boliden"); err != nil {
		t.Fatal(err)
	}
	output, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{"# portfolio", "# keep tags", "# local market", "# starts empty", "# Boliden"} {
		if !strings.Contains(string(output), comment) {
			t.Fatalf("comment %q was not preserved:\n%s", comment, output)
		}
	}
}

func TestRemoveFinalSymbolPreservesComments(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	path := "/ticker/.ticker.yaml"
	if err := fs.MkdirAll("/ticker", 0o755); err != nil {
		t.Fatal(err)
	}
	input := "# portfolio\nshow-tags: true # keep tags\ngroups:\n  - name: Stockholm # local market\n    watchlist:\n      - BOL.ST # removed with stock\n  - name: USA # keep group\n    watchlist:\n      - AMD # keep stock\n"
	if err := afero.WriteFile(fs, path, []byte(input), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSymbolFromConfig(fs, path, "Stockholm", "BOL.ST"); err != nil {
		t.Fatal(err)
	}
	output, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{"# portfolio", "# keep tags", "# local market", "# keep group", "# keep stock"} {
		if !strings.Contains(string(output), comment) {
			t.Fatalf("comment %q was not preserved:\n%s", comment, output)
		}
	}
	if strings.Contains(string(output), "# removed with stock") {
		t.Fatalf("deleted stock comment remains:\n%s", output)
	}
}
