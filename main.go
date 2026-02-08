package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"context-doctor/rules"
)

var (
	// Set by ldflags at build time
	Version   = "dev"
	BuildTime = "unknown"
)

var (
	customRulesDir  string
	noBuiltin       bool
	verbose         bool
	showScore       bool
	categoriesFlag  string
	severitiesFlag  string
	showVersion     bool
	staleThreshold  int
)

func init() {
	flag.StringVar(&customRulesDir, "rules-dir", "", "Directory containing custom rules (default: .context-doctor/)")
	flag.BoolVar(&noBuiltin, "no-builtin", false, "Disable built-in rules")
	flag.BoolVar(&verbose, "verbose", false, "Show detailed output including passed checks")
	flag.BoolVar(&showScore, "score", true, "Show overall score")
	flag.StringVar(&categoriesFlag, "categories", "", "Filter by categories (comma-separated)")
	flag.StringVar(&severitiesFlag, "severities", "", "Filter by severities (comma-separated: error,warning,info)")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.IntVar(&staleThreshold, "stale-threshold", 90, "Days before a referenced doc is considered stale")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("context-doctor %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		fmt.Println("Usage: context-doctor [options] <path-to-CLAUDE.md | directory>")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	target := flag.Arg(0)

	// Check if target is a directory
	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		files := findCLAUDEFiles(target)
		if len(files) == 0 {
			fmt.Fprintf(os.Stderr, "No CLAUDE.md files found in %s\n", target)
			os.Exit(1)
		}
		printRepoReport(target, files)
	} else {
		analyzeFile(target)
	}
}

// findCLAUDEFiles finds all CLAUDE.md files in a directory, respecting .gitignore
func findCLAUDEFiles(dir string) []string {
	// Try git ls-files first — respects .gitignore automatically
	if files := findCLAUDEFilesGit(dir); files != nil {
		return files
	}
	// Fallback for non-git directories
	return findCLAUDEFilesWalk(dir)
}

func findCLAUDEFilesGit(dir string) []string {
	// --cached: tracked files, --others: untracked, --exclude-standard: respect .gitignore
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "CLAUDE.md", "*/CLAUDE.md")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil // not a git repo or git not available
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return []string{}
	}

	var files []string
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, filepath.Join(dir, line))
		}
	}
	return files
}

// findAllMDFiles finds all .md files in a directory, respecting .gitignore
func findAllMDFiles(dir string) []string {
	if files := findAllMDFilesGit(dir); files != nil {
		return files
	}
	return findAllMDFilesWalk(dir)
}

func findAllMDFilesGit(dir string) []string {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "*.md", "**/*.md")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return []string{}
	}

	var files []string
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

func findAllMDFilesWalk(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base != "." && strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				rel = path
			}
			files = append(files, rel)
		}
		return nil
	})
	return files
}

// findOrphanMDFiles returns .md files not referenced by any CLAUDE.md and not CLAUDE.md themselves
func findOrphanMDFiles(dir string, analyses []*fileAnalysis) []string {
	allMD := findAllMDFiles(dir)

	// Build set of referenced paths (relative to dir)
	referenced := make(map[string]bool)
	for _, fa := range analyses {
		// Mark the CLAUDE.md file itself
		rel, err := filepath.Rel(dir, fa.FilePath)
		if err != nil {
			rel = fa.FilePath
		}
		referenced[rel] = true

		// Mark all files it references
		for _, ref := range fa.Refs {
			referenced[ref.Path] = true
		}
	}

	// Common non-context files to skip
	skip := map[string]bool{
		"README.md": true, "readme.md": true,
		"CHANGELOG.md": true, "changelog.md": true,
		"LICENSE.md": true, "license.md": true,
		"CONTRIBUTING.md": true, "contributing.md": true,
		"CODE_OF_CONDUCT.md": true,
		"SECURITY.md": true,
	}

	var orphans []string
	for _, md := range allMD {
		if referenced[md] {
			continue
		}
		base := filepath.Base(md)
		if skip[base] {
			continue
		}
		orphans = append(orphans, md)
	}
	return orphans
}

func findCLAUDEFilesWalk(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base != "." && strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "CLAUDE.md" {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// fileAnalysis holds all analysis results for a single CLAUDE.md file
type fileAnalysis struct {
	FilePath   string
	Ctx        *rules.AnalysisContext
	Results    []rules.RuleResult
	Refs       []rules.RefInfo
	RefResults map[string][]rules.RuleResult
	AggMetrics rules.AggregateMetrics
	Score      int
	Errors     int
	Warnings   int
}

func buildAnalysis(filePath string) (*fileAnalysis, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	rulesDir := customRulesDir
	if rulesDir == "" {
		rulesDir = filepath.Dir(filePath)
	}

	allRules, err := rules.LoadAllRules(rulesDir, !noBuiltin)
	if err != nil {
		return nil, err
	}

	ctx := rules.BuildContext(filePath, string(content))

	baseDir := filepath.Dir(filePath)
	refs := rules.ResolveReferences(ctx, baseDir, staleThreshold)
	rules.EnrichContextWithRefMetrics(ctx, refs)

	aggMetrics := rules.ComputeAggregateMetrics(ctx, refs)
	ctx.Metrics["total_instruction_count"] = aggMetrics.TotalInstructionCount
	ctx.Metrics["duplicate_instruction_count"] = len(aggMetrics.Duplicates)

	engine := rules.NewEngine(allRules)
	results := engine.Evaluate(ctx)

	var refResults map[string][]rules.RuleResult
	if len(refs) > 0 {
		refResults = make(map[string][]rules.RuleResult)
		for _, ref := range refs {
			if ref.Exists && ref.Context != nil {
				refResults[ref.Path] = engine.Evaluate(ref.Context)
			}
		}
	}

	score := calculateScore(ctx, results)

	errors := 0
	warnings := 0
	for _, r := range results {
		if r.Rule.Category == "good-practice" || !r.Passed {
			continue
		}
		switch r.Rule.Severity {
		case rules.SeverityError:
			errors++
		case rules.SeverityWarning:
			warnings++
		}
	}

	return &fileAnalysis{
		FilePath:   filePath,
		Ctx:        ctx,
		Results:    results,
		Refs:       refs,
		RefResults: refResults,
		AggMetrics: aggMetrics,
		Score:      score,
		Errors:     errors,
		Warnings:   warnings,
	}, nil
}

func analyzeFile(filePath string) {
	fa, err := buildAnalysis(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	filterOpts := buildFilterOpts()
	printReport(fa.Ctx, fa.Results, filterOpts, fa.Refs, fa.RefResults, fa.AggMetrics)
}

func printRepoReport(dir string, files []string) {
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("  Repository Context Report")
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println()

	var analyses []*fileAnalysis
	for _, f := range files {
		fa, err := buildAnalysis(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error analyzing %s: %v\n", f, err)
			continue
		}
		analyses = append(analyses, fa)
	}

	if len(analyses) == 0 {
		fmt.Println("  No files could be analyzed.")
		return
	}

	// Multiple CLAUDE.md violation
	if len(analyses) > 1 {
		fmt.Println("✗ [CD060] MULTIPLE CLAUDE.md FILES DETECTED")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println("  A repository should have exactly one CLAUDE.md at the root.")
		fmt.Println("  Multiple files fragment context and confuse the LLM.")
		fmt.Println("  Consolidate into root CLAUDE.md and use progressive")
		fmt.Println("  disclosure to reference supporting docs.")
		fmt.Println()
		for _, fa := range analyses {
			relPath, err := filepath.Rel(dir, fa.FilePath)
			if err != nil {
				relPath = fa.FilePath
			}
			fmt.Printf("  ✗ %s\n", relPath)
		}
		fmt.Println()
	}

	// Summary table
	fmt.Printf("FILES (%d CLAUDE.md found)\n", len(analyses))
	fmt.Println(strings.Repeat("-", 40))

	totalScore := 0
	totalErrors := 0
	totalWarnings := 0
	totalInstructions := 0
	totalLines := 0

	for _, fa := range analyses {
		relPath, err := filepath.Rel(dir, fa.FilePath)
		if err != nil {
			relPath = fa.FilePath
		}

		icon := "✓"
		if fa.Errors > 0 {
			icon = "✗"
		} else if fa.Warnings > 0 {
			icon = "⚠"
		}

		fmt.Printf("  %s %s\n", icon, relPath)
		fmt.Printf("      Score: %d/100  Lines: %d  Instructions: ~%d  Errors: %d  Warnings: %d\n",
			fa.Score, fa.Ctx.LineCount, fa.Ctx.InstructionCount, fa.Errors, fa.Warnings)

		// Show referenced docs inline
		if len(fa.Refs) > 0 {
			for _, ref := range fa.Refs {
				if !ref.Exists {
					fmt.Printf("      ✗ ref: %s (not found!)\n", ref.Path)
				} else if ref.IsStale {
					fmt.Printf("      ⚠ ref: %s (stale — %d days)\n", ref.Path, ref.DaysSinceUpdate)
				} else {
					fmt.Printf("      ✓ ref: %s (%d days ago)\n", ref.Path, ref.DaysSinceUpdate)
				}
			}
		}

		totalScore += fa.Score
		totalErrors += fa.Errors
		totalWarnings += fa.Warnings
		totalInstructions += fa.AggMetrics.TotalInstructionCount
		totalLines += fa.AggMetrics.TotalLineCount
	}
	fmt.Println()

	// Issues section — only show files that have problems
	hasIssues := false
	for _, fa := range analyses {
		if fa.Errors == 0 && fa.Warnings == 0 {
			continue
		}

		relPath, err := filepath.Rel(dir, fa.FilePath)
		if err != nil {
			relPath = fa.FilePath
		}

		if !hasIssues {
			fmt.Println("ISSUES")
			fmt.Println(strings.Repeat("-", 40))
			hasIssues = true
		}

		fmt.Printf("  %s\n", relPath)
		for _, r := range fa.Results {
			if r.Rule.Category == "good-practice" || !r.Passed {
				continue
			}
			if r.Rule.Severity == rules.SeverityInfo {
				continue
			}
			icon := getSeverityIcon(r.Rule.Severity)
			fmt.Printf("    %s [%s] %s\n", icon, r.Rule.Code, r.Rule.ErrorMessage)
		}
	}
	if hasIssues {
		fmt.Println()
	}

	// Orphan docs section
	orphans := findOrphanMDFiles(dir, analyses)
	if len(orphans) > 0 {
		fmt.Println("ORPHAN DOCS (not referenced by any CLAUDE.md)")
		fmt.Println(strings.Repeat("-", 40))
		for _, o := range orphans {
			fmt.Printf("  ? %s\n", o)
		}
		fmt.Println()
	}

	// Repo totals
	avgScore := totalScore / len(analyses)
	if len(analyses) > 1 {
		// Heavy penalty for multiple CLAUDE.md files
		avgScore = max(0, avgScore-30)
		totalErrors++
	}
	fmt.Println("REPO SUMMARY")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("  Files:        %d\n", len(analyses))
	fmt.Printf("  Total lines:  %d\n", totalLines)
	fmt.Printf("  Total instr:  ~%d\n", totalInstructions)
	fmt.Printf("  Errors:       %d\n", totalErrors)
	fmt.Printf("  Warnings:     %d\n", totalWarnings)
	fmt.Printf("  Avg score:    %d/100\n", avgScore)
	fmt.Println()
}

func buildFilterOpts() rules.FilterOptions {
	filterOpts := rules.FilterOptions{
		FailuresOnly:     !verbose,
		HideGoodPractice: !verbose,
	}
	if categoriesFlag != "" {
		filterOpts.Categories = strings.Split(categoriesFlag, ",")
	}
	if severitiesFlag != "" {
		for _, s := range strings.Split(severitiesFlag, ",") {
			filterOpts.Severities = append(filterOpts.Severities, rules.Severity(strings.TrimSpace(s)))
		}
	}
	return filterOpts
}

func printReport(ctx *rules.AnalysisContext, results []rules.RuleResult, filterOpts rules.FilterOptions, refs []rules.RefInfo, refResults map[string][]rules.RuleResult, aggMetrics rules.AggregateMetrics) {
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("  CLAUDE.md Analysis Report")
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println()

	fmt.Printf("File: %s\n\n", ctx.FilePath)

	// Metrics section
	fmt.Println("METRICS")
	fmt.Println(strings.Repeat("-", 40))

	lineStatus := "OK"
	if ctx.LineCount > 300 {
		lineStatus = "HIGH"
	} else if ctx.LineCount > 100 {
		lineStatus = "MODERATE"
	}
	fmt.Printf("  Lines:        %d (%s)\n", ctx.LineCount, lineStatus)

	effective := ctx.InstructionCount + 50
	instrStatus := "OK"
	if effective > 150 {
		instrStatus = "HIGH"
	} else if effective > 100 {
		instrStatus = "MODERATE"
	}
	fmt.Printf("  Instructions: ~%d (+50 Claude = ~%d) (%s)\n", ctx.InstructionCount, effective, instrStatus)

	hasProgDisc := ctx.Metrics["hasProgressiveDisclosure"].(bool)
	pdStatus := "NO"
	if hasProgDisc {
		pdStatus = "YES"
	}
	fmt.Printf("  Progressive Disclosure: %s\n", pdStatus)
	fmt.Println()

	// Group results by category
	problemsByCategory := make(map[string][]rules.RuleResult)
	goodPractices := []rules.RuleResult{}

	for _, r := range results {
		if r.Rule.Category == "good-practice" {
			if r.Passed {
				goodPractices = append(goodPractices, r)
			}
			continue
		}

		// For problem rules, "passed" means the problem was detected
		if r.Passed {
			cat := r.Rule.Category
			if cat == "" {
				cat = "other"
			}
			problemsByCategory[cat] = append(problemsByCategory[cat], r)
		}
	}

	// Print problems by category
	categoryOrder := []string{"length", "instructions", "linter-abuse", "auto-generated", "progressive-disclosure", "referenced-docs", "cross-file-consistency"}
	categoryNames := map[string]string{
		"length":                  "LENGTH ISSUES",
		"instructions":            "INSTRUCTION COUNT ISSUES",
		"linter-abuse":            "LINTER ABUSE DETECTED",
		"auto-generated":          "AUTO-GENERATED CONTENT",
		"progressive-disclosure":  "PROGRESSIVE DISCLOSURE",
		"referenced-docs":         "REFERENCED DOCS",
		"cross-file-consistency":  "CROSS-FILE CONSISTENCY",
	}

	// Add any custom categories found in results
	for _, r := range results {
		if r.Rule.Category != "" && r.Rule.Category != "good-practice" {
			found := false
			for _, c := range categoryOrder {
				if c == r.Rule.Category {
					found = true
					break
				}
			}
			if !found {
				categoryOrder = append(categoryOrder, r.Rule.Category)
			}
		}
	}
	categoryOrder = append(categoryOrder, "other")

	hasProblems := false
	for _, cat := range categoryOrder {
		problems, ok := problemsByCategory[cat]
		if !ok || len(problems) == 0 {
			continue
		}

		// Apply filters
		if len(filterOpts.Categories) > 0 {
			found := false
			for _, c := range filterOpts.Categories {
				if c == cat {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		hasProblems = true
		name := categoryNames[cat]
		if name == "" {
			name = strings.ToUpper(cat)
		}
		fmt.Println(name)
		fmt.Println(strings.Repeat("-", 40))

		for _, p := range problems {
			// Apply severity filter
			if len(filterOpts.Severities) > 0 {
				found := false
				for _, s := range filterOpts.Severities {
					if p.Rule.Severity == s {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			severityIcon := getSeverityIcon(p.Rule.Severity)
			fmt.Printf("  %s [%s] %s\n", severityIcon, p.Rule.Code, p.Rule.ErrorMessage)
			if p.Rule.Suggestion != "" {
				fmt.Printf("     → %s\n", p.Rule.Suggestion)
			}
		}
		fmt.Println()
	}

	// Print good practices if verbose
	if verbose && len(goodPractices) > 0 {
		fmt.Println("GOOD PRACTICES DETECTED")
		fmt.Println(strings.Repeat("-", 40))
		for _, p := range goodPractices {
			fmt.Printf("  ✓ [%s] %s\n", p.Rule.Code, p.Rule.ErrorMessage)
		}
		fmt.Println()
	}

	// Print referenced docs section
	if len(refs) > 0 {
		printReferencedDocs(refs)
		printReferencedDocIssues(refs, refResults, filterOpts)
	} else {
		// Print progressive disclosure refs if found (legacy section for when refs aren't resolved)
		if rawRefs, ok := ctx.Metrics["progressiveDisclosureRefs"].([]string); ok && len(rawRefs) > 0 {
			fmt.Println("PROGRESSIVE DISCLOSURE REFERENCES")
			fmt.Println(strings.Repeat("-", 40))
			for _, ref := range rawRefs {
				fmt.Printf("  - %s\n", ref)
			}
			fmt.Println()
		}
	}

	// Print cross-file analysis section
	if len(refs) > 0 {
		printCrossFileAnalysis(aggMetrics)
	}

	// Calculate and print score
	if showScore {
		score := calculateScore(ctx, results)
		fmt.Println("OVERALL SCORE")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  %d/100\n", score)

		if !hasProblems && score == 100 {
			fmt.Println("\n  ✓ Excellent! Your CLAUDE.md follows best practices.")
		}
		fmt.Println()
	}
}

func printReferencedDocs(refs []rules.RefInfo) {
	fmt.Println("REFERENCED DOCS")
	fmt.Println(strings.Repeat("-", 40))

	for _, ref := range refs {
		if !ref.Exists {
			fmt.Printf("  ✗ %s (file not found!)\n", ref.Path)
		} else if ref.IsStale {
			fmt.Printf("  ⚠ %s (last updated %d days ago — stale)\n", ref.Path, ref.DaysSinceUpdate)
		} else {
			fmt.Printf("  ✓ %s (last updated %d days ago)\n", ref.Path, ref.DaysSinceUpdate)
		}
	}
	fmt.Println()
}

func printReferencedDocIssues(refs []rules.RefInfo, refResults map[string][]rules.RuleResult, filterOpts rules.FilterOptions) {
	if refResults == nil {
		return
	}

	for _, ref := range refs {
		if !ref.Exists {
			continue
		}
		results, ok := refResults[ref.Path]
		if !ok {
			continue
		}

		// Collect issues for this ref
		var issues []rules.RuleResult
		for _, r := range results {
			if r.Rule.Category == "good-practice" {
				continue
			}
			// Skip referenced-docs and cross-file-consistency rules for sub-files
			if r.Rule.Category == "referenced-docs" || r.Rule.Category == "cross-file-consistency" {
				continue
			}
			if r.Passed {
				issues = append(issues, r)
			}
		}

		if len(issues) == 0 {
			continue
		}

		fmt.Printf("REFERENCED DOC ISSUES: %s\n", ref.Path)
		fmt.Println(strings.Repeat("-", 40))

		for _, p := range issues {
			if len(filterOpts.Severities) > 0 {
				found := false
				for _, s := range filterOpts.Severities {
					if p.Rule.Severity == s {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			severityIcon := getSeverityIcon(p.Rule.Severity)
			fmt.Printf("  %s [%s] %s\n", severityIcon, p.Rule.Code, p.Rule.ErrorMessage)
			if p.Rule.Suggestion != "" {
				fmt.Printf("     → %s\n", p.Rule.Suggestion)
			}
		}
		fmt.Println()
	}
}

func printCrossFileAnalysis(agg rules.AggregateMetrics) {
	fmt.Println("CROSS-FILE ANALYSIS")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("  Total instructions across %d files: %d\n", agg.FileCount, agg.TotalInstructionCount)
	fmt.Printf("  Total lines across %d files: %d\n", agg.FileCount, agg.TotalLineCount)

	if len(agg.Duplicates) > 0 {
		fmt.Printf("  ⚠ %d duplicated instructions found across files\n", len(agg.Duplicates))
		for _, dup := range agg.Duplicates {
			fmt.Printf("     → \"%s\" in %s\n", truncate(dup.Instruction, 60), strings.Join(dup.Files, ", "))
		}
	}
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getSeverityIcon(severity rules.Severity) string {
	switch severity {
	case rules.SeverityError:
		return "✗"
	case rules.SeverityWarning:
		return "⚠"
	case rules.SeverityInfo:
		return "ℹ"
	default:
		return " "
	}
}

func calculateScore(ctx *rules.AnalysisContext, results []rules.RuleResult) int {
	score := 100

	for _, r := range results {
		if r.Rule.Category == "good-practice" {
			continue
		}

		// Problem was detected
		if r.Passed {
			switch r.Rule.Severity {
			case rules.SeverityError:
				score -= 15
			case rules.SeverityWarning:
				score -= 5
			case rules.SeverityInfo:
				score -= 2
			}
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}
