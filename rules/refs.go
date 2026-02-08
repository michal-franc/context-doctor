package rules

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RefInfo holds information about a referenced documentation file
type RefInfo struct {
	Path            string           // referenced path (e.g., "docs/architecture.md")
	ResolvedPath    string           // absolute path on disk
	Exists          bool             // whether the file exists
	LastModified    time.Time        // last modification time
	DaysSinceUpdate int              // days since last update
	IsStale         bool             // exceeds stale threshold
	Context         *AnalysisContext // analysis context if file exists
}

// ResolveReferences takes progressive disclosure refs from the context and resolves them
func ResolveReferences(ctx *AnalysisContext, baseDir string, staleThresholdDays int) []RefInfo {
	rawRefs, ok := ctx.Metrics["progressiveDisclosureRefs"].([]string)
	if !ok || len(rawRefs) == 0 {
		return nil
	}

	// Deduplicate refs
	seen := make(map[string]bool)
	var refs []RefInfo

	for _, ref := range rawRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true

		resolved := filepath.Join(baseDir, ref)
		info := RefInfo{
			Path:         ref,
			ResolvedPath: resolved,
		}

		stat, err := os.Stat(resolved)
		if err != nil {
			// File doesn't exist
			refs = append(refs, info)
			continue
		}

		info.Exists = true

		// Try git log first for last modified time
		lastMod := getGitLastModified(resolved)
		if lastMod.IsZero() {
			lastMod = stat.ModTime()
		}
		info.LastModified = lastMod
		info.DaysSinceUpdate = int(time.Since(lastMod).Hours() / 24)
		info.IsStale = staleThresholdDays > 0 && info.DaysSinceUpdate > staleThresholdDays

		// Build analysis context for the referenced file
		content, err := os.ReadFile(resolved)
		if err == nil {
			info.Context = BuildContext(resolved, string(content))
		}

		refs = append(refs, info)
	}

	return refs
}

// getGitLastModified tries to get the last commit date for a file using git
func getGitLastModified(filePath string) time.Time {
	cmd := exec.Command("git", "log", "-1", "--format=%ci", filePath)
	cmd.Dir = filepath.Dir(filePath)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return time.Time{}
	}

	// Parse git date format: "2024-01-15 10:30:00 +0100"
	t, err := time.Parse("2006-01-02 15:04:05 -0700", trimmed)
	if err != nil {
		return time.Time{}
	}
	return t
}

// EnrichContextWithRefMetrics adds reference-related metrics to the context
func EnrichContextWithRefMetrics(ctx *AnalysisContext, refs []RefInfo) {
	brokenCount := 0
	staleCount := 0
	var refFiles []string

	for _, ref := range refs {
		refFiles = append(refFiles, ref.Path)
		if !ref.Exists {
			brokenCount++
		} else if ref.IsStale {
			staleCount++
		}
	}

	ctx.Metrics["broken_references_count"] = brokenCount
	ctx.Metrics["stale_references_count"] = staleCount
	ctx.Metrics["referenced_files"] = refFiles
}
