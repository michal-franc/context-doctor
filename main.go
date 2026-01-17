package main

import (
	"flag"
	"fmt"
	"os"
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
)

func init() {
	flag.StringVar(&customRulesDir, "rules-dir", "", "Directory containing custom rules (default: .context-doctor/)")
	flag.BoolVar(&noBuiltin, "no-builtin", false, "Disable built-in rules")
	flag.BoolVar(&verbose, "verbose", false, "Show detailed output including passed checks")
	flag.BoolVar(&showScore, "score", true, "Show overall score")
	flag.StringVar(&categoriesFlag, "categories", "", "Filter by categories (comma-separated)")
	flag.StringVar(&severitiesFlag, "severities", "", "Filter by severities (comma-separated: error,warning,info)")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("context-doctor %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		fmt.Println("Usage: context-doctor [options] <path-to-CLAUDE.md>")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	filePath := flag.Arg(0)

	// Load file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Determine custom rules directory
	rulesDir := customRulesDir
	if rulesDir == "" {
		rulesDir = filepath.Dir(filePath)
	}

	// Load rules
	allRules, err := rules.LoadAllRules(rulesDir, !noBuiltin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
		os.Exit(1)
	}

	// Build analysis context
	ctx := rules.BuildContext(filePath, string(content))

	// Create engine and evaluate
	engine := rules.NewEngine(allRules)
	results := engine.Evaluate(ctx)

	// Filter results
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

	// Print report
	printReport(ctx, results, filterOpts)
}

func printReport(ctx *rules.AnalysisContext, results []rules.RuleResult, filterOpts rules.FilterOptions) {
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
	categoryOrder := []string{"length", "instructions", "linter-abuse", "auto-generated", "progressive-disclosure"}
	categoryNames := map[string]string{
		"length":                 "LENGTH ISSUES",
		"instructions":           "INSTRUCTION COUNT ISSUES",
		"linter-abuse":           "LINTER ABUSE DETECTED",
		"auto-generated":         "AUTO-GENERATED CONTENT",
		"progressive-disclosure": "PROGRESSIVE DISCLOSURE",
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
				fmt.Printf("     \u2192 %s\n", p.Rule.Suggestion)
			}
		}
		fmt.Println()
	}

	// Print good practices if verbose
	if verbose && len(goodPractices) > 0 {
		fmt.Println("GOOD PRACTICES DETECTED")
		fmt.Println(strings.Repeat("-", 40))
		for _, p := range goodPractices {
			fmt.Printf("  \u2713 [%s] %s\n", p.Rule.Code, p.Rule.ErrorMessage)
		}
		fmt.Println()
	}

	// Print progressive disclosure refs if found
	if refs, ok := ctx.Metrics["progressiveDisclosureRefs"].([]string); ok && len(refs) > 0 {
		fmt.Println("PROGRESSIVE DISCLOSURE REFERENCES")
		fmt.Println(strings.Repeat("-", 40))
		for _, ref := range refs {
			fmt.Printf("  - %s\n", ref)
		}
		fmt.Println()
	}

	// Calculate and print score
	if showScore {
		score := calculateScore(ctx, results)
		fmt.Println("OVERALL SCORE")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  %d/100\n", score)

		if !hasProblems && score == 100 {
			fmt.Println("\n  \u2713 Excellent! Your CLAUDE.md follows best practices.")
		}
		fmt.Println()
	}
}

func getSeverityIcon(severity rules.Severity) string {
	switch severity {
	case rules.SeverityError:
		return "\u2717"
	case rules.SeverityWarning:
		return "\u26a0"
	case rules.SeverityInfo:
		return "\u2139"
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
