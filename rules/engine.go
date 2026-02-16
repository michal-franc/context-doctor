package rules

import (
	"regexp"
	"strings"
)

// imperativeVerbPattern matches lines that look like instructions
var imperativeVerbPattern = regexp.MustCompile(`(?i)^[-*]?\s*(always|never|do not|don't|must|should|ensure|make sure|use|avoid|prefer|run|execute|check|verify|include|exclude|add|remove|create|delete|update|follow|implement)`)

// Engine evaluates rules against analysis context
type Engine struct {
	Rules []Rule
}

// NewEngine creates a new rules engine with the given rules
func NewEngine(rules []Rule) *Engine {
	return &Engine{Rules: rules}
}

// Evaluate runs all rules against the given context
func (e *Engine) Evaluate(ctx *AnalysisContext) []RuleResult {
	var results []RuleResult

	for _, rule := range e.Rules {
		result := e.evaluateRule(ctx, rule)
		results = append(results, result)
	}

	return results
}

// EvaluateSecondary runs only non-primaryOnly rules (for referenced docs)
func (e *Engine) EvaluateSecondary(ctx *AnalysisContext) []RuleResult {
	var results []RuleResult

	for _, rule := range e.Rules {
		if rule.PrimaryOnly {
			continue
		}
		result := e.evaluateRule(ctx, rule)
		results = append(results, result)
	}

	return results
}

func (e *Engine) evaluateRule(ctx *AnalysisContext, rule Rule) RuleResult {
	passed := EvaluateSpec(ctx, &rule.MatchSpec)

	// For "positive" checks (good practices), passing means the pattern was found
	// For "negative" checks (problems), passing means the problem was found
	// We need to invert for problem-detection rules
	isPositiveRule := rule.Category == "good-practice"

	result := RuleResult{
		Rule:    rule,
		Passed:  passed,
		Details: make(map[string]any),
	}

	if isPositiveRule {
		// Good practice rule: passing = good
		if passed {
			result.Message = rule.ErrorMessage // This is actually a positive message
		}
	} else {
		// Problem detection rule: passing = problem found
		if passed {
			result.Message = rule.ErrorMessage
		}
	}

	return result
}

// BuildContext creates an AnalysisContext from file content
func BuildContext(filePath string, content string) *AnalysisContext {
	lines := strings.Split(content, "\n")

	ctx := &AnalysisContext{
		FilePath:         filePath,
		Content:          content,
		Lines:            lines,
		LineCount:        len(lines),
		InstructionCount: CountInstructions(lines),
		Metrics:          make(map[string]any),
	}

	// Add derived metrics
	ctx.Metrics["hasProgressiveDisclosure"] = hasProgressiveDisclosure(content)
	ctx.Metrics["progressiveDisclosureRefs"] = findProgressiveDisclosureRefs(content)

	return ctx
}

// listItemPattern matches list items (bullets and numbered)
var listItemPattern = regexp.MustCompile(`^[-*]|\d+\.`)

// CountInstructions estimates the number of instructions in the content
func CountInstructions(lines []string) int {
	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if imperativeVerbPattern.MatchString(line) {
			count++
			continue
		}

		if listItemPattern.MatchString(line) {
			trimmed := strings.TrimLeft(line, "-*0123456789. ")
			if len(trimmed) > 10 {
				count++
			}
		}
	}

	return count
}

// hasProgressiveDisclosure checks if the content references other docs
func hasProgressiveDisclosure(content string) bool {
	patterns := []string{
		`(?i)see\s+[\w/.-]+\.md`,
		`(?i)refer\s+to\s+[\w/.-]+\.md`,
		`(?i)read\s+[\w/.-]+\.md`,
		`(?i)docs?/[\w/.-]+\.md`,
		`(?i)check\s+[\w/.-]+\.md`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(content) {
			return true
		}
	}

	return false
}

// findProgressiveDisclosureRefs extracts references to other docs
func findProgressiveDisclosureRefs(content string) []string {
	var refs []string

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)see\s+([\w/.-]+\.md)`),
		regexp.MustCompile(`(?i)refer\s+to\s+([\w/.-]+\.md)`),
		regexp.MustCompile(`(?i)read\s+([\w/.-]+\.md)`),
		regexp.MustCompile(`((?:\.\./)*docs?/[\w/.-]+\.md)`),
		regexp.MustCompile(`(?i)check\s+([\w/.-]+\.md)`),
		regexp.MustCompile(`[-*]\s*\x60?([\w/.-]+\.md)\x60?\s*[-:]`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				refs = append(refs, match[1])
			}
		}
	}

	return refs
}

// FilterResults filters results by various criteria
func FilterResults(results []RuleResult, opts FilterOptions) []RuleResult {
	var filtered []RuleResult

	for _, r := range results {
		// Skip passed rules if we only want failures
		if opts.FailuresOnly && !r.Passed {
			continue
		}

		// Skip if severity doesn't match
		if len(opts.Severities) > 0 {
			found := false
			for _, s := range opts.Severities {
				if r.Rule.Severity == s {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Skip if category doesn't match
		if len(opts.Categories) > 0 {
			found := false
			for _, c := range opts.Categories {
				if r.Rule.Category == c {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Skip good-practice rules in normal output (they're informational)
		if opts.HideGoodPractice && r.Rule.Category == "good-practice" {
			continue
		}

		filtered = append(filtered, r)
	}

	return filtered
}

// FilterOptions controls which results to include
type FilterOptions struct {
	FailuresOnly     bool
	Severities       []Severity
	Categories       []string
	HideGoodPractice bool
}
