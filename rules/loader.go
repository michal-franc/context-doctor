package rules

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed builtin.yaml
var builtinRulesYAML []byte

// LoadBuiltinRules loads the embedded default rules
func LoadBuiltinRules() ([]Rule, error) {
	var rulesFile RulesFile
	if err := yaml.Unmarshal(builtinRulesYAML, &rulesFile); err != nil {
		return nil, fmt.Errorf("failed to parse builtin rules: %w", err)
	}
	return rulesFile.Rules, nil
}

// LoadRulesFromFile loads rules from a YAML or JSON file
func LoadRulesFromFile(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	var rulesFile RulesFile
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &rulesFile); err != nil {
			return nil, fmt.Errorf("failed to parse YAML rules: %w", err)
		}
	case ".json":
		if err := yaml.Unmarshal(data, &rulesFile); err != nil {
			return nil, fmt.Errorf("failed to parse JSON rules: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	return rulesFile.Rules, nil
}

// DiscoverCustomRules finds and loads custom rules from a directory
func DiscoverCustomRules(dir string) ([]Rule, error) {
	var allRules []Rule

	// Check default locations
	checkDirs := []string{
		filepath.Join(dir, ".context-doctor"),
		dir,
	}

	for _, checkDir := range checkDirs {
		info, err := os.Stat(checkDir)
		if err != nil || !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(checkDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			// Match patterns like *_rules.yaml, *_rules.yml, rules.yaml
			if strings.HasSuffix(name, "_rules.yaml") ||
				strings.HasSuffix(name, "_rules.yml") ||
				name == "rules.yaml" ||
				name == "rules.yml" {

				path := filepath.Join(checkDir, name)
				rules, err := LoadRulesFromFile(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", path, err)
					continue
				}
				allRules = append(allRules, rules...)
			}
		}
	}

	return allRules, nil
}

// LoadAllRules loads builtin rules and discovers custom rules
func LoadAllRules(customDir string, includeBuiltin bool) ([]Rule, error) {
	var allRules []Rule

	if includeBuiltin {
		builtin, err := LoadBuiltinRules()
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, builtin...)
	}

	if customDir != "" {
		custom, err := DiscoverCustomRules(customDir)
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, custom...)
	}

	return allRules, nil
}
