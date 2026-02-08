package main

import (
	"testing"

	"context-doctor/rules"
)

// =============================================================================
// calculateScore
// =============================================================================

// result is a shorthand for building RuleResult test data.
func result(severity rules.Severity, category string, passed bool) rules.RuleResult {
	return rules.RuleResult{
		Rule:   rules.Rule{Severity: severity, Category: category},
		Passed: passed,
	}
}

func repeat(r rules.RuleResult, n int) []rules.RuleResult {
	out := make([]rules.RuleResult, n)
	for i := range out {
		out[i] = r
	}
	return out
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name    string
		results []rules.RuleResult
		want    int
	}{
		{"no problems detected",
			[]rules.RuleResult{
				result(rules.SeverityError, "length", false),
				result(rules.SeverityWarning, "length", false),
			}, 100},
		{"one error: -15",
			[]rules.RuleResult{result(rules.SeverityError, "length", true)},
			85},
		{"one warning: -5",
			[]rules.RuleResult{result(rules.SeverityWarning, "length", true)},
			95},
		{"one info: -2",
			[]rules.RuleResult{result(rules.SeverityInfo, "length", true)},
			98},
		{"floor at zero (8 errors = -120)",
			repeat(result(rules.SeverityError, "length", true), 8),
			0},
		{"good-practice skipped",
			[]rules.RuleResult{
				result(rules.SeverityError, "good-practice", true),
				result(rules.SeverityError, "length", true),
			}, 85},
		{"mixed: error + warning + info = 100-15-5-2",
			[]rules.RuleResult{
				result(rules.SeverityError, "length", true),
				result(rules.SeverityWarning, "length", true),
				result(rules.SeverityInfo, "length", true),
			}, 78},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &rules.AnalysisContext{Metrics: make(map[string]any)}
			if got := calculateScore(ctx, tc.results); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// =============================================================================
// truncate
// =============================================================================

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world this is long", 10, "hello w..."},
		{"hello", 3, "..."},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := truncate(tc.input, tc.maxLen); got != tc.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
			}
		})
	}
}

// =============================================================================
// getSeverityIcon
// =============================================================================

func TestGetSeverityIcon(t *testing.T) {
	tests := []struct {
		severity rules.Severity
		want     string
	}{
		{rules.SeverityError, "✗"},
		{rules.SeverityWarning, "⚠"},
		{rules.SeverityInfo, "ℹ"},
		{rules.Severity("unknown"), " "},
	}
	for _, tc := range tests {
		t.Run(string(tc.severity), func(t *testing.T) {
			if got := getSeverityIcon(tc.severity); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// =============================================================================
// buildFilterOpts
// =============================================================================

func TestBuildFilterOpts(t *testing.T) {
	// Save and restore globals to avoid test pollution.
	save := func() (bool, string, string) {
		return verbose, categoriesFlag, severitiesFlag
	}
	restore := func(v bool, c, s string) {
		verbose, categoriesFlag, severitiesFlag = v, c, s
	}

	t.Run("defaults (non-verbose)", func(t *testing.T) {
		v, c, s := save()
		defer restore(v, c, s)
		verbose, categoriesFlag, severitiesFlag = false, "", ""

		opts := buildFilterOpts()
		if !opts.FailuresOnly {
			t.Error("expected FailuresOnly=true")
		}
		if !opts.HideGoodPractice {
			t.Error("expected HideGoodPractice=true")
		}
		if len(opts.Categories) != 0 || len(opts.Severities) != 0 {
			t.Error("expected empty categories and severities")
		}
	})

	t.Run("verbose disables filters", func(t *testing.T) {
		v, c, s := save()
		defer restore(v, c, s)
		verbose, categoriesFlag, severitiesFlag = true, "", ""

		opts := buildFilterOpts()
		if opts.FailuresOnly || opts.HideGoodPractice {
			t.Error("expected both filters disabled in verbose mode")
		}
	})

	t.Run("parses categories", func(t *testing.T) {
		v, c, s := save()
		defer restore(v, c, s)
		categoriesFlag = "length,instructions"

		opts := buildFilterOpts()
		if len(opts.Categories) != 2 {
			t.Fatalf("expected 2 categories, got %d", len(opts.Categories))
		}
		if opts.Categories[0] != "length" || opts.Categories[1] != "instructions" {
			t.Errorf("got %v", opts.Categories)
		}
	})

	t.Run("parses severities", func(t *testing.T) {
		v, c, s := save()
		defer restore(v, c, s)
		severitiesFlag = "error,warning"

		opts := buildFilterOpts()
		if len(opts.Severities) != 2 {
			t.Fatalf("expected 2 severities, got %d", len(opts.Severities))
		}
		if opts.Severities[0] != rules.SeverityError || opts.Severities[1] != rules.SeverityWarning {
			t.Errorf("got %v", opts.Severities)
		}
	})
}
