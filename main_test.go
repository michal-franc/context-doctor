package main

import (
	"testing"

	"context-doctor/rules"
)

// --- calculateScore ---

func TestCalculateScore_Perfect(t *testing.T) {
	ctx := &rules.AnalysisContext{Metrics: make(map[string]any)}
	results := []rules.RuleResult{
		{Rule: rules.Rule{Severity: rules.SeverityError, Category: "length"}, Passed: false},
		{Rule: rules.Rule{Severity: rules.SeverityWarning, Category: "length"}, Passed: false},
	}
	score := calculateScore(ctx, results)
	if score != 100 {
		t.Errorf("expected 100 for no detected problems, got %d", score)
	}
}

func TestCalculateScore_ErrorDeduction(t *testing.T) {
	ctx := &rules.AnalysisContext{Metrics: make(map[string]any)}
	results := []rules.RuleResult{
		{Rule: rules.Rule{Severity: rules.SeverityError, Category: "length"}, Passed: true},
	}
	score := calculateScore(ctx, results)
	if score != 85 {
		t.Errorf("expected 85 (100 - 15), got %d", score)
	}
}

func TestCalculateScore_WarningDeduction(t *testing.T) {
	ctx := &rules.AnalysisContext{Metrics: make(map[string]any)}
	results := []rules.RuleResult{
		{Rule: rules.Rule{Severity: rules.SeverityWarning, Category: "length"}, Passed: true},
	}
	score := calculateScore(ctx, results)
	if score != 95 {
		t.Errorf("expected 95 (100 - 5), got %d", score)
	}
}

func TestCalculateScore_InfoDeduction(t *testing.T) {
	ctx := &rules.AnalysisContext{Metrics: make(map[string]any)}
	results := []rules.RuleResult{
		{Rule: rules.Rule{Severity: rules.SeverityInfo, Category: "length"}, Passed: true},
	}
	score := calculateScore(ctx, results)
	if score != 98 {
		t.Errorf("expected 98 (100 - 2), got %d", score)
	}
}

func TestCalculateScore_FloorAtZero(t *testing.T) {
	ctx := &rules.AnalysisContext{Metrics: make(map[string]any)}
	// 8 errors = 8 * 15 = 120 deduction
	var results []rules.RuleResult
	for i := 0; i < 8; i++ {
		results = append(results, rules.RuleResult{
			Rule:   rules.Rule{Severity: rules.SeverityError, Category: "length"},
			Passed: true,
		})
	}
	score := calculateScore(ctx, results)
	if score != 0 {
		t.Errorf("expected 0 (floor), got %d", score)
	}
}

func TestCalculateScore_SkipsGoodPractice(t *testing.T) {
	ctx := &rules.AnalysisContext{Metrics: make(map[string]any)}
	results := []rules.RuleResult{
		{Rule: rules.Rule{Severity: rules.SeverityError, Category: "good-practice"}, Passed: true},
		{Rule: rules.Rule{Severity: rules.SeverityError, Category: "length"}, Passed: true},
	}
	score := calculateScore(ctx, results)
	// Only the non-good-practice error should count
	if score != 85 {
		t.Errorf("expected 85 (good-practice skipped), got %d", score)
	}
}

func TestCalculateScore_MixedSeverities(t *testing.T) {
	ctx := &rules.AnalysisContext{Metrics: make(map[string]any)}
	results := []rules.RuleResult{
		{Rule: rules.Rule{Severity: rules.SeverityError, Category: "length"}, Passed: true},
		{Rule: rules.Rule{Severity: rules.SeverityWarning, Category: "length"}, Passed: true},
		{Rule: rules.Rule{Severity: rules.SeverityInfo, Category: "length"}, Passed: true},
	}
	score := calculateScore(ctx, results)
	// 100 - 15 - 5 - 2 = 78
	if score != 78 {
		t.Errorf("expected 78, got %d", score)
	}
}

// --- truncate ---

func TestTruncate_ShortString(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	result := truncate("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncate_LongString(t *testing.T) {
	result := truncate("hello world this is long", 10)
	if result != "hello w..." {
		t.Errorf("expected 'hello w...', got %q", result)
	}
}

func TestTruncate_VeryShortMax(t *testing.T) {
	result := truncate("hello", 3)
	if result != "..." {
		t.Errorf("expected '...', got %q", result)
	}
}

// --- getSeverityIcon ---

func TestGetSeverityIcon_Error(t *testing.T) {
	icon := getSeverityIcon(rules.SeverityError)
	if icon != "✗" {
		t.Errorf("expected '✗', got %q", icon)
	}
}

func TestGetSeverityIcon_Warning(t *testing.T) {
	icon := getSeverityIcon(rules.SeverityWarning)
	if icon != "⚠" {
		t.Errorf("expected '⚠', got %q", icon)
	}
}

func TestGetSeverityIcon_Info(t *testing.T) {
	icon := getSeverityIcon(rules.SeverityInfo)
	if icon != "ℹ" {
		t.Errorf("expected 'ℹ', got %q", icon)
	}
}

func TestGetSeverityIcon_Unknown(t *testing.T) {
	icon := getSeverityIcon(rules.Severity("unknown"))
	if icon != " " {
		t.Errorf("expected ' ', got %q", icon)
	}
}

// --- buildFilterOpts ---

func TestBuildFilterOpts_Defaults(t *testing.T) {
	// Reset global flags to defaults
	oldVerbose := verbose
	oldCategories := categoriesFlag
	oldSeverities := severitiesFlag
	defer func() {
		verbose = oldVerbose
		categoriesFlag = oldCategories
		severitiesFlag = oldSeverities
	}()

	verbose = false
	categoriesFlag = ""
	severitiesFlag = ""

	opts := buildFilterOpts()
	if !opts.FailuresOnly {
		t.Error("expected FailuresOnly=true when not verbose")
	}
	if !opts.HideGoodPractice {
		t.Error("expected HideGoodPractice=true when not verbose")
	}
	if len(opts.Categories) != 0 {
		t.Errorf("expected no categories, got %v", opts.Categories)
	}
	if len(opts.Severities) != 0 {
		t.Errorf("expected no severities, got %v", opts.Severities)
	}
}

func TestBuildFilterOpts_Verbose(t *testing.T) {
	oldVerbose := verbose
	defer func() { verbose = oldVerbose }()

	verbose = true
	categoriesFlag = ""
	severitiesFlag = ""

	opts := buildFilterOpts()
	if opts.FailuresOnly {
		t.Error("expected FailuresOnly=false when verbose")
	}
	if opts.HideGoodPractice {
		t.Error("expected HideGoodPractice=false when verbose")
	}
}

func TestBuildFilterOpts_WithCategories(t *testing.T) {
	oldCategories := categoriesFlag
	defer func() { categoriesFlag = oldCategories }()

	categoriesFlag = "length,instructions"

	opts := buildFilterOpts()
	if len(opts.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(opts.Categories))
	}
	if opts.Categories[0] != "length" || opts.Categories[1] != "instructions" {
		t.Errorf("expected [length, instructions], got %v", opts.Categories)
	}
}

func TestBuildFilterOpts_WithSeverities(t *testing.T) {
	oldSeverities := severitiesFlag
	defer func() { severitiesFlag = oldSeverities }()

	severitiesFlag = "error,warning"

	opts := buildFilterOpts()
	if len(opts.Severities) != 2 {
		t.Fatalf("expected 2 severities, got %d", len(opts.Severities))
	}
	if opts.Severities[0] != rules.SeverityError || opts.Severities[1] != rules.SeverityWarning {
		t.Errorf("expected [error, warning], got %v", opts.Severities)
	}
}
