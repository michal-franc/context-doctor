package rules

import (
	"path/filepath"
	"testing"
)

// =============================================================================
// ScoreFromDays
// =============================================================================

func TestScoreFromDays(t *testing.T) {
	tests := []struct {
		name string
		days int
		want int
	}{
		{"today", 0, 100},
		{"7 days (upper bound of first tier)", 7, 100},
		{"8 days (just past first tier)", 8, 90},
		{"30 days", 30, 90},
		{"31 days", 31, 75},
		{"60 days", 60, 75},
		{"61 days", 61, 50},
		{"90 days", 90, 50},
		{"91 days", 91, 25},
		{"180 days", 180, 25},
		{"181 days", 181, 10},
		{"365 days", 365, 10},
		{"366 days", 366, 0},
		{"1000 days", 1000, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ScoreFromDays(tc.days); got != tc.want {
				t.Errorf("ScoreFromDays(%d) = %d, want %d", tc.days, got, tc.want)
			}
		})
	}
}

// =============================================================================
// ScopeActivitySinceUpdate
// =============================================================================

func TestScopeActivitySinceUpdate_NoGitHistory(t *testing.T) {
	tmpDir := t.TempDir()
	fakePath := filepath.Join(tmpDir, "CLAUDE.md")

	scopeCommits, days := ScopeActivitySinceUpdate(fakePath)
	if scopeCommits != 0 {
		t.Errorf("expected 0 scope commits, got %d", scopeCommits)
	}
	if days != -1 {
		t.Errorf("expected -1 days, got %d", days)
	}
}
