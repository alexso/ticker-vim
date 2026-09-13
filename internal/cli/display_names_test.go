package cli

import (
	"testing"

	c "github.com/alexso/ticker-vim/v5/internal/common"
	"github.com/spf13/afero"
)

func TestApplyDisplayNames(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	path := "/ticker/.ticker.yaml"
	if err := fs.MkdirAll("/ticker", 0o755); err != nil {
		t.Fatal(err)
	}
	input := "groups:\n  - name: funds\n    watchlist:\n      - 0P0001ECQR.ST # Avanza Global\n      - 0P00005U1J.ST\n"
	if err := afero.WriteFile(fs, path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	groups := []c.AssetGroup{{ConfigAssetGroup: c.ConfigAssetGroup{Name: "funds"}}}
	if err := ApplyDisplayNames(fs, path, groups); err != nil {
		t.Fatal(err)
	}
	if got := groups[0].DisplayNames["0p0001ecqr.st"]; got != "Avanza Global" {
		t.Fatalf("display name = %q, want Avanza Global", got)
	}
	if _, exists := groups[0].DisplayNames["0p00005u1j.st"]; exists {
		t.Fatal("symbol without a comment unexpectedly got a display name")
	}
}
