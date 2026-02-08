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

// --- getMetricValue ---

func TestGetMetricValue_LineCount(t *testing.T) {
	ctx := makeCtx("", 42, 0, nil)
	v := getMetricValue(ctx, MetricLineCount)
	if v != 42 {
		t.Errorf("expected 42, got %v", v)
	}
}

func TestGetMetricValue_InstructionCount(t *testing.T) {
	ctx := makeCtx("", 0, 15, nil)
	v := getMetricValue(ctx, MetricInstructionCount)
	if v != 15 {
		t.Errorf("expected 15, got %v", v)
	}
}

func TestGetMetricValue_Content(t *testing.T) {
	ctx := makeCtx("hello world", 0, 0, nil)
	v := getMetricValue(ctx, MetricContent)
	if v != "hello world" {
		t.Errorf("expected 'hello world', got %v", v)
	}
}

func TestGetMetricValue_CustomMetric(t *testing.T) {
	ctx := makeCtx("", 0, 0, map[string]any{"custom_key": 99})
	v := getMetricValue(ctx, MetricType("custom_key"))
	if v != 99 {
		t.Errorf("expected 99, got %v", v)
	}
}

func TestGetMetricValue_UnknownMetric(t *testing.T) {
	ctx := makeCtx("", 0, 0, nil)
	v := getMetricValue(ctx, MetricType("nonexistent"))
	if v != nil {
		t.Errorf("expected nil, got %v", v)
	}
}

// --- toInt ---

func TestToInt_Int(t *testing.T) {
	v, ok := toInt(42)
	if !ok || v != 42 {
		t.Errorf("expected (42, true), got (%d, %v)", v, ok)
	}
}

func TestToInt_Int64(t *testing.T) {
	v, ok := toInt(int64(100))
	if !ok || v != 100 {
		t.Errorf("expected (100, true), got (%d, %v)", v, ok)
	}
}

func TestToInt_Float64(t *testing.T) {
	v, ok := toInt(float64(7.9))
	if !ok || v != 7 {
		t.Errorf("expected (7, true), got (%d, %v)", v, ok)
	}
}

func TestToInt_String(t *testing.T) {
	_, ok := toInt("not a number")
	if ok {
		t.Error("expected ok=false for string input")
	}
}

func TestToInt_Nil(t *testing.T) {
	_, ok := toInt(nil)
	if ok {
		t.Error("expected ok=false for nil")
	}
}

// --- toString ---

func TestToString_String(t *testing.T) {
	if toString("hello") != "hello" {
		t.Error("expected 'hello'")
	}
}

func TestToString_NonString(t *testing.T) {
	if toString(42) != "" {
		t.Error("expected empty string for non-string type")
	}
}

func TestToString_Nil(t *testing.T) {
	if toString(nil) != "" {
		t.Error("expected empty string for nil")
	}
}

// --- checkLessThan ---

func TestCheckLessThan_True(t *testing.T) {
	ctx := makeCtx("", 50, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionLessThan, Value: 100}
	if !checkLessThan(ctx, spec) {
		t.Error("expected 50 < 100 to be true")
	}
}

func TestCheckLessThan_False(t *testing.T) {
	ctx := makeCtx("", 200, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionLessThan, Value: 100}
	if checkLessThan(ctx, spec) {
		t.Error("expected 200 < 100 to be false")
	}
}

func TestCheckLessThan_Equal(t *testing.T) {
	ctx := makeCtx("", 100, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionLessThan, Value: 100}
	if checkLessThan(ctx, spec) {
		t.Error("expected 100 < 100 to be false")
	}
}

func TestCheckLessThan_NonNumericMetric(t *testing.T) {
	ctx := makeCtx("text", 0, 0, nil)
	spec := &MatchSpec{Metric: MetricContent, Action: ActionLessThan, Value: 100}
	if checkLessThan(ctx, spec) {
		t.Error("expected false for non-numeric metric")
	}
}

func TestCheckLessThan_NonNumericValue(t *testing.T) {
	ctx := makeCtx("", 50, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionLessThan, Value: "not a number"}
	if checkLessThan(ctx, spec) {
		t.Error("expected false for non-numeric value")
	}
}

// --- checkGreaterThan ---

func TestCheckGreaterThan_True(t *testing.T) {
	ctx := makeCtx("", 200, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionGreaterThan, Value: 100}
	if !checkGreaterThan(ctx, spec) {
		t.Error("expected 200 > 100 to be true")
	}
}

func TestCheckGreaterThan_False(t *testing.T) {
	ctx := makeCtx("", 50, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionGreaterThan, Value: 100}
	if checkGreaterThan(ctx, spec) {
		t.Error("expected 50 > 100 to be false")
	}
}

func TestCheckGreaterThan_InstructionCount(t *testing.T) {
	ctx := makeCtx("", 0, 30, nil)
	spec := &MatchSpec{Metric: MetricInstructionCount, Action: ActionGreaterThan, Value: 20}
	if !checkGreaterThan(ctx, spec) {
		t.Error("expected 30 > 20 to be true")
	}
}

// --- checkEquals ---

func TestCheckEquals_IntMatch(t *testing.T) {
	ctx := makeCtx("", 50, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionEquals, Value: 50}
	if !checkEquals(ctx, spec) {
		t.Error("expected 50 == 50")
	}
}

func TestCheckEquals_IntMismatch(t *testing.T) {
	ctx := makeCtx("", 50, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionEquals, Value: 99}
	if checkEquals(ctx, spec) {
		t.Error("expected 50 != 99")
	}
}

func TestCheckEquals_StringMatch(t *testing.T) {
	ctx := makeCtx("hello", 0, 0, nil)
	spec := &MatchSpec{Metric: MetricContent, Action: ActionEquals, Value: "hello"}
	if !checkEquals(ctx, spec) {
		t.Error("expected content match")
	}
}

// --- checkNotEquals ---

func TestCheckNotEquals_True(t *testing.T) {
	ctx := makeCtx("", 50, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionNotEquals, Value: 99}
	if !checkNotEquals(ctx, spec) {
		t.Error("expected 50 != 99")
	}
}

func TestCheckNotEquals_False(t *testing.T) {
	ctx := makeCtx("", 50, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionNotEquals, Value: 50}
	if checkNotEquals(ctx, spec) {
		t.Error("expected 50 == 50 to make notEquals false")
	}
}

// --- checkContains ---

func TestCheckContains_SingleValue(t *testing.T) {
	ctx := makeCtx("Always use gofmt for formatting", 0, 0, nil)
	spec := &MatchSpec{Action: ActionContains, Value: "gofmt"}
	if !checkContains(ctx, spec) {
		t.Error("expected content to contain 'gofmt'")
	}
}

func TestCheckContains_CaseInsensitive(t *testing.T) {
	ctx := makeCtx("Always use GOFMT for formatting", 0, 0, nil)
	spec := &MatchSpec{Action: ActionContains, Value: "gofmt"}
	if !checkContains(ctx, spec) {
		t.Error("expected case-insensitive match")
	}
}

func TestCheckContains_NotFound(t *testing.T) {
	ctx := makeCtx("Use eslint for linting", 0, 0, nil)
	spec := &MatchSpec{Action: ActionContains, Value: "gofmt"}
	if checkContains(ctx, spec) {
		t.Error("expected content to not contain 'gofmt'")
	}
}

func TestCheckContains_Patterns_OneMatches(t *testing.T) {
	ctx := makeCtx("Use eslint for linting", 0, 0, nil)
	spec := &MatchSpec{Action: ActionContains, Patterns: []string{"gofmt", "eslint", "prettier"}}
	if !checkContains(ctx, spec) {
		t.Error("expected one pattern to match")
	}
}

func TestCheckContains_Patterns_NoneMatch(t *testing.T) {
	ctx := makeCtx("Use standard library only", 0, 0, nil)
	spec := &MatchSpec{Action: ActionContains, Patterns: []string{"gofmt", "eslint", "prettier"}}
	if checkContains(ctx, spec) {
		t.Error("expected no patterns to match")
	}
}

func TestCheckContains_NilValue(t *testing.T) {
	ctx := makeCtx("some content", 0, 0, nil)
	spec := &MatchSpec{Action: ActionContains}
	if checkContains(ctx, spec) {
		t.Error("expected false for nil value and empty patterns")
	}
}

func TestCheckContains_CustomMetric(t *testing.T) {
	ctx := makeCtx("", 0, 0, map[string]any{"custom": "hello world"})
	spec := &MatchSpec{Metric: MetricType("custom"), Action: ActionContains, Value: "hello"}
	if !checkContains(ctx, spec) {
		t.Error("expected custom metric to contain 'hello'")
	}
}

// --- checkNotContains ---

func TestCheckNotContains_True(t *testing.T) {
	ctx := makeCtx("Use standard library only", 0, 0, nil)
	spec := &MatchSpec{Action: ActionNotContains, Value: "gofmt"}
	if !checkNotContains(ctx, spec) {
		t.Error("expected notContains to be true")
	}
}

func TestCheckNotContains_False(t *testing.T) {
	ctx := makeCtx("Always use gofmt", 0, 0, nil)
	spec := &MatchSpec{Action: ActionNotContains, Value: "gofmt"}
	if checkNotContains(ctx, spec) {
		t.Error("expected notContains to be false when content contains value")
	}
}

func TestCheckNotContains_Patterns_AllAbsent(t *testing.T) {
	ctx := makeCtx("Clean code", 0, 0, nil)
	spec := &MatchSpec{Action: ActionNotContains, Patterns: []string{"gofmt", "eslint"}}
	if !checkNotContains(ctx, spec) {
		t.Error("expected true when no patterns match")
	}
}

func TestCheckNotContains_Patterns_OnePresent(t *testing.T) {
	ctx := makeCtx("Use gofmt for formatting", 0, 0, nil)
	spec := &MatchSpec{Action: ActionNotContains, Patterns: []string{"gofmt", "eslint"}}
	if checkNotContains(ctx, spec) {
		t.Error("expected false when one pattern matches")
	}
}

func TestCheckNotContains_NilValue(t *testing.T) {
	ctx := makeCtx("some content", 0, 0, nil)
	spec := &MatchSpec{Action: ActionNotContains}
	if !checkNotContains(ctx, spec) {
		t.Error("expected true for nil value and empty patterns")
	}
}

// --- checkRegexMatch ---

func TestCheckRegexMatch_SingleValue(t *testing.T) {
	ctx := makeCtx("Version 2.3.1 released", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexMatch, Value: `\d+\.\d+\.\d+`}
	if !checkRegexMatch(ctx, spec) {
		t.Error("expected regex to match version number")
	}
}

func TestCheckRegexMatch_NoMatch(t *testing.T) {
	ctx := makeCtx("No version here", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexMatch, Value: `\d+\.\d+\.\d+`}
	if checkRegexMatch(ctx, spec) {
		t.Error("expected regex to not match")
	}
}

func TestCheckRegexMatch_Patterns(t *testing.T) {
	ctx := makeCtx("disable all checks", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexMatch, Patterns: []string{`^version`, `disable.*checks`}}
	if !checkRegexMatch(ctx, spec) {
		t.Error("expected one regex pattern to match")
	}
}

func TestCheckRegexMatch_Patterns_NoneMatch(t *testing.T) {
	ctx := makeCtx("clean code", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexMatch, Patterns: []string{`^version`, `disable.*checks`}}
	if checkRegexMatch(ctx, spec) {
		t.Error("expected no regex patterns to match")
	}
}

func TestCheckRegexMatch_InvalidRegex(t *testing.T) {
	ctx := makeCtx("test", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexMatch, Value: `[invalid`}
	if checkRegexMatch(ctx, spec) {
		t.Error("expected false for invalid regex")
	}
}

func TestCheckRegexMatch_InvalidPatternSkipped(t *testing.T) {
	ctx := makeCtx("good match here", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexMatch, Patterns: []string{`[invalid`, `good match`}}
	if !checkRegexMatch(ctx, spec) {
		t.Error("expected valid pattern to match even with invalid pattern in list")
	}
}

func TestCheckRegexMatch_NilValue(t *testing.T) {
	ctx := makeCtx("some content", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexMatch}
	if checkRegexMatch(ctx, spec) {
		t.Error("expected false for nil value and empty patterns")
	}
}

func TestCheckRegexMatch_CaseInsensitive(t *testing.T) {
	ctx := makeCtx("ALWAYS USE GOFMT", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexMatch, Value: "always use gofmt"}
	if !checkRegexMatch(ctx, spec) {
		t.Error("expected case-insensitive match")
	}
}

// --- checkRegexNotMatch ---

func TestCheckRegexNotMatch_True(t *testing.T) {
	ctx := makeCtx("clean code", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexNotMatch, Value: `\d+\.\d+`}
	if !checkRegexNotMatch(ctx, spec) {
		t.Error("expected true when regex does not match")
	}
}

func TestCheckRegexNotMatch_False(t *testing.T) {
	ctx := makeCtx("version 1.2", 0, 0, nil)
	spec := &MatchSpec{Action: ActionRegexNotMatch, Value: `\d+\.\d+`}
	if checkRegexNotMatch(ctx, spec) {
		t.Error("expected false when regex matches")
	}
}

// --- checkIsPresent ---

func TestCheckIsPresent_PatternFound(t *testing.T) {
	ctx := makeCtx("# Build & Test\nmake build\nmake test", 0, 0, nil)
	spec := &MatchSpec{Action: ActionIsPresent, Patterns: []string{`make\s+test`, `npm\s+test`}}
	if !checkIsPresent(ctx, spec) {
		t.Error("expected pattern to be found")
	}
}

func TestCheckIsPresent_PatternNotFound(t *testing.T) {
	ctx := makeCtx("# Simple readme", 0, 0, nil)
	spec := &MatchSpec{Action: ActionIsPresent, Patterns: []string{`make\s+test`, `npm\s+test`}}
	if checkIsPresent(ctx, spec) {
		t.Error("expected pattern to not be found")
	}
}

func TestCheckIsPresent_SingleValue(t *testing.T) {
	ctx := makeCtx("use gofmt for formatting", 0, 0, nil)
	spec := &MatchSpec{Action: ActionIsPresent, Value: "gofmt"}
	if !checkIsPresent(ctx, spec) {
		t.Error("expected value to be present")
	}
}

func TestCheckIsPresent_InvalidRegexFallsBackToLiteral(t *testing.T) {
	ctx := makeCtx("some [bracket content", 0, 0, nil)
	spec := &MatchSpec{Action: ActionIsPresent, Patterns: []string{`[bracket`}}
	if !checkIsPresent(ctx, spec) {
		t.Error("expected literal fallback to find the string")
	}
}

func TestCheckIsPresent_NilValue(t *testing.T) {
	ctx := makeCtx("some content", 0, 0, nil)
	spec := &MatchSpec{Action: ActionIsPresent}
	if checkIsPresent(ctx, spec) {
		t.Error("expected false for nil value and empty patterns")
	}
}

// --- checkNotPresent ---

func TestCheckNotPresent_True(t *testing.T) {
	ctx := makeCtx("clean code", 0, 0, nil)
	spec := &MatchSpec{Action: ActionNotPresent, Patterns: []string{`deprecated`}}
	if !checkNotPresent(ctx, spec) {
		t.Error("expected true when pattern not present")
	}
}

func TestCheckNotPresent_False(t *testing.T) {
	ctx := makeCtx("this is deprecated", 0, 0, nil)
	spec := &MatchSpec{Action: ActionNotPresent, Patterns: []string{`deprecated`}}
	if checkNotPresent(ctx, spec) {
		t.Error("expected false when pattern is present")
	}
}

// --- checkAnd ---

func TestCheckAnd_AllTrue(t *testing.T) {
	ctx := makeCtx("use gofmt and eslint", 50, 10, nil)
	spec := &MatchSpec{
		Action: ActionAnd,
		SubMatch: []MatchSpec{
			{Action: ActionContains, Value: "gofmt"},
			{Metric: MetricLineCount, Action: ActionLessThan, Value: 100},
		},
	}
	if !checkAnd(ctx, spec) {
		t.Error("expected AND to be true when all sub-specs pass")
	}
}

func TestCheckAnd_OneFalse(t *testing.T) {
	ctx := makeCtx("use gofmt", 200, 0, nil)
	spec := &MatchSpec{
		Action: ActionAnd,
		SubMatch: []MatchSpec{
			{Action: ActionContains, Value: "gofmt"},
			{Metric: MetricLineCount, Action: ActionLessThan, Value: 100},
		},
	}
	if checkAnd(ctx, spec) {
		t.Error("expected AND to be false when one sub-spec fails")
	}
}

func TestCheckAnd_EmptySubMatch(t *testing.T) {
	ctx := makeCtx("", 0, 0, nil)
	spec := &MatchSpec{Action: ActionAnd, SubMatch: []MatchSpec{}}
	if !checkAnd(ctx, spec) {
		t.Error("expected AND with empty subMatch to return true")
	}
}

// --- checkOr ---

func TestCheckOr_OneTrue(t *testing.T) {
	ctx := makeCtx("use gofmt", 200, 0, nil)
	spec := &MatchSpec{
		Action: ActionOr,
		SubMatch: []MatchSpec{
			{Action: ActionContains, Value: "gofmt"},
			{Metric: MetricLineCount, Action: ActionLessThan, Value: 100},
		},
	}
	if !checkOr(ctx, spec) {
		t.Error("expected OR to be true when one sub-spec passes")
	}
}

func TestCheckOr_AllFalse(t *testing.T) {
	ctx := makeCtx("clean code", 200, 0, nil)
	spec := &MatchSpec{
		Action: ActionOr,
		SubMatch: []MatchSpec{
			{Action: ActionContains, Value: "gofmt"},
			{Metric: MetricLineCount, Action: ActionLessThan, Value: 100},
		},
	}
	if checkOr(ctx, spec) {
		t.Error("expected OR to be false when all sub-specs fail")
	}
}

func TestCheckOr_EmptySubMatch(t *testing.T) {
	ctx := makeCtx("", 0, 0, nil)
	spec := &MatchSpec{Action: ActionOr, SubMatch: []MatchSpec{}}
	if checkOr(ctx, spec) {
		t.Error("expected OR with empty subMatch to return false")
	}
}

// --- EvaluateSpec ---

func TestEvaluateSpec_Dispatches(t *testing.T) {
	ctx := makeCtx("", 50, 0, nil)
	spec := &MatchSpec{Metric: MetricLineCount, Action: ActionLessThan, Value: 100}
	if !EvaluateSpec(ctx, spec) {
		t.Error("expected EvaluateSpec to dispatch to checkLessThan")
	}
}

func TestEvaluateSpec_UnknownAction(t *testing.T) {
	ctx := makeCtx("", 0, 0, nil)
	spec := &MatchSpec{Action: CheckAction("unknownAction")}
	if EvaluateSpec(ctx, spec) {
		t.Error("expected false for unknown action")
	}
}
