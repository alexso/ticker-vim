package sorter

import (
	"cmp"
	"slices"
	"strings"

	c "github.com/alexso/ticker-vim/v5/internal/common"
)

// Sorter represents a function that sorts quotes
type Sorter func([]*c.Asset) []*c.Asset

// NewSorter creates a sorting function
func NewSorter(sort string, firstLine ...string) Sorter {
	if sort == "alpha" {
		if len(firstLine) > 0 && firstLine[0] == "name" {
			return sortByAlphaName
		}

		return sortByAlphaSymbol
	}

	var sortDict = map[string]Sorter{
		"value": sortByValue,
		"user":  sortByUser,
	}
	if sorter, ok := sortDict[sort]; ok {
		return sorter
	}

	return sortByChange
}

func sortByUser(assets []*c.Asset) []*c.Asset {

	assetCount := len(assets)

	if assetCount <= 0 {
		return assets
	}

	slices.SortStableFunc(assets, func(a, b *c.Asset) int {
		return cmp.Compare(a.Meta.OrderIndex, b.Meta.OrderIndex)
	})

	return assets

}

func sortByAlphaSymbol(assetsIn []*c.Asset) []*c.Asset {
	return sortByAlpha(assetsIn, func(asset *c.Asset) string { return asset.Symbol })
}

func sortByAlphaName(assetsIn []*c.Asset) []*c.Asset {
	return sortByAlpha(assetsIn, func(asset *c.Asset) string {
		if strings.TrimSpace(asset.Name) == "" {
			return asset.Symbol
		}

		return asset.Name
	})
}

func sortByAlpha(assetsIn []*c.Asset, value func(*c.Asset) string) []*c.Asset {

	assetCount := len(assetsIn)

	if assetCount <= 0 {
		return assetsIn
	}

	assets := make([]*c.Asset, assetCount)
	copy(assets, assetsIn)

	slices.SortStableFunc(assets, func(a, b *c.Asset) int {
		comparison := strings.Compare(strings.ToLower(value(a)), strings.ToLower(value(b)))
		if comparison != 0 {
			return comparison
		}

		return cmp.Compare(a.Symbol, b.Symbol)
	})

	return assets
}

func sortByValue(assetsIn []*c.Asset) []*c.Asset {

	assetCount := len(assetsIn)

	if assetCount <= 0 {
		return assetsIn
	}

	assets := make([]*c.Asset, assetCount)
	copy(assets, assetsIn)

	activeAssets, inactiveAssets := splitActiveAssets(assets)

	slices.SortStableFunc(inactiveAssets, func(a, b *c.Asset) int {
		return cmp.Compare(b.Position.Value, a.Position.Value)
	})

	slices.SortStableFunc(activeAssets, func(a, b *c.Asset) int {
		return cmp.Compare(b.Position.Value, a.Position.Value)
	})

	return append(activeAssets, inactiveAssets...)
}

func sortByChange(assetsIn []*c.Asset) []*c.Asset {

	assetCount := len(assetsIn)

	if assetCount <= 0 {
		return assetsIn
	}

	assets := make([]*c.Asset, assetCount)
	copy(assets, assetsIn)

	activeAssets, inactiveAssets := splitActiveAssets(assets)

	slices.SortStableFunc(activeAssets, func(a, b *c.Asset) int {
		return cmp.Compare(b.QuotePrice.ChangePercent, a.QuotePrice.ChangePercent)
	})

	slices.SortStableFunc(inactiveAssets, func(a, b *c.Asset) int {
		return cmp.Compare(b.QuotePrice.ChangePercent, a.QuotePrice.ChangePercent)
	})

	return append(activeAssets, inactiveAssets...)

}

func splitActiveAssets(assets []*c.Asset) ([]*c.Asset, []*c.Asset) {

	activeAssets := make([]*c.Asset, 0)
	inactiveAssets := make([]*c.Asset, 0)

	for _, asset := range assets {
		if asset.Exchange.IsActive {
			activeAssets = append(activeAssets, asset)
		} else {
			inactiveAssets = append(inactiveAssets, asset)
		}
	}

	return activeAssets, inactiveAssets
}
