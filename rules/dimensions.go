package rules

// DimensionScoreResult holds the score breakdown for a single dimension.
type DimensionScoreResult struct {
	Dimension  Dimension
	Score      int
	Violations int
	Bonuses    int
}

// DimensionScores holds per-dimension scores and the weighted overall.
type DimensionScores struct {
	Scores  map[Dimension]*DimensionScoreResult
	Overall int
}

// categoryToDimension maps rule categories to dimensions for custom rules
// that don't set an explicit dimension field.
var categoryToDimension = map[string]Dimension{
	"length":                  DimensionCorrectness,
	"instructions":            DimensionCorrectness,
	"referenced-docs":         DimensionCorrectness,
	"linter-abuse":            DimensionStyle,
	"progressive-disclosure":  DimensionStyle,
	"generic-advice":          DimensionStyle,
	"auto-generated":          DimensionCompliance,
	"content-quality":         DimensionCompliance,
	"cross-file-consistency":  DimensionCompliance,
}

// ResolveDimension returns the dimension for a rule. It prefers the explicit
// YAML field, then falls back to category-based mapping, and finally defaults
// to compliance.
func ResolveDimension(r Rule) Dimension {
	if r.Dimension != "" {
		return r.Dimension
	}
	if d, ok := categoryToDimension[r.Category]; ok {
		return d
	}
	return DimensionCompliance
}

// DefaultDimensionWeights returns the default scoring weights per dimension.
func DefaultDimensionWeights() map[Dimension]float64 {
	return map[Dimension]float64{
		DimensionCorrectness: 0.40,
		DimensionStyle:       0.20,
		DimensionCompliance:  0.20,
		DimensionFreshness:   0.20,
	}
}

// CalculateDimensionScores computes per-dimension scores from rule results
// and a freshness score, then produces a weighted overall score.
func CalculateDimensionScores(results []RuleResult, freshnessScore int) *DimensionScores {
	ds := &DimensionScores{
		Scores: make(map[Dimension]*DimensionScoreResult),
	}

	// Initialise every dimension at 100.
	for _, dim := range AllDimensions() {
		ds.Scores[dim] = &DimensionScoreResult{
			Dimension: dim,
			Score:     100,
		}
	}

	for _, r := range results {
		dim := ResolveDimension(r.Rule)

		entry := ds.Scores[dim]
		if entry == nil {
			// Safety: should not happen, but create if missing.
			entry = &DimensionScoreResult{Dimension: dim, Score: 100}
			ds.Scores[dim] = entry
		}

		if r.Rule.Category == "good-practice" {
			if r.Passed {
				entry.Bonuses++
				entry.Score += 5
			}
			continue
		}

		// Problem detected (Passed == true means the bad pattern matched).
		if r.Passed {
			entry.Violations++
			switch r.Rule.Severity {
			case SeverityError:
				entry.Score -= 15
			case SeverityWarning:
				entry.Score -= 5
			case SeverityInfo:
				entry.Score -= 2
			}
		}
	}

	// Apply freshness directly.
	ds.Scores[DimensionFreshness].Score = freshnessScore

	// Clamp all to [0, 100].
	for _, entry := range ds.Scores {
		if entry.Score < 0 {
			entry.Score = 0
		}
		if entry.Score > 100 {
			entry.Score = 100
		}
	}

	// Weighted average for Overall.
	weights := DefaultDimensionWeights()
	total := 0.0
	for dim, w := range weights {
		if entry, ok := ds.Scores[dim]; ok {
			total += float64(entry.Score) * w
		}
	}
	ds.Overall = int(total + 0.5) // round to nearest

	return ds
}
