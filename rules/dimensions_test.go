package rules

import (
	"math"
	"testing"
)

// =============================================================================
// ResolveDimension
// =============================================================================

func TestResolveDimension(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
		want Dimension
	}{
		{"explicit yaml dimension",
			Rule{Dimension: DimensionStyle, Category: "length"},
			DimensionStyle},
		{"category fallback: length -> correctness",
			Rule{Category: "length"},
			DimensionCorrectness},
		{"category fallback: linter-abuse -> style",
			Rule{Category: "linter-abuse"},
			DimensionStyle},
		{"category fallback: content-quality -> compliance",
			Rule{Category: "content-quality"},
			DimensionCompliance},
		{"category fallback: auto-generated -> compliance",
			Rule{Category: "auto-generated"},
			DimensionCompliance},
		{"category fallback: cross-file-consistency -> compliance",
			Rule{Category: "cross-file-consistency"},
			DimensionCompliance},
		{"category fallback: stack-suggestions -> compliance",
			Rule{Category: "stack-suggestions"},
			DimensionCompliance},
		{"unknown category defaults to compliance",
			Rule{Category: "custom-unknown"},
			DimensionCompliance},
		{"no dimension or category defaults to compliance",
			Rule{},
			DimensionCompliance},
		{"good-practice with explicit dimension",
			Rule{Category: "good-practice", Dimension: DimensionStyle},
			DimensionStyle},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveDimension(tc.rule); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// =============================================================================
// CalculateDimensionScores
// =============================================================================

func dimResult(severity Severity, category string, dimension Dimension, passed bool) RuleResult {
	return RuleResult{
		Rule:   Rule{Severity: severity, Category: category, Dimension: dimension},
		Passed: passed,
	}
}

func TestCalculateDimensionScores(t *testing.T) {
	t.Run("no problems all 100", func(t *testing.T) {
		ds := CalculateDimensionScores(nil, 100)
		for _, dim := range AllDimensions() {
			if ds.Scores[dim].Score != 100 {
				t.Errorf("%s: got %d, want 100", dim, ds.Scores[dim].Score)
			}
		}
		if ds.Overall != 100 {
			t.Errorf("overall: got %d, want 100", ds.Overall)
		}
	})

	t.Run("single error deducts from correct dimension", func(t *testing.T) {
		results := []RuleResult{
			dimResult(SeverityError, "length", DimensionCorrectness, true),
		}
		ds := CalculateDimensionScores(results, 100)

		if ds.Scores[DimensionCorrectness].Score != 85 {
			t.Errorf("correctness: got %d, want 85", ds.Scores[DimensionCorrectness].Score)
		}
		if ds.Scores[DimensionStyle].Score != 100 {
			t.Errorf("style: got %d, want 100", ds.Scores[DimensionStyle].Score)
		}
	})

	t.Run("warning deducts 5", func(t *testing.T) {
		results := []RuleResult{
			dimResult(SeverityWarning, "linter-abuse", DimensionStyle, true),
		}
		ds := CalculateDimensionScores(results, 100)
		if ds.Scores[DimensionStyle].Score != 95 {
			t.Errorf("style: got %d, want 95", ds.Scores[DimensionStyle].Score)
		}
	})

	t.Run("info deducts 2", func(t *testing.T) {
		results := []RuleResult{
			dimResult(SeverityInfo, "content-quality", DimensionCompliance, true),
		}
		ds := CalculateDimensionScores(results, 100)
		if ds.Scores[DimensionCompliance].Score != 98 {
			t.Errorf("compliance: got %d, want 98", ds.Scores[DimensionCompliance].Score)
		}
	})

	t.Run("good-practice bonus +5", func(t *testing.T) {
		results := []RuleResult{
			dimResult(SeverityInfo, "good-practice", DimensionStyle, true),
		}
		ds := CalculateDimensionScores(results, 100)
		// 100 + 5 = 105, clamped to 100
		if ds.Scores[DimensionStyle].Score != 100 {
			t.Errorf("style: got %d, want 100 (clamped)", ds.Scores[DimensionStyle].Score)
		}
		if ds.Scores[DimensionStyle].Bonuses != 1 {
			t.Errorf("bonuses: got %d, want 1", ds.Scores[DimensionStyle].Bonuses)
		}
	})

	t.Run("good-practice bonus compensates deduction", func(t *testing.T) {
		results := []RuleResult{
			dimResult(SeverityWarning, "linter-abuse", DimensionStyle, true),  // -5
			dimResult(SeverityInfo, "good-practice", DimensionStyle, true),    // +5
		}
		ds := CalculateDimensionScores(results, 100)
		if ds.Scores[DimensionStyle].Score != 100 {
			t.Errorf("style: got %d, want 100", ds.Scores[DimensionStyle].Score)
		}
	})

	t.Run("floor at 0 with many errors", func(t *testing.T) {
		var results []RuleResult
		for i := 0; i < 10; i++ {
			results = append(results, dimResult(SeverityError, "length", DimensionCorrectness, true))
		}
		ds := CalculateDimensionScores(results, 100)
		if ds.Scores[DimensionCorrectness].Score != 0 {
			t.Errorf("correctness: got %d, want 0", ds.Scores[DimensionCorrectness].Score)
		}
	})

	t.Run("stale freshness impacts overall", func(t *testing.T) {
		ds := CalculateDimensionScores(nil, 25) // 180 days
		if ds.Scores[DimensionFreshness].Score != 25 {
			t.Errorf("freshness: got %d, want 25", ds.Scores[DimensionFreshness].Score)
		}
		// Overall = 100*0.4 + 100*0.2 + 100*0.2 + 25*0.2 = 40+20+20+5 = 85
		if ds.Overall != 85 {
			t.Errorf("overall: got %d, want 85", ds.Overall)
		}
	})

	t.Run("weighted overall with mixed dimensions", func(t *testing.T) {
		results := []RuleResult{
			dimResult(SeverityError, "length", DimensionCorrectness, true), // correctness: 85
		}
		ds := CalculateDimensionScores(results, 90)
		// 85*0.4 + 100*0.2 + 100*0.2 + 90*0.2 = 34+20+20+18 = 92
		if ds.Overall != 92 {
			t.Errorf("overall: got %d, want 92", ds.Overall)
		}
	})
}

// =============================================================================
// DefaultDimensionWeights
// =============================================================================

func TestDefaultDimensionWeights(t *testing.T) {
	weights := DefaultDimensionWeights()

	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if math.Abs(sum-1.0) > 0.001 {
		t.Errorf("weights sum to %f, want 1.0", sum)
	}

	for _, dim := range AllDimensions() {
		if _, ok := weights[dim]; !ok {
			t.Errorf("missing weight for dimension %q", dim)
		}
	}
}
