package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// --- LoadBuiltinRules ---

func TestLoadBuiltinRules_Loads(t *testing.T) {
	rules, err := LoadBuiltinRules()
	if err != nil {
		t.Fatalf("failed to load builtin rules: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("expected at least 1 builtin rule")
	}
}

func TestLoadBuiltinRules_HasExpectedCount(t *testing.T) {
	rules, err := LoadBuiltinRules()
	if err != nil {
		t.Fatalf("failed to load builtin rules: %v", err)
	}
	// We know there are 25 rules in builtin.yaml
	if len(rules) != 25 {
		t.Errorf("expected 25 builtin rules, got %d", len(rules))
	}
}

func TestLoadBuiltinRules_HasRequiredFields(t *testing.T) {
	rules, err := LoadBuiltinRules()
	if err != nil {
		t.Fatalf("failed to load builtin rules: %v", err)
	}
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
}

// --- LoadRulesFromFile ---

func TestLoadRulesFromFile_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "test_rules.yaml")
	content := `version: "1"
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
	if err := os.WriteFile(rulesFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadRulesFromFile(rulesFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Code != "T001" {
		t.Errorf("expected code T001, got %s", rules[0].Code)
	}
	if rules[0].Severity != SeverityWarning {
		t.Errorf("expected severity warning, got %s", rules[0].Severity)
	}
}

func TestLoadRulesFromFile_ValidYML(t *testing.T) {
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "test_rules.yml")
	content := `rules:
  - code: "T002"
    description: "YML rule"
    severity: info
    matchSpec:
      action: contains
      value: "test"
    errorMessage: "Contains test"
`
	if err := os.WriteFile(rulesFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadRulesFromFile(rulesFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Code != "T002" {
		t.Errorf("expected code T002, got %s", rules[0].Code)
	}
}

func TestLoadRulesFromFile_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	// The loader uses yaml.Unmarshal for JSON too (YAML is a superset of JSON)
	rulesFile := filepath.Join(tmpDir, "test_rules.json")
	content := `{"rules": [{"code": "T003", "description": "JSON rule", "severity": "error", "matchSpec": {"action": "contains", "value": "test"}, "errorMessage": "Test error"}]}`
	if err := os.WriteFile(rulesFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadRulesFromFile(rulesFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Code != "T003" {
		t.Errorf("expected code T003, got %s", rules[0].Code)
	}
}

func TestLoadRulesFromFile_MalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(rulesFile, []byte("{{{{not yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadRulesFromFile(rulesFile)
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}

func TestLoadRulesFromFile_MissingFile(t *testing.T) {
	_, err := LoadRulesFromFile("/nonexistent/path/rules.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadRulesFromFile_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "rules.txt")
	if err := os.WriteFile(rulesFile, []byte("some text"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadRulesFromFile(rulesFile)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestLoadRulesFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(rulesFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadRulesFromFile(rulesFile)
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules for empty file, got %d", len(rules))
	}
}

// --- DiscoverCustomRules ---

func TestDiscoverCustomRules_FindsRulesInContextDoctorDir(t *testing.T) {
	tmpDir := t.TempDir()
	cdDir := filepath.Join(tmpDir, ".context-doctor")
	if err := os.MkdirAll(cdDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `rules:
  - code: "C001"
    description: "Custom rule"
    severity: warning
    matchSpec:
      action: contains
      value: "custom"
    errorMessage: "Custom check"
`
	if err := os.WriteFile(filepath.Join(cdDir, "my_rules.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := DiscoverCustomRules(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 custom rule, got %d", len(rules))
	}
	if rules[0].Code != "C001" {
		t.Errorf("expected code C001, got %s", rules[0].Code)
	}
}

func TestDiscoverCustomRules_FindsRulesYamlInDir(t *testing.T) {
	tmpDir := t.TempDir()
	content := `rules:
  - code: "C002"
    description: "Direct rule"
    severity: info
    matchSpec:
      action: contains
      value: "direct"
    errorMessage: "Direct check"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "rules.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := DiscoverCustomRules(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Code != "C002" {
		t.Errorf("expected code C002, got %s", rules[0].Code)
	}
}

func TestDiscoverCustomRules_YmlSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	cdDir := filepath.Join(tmpDir, ".context-doctor")
	if err := os.MkdirAll(cdDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `rules:
  - code: "C003"
    description: "YML rule"
    severity: info
    matchSpec:
      action: contains
      value: "yml"
    errorMessage: "YML check"
`
	if err := os.WriteFile(filepath.Join(cdDir, "custom_rules.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := DiscoverCustomRules(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestDiscoverCustomRules_IgnoresNonRuleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cdDir := filepath.Join(tmpDir, ".context-doctor")
	if err := os.MkdirAll(cdDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Files that don't match the rule naming patterns
	if err := os.WriteFile(filepath.Join(cdDir, "README.md"), []byte("# readme"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cdDir, "config.yaml"), []byte("key: value"), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := DiscoverCustomRules(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules (non-rule files), got %d", len(rules))
	}
}

func TestDiscoverCustomRules_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	rules, err := DiscoverCustomRules(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestDiscoverCustomRules_NonexistentDir(t *testing.T) {
	rules, err := DiscoverCustomRules("/nonexistent/path/12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

// --- LoadAllRules ---

func TestLoadAllRules_BuiltinOnly(t *testing.T) {
	rules, err := LoadAllRules("", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) == 0 {
		t.Error("expected builtin rules to be loaded")
	}
}

func TestLoadAllRules_NoBuiltin(t *testing.T) {
	rules, err := LoadAllRules("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules with no builtin and no custom dir, got %d", len(rules))
	}
}

func TestLoadAllRules_WithCustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	cdDir := filepath.Join(tmpDir, ".context-doctor")
	if err := os.MkdirAll(cdDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `rules:
  - code: "C010"
    description: "Custom"
    severity: info
    matchSpec:
      action: contains
      value: "custom"
    errorMessage: "Custom found"
`
	if err := os.WriteFile(filepath.Join(cdDir, "my_rules.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadAllRules(tmpDir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have builtin + 1 custom
	builtinCount := 25
	if len(rules) != builtinCount+1 {
		t.Errorf("expected %d rules (builtin+custom), got %d", builtinCount+1, len(rules))
	}
}

func TestLoadAllRules_CustomOnly(t *testing.T) {
	tmpDir := t.TempDir()
	content := `rules:
  - code: "C011"
    description: "Custom only"
    severity: warning
    matchSpec:
      action: contains
      value: "only"
    errorMessage: "Only custom"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "rules.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadAllRules(tmpDir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("expected 1 custom rule, got %d", len(rules))
	}
}
