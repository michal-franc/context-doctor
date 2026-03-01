package rules

import (
	"path/filepath"
	"time"
)

// ScoreFromDays maps days since last modification to a freshness score.
func ScoreFromDays(days int) int {
	switch {
	case days <= 7:
		return 100
	case days <= 30:
		return 90
	case days <= 60:
		return 75
	case days <= 90:
		return 50
	case days <= 180:
		return 25
	case days <= 365:
		return 10
	default:
		return 0
	}
}

// CalculateFreshnessScore returns a freshness score and the number of days
// since the file was last modified in git. Returns (75, -1) if git history
// is unavailable.
func CalculateFreshnessScore(filePath string) (score int, days int) {
	lastMod := getGitLastModified(filePath)
	if lastMod.IsZero() {
		return 75, -1
	}
	days = int(time.Since(lastMod).Hours() / 24)
	return ScoreFromDays(days), days
}

// ScopeActivitySinceUpdate returns the number of commits in the CLAUDE.md's
// directory since the file was last updated, and the number of days since
// that update. Returns (0, -1) if git history is unavailable.
func ScopeActivitySinceUpdate(filePath string) (scopeCommits int, daysSinceUpdate int) {
	lastMod := getGitLastModified(filePath)
	if lastMod.IsZero() {
		return 0, -1
	}
	daysSinceUpdate = int(time.Since(lastMod).Hours() / 24)
	dir := filepath.Dir(filePath)
	scopeCommits = countCommitsSince(dir, lastMod.Format(time.RFC3339))
	return scopeCommits, daysSinceUpdate
}
