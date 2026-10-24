package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

// =============================================================================
// renderProgressBar
// =============================================================================

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name  string
		score int
		width int
		want  string
	}{
		{"score 0", 0, 10, "[░░░░░░░░░░]"},
		{"score 50", 50, 10, "[█████░░░░░]"},
		{"score 100", 100, 10, "[██████████]"},
		{"score 100 width 20", 100, 20, "[████████████████████]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderProgressBar(tc.score, tc.width); got != tc.want {
				t.Errorf("renderProgressBar(%d, %d) = %q, want %q", tc.score, tc.width, got, tc.want)
			}
		})
	}
}

// =============================================================================
// formatDimensionCompact
// =============================================================================

func TestFormatDimensionCompact(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		if got := formatDimensionCompact(nil); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("formats all dimensions", func(t *testing.T) {
		ds := rules.CalculateDimensionScores(nil, 90)
		got := formatDimensionCompact(ds)
		want := "[C:100 S:100 M:100 F:90]"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// =============================================================================
// findOrphanMDFiles
// =============================================================================

// writeMDFiles creates empty files under dir (no git repo, so the walk fallback is used).
func writeMDFiles(t *testing.T, dir string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("# doc"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindOrphanMDFiles_IgnorePatterns(t *testing.T) {
	dir := t.TempDir()
	writeMDFiles(t, dir, "README.md", "docs/guide.md", "notes/2026/jan.md", "sessions/log.md")

	tests := []struct {
		name   string
		ignore []string
		want   []string
	}{
		{"no ignore patterns", nil,
			[]string{"README.md", "docs/guide.md", "notes/2026/jan.md", "sessions/log.md"}},
		{"double star and trailing slash", []string{"notes/**", "sessions/"},
			[]string{"README.md", "docs/guide.md"}},
		{"single file pattern", []string{"README.md"},
			[]string{"docs/guide.md", "notes/2026/jan.md", "sessions/log.md"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findOrphanMDFiles(dir, dir, tt.ignore, nil)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindOrphanMDFiles_ScanningSubdirOfConfigRoot(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "project")
	writeMDFiles(t, sub, "notes/a.md", "docs/b.md")

	// Patterns are written relative to the config root, not the scanned dir
	got := findOrphanMDFiles(sub, root, []string{"project/notes/**"}, nil)
	want := []string{"docs/b.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRelativeDir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		base string
		dir  string
		want string
	}{
		{"same directory", root, root, "."},
		{"nested directory", root, sub, filepath.Join("a", "b")},
		{"dir outside base", sub, root, "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relativeDir(tt.base, tt.dir); got != tt.want {
				t.Errorf("relativeDir(%q, %q) = %q, want %q", tt.base, tt.dir, got, tt.want)
			}
		})
	}
}

// =============================================================================
// summarizeRepo / findAgentRoots
// =============================================================================

func scored(score, errors, warnings int) *fileAnalysis {
	return &fileAnalysis{
		Score:      score,
		Errors:     errors,
		Warnings:   warnings,
		AggMetrics: rules.AggregateMetrics{TotalLineCount: 10, TotalInstructionCount: 4},
	}
}

func TestSummarizeRepo(t *testing.T) {
	files := []*fileAnalysis{scored(92, 0, 1), scored(96, 1, 0), scored(98, 0, 0)}

	tests := []struct {
		name          string
		multipleFiles bool
		wantAvg       int
		wantRepo      int
		wantErrors    int
		wantIssues    []string
	}{
		{"single-file structure is fine", false, 95, 95, 1, nil},
		{"CD060 lowers only the repo score", true, 95, 65, 1, []string{"CD060"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeRepo(files, tt.multipleFiles)
			if got.AvgFileScore != tt.wantAvg || got.RepoScore != tt.wantRepo {
				t.Errorf("scores: avg %d repo %d, want avg %d repo %d", got.AvgFileScore, got.RepoScore, tt.wantAvg, tt.wantRepo)
			}
			if got.Errors != tt.wantErrors || got.Warnings != 1 {
				t.Errorf("errors/warnings: %d/%d, want %d/1", got.Errors, got.Warnings, tt.wantErrors)
			}
			if got.TotalLines != 30 || got.TotalInstructions != 12 {
				t.Errorf("totals: lines %d instr %d, want 30/12", got.TotalLines, got.TotalInstructions)
			}
			if !reflect.DeepEqual(got.StructureIssues, tt.wantIssues) {
				t.Errorf("issues: %v, want %v", got.StructureIssues, tt.wantIssues)
			}
		})
	}
}

func TestSummarizeRepo_PenaltyFloorsAtZero(t *testing.T) {
	got := summarizeRepo([]*fileAnalysis{scored(20, 0, 0)}, true)
	if got.RepoScore != 0 || got.AvgFileScore != 20 {
		t.Errorf("got avg %d repo %d, want avg 20 repo 0", got.AvgFileScore, got.RepoScore)
	}
}

func TestFindAgentRoots(t *testing.T) {
	root := t.TempDir()
	analysis := func(rel, content string) *fileAnalysis {
		return &fileAnalysis{
			FilePath: filepath.Join(root, rel),
			Ctx:      rules.BuildContext(rel, content),
		}
	}
	for _, d := range []string{"agent-a", "agent-b", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	analyses := []*fileAnalysis{
		analysis("CLAUDE.md", "# Root"),
		analysis("agent-a/CLAUDE.md", "# A"),
		analysis("agent-b/CLAUDE.md", rules.AgentRootMarker+"\n# B"),
		analysis("docs/CLAUDE.md", "# Docs"),
	}

	cfg = rules.Config{AgentRoots: []string{"agent-a/"}}
	t.Cleanup(func() { cfg = rules.Config{} })

	got := findAgentRoots(root, analyses)
	want := map[string]bool{
		filepath.Join(root, "agent-a/CLAUDE.md"): true,
		filepath.Join(root, "agent-b/CLAUDE.md"): true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
