package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// LoadBuiltinRules
// =============================================================================

func TestLoadBuiltinRules(t *testing.T) {
	rules, err := LoadBuiltinRules()
	if err != nil {
		t.Fatalf("failed to load builtin rules: %v", err)
	}

	t.Run("has expected count", func(t *testing.T) {
		if len(rules) != 36 {
			t.Errorf("expected 36 rules, got %d", len(rules))
		}
	})

	t.Run("all rules have required fields", func(t *testing.T) {
		for _, rule := range rules {
			if rule.Code == "" {
				t.Error("rule missing code")
			}
			if rule.Severity == "" {
				t.Errorf("rule %s missing severity", rule.Code)
			}
			if rule.ErrorMessage == "" {
				t.Errorf("rule %s missing errorMessage", rule.Code)
			}
		}
	})

	t.Run("all rules have dimension set", func(t *testing.T) {
		for _, rule := range rules {
			if rule.Dimension == "" {
				t.Errorf("rule %s missing dimension", rule.Code)
			}
		}
	})
}

// =============================================================================
// LoadRulesFromFile
// =============================================================================

// writeRulesFile is a test helper that writes content to a temp file and returns its path.
func writeRulesFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadRulesFromFile(t *testing.T) {
	validYAML := `version: "1"
rules:
  - code: "T001"
    description: "Test rule"
    severity: warning
    matchSpec:
      metric: lineCount
      action: greaterThan
      value: 50
    errorMessage: "File is long"
`

	t.Run("valid .yaml", func(t *testing.T) {
		rules, err := LoadRulesFromFile(writeRulesFile(t, "test.yaml", validYAML))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 1 || rules[0].Code != "T001" {
			t.Errorf("got %d rules, code=%v", len(rules), rules)
		}
	})

	t.Run("valid .yml", func(t *testing.T) {
		rules, err := LoadRulesFromFile(writeRulesFile(t, "test.yml", validYAML))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 1 {
			t.Errorf("expected 1 rule, got %d", len(rules))
		}
	})

	t.Run("valid .json", func(t *testing.T) {
		json := `{"rules": [{"code": "T003", "description": "JSON rule", "severity": "error", "matchSpec": {"action": "contains", "value": "test"}, "errorMessage": "err"}]}`
		rules, err := LoadRulesFromFile(writeRulesFile(t, "test.json", json))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 1 || rules[0].Code != "T003" {
			t.Errorf("got %d rules, code=%v", len(rules), rules)
		}
	})

	t.Run("malformed YAML returns error", func(t *testing.T) {
		if _, err := LoadRulesFromFile(writeRulesFile(t, "bad.yaml", "{{{{not yaml")); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		if _, err := LoadRulesFromFile("/nonexistent/path/rules.yaml"); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		if _, err := LoadRulesFromFile(writeRulesFile(t, "rules.txt", "text")); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("empty file returns zero rules", func(t *testing.T) {
		rules, err := LoadRulesFromFile(writeRulesFile(t, "empty.yaml", ""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 0 {
			t.Errorf("expected 0 rules, got %d", len(rules))
		}
	})
}

// =============================================================================
// DiscoverCustomRules
// =============================================================================

// writeCustomRule creates a rules file in the given directory and returns the rule count.
func writeCustomRule(t *testing.T, dir, filename string) {
	t.Helper()
	content := `rules:
  - code: "C001"
    description: "Custom"
    severity: warning
    matchSpec:
      action: contains
      value: "custom"
    errorMessage: "Custom check"
`
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverCustomRules(t *testing.T) {
	t.Run("finds *_rules.yaml in .context-doctor/", func(t *testing.T) {
		tmpDir := t.TempDir()
		cdDir := filepath.Join(tmpDir, ".context-doctor")
		if err := os.MkdirAll(cdDir, 0755); err != nil {
			t.Fatal(err)
		}
		writeCustomRule(t, cdDir, "my_rules.yaml")

		rules, err := DiscoverCustomRules(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 1 {
			t.Errorf("expected 1 rule, got %d", len(rules))
		}
	})

	t.Run("finds rules.yaml in dir root", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeCustomRule(t, tmpDir, "rules.yaml")

		rules, err := DiscoverCustomRules(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 1 {
			t.Errorf("expected 1 rule, got %d", len(rules))
		}
	})

	t.Run("finds *_rules.yml suffix", func(t *testing.T) {
		tmpDir := t.TempDir()
		cdDir := filepath.Join(tmpDir, ".context-doctor")
		if err := os.MkdirAll(cdDir, 0755); err != nil {
			t.Fatal(err)
		}
		writeCustomRule(t, cdDir, "custom_rules.yml")

		rules, err := DiscoverCustomRules(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 1 {
			t.Errorf("expected 1 rule, got %d", len(rules))
		}
	})

	t.Run("ignores non-rule files", func(t *testing.T) {
		tmpDir := t.TempDir()
		cdDir := filepath.Join(tmpDir, ".context-doctor")
		if err := os.MkdirAll(cdDir, 0755); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(cdDir, "README.md"), []byte("# readme"), 0644)
		_ = os.WriteFile(filepath.Join(cdDir, "config.yaml"), []byte("key: value"), 0644)

		rules, err := DiscoverCustomRules(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 0 {
			t.Errorf("expected 0 rules, got %d", len(rules))
		}
	})

	t.Run("empty dir returns zero rules", func(t *testing.T) {
		rules, err := DiscoverCustomRules(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 0 {
			t.Errorf("expected 0 rules, got %d", len(rules))
		}
	})

	t.Run("nonexistent dir returns zero rules", func(t *testing.T) {
		rules, err := DiscoverCustomRules("/nonexistent/path/12345")
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 0 {
			t.Errorf("expected 0 rules, got %d", len(rules))
		}
	})
}

// =============================================================================
// LoadAllRules
// =============================================================================

func TestLoadAllRules(t *testing.T) {
	t.Run("builtin only", func(t *testing.T) {
		rules, err := LoadAllRules("", true)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) == 0 {
			t.Error("expected builtin rules")
		}
	})

	t.Run("no builtin, no custom dir", func(t *testing.T) {
		rules, err := LoadAllRules("", false)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 0 {
			t.Errorf("expected 0 rules, got %d", len(rules))
		}
	})

	t.Run("builtin + custom combined", func(t *testing.T) {
		tmpDir := t.TempDir()
		cdDir := filepath.Join(tmpDir, ".context-doctor")
		if err := os.MkdirAll(cdDir, 0755); err != nil {
			t.Fatal(err)
		}
		writeCustomRule(t, cdDir, "my_rules.yaml")

		rules, err := LoadAllRules(tmpDir, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 37 { // 36 builtin + 1 custom
			t.Errorf("expected 37 rules, got %d", len(rules))
		}
	})

	t.Run("custom only (no builtin)", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeCustomRule(t, tmpDir, "rules.yaml")

		rules, err := LoadAllRules(tmpDir, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 1 {
			t.Errorf("expected 1 rule, got %d", len(rules))
		}
	})
}
