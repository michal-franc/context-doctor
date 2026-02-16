package rules

import (
	"testing"
)

// =============================================================================
// Engine construction
// =============================================================================

func TestNewEngine(t *testing.T) {
	t.Run("stores rules", func(t *testing.T) {
		engine := NewEngine([]Rule{
			{Code: "R001"}, {Code: "R002"},
		})
		if len(engine.Rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(engine.Rules))
		}
	})

	t.Run("nil input", func(t *testing.T) {
		engine := NewEngine(nil)
		if engine.Rules != nil {
			t.Error("expected nil rules")
		}
	})
}

// =============================================================================
// Evaluate / EvaluateSecondary
// =============================================================================

func TestEvaluate(t *testing.T) {
	t.Run("runs all rules and returns correct pass/fail", func(t *testing.T) {
		engine := NewEngine([]Rule{
			{Code: "R001", Severity: SeverityError,
				MatchSpec:    MatchSpec{Metric: MetricLineCount, Action: ActionGreaterThan, Value: 100},
				ErrorMessage: "Too many lines"},
			{Code: "R002", Severity: SeverityWarning,
				MatchSpec:    MatchSpec{Action: ActionContains, Value: "TODO"},
				ErrorMessage: "Contains TODO"},
		})
		ctx := makeCtx("This has a TODO item", 50, 5, nil)
		results := engine.Evaluate(ctx)

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].Passed {
			t.Error("R001: expected passed=false (50 is not > 100)")
		}
		if !results[1].Passed {
			t.Error("R002: expected passed=true (content contains TODO)")
		}
	})

	t.Run("empty rules returns empty results", func(t *testing.T) {
		results := NewEngine(nil).Evaluate(makeCtx("content", 10, 2, nil))
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

func TestEvaluateSecondary_SkipsPrimaryOnly(t *testing.T) {
	engine := NewEngine([]Rule{
		{Code: "R001", PrimaryOnly: true,
			MatchSpec: MatchSpec{Metric: MetricLineCount, Action: ActionGreaterThan, Value: 100}},
		{Code: "R002", PrimaryOnly: false,
			MatchSpec: MatchSpec{Action: ActionContains, Value: "hello"}},
	})
	results := engine.EvaluateSecondary(makeCtx("hello world", 50, 5, nil))

	if len(results) != 1 {
		t.Fatalf("expected 1 result (primaryOnly skipped), got %d", len(results))
	}
	if results[0].Rule.Code != "R002" {
		t.Errorf("expected R002, got %s", results[0].Rule.Code)
	}
}

// =============================================================================
// evaluateRule — good-practice vs problem detection semantics
// =============================================================================

func TestEvaluateRule(t *testing.T) {
	engine := NewEngine(nil)

	t.Run("good-practice detected", func(t *testing.T) {
		rule := Rule{Code: "GP001", Category: "good-practice",
			MatchSpec:    MatchSpec{Action: ActionContains, Value: "## build"},
			ErrorMessage: "Has build section"}
		result := engine.evaluateRule(makeCtx("## Build\nrun make", 2, 1, nil), rule)

		if !result.Passed {
			t.Error("expected passed=true when pattern found")
		}
		if result.Message != "Has build section" {
			t.Errorf("expected message 'Has build section', got %q", result.Message)
		}
	})

	t.Run("good-practice not detected", func(t *testing.T) {
		rule := Rule{Code: "GP001", Category: "good-practice",
			MatchSpec:    MatchSpec{Action: ActionContains, Value: "## testing"},
			ErrorMessage: "Has testing section"}
		result := engine.evaluateRule(makeCtx("# Readme\nNo testing.", 2, 0, nil), rule)

		if result.Passed {
			t.Error("expected passed=false when pattern not found")
		}
		if result.Message != "" {
			t.Errorf("expected empty message, got %q", result.Message)
		}
	})

	t.Run("problem detected", func(t *testing.T) {
		rule := Rule{Code: "CD001", Category: "length", Severity: SeverityError,
			MatchSpec:    MatchSpec{Metric: MetricLineCount, Action: ActionGreaterThan, Value: 300},
			ErrorMessage: "File too long"}
		result := engine.evaluateRule(makeCtx("", 500, 0, nil), rule)

		if !result.Passed {
			t.Error("expected passed=true (problem found: 500 > 300)")
		}
		if result.Message != "File too long" {
			t.Errorf("expected 'File too long', got %q", result.Message)
		}
	})

	t.Run("problem not detected", func(t *testing.T) {
		rule := Rule{Code: "CD001", Category: "length", Severity: SeverityError,
			MatchSpec:    MatchSpec{Metric: MetricLineCount, Action: ActionGreaterThan, Value: 300},
			ErrorMessage: "File too long"}
		result := engine.evaluateRule(makeCtx("", 50, 0, nil), rule)

		if result.Passed {
			t.Error("expected passed=false (no problem: 50 is not > 300)")
		}
	})
}

// =============================================================================
// CountInstructions
// =============================================================================

func TestCountInstructions(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		wantMin int
		wantMax int
	}{
		{"imperative verbs", []string{
			"- Always use TypeScript",
			"- Never use var",
			"- Must follow linting rules",
			"- Should prefer const",
		}, 4, 4},
		{"long list items count", []string{
			"- This is a longer list item with enough text",
			"* Another bullet that is long enough to count",
			"1. A numbered item with enough text to count",
		}, 3, 3},
		{"short list items excluded", []string{
			"- Short", "- Tiny", "* Mini",
		}, 0, 0},
		{"headers excluded", []string{
			"# Header", "## Sub Header", "### Sub Sub Header",
		}, 0, 0},
		{"empty lines excluded", []string{
			"", "  ", "",
		}, 0, 0},
		{"mixed content", []string{
			"# Build Instructions", "",
			"- Always run tests before committing",
			"- Use gofmt", "",
			"## Style",
			"Some plain text.",
			"- Prefer explicit error handling over panics",
		}, 2, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CountInstructions(tc.lines)
			if got < tc.wantMin || got > tc.wantMax {
				t.Errorf("got %d, want [%d, %d]", got, tc.wantMin, tc.wantMax)
			}
		})
	}
}

// =============================================================================
// BuildContext
// =============================================================================

func TestBuildContext(t *testing.T) {
	t.Run("populates all fields", func(t *testing.T) {
		content := "# Title\n\nSome content\n- Always do X\n"
		ctx := BuildContext("/path/to/file.md", content)

		if ctx.FilePath != "/path/to/file.md" {
			t.Errorf("FilePath = %q", ctx.FilePath)
		}
		if ctx.Content != content {
			t.Error("Content mismatch")
		}
		// trailing newline produces 5 lines via strings.Split
		if ctx.LineCount != 5 {
			t.Errorf("LineCount = %d, want 5", ctx.LineCount)
		}
		if ctx.Metrics == nil {
			t.Fatal("Metrics not initialized")
		}
		if _, ok := ctx.Metrics["hasProgressiveDisclosure"]; !ok {
			t.Error("missing hasProgressiveDisclosure metric")
		}
		if _, ok := ctx.Metrics["progressiveDisclosureRefs"]; !ok {
			t.Error("missing progressiveDisclosureRefs metric")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		ctx := BuildContext("empty.md", "")
		if ctx.LineCount != 1 { // strings.Split("", "\n") = [""]
			t.Errorf("LineCount = %d, want 1", ctx.LineCount)
		}
		if ctx.InstructionCount != 0 {
			t.Errorf("InstructionCount = %d, want 0", ctx.InstructionCount)
		}
	})
}

// =============================================================================
// Progressive disclosure detection
// =============================================================================

func TestHasProgressiveDisclosure(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"see pattern", "For details see docs/guide.md", true},
		{"refer to pattern", "Please refer to setup.md", true},
		{"read pattern", "First read CONTRIBUTING.md", true},
		{"docs/ path", "- docs/architecture.md - system design", true},
		{"check pattern", "Also check RULES.md for details", true},
		{"no match", "Just a normal line with no references.", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasProgressiveDisclosure(tc.content); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFindProgressiveDisclosureRefs(t *testing.T) {
	t.Run("extracts multiple refs", func(t *testing.T) {
		content := "See docs/setup.md for setup.\nRefer to CONTRIBUTING.md.\nCheck RULES.md.\n- docs/api.md - API ref\n"
		refs := findProgressiveDisclosureRefs(content)
		if len(refs) < 3 {
			t.Errorf("expected at least 3 refs, got %d: %v", len(refs), refs)
		}
	})

	t.Run("no refs", func(t *testing.T) {
		refs := findProgressiveDisclosureRefs("No references here")
		if len(refs) != 0 {
			t.Errorf("expected 0 refs, got %d", len(refs))
		}
	})

	t.Run("preserves parent-dir prefix in docs path", func(t *testing.T) {
		content := "Jira CLI reference at `../docs/jira-cli.md`.\n"
		refs := findProgressiveDisclosureRefs(content)

		found := false
		for _, ref := range refs {
			if ref == "../docs/jira-cli.md" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected '../docs/jira-cli.md' in refs, got %v", refs)
		}
	})

	t.Run("preserves nested parent-dir prefix", func(t *testing.T) {
		content := "See ../../docs/guide.md for details.\n"
		refs := findProgressiveDisclosureRefs(content)

		found := false
		for _, ref := range refs {
			if ref == "../../docs/guide.md" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected '../../docs/guide.md' in refs, got %v", refs)
		}
	})
}

// =============================================================================
// FilterResults
// =============================================================================

func TestFilterResults(t *testing.T) {
	// shared test data
	allResults := []RuleResult{
		{Rule: Rule{Code: "R001", Severity: SeverityError, Category: "length"}, Passed: true},
		{Rule: Rule{Code: "R002", Severity: SeverityWarning, Category: "length"}, Passed: true},
		{Rule: Rule{Code: "R003", Severity: SeverityError, Category: "instructions"}, Passed: true},
		{Rule: Rule{Code: "R004", Severity: SeverityInfo, Category: "length"}, Passed: false},
		{Rule: Rule{Code: "GP001", Category: "good-practice"}, Passed: true},
	}

	tests := []struct {
		name      string
		opts      FilterOptions
		wantCodes []string
	}{
		{"no filters — all pass through",
			FilterOptions{},
			[]string{"R001", "R002", "R003", "R004", "GP001"}},
		{"failures only — keeps Passed=true",
			FilterOptions{FailuresOnly: true},
			[]string{"R001", "R002", "R003", "GP001"}},
		{"filter by severity=error",
			FilterOptions{Severities: []Severity{SeverityError}},
			[]string{"R001", "R003"}},
		{"filter by category=length",
			FilterOptions{Categories: []string{"length"}},
			[]string{"R001", "R002", "R004"}},
		{"hide good-practice",
			FilterOptions{HideGoodPractice: true},
			[]string{"R001", "R002", "R003", "R004"}},
		{"combined severity+category",
			FilterOptions{Severities: []Severity{SeverityError}, Categories: []string{"length"}},
			[]string{"R001"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filtered := FilterResults(allResults, tc.opts)
			got := make([]string, len(filtered))
			for i, r := range filtered {
				got[i] = r.Rule.Code
			}
			if len(got) != len(tc.wantCodes) {
				t.Fatalf("got %v, want %v", got, tc.wantCodes)
			}
			for i, code := range tc.wantCodes {
				if got[i] != code {
					t.Errorf("result[%d] = %s, want %s", i, got[i], code)
				}
			}
		})
	}
}
