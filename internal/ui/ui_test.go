package ui

import (
	"testing"

	"github.com/acarl005/stripansi"
	c "github.com/alexso/ticker-vim/v5/internal/common"
)

func TestRenderGroupTabs(t *testing.T) {
	t.Parallel()

	groups := []c.AssetGroup{
		{ConfigAssetGroup: c.ConfigAssetGroup{Name: "Stockholm"}},
		{ConfigAssetGroup: c.ConfigAssetGroup{Name: "USA"}},
		{ConfigAssetGroup: c.ConfigAssetGroup{Name: "ETC"}},
	}

	view, width := renderGroupTabs(groups, 1)
	if got, want := stripansi.Strip(view), " Stockholm  USA  ETC "; got != want {
		t.Fatalf("group tabs = %q, want %q", got, want)
	}
	if got, want := width, len(" Stockholm  USA  ETC "); got != want {
		t.Fatalf("group tab width = %d, want %d", got, want)
	}
}
