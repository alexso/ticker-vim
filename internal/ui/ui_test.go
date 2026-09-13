package ui

import (
	"testing"

	"github.com/acarl005/stripansi"
	c "github.com/alexso/ticker-vim/v5/internal/common"
	"github.com/alexso/ticker-vim/v5/internal/stocksearch"
)

func TestRenderGroupTabs(t *testing.T) {
	t.Parallel()

	groups := []c.AssetGroup{
		{ConfigAssetGroup: c.ConfigAssetGroup{Name: "Stockholm"}},
		{ConfigAssetGroup: c.ConfigAssetGroup{Name: "USA"}},
		{ConfigAssetGroup: c.ConfigAssetGroup{Name: "ETC"}},
	}

	view, width := renderGroupTabs(groups, 1)
	if got, want := stripansi.Strip(view), " 1 Stockholm  2 USA  3 ETC "; got != want {
		t.Fatalf("group tabs = %q, want %q", got, want)
	}
	if got, want := width, len(" 1 Stockholm  2 USA  3 ETC "); got != want {
		t.Fatalf("group tab width = %d, want %d", got, want)
	}
}

func TestPrioritizeCandidatesForActiveGroup(t *testing.T) {
	t.Parallel()
	candidates := []stocksearch.Candidate{
		{Symbol: "BINI", Exchange: "OTC Markets"},
		{Symbol: "BOL.ST", Exchange: "Stockholm"},
		{Symbol: "BOLT", Exchange: "NASDAQ"},
	}

	prioritized := prioritizeCandidates(candidates, "Stockholm")
	if prioritized[0].Symbol != "BOL.ST" {
		t.Fatalf("first candidate = %s, want BOL.ST", prioritized[0].Symbol)
	}
	if candidates[0].Symbol != "BINI" {
		t.Fatal("input candidates were mutated")
	}
}
