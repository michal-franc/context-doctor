package rules

import (
	"testing"
)

func makeCtx(content string, lineCount, instrCount int, metrics map[string]any) *AnalysisContext {
	if metrics == nil {
		metrics = make(map[string]any)
	}
	return &AnalysisContext{
		FilePath:         "test.md",
		Content:          content,
		Lines:            nil,
		LineCount:        lineCount,
		InstructionCount: instrCount,
		Metrics:          metrics,
	}
}

// =============================================================================
// Type conversion helpers
// =============================================================================

func TestGetMetricValue(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *AnalysisContext
		metric  MetricType
		want    any
	}{
		{"lineCount", makeCtx("", 42, 0, nil), MetricLineCount, 42},
		{"instructionCount", makeCtx("", 0, 15, nil), MetricInstructionCount, 15},
		{"content", makeCtx("hello world", 0, 0, nil), MetricContent, "hello world"},
		{"custom metric from map", makeCtx("", 0, 0, map[string]any{"custom_key": 99}), MetricType("custom_key"), 99},
		{"unknown metric returns nil", makeCtx("", 0, 0, nil), MetricType("nonexistent"), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := getMetricValue(tc.ctx, tc.metric); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		want   int
		wantOK bool
	}{
		{"int", 42, 42, true},
		{"int64", int64(100), 100, true},
		{"float64 truncates", float64(7.9), 7, true},
		{"string fails", "not a number", 0, false},
		{"nil fails", nil, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := toInt(tc.input)
			if ok != tc.wantOK || (ok && got != tc.want) {
				t.Errorf("toInt(%v) = (%d, %v), want (%d, %v)", tc.input, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string passes through", "hello", "hello"},
		{"int returns empty", 42, ""},
		{"nil returns empty", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := toString(tc.input); got != tc.want {
				t.Errorf("toString(%v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// =============================================================================
// Numeric comparison actions
// =============================================================================

func TestCheckLessThan(t *testing.T) {
	tests := []struct {
		name      string
		metric    MetricType
		lineCount int
		content   string
		value     any
		want      bool
	}{
		{"below threshold", MetricLineCount, 50, "", 100, true},
		{"above threshold", MetricLineCount, 200, "", 100, false},
		{"equal is not less", MetricLineCount, 100, "", 100, false},
		{"non-numeric metric", MetricContent, 0, "text", 100, false},
		{"non-numeric value", MetricLineCount, 50, "", "NaN", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := makeCtx(tc.content, tc.lineCount, 0, nil)
			spec := &MatchSpec{Metric: tc.metric, Action: ActionLessThan, Value: tc.value}
			if got := checkLessThan(ctx, spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckGreaterThan(t *testing.T) {
	tests := []struct {
		name       string
		metric     MetricType
		lineCount  int
		instrCount int
		value      any
		want       bool
	}{
		{"above threshold", MetricLineCount, 200, 0, 100, true},
		{"below threshold", MetricLineCount, 50, 0, 100, false},
		{"works with instructionCount", MetricInstructionCount, 0, 30, 20, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := makeCtx("", tc.lineCount, tc.instrCount, nil)
			spec := &MatchSpec{Metric: tc.metric, Action: ActionGreaterThan, Value: tc.value}
			if got := checkGreaterThan(ctx, spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// =============================================================================
// Equality actions
// =============================================================================

func TestCheckEquals(t *testing.T) {
	tests := []struct {
		name string
		ctx  *AnalysisContext
		spec *MatchSpec
		want bool
	}{
		{"int match", makeCtx("", 50, 0, nil), &MatchSpec{Metric: MetricLineCount, Value: 50}, true},
		{"int mismatch", makeCtx("", 50, 0, nil), &MatchSpec{Metric: MetricLineCount, Value: 99}, false},
		{"string match", makeCtx("hello", 0, 0, nil), &MatchSpec{Metric: MetricContent, Value: "hello"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkEquals(tc.ctx, tc.spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckNotEquals(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{"different values", 99, true},
		{"same value", 50, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := makeCtx("", 50, 0, nil)
			spec := &MatchSpec{Metric: MetricLineCount, Value: tc.value}
			if got := checkNotEquals(ctx, spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// =============================================================================
// Substring actions (contains / notContains)
// =============================================================================

func TestCheckContains(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *AnalysisContext
		spec    *MatchSpec
		want    bool
	}{
		{"single value found",
			makeCtx("Always use gofmt for formatting", 0, 0, nil),
			&MatchSpec{Action: ActionContains, Value: "gofmt"}, true},
		{"case insensitive",
			makeCtx("Always use GOFMT for formatting", 0, 0, nil),
			&MatchSpec{Action: ActionContains, Value: "gofmt"}, true},
		{"value not found",
			makeCtx("Use eslint for linting", 0, 0, nil),
			&MatchSpec{Action: ActionContains, Value: "gofmt"}, false},
		{"patterns — one matches",
			makeCtx("Use eslint for linting", 0, 0, nil),
			&MatchSpec{Action: ActionContains, Patterns: []string{"gofmt", "eslint", "prettier"}}, true},
		{"patterns — none match",
			makeCtx("Use standard library only", 0, 0, nil),
			&MatchSpec{Action: ActionContains, Patterns: []string{"gofmt", "eslint", "prettier"}}, false},
		{"nil value and no patterns",
			makeCtx("some content", 0, 0, nil),
			&MatchSpec{Action: ActionContains}, false},
		{"custom metric",
			makeCtx("", 0, 0, map[string]any{"custom": "hello world"}),
			&MatchSpec{Metric: MetricType("custom"), Action: ActionContains, Value: "hello"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkContains(tc.ctx, tc.spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckNotContains(t *testing.T) {
	tests := []struct {
		name string
		ctx  *AnalysisContext
		spec *MatchSpec
		want bool
	}{
		{"value absent",
			makeCtx("Use standard library only", 0, 0, nil),
			&MatchSpec{Action: ActionNotContains, Value: "gofmt"}, true},
		{"value present",
			makeCtx("Always use gofmt", 0, 0, nil),
			&MatchSpec{Action: ActionNotContains, Value: "gofmt"}, false},
		{"patterns — all absent",
			makeCtx("Clean code", 0, 0, nil),
			&MatchSpec{Action: ActionNotContains, Patterns: []string{"gofmt", "eslint"}}, true},
		{"patterns — one present",
			makeCtx("Use gofmt for formatting", 0, 0, nil),
			&MatchSpec{Action: ActionNotContains, Patterns: []string{"gofmt", "eslint"}}, false},
		{"nil value and no patterns",
			makeCtx("some content", 0, 0, nil),
			&MatchSpec{Action: ActionNotContains}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkNotContains(tc.ctx, tc.spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// =============================================================================
// Regex actions
// =============================================================================

func TestCheckRegexMatch(t *testing.T) {
	tests := []struct {
		name string
		ctx  *AnalysisContext
		spec *MatchSpec
		want bool
	}{
		{"single value matches",
			makeCtx("Version 2.3.1 released", 0, 0, nil),
			&MatchSpec{Action: ActionRegexMatch, Value: `\d+\.\d+\.\d+`}, true},
		{"single value no match",
			makeCtx("No version here", 0, 0, nil),
			&MatchSpec{Action: ActionRegexMatch, Value: `\d+\.\d+\.\d+`}, false},
		{"patterns — second matches",
			makeCtx("disable all checks", 0, 0, nil),
			&MatchSpec{Action: ActionRegexMatch, Patterns: []string{`^version`, `disable.*checks`}}, true},
		{"patterns — none match",
			makeCtx("clean code", 0, 0, nil),
			&MatchSpec{Action: ActionRegexMatch, Patterns: []string{`^version`, `disable.*checks`}}, false},
		{"invalid regex returns false",
			makeCtx("test", 0, 0, nil),
			&MatchSpec{Action: ActionRegexMatch, Value: `[invalid`}, false},
		{"invalid pattern skipped, valid one matches",
			makeCtx("good match here", 0, 0, nil),
			&MatchSpec{Action: ActionRegexMatch, Patterns: []string{`[invalid`, `good match`}}, true},
		{"nil value and no patterns",
			makeCtx("some content", 0, 0, nil),
			&MatchSpec{Action: ActionRegexMatch}, false},
		{"case insensitive",
			makeCtx("ALWAYS USE GOFMT", 0, 0, nil),
			&MatchSpec{Action: ActionRegexMatch, Value: "always use gofmt"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkRegexMatch(tc.ctx, tc.spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckRegexNotMatch(t *testing.T) {
	tests := []struct {
		name string
		ctx  *AnalysisContext
		spec *MatchSpec
		want bool
	}{
		{"no match returns true",
			makeCtx("clean code", 0, 0, nil),
			&MatchSpec{Action: ActionRegexNotMatch, Value: `\d+\.\d+`}, true},
		{"match returns false",
			makeCtx("version 1.2", 0, 0, nil),
			&MatchSpec{Action: ActionRegexNotMatch, Value: `\d+\.\d+`}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkRegexNotMatch(tc.ctx, tc.spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// =============================================================================
// Presence actions (isPresent / notPresent)
// =============================================================================

func TestCheckIsPresent(t *testing.T) {
	tests := []struct {
		name string
		ctx  *AnalysisContext
		spec *MatchSpec
		want bool
	}{
		{"regex pattern found",
			makeCtx("# Build & Test\nmake build\nmake test", 0, 0, nil),
			&MatchSpec{Action: ActionIsPresent, Patterns: []string{`make\s+test`, `npm\s+test`}}, true},
		{"regex pattern not found",
			makeCtx("# Simple readme", 0, 0, nil),
			&MatchSpec{Action: ActionIsPresent, Patterns: []string{`make\s+test`, `npm\s+test`}}, false},
		{"single value found",
			makeCtx("use gofmt for formatting", 0, 0, nil),
			&MatchSpec{Action: ActionIsPresent, Value: "gofmt"}, true},
		{"invalid regex falls back to literal match",
			makeCtx("some [bracket content", 0, 0, nil),
			&MatchSpec{Action: ActionIsPresent, Patterns: []string{`[bracket`}}, true},
		{"nil value and no patterns",
			makeCtx("some content", 0, 0, nil),
			&MatchSpec{Action: ActionIsPresent}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkIsPresent(tc.ctx, tc.spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckNotPresent(t *testing.T) {
	tests := []struct {
		name string
		ctx  *AnalysisContext
		spec *MatchSpec
		want bool
	}{
		{"pattern absent",
			makeCtx("clean code", 0, 0, nil),
			&MatchSpec{Action: ActionNotPresent, Patterns: []string{`deprecated`}}, true},
		{"pattern present",
			makeCtx("this is deprecated", 0, 0, nil),
			&MatchSpec{Action: ActionNotPresent, Patterns: []string{`deprecated`}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkNotPresent(tc.ctx, tc.spec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// =============================================================================
// Logical combinators (and / or)
// =============================================================================

func TestCheckAnd(t *testing.T) {
	t.Run("all sub-specs pass", func(t *testing.T) {
		ctx := makeCtx("use gofmt and eslint", 50, 10, nil)
		spec := &MatchSpec{
			Action: ActionAnd,
			SubMatch: []MatchSpec{
				{Action: ActionContains, Value: "gofmt"},
				{Metric: MetricLineCount, Action: ActionLessThan, Value: 100},
			},
		}
		if !checkAnd(ctx, spec) {
			t.Error("expected true")
		}
	})

	t.Run("one sub-spec fails", func(t *testing.T) {
		ctx := makeCtx("use gofmt", 200, 0, nil)
		spec := &MatchSpec{
			Action: ActionAnd,
			SubMatch: []MatchSpec{
				{Action: ActionContains, Value: "gofmt"},
				{Metric: MetricLineCount, Action: ActionLessThan, Value: 100},
			},
		}
		if checkAnd(ctx, spec) {
			t.Error("expected false")
		}
	})

	t.Run("empty subMatch returns true", func(t *testing.T) {
		ctx := makeCtx("", 0, 0, nil)
		spec := &MatchSpec{Action: ActionAnd, SubMatch: []MatchSpec{}}
		if !checkAnd(ctx, spec) {
			t.Error("expected true (vacuous truth)")
		}
	})
}

func TestCheckOr(t *testing.T) {
	t.Run("one sub-spec passes", func(t *testing.T) {
		ctx := makeCtx("use gofmt", 200, 0, nil)
		spec := &MatchSpec{
			Action: ActionOr,
			SubMatch: []MatchSpec{
				{Action: ActionContains, Value: "gofmt"},
				{Metric: MetricLineCount, Action: ActionLessThan, Value: 100},
			},
		}
		if !checkOr(ctx, spec) {
			t.Error("expected true")
		}
	})

	t.Run("all sub-specs fail", func(t *testing.T) {
		ctx := makeCtx("clean code", 200, 0, nil)
		spec := &MatchSpec{
			Action: ActionOr,
			SubMatch: []MatchSpec{
				{Action: ActionContains, Value: "gofmt"},
				{Metric: MetricLineCount, Action: ActionLessThan, Value: 100},
			},
		}
		if checkOr(ctx, spec) {
			t.Error("expected false")
		}
	})

	t.Run("empty subMatch returns false", func(t *testing.T) {
		ctx := makeCtx("", 0, 0, nil)
		spec := &MatchSpec{Action: ActionOr, SubMatch: []MatchSpec{}}
		if checkOr(ctx, spec) {
			t.Error("expected false")
		}
	})
}

// =============================================================================
// EvaluateSpec dispatcher
// =============================================================================

func TestEvaluateSpec(t *testing.T) {
	t.Run("dispatches to correct action", func(t *testing.T) {
		ctx := makeCtx("", 50, 0, nil)
		spec := &MatchSpec{Metric: MetricLineCount, Action: ActionLessThan, Value: 100}
		if !EvaluateSpec(ctx, spec) {
			t.Error("expected true (50 < 100)")
		}
	})

	t.Run("unknown action returns false", func(t *testing.T) {
		ctx := makeCtx("", 0, 0, nil)
		spec := &MatchSpec{Action: CheckAction("unknownAction")}
		if EvaluateSpec(ctx, spec) {
			t.Error("expected false")
		}
	})
}
