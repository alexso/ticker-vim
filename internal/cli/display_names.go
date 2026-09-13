package cli

import (
	"fmt"
	"strings"

	c "github.com/alexso/ticker-vim/v5/internal/common"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// ApplyDisplayNames loads optional inline watchlist comments as UI display names.
func ApplyDisplayNames(fs afero.Fs, configPath string, groups []c.AssetGroup) error {
	data, err := afero.ReadFile(fs, configPath)
	if err != nil {
		return fmt.Errorf("read stock display names: %w", err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse stock display names: %w", err)
	}
	if len(document.Content) == 0 {
		return nil
	}
	root := document.Content[0]
	for index := range groups {
		watchlist := findWatchlist(root, groups[index].Name)
		groups[index].DisplayNames = displayNamesFromWatchlist(watchlist)
	}

	return nil
}

func displayNamesFromWatchlist(watchlist *yaml.Node) map[string]string {
	names := make(map[string]string)
	if watchlist == nil || watchlist.Kind != yaml.SequenceNode {
		return names
	}
	for _, symbol := range watchlist.Content {
		name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(symbol.LineComment), "#"))
		if name != "" {
			names[strings.ToLower(symbol.Value)] = name
		}
	}

	return names
}
