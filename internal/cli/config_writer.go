package cli

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// AddSymbolToConfig appends a Yahoo symbol and company name to a group's watchlist.
func AddSymbolToConfig(fs afero.Fs, configPath string, groupName string, symbol string, name string) error {
	data, err := afero.ReadFile(fs, configPath)
	if err != nil {
		return fmt.Errorf("read stock config: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse stock config: %w", err)
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return errors.New("stock config must contain a YAML mapping")
	}

	root := document.Content[0]
	var watchlist *yaml.Node
	if groupName == "default" {
		watchlist = mappingValue(root, "watchlist")
		if watchlist == nil {
			watchlist = appendSequence(root, "watchlist")
		}
	} else {
		groups := mappingValue(root, "groups")
		if groups == nil || groups.Kind != yaml.SequenceNode {
			return fmt.Errorf("group %q was not found in stock config", groupName) //nolint:goerr113
		}
		for _, group := range groups.Content {
			name := mappingValue(group, "name")
			if name != nil && name.Value == groupName {
				watchlist = mappingValue(group, "watchlist")
				if watchlist == nil {
					watchlist = appendSequence(group, "watchlist")
				}

				break
			}
		}
	}

	if watchlist == nil || watchlist.Kind != yaml.SequenceNode {
		return fmt.Errorf("watchlist for group %q is not a YAML list", groupName) //nolint:goerr113
	}
	for _, existing := range watchlist.Content {
		if strings.EqualFold(existing.Value, symbol) {
			return fmt.Errorf("%s is already in %s", symbol, groupName) //nolint:goerr113
		}
	}
	if len(watchlist.Content) > 0 {
		lastSymbol := watchlist.Content[len(watchlist.Content)-1]
		lines := strings.SplitAfter(string(data), "\n")
		if lastSymbol.Line > 0 && lastSymbol.Line <= len(lines) {
			line := lines[lastSymbol.Line-1]
			trimmed := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(trimmed, "- ") {
				indent := line[:len(line)-len(trimmed)]
				insertion := indent + "- " + symbol + symbolComment(name) + "\n"
				output := strings.Join(lines[:lastSymbol.Line], "") + insertion + strings.Join(lines[lastSymbol.Line:], "")

				return writeConfigAtomically(fs, configPath, []byte(output))
			}
		}
	}

	// Empty or unusual YAML lists use the node encoder as a safe fallback.
	watchlist.Content = append(watchlist.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: symbol, LineComment: strings.TrimSpace(strings.Join(strings.Fields(name), " "))})

	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return fmt.Errorf("encode stock config: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("finish stock config: %w", err)
	}

	return writeConfigAtomically(fs, configPath, encoded.Bytes())
}

func symbolComment(name string) string {
	name = strings.TrimSpace(strings.Join(strings.Fields(name), " "))
	if name == "" {
		return ""
	}

	return " # " + name
}

// UpdateSymbolComment changes the display-name comment for one watchlist symbol.
func UpdateSymbolComment(fs afero.Fs, configPath string, groupName string, symbol string, name string) error {
	data, err := afero.ReadFile(fs, configPath)
	if err != nil {
		return fmt.Errorf("read stock config: %w", err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse stock config: %w", err)
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return errors.New("stock config must contain a YAML mapping")
	}
	watchlist := findWatchlist(document.Content[0], groupName)
	if watchlist == nil || watchlist.Kind != yaml.SequenceNode {
		return fmt.Errorf("watchlist for group %q is not a YAML list", groupName) //nolint:goerr113
	}
	for _, existing := range watchlist.Content {
		if !strings.EqualFold(existing.Value, symbol) {
			continue
		}
		lines := strings.SplitAfter(string(data), "\n")
		if existing.Line <= 0 || existing.Line > len(lines) {
			break
		}
		line := lines[existing.Line-1]
		newline := ""
		if strings.HasSuffix(line, "\n") {
			newline = "\n"
		}
		trimmed := strings.TrimLeft(line, " \t")
		indent := line[:len(line)-len(trimmed)]
		replacement := indent + "- " + existing.Value + symbolComment(name) + newline
		lines[existing.Line-1] = replacement

		return writeConfigAtomically(fs, configPath, []byte(strings.Join(lines, "")))
	}

	return fmt.Errorf("%s is not in the %s watchlist", symbol, groupName) //nolint:goerr113
}

// RemoveSymbolFromConfig removes a symbol from a group's watchlist while retaining YAML comments.
func RemoveSymbolFromConfig(fs afero.Fs, configPath string, groupName string, symbol string) error {
	data, err := afero.ReadFile(fs, configPath)
	if err != nil {
		return fmt.Errorf("read stock config: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse stock config: %w", err)
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return errors.New("stock config must contain a YAML mapping")
	}

	watchlist := findWatchlist(document.Content[0], groupName)
	if watchlist == nil || watchlist.Kind != yaml.SequenceNode {
		return fmt.Errorf("watchlist for group %q is not a YAML list", groupName) //nolint:goerr113
	}
	removeIndex := -1
	for index, existing := range watchlist.Content {
		if strings.EqualFold(existing.Value, symbol) {
			removeIndex = index

			break
		}
	}
	if removeIndex < 0 {
		return fmt.Errorf("%s is not in the %s watchlist", symbol, groupName) //nolint:goerr113
	}

	node := watchlist.Content[removeIndex]
	if len(watchlist.Content) > 1 && node.Line > 0 {
		lines := strings.SplitAfter(string(data), "\n")
		if node.Line <= len(lines) {
			output := strings.Join(lines[:node.Line-1], "") + strings.Join(lines[node.Line:], "")

			return writeConfigAtomically(fs, configPath, []byte(output))
		}
	}

	// Encoding is needed when the last item is removed so watchlist remains an explicit empty list.
	watchlist.Content = append(watchlist.Content[:removeIndex], watchlist.Content[removeIndex+1:]...)
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return fmt.Errorf("encode stock config: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("finish stock config: %w", err)
	}

	return writeConfigAtomically(fs, configPath, encoded.Bytes())
}

func findWatchlist(root *yaml.Node, groupName string) *yaml.Node {
	if groupName == "default" {
		return mappingValue(root, "watchlist")
	}
	groups := mappingValue(root, "groups")
	if groups == nil || groups.Kind != yaml.SequenceNode {
		return nil
	}
	for _, group := range groups.Content {
		name := mappingValue(group, "name")
		if name != nil && name.Value == groupName {
			return mappingValue(group, "watchlist")
		}
	}

	return nil
}

func writeConfigAtomically(fs afero.Fs, configPath string, data []byte) error {
	info, err := fs.Stat(configPath)
	if err != nil {
		return fmt.Errorf("inspect stock config: %w", err)
	}
	temporary, err := afero.TempFile(fs, filepath.Dir(configPath), ".ticker-vim-*.yaml")
	if err != nil {
		return fmt.Errorf("create temporary stock config: %w", err)
	}
	temporaryPath := temporary.Name()
	defer fs.Remove(temporaryPath) //nolint:errcheck

	if _, err := temporary.Write(data); err != nil {
		temporary.Close() //nolint:errcheck

		return fmt.Errorf("write temporary stock config: %w", err)
	}
	if err := fs.Chmod(temporaryPath, info.Mode().Perm()); err != nil {
		temporary.Close() //nolint:errcheck

		return fmt.Errorf("preserve stock config permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary stock config: %w", err)
	}
	if err := fs.Rename(temporaryPath, configPath); err != nil {
		return fmt.Errorf("replace stock config: %w", err)
	}

	return nil
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}

	return nil
}

func appendSequence(mapping *yaml.Node, key string) *yaml.Node {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valueNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	mapping.Content = append(mapping.Content, keyNode, valueNode)

	return valueNode
}
