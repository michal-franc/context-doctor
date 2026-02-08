package rules

import (
	"testing"
)

// --- NewEngine ---

func TestNewEngine(t *testing.T) {
	rules := []Rule{
		{Code: "R001", Description: "test rule"},
		{Code: "R002", Description: "another rule"},
	}
	engine := NewEngine(rules)
	if len(engine.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(engine.Rules))
	}
}

func TestNewEngine_Empty(t *testing.T) {
	engine := NewEngine(nil)
	if engine.Rules != nil {
		t.Errorf("expected nil rules, got %v", engine.Rules)
	}
}

// --- Evaluate ---

func TestEvaluate_RunsAllRules(t *testing.T) {
	rules := []Rule{
		{
			Code:     "R001",
			Severity: SeverityError,
			MatchSpec: MatchSpec{
				Metric: MetricLineCount,
				Action: ActionGreaterThan,
				Value:  100,
			},
			ErrorMessage: "Too many lines",
		},
		{
			Code:     "R002",
			Severity: SeverityWarning,
			MatchSpec: MatchSpec{
				Action: ActionContains,
				Value:  "TODO",
			},
			ErrorMessage: "Contains TODO",
		},
	}

	ctx := makeCtx("This has a TODO item", 50, 5, nil)
	engine := NewEngine(rules)
	results := engine.Evaluate(ctx)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// R001: 50 > 100 is false, so problem NOT detected
	if results[0].Passed {
		t.Error("R001: expected passed=false (50 is not > 100)")
	}

	// R002: content contains "TODO", so problem detected
	if !results[1].Passed {
		t.Error("R002: expected passed=true (content contains TODO)")
	}
}

func TestEvaluate_EmptyRules(t *testing.T) {
	engine := NewEngine(nil)
	ctx := makeCtx("some content", 10, 2, nil)
	results := engine.Evaluate(ctx)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil rules, got %d", len(results))
	}
}

// --- EvaluateSecondary ---

func TestEvaluateSecondary_SkipsPrimaryOnly(t *testing.T) {
	rules := []Rule{
		{
			Code:        "R001",
			PrimaryOnly: true,
			MatchSpec: MatchSpec{
				Metric: MetricLineCount,
				Action: ActionGreaterThan,
				Value:  100,
			},
			ErrorMessage: "Primary only rule",
		},
		{
			Code:        "R002",
			PrimaryOnly: false,
			MatchSpec: MatchSpec{
				Action: ActionContains,
				Value:  "hello",
			},
			ErrorMessage: "Secondary rule",
		},
	}

	ctx := makeCtx("hello world", 50, 5, nil)
	engine := NewEngine(rules)
	results := engine.EvaluateSecondary(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result (primaryOnly skipped), got %d", len(results))
	}
	if results[0].Rule.Code != "R002" {
		t.Errorf("expected R002, got %s", results[0].Rule.Code)
	}
}

// --- evaluateRule ---

func TestEvaluateRule_GoodPracticeDetected(t *testing.T) {
	rule := Rule{
		Code:     "GP001",
		Category: "good-practice",
		MatchSpec: MatchSpec{
			Action: ActionContains,
			Value:  "## build",
		},
		ErrorMessage: "Has build section",
	}

	ctx := makeCtx("## Build\nrun make", 2, 1, nil)
	engine := NewEngine(nil)
	result := engine.evaluateRule(ctx, rule)

	if !result.Passed {
		t.Error("expected good-practice to pass when pattern found")
	}
	if result.Message != "Has build section" {
		t.Errorf("expected message 'Has build section', got %q", result.Message)
	}
}

func TestEvaluateRule_GoodPracticeNotDetected(t *testing.T) {
	rule := Rule{
		Code:     "GP001",
		Category: "good-practice",
		MatchSpec: MatchSpec{
			Action: ActionContains,
			Value:  "## testing",
		},
		ErrorMessage: "Has testing section",
	}

	ctx := makeCtx("# Readme\nNo testing section here.", 2, 0, nil)
	engine := NewEngine(nil)
	result := engine.evaluateRule(ctx, rule)

	if result.Passed {
		t.Error("expected good-practice not to pass when pattern not found")
	}
	if result.Message != "" {
		t.Errorf("expected empty message for non-passing good practice, got %q", result.Message)
	}
}

func TestEvaluateRule_ProblemDetected(t *testing.T) {
	rule := Rule{
		Code:     "CD001",
		Category: "length",
		Severity: SeverityError,
		MatchSpec: MatchSpec{
			Metric: MetricLineCount,
			Action: ActionGreaterThan,
			Value:  300,
		},
		ErrorMessage: "File too long",
	}

	ctx := makeCtx("", 500, 0, nil)
	engine := NewEngine(nil)
	result := engine.evaluateRule(ctx, rule)

	if !result.Passed {
		t.Error("expected problem to be detected (500 > 300)")
	}
	if result.Message != "File too long" {
		t.Errorf("expected 'File too long', got %q", result.Message)
	}
}

func TestEvaluateRule_ProblemNotDetected(t *testing.T) {
	rule := Rule{
		Code:     "CD001",
		Category: "length",
		Severity: SeverityError,
		MatchSpec: MatchSpec{
			Metric: MetricLineCount,
			Action: ActionGreaterThan,
			Value:  300,
		},
		ErrorMessage: "File too long",
	}

	ctx := makeCtx("", 50, 0, nil)
	engine := NewEngine(nil)
	result := engine.evaluateRule(ctx, rule)

	if result.Passed {
		t.Error("expected problem NOT detected (50 is not > 300)")
	}
}

// --- CountInstructions ---

func TestCountInstructions_ImperativeVerbs(t *testing.T) {
	lines := []string{
		"- Always use TypeScript",
		"- Never use var",
		"- Must follow linting rules",
		"- Should prefer const",
	}
	count := CountInstructions(lines)
	if count != 4 {
		t.Errorf("expected 4 instructions, got %d", count)
	}
}

func TestCountInstructions_ListItems(t *testing.T) {
	lines := []string{
		"- This is a longer list item with enough text",
		"* Another bullet that is long enough to count",
		"1. A numbered item with enough text to count",
	}
	count := CountInstructions(lines)
	if count != 3 {
		t.Errorf("expected 3 instructions, got %d", count)
	}
}

func TestCountInstructions_ShortListItemsExcluded(t *testing.T) {
	lines := []string{
		"- Short",
		"- Tiny",
		"* Mini",
	}
	count := CountInstructions(lines)
	if count != 0 {
		t.Errorf("expected 0 instructions for short list items, got %d", count)
	}
}

func TestCountInstructions_HeadersExcluded(t *testing.T) {
	lines := []string{
		"# Header",
		"## Sub Header",
		"### Sub Sub Header",
	}
	count := CountInstructions(lines)
	if count != 0 {
		t.Errorf("expected 0 instructions for headers, got %d", count)
	}
}

func TestCountInstructions_EmptyLinesExcluded(t *testing.T) {
	lines := []string{
		"",
		"  ",
		"",
	}
	count := CountInstructions(lines)
	if count != 0 {
		t.Errorf("expected 0 instructions for empty lines, got %d", count)
	}
}

func TestCountInstructions_MixedContent(t *testing.T) {
	lines := []string{
		"# Build Instructions",
		"",
		"- Always run tests before committing",
		"- Use gofmt",
		"",
		"## Style",
		"Some plain text that's not an instruction.",
		"- Prefer explicit error handling over panics",
	}
	count := CountInstructions(lines)
	// "Always run tests before committing" - imperative verb match
	// "Use gofmt" - imperative verb match (but short list item < 10 chars for trimmed? Let's check)
	// "Prefer explicit error handling over panics" - imperative verb match
	if count < 2 {
		t.Errorf("expected at least 2 instructions, got %d", count)
	}
}

// --- BuildContext ---

func TestBuildContext(t *testing.T) {
	content := "# Title\n\nSome content\n- Always do X\n"
	ctx := BuildContext("/path/to/file.md", content)

	if ctx.FilePath != "/path/to/file.md" {
		t.Errorf("expected FilePath '/path/to/file.md', got %q", ctx.FilePath)
	}
	if ctx.Content != content {
		t.Error("expected Content to match input")
	}
	// "# Title\n\nSome content\n- Always do X\n" splits into 5 lines (trailing newline)
	if ctx.LineCount != 5 {
		t.Errorf("expected 5 lines, got %d", ctx.LineCount)
	}
	if ctx.Metrics == nil {
		t.Fatal("expected Metrics to be initialized")
	}
	if _, ok := ctx.Metrics["hasProgressiveDisclosure"]; !ok {
		t.Error("expected hasProgressiveDisclosure metric")
	}
	if _, ok := ctx.Metrics["progressiveDisclosureRefs"]; !ok {
		t.Error("expected progressiveDisclosureRefs metric")
	}
}

func TestBuildContext_EmptyContent(t *testing.T) {
	ctx := BuildContext("empty.md", "")
	if ctx.LineCount != 1 {
		// strings.Split("", "\n") gives [""]
		t.Errorf("expected 1 line for empty content, got %d", ctx.LineCount)
	}
	if ctx.InstructionCount != 0 {
		t.Errorf("expected 0 instructions, got %d", ctx.InstructionCount)
	}
}

// --- hasProgressiveDisclosure ---

func TestHasProgressiveDisclosure_SeePattern(t *testing.T) {
	if !hasProgressiveDisclosure("For details see docs/guide.md") {
		t.Error("expected 'see X.md' to be detected")
	}
}

func TestHasProgressiveDisclosure_ReferToPattern(t *testing.T) {
	if !hasProgressiveDisclosure("Please refer to setup.md") {
		t.Error("expected 'refer to X.md' to be detected")
	}
}

func TestHasProgressiveDisclosure_ReadPattern(t *testing.T) {
	if !hasProgressiveDisclosure("First read CONTRIBUTING.md") {
		t.Error("expected 'read X.md' to be detected")
	}
}

func TestHasProgressiveDisclosure_DocsPath(t *testing.T) {
	if !hasProgressiveDisclosure("- docs/architecture.md - system design") {
		t.Error("expected 'docs/X.md' to be detected")
	}
}

func TestHasProgressiveDisclosure_CheckPattern(t *testing.T) {
	if !hasProgressiveDisclosure("Also check RULES.md for details") {
		t.Error("expected 'check X.md' to be detected")
	}
}

func TestHasProgressiveDisclosure_NoMatch(t *testing.T) {
	if hasProgressiveDisclosure("Just a normal line with no references.") {
		t.Error("expected no progressive disclosure detected")
	}
}

// --- findProgressiveDisclosureRefs ---

func TestFindProgressiveDisclosureRefs_MultiplePatterns(t *testing.T) {
	content := `# Guide
See docs/setup.md for setup.
Refer to CONTRIBUTING.md for guidelines.
Check RULES.md for rules.
- docs/api.md - API reference
`
	refs := findProgressiveDisclosureRefs(content)
	if len(refs) < 3 {
		t.Errorf("expected at least 3 refs, got %d: %v", len(refs), refs)
	}
}

func TestFindProgressiveDisclosureRefs_NoRefs(t *testing.T) {
	refs := findProgressiveDisclosureRefs("No references here")
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

// --- FilterResults ---

func TestFilterResults_FailuresOnly(t *testing.T) {
	results := []RuleResult{
		{Rule: Rule{Code: "R001", Severity: SeverityError}, Passed: true},
		{Rule: Rule{Code: "R002", Severity: SeverityWarning}, Passed: false},
	}
	filtered := FilterResults(results, FilterOptions{FailuresOnly: true})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
	if filtered[0].Rule.Code != "R001" {
		t.Errorf("expected R001, got %s", filtered[0].Rule.Code)
	}
}

func TestFilterResults_BySeverity(t *testing.T) {
	results := []RuleResult{
		{Rule: Rule{Code: "R001", Severity: SeverityError}, Passed: true},
		{Rule: Rule{Code: "R002", Severity: SeverityWarning}, Passed: true},
		{Rule: Rule{Code: "R003", Severity: SeverityInfo}, Passed: true},
	}
	filtered := FilterResults(results, FilterOptions{Severities: []Severity{SeverityError}})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
	if filtered[0].Rule.Code != "R001" {
		t.Errorf("expected R001, got %s", filtered[0].Rule.Code)
	}
}

func TestFilterResults_ByCategory(t *testing.T) {
	results := []RuleResult{
		{Rule: Rule{Code: "R001", Category: "length"}, Passed: true},
		{Rule: Rule{Code: "R002", Category: "instructions"}, Passed: true},
	}
	filtered := FilterResults(results, FilterOptions{Categories: []string{"length"}})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
	if filtered[0].Rule.Code != "R001" {
		t.Errorf("expected R001, got %s", filtered[0].Rule.Code)
	}
}

func TestFilterResults_HideGoodPractice(t *testing.T) {
	results := []RuleResult{
		{Rule: Rule{Code: "R001", Category: "length"}, Passed: true},
		{Rule: Rule{Code: "GP001", Category: "good-practice"}, Passed: true},
	}
	filtered := FilterResults(results, FilterOptions{HideGoodPractice: true})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
	if filtered[0].Rule.Code != "R001" {
		t.Errorf("expected R001, got %s", filtered[0].Rule.Code)
	}
}

func TestFilterResults_NoFilters(t *testing.T) {
	results := []RuleResult{
		{Rule: Rule{Code: "R001"}, Passed: true},
		{Rule: Rule{Code: "R002"}, Passed: false},
		{Rule: Rule{Code: "GP001", Category: "good-practice"}, Passed: true},
	}
	filtered := FilterResults(results, FilterOptions{})
	if len(filtered) != 3 {
		t.Errorf("expected 3 results with no filters, got %d", len(filtered))
	}
}

func TestFilterResults_CombinedFilters(t *testing.T) {
	results := []RuleResult{
		{Rule: Rule{Code: "R001", Severity: SeverityError, Category: "length"}, Passed: true},
		{Rule: Rule{Code: "R002", Severity: SeverityWarning, Category: "length"}, Passed: true},
		{Rule: Rule{Code: "R003", Severity: SeverityError, Category: "instructions"}, Passed: true},
	}
	filtered := FilterResults(results, FilterOptions{
		Severities: []Severity{SeverityError},
		Categories: []string{"length"},
	})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
	if filtered[0].Rule.Code != "R001" {
		t.Errorf("expected R001, got %s", filtered[0].Rule.Code)
	}
}
