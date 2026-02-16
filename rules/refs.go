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
	ReferencedBy    string           // which file references this one
	Depth           int              // depth in the reference tree (0 = direct from CLAUDE.md)
	Children        []RefInfo        // files referenced by this file
}

// ResolveReferences recursively resolves progressive disclosure refs from the context.
// It follows references in referenced files, detecting circular references.
func ResolveReferences(ctx *AnalysisContext, baseDir string, staleThresholdDays int) []RefInfo {
	seen := make(map[string]bool)
	repoRoot := getGitRoot(baseDir)
	return resolveRefsRecursive(ctx, baseDir, repoRoot, staleThresholdDays, ctx.FilePath, 0, seen)
}

func resolveRefsRecursive(ctx *AnalysisContext, baseDir string, repoRoot string, staleThresholdDays int, referencedBy string, depth int, seen map[string]bool) []RefInfo {
	rawRefs, ok := ctx.Metrics["progressiveDisclosureRefs"].([]string)
	if !ok || len(rawRefs) == 0 {
		return nil
	}

	var refs []RefInfo

	for _, ref := range rawRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}

		resolved := filepath.Join(baseDir, ref)

		// Fallback: if not found relative to baseDir, try repo root
		if repoRoot != "" {
			if _, err := os.Stat(resolved); err != nil {
				fromRoot := filepath.Join(repoRoot, ref)
				if _, err := os.Stat(fromRoot); err == nil {
					resolved = fromRoot
				}
			}
		}

		absResolved, err := filepath.Abs(resolved)
		if err != nil {
			absResolved = resolved
		}

		// Cycle detection
		if seen[absResolved] {
			continue
		}
		seen[absResolved] = true

		info := RefInfo{
			Path:         ref,
			ResolvedPath: resolved,
			ReferencedBy: referencedBy,
			Depth:        depth,
		}

		stat, err := os.Stat(resolved)
		if err != nil {
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

			// Recurse into this file's references
			childBaseDir := filepath.Dir(resolved)
			info.Children = resolveRefsRecursive(info.Context, childBaseDir, repoRoot, staleThresholdDays, ref, depth+1, seen)
		}

		refs = append(refs, info)
	}

	return refs
}

// FlattenRefs returns all refs in a flat list (depth-first), including children
func FlattenRefs(refs []RefInfo) []RefInfo {
	var flat []RefInfo
	for _, ref := range refs {
		flat = append(flat, ref)
		if len(ref.Children) > 0 {
			flat = append(flat, FlattenRefs(ref.Children)...)
		}
	}
	return flat
}

// getGitRoot returns the top-level directory of the git repository, or "" if not in a repo.
func getGitRoot(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// countCommitsSince counts commits in dir since the given date string (RFC3339 or git-compatible).
// Returns 0 on error (consistent with other git helpers).
func countCommitsSince(dir string, since string) int {
	cmd := exec.Command("git", "log", "--since="+since, "--oneline", "--", ".")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "\n"))
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
	allRefs := FlattenRefs(refs)

	brokenCount := 0
	staleCount := 0
	var refFiles []string

	for _, ref := range allRefs {
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
