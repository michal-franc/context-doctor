package rules

import (
	"regexp"
	"strings"
)

// ActionFunc is the signature for check action functions
type ActionFunc func(ctx *AnalysisContext, spec *MatchSpec) bool

// ActionRegistry holds all registered check actions
var ActionRegistry map[CheckAction]ActionFunc

func init() {
	ActionRegistry = map[CheckAction]ActionFunc{
		ActionLessThan:      checkLessThan,
		ActionGreaterThan:   checkGreaterThan,
		ActionEquals:        checkEquals,
		ActionNotEquals:     checkNotEquals,
		ActionContains:      checkContains,
		ActionNotContains:   checkNotContains,
		ActionRegexMatch:    checkRegexMatch,
		ActionRegexNotMatch: checkRegexNotMatch,
		ActionIsPresent:     checkIsPresent,
		ActionNotPresent:    checkNotPresent,
		ActionListContains:  checkListContains,
		ActionAnd:           checkAnd,
		ActionOr:            checkOr,
	}
}

// getMetricValue retrieves the value for a given metric
func getMetricValue(ctx *AnalysisContext, metric MetricType) any {
	switch metric {
	case MetricLineCount:
		return ctx.LineCount
	case MetricInstructionCount:
		return ctx.InstructionCount
	case MetricContent:
		return ctx.Content
	default:
		if val, ok := ctx.Metrics[string(metric)]; ok {
			return val
		}
		return nil
	}
}

// toInt converts any numeric value to int
func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

// toString converts any value to string
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return ""
	}
}

func checkLessThan(ctx *AnalysisContext, spec *MatchSpec) bool {
	metricVal := getMetricValue(ctx, spec.Metric)
	actual, ok := toInt(metricVal)
	if !ok {
		return false
	}
	threshold, ok := toInt(spec.Value)
	if !ok {
		return false
	}
	return actual < threshold
}

func checkGreaterThan(ctx *AnalysisContext, spec *MatchSpec) bool {
	metricVal := getMetricValue(ctx, spec.Metric)
	actual, ok := toInt(metricVal)
	if !ok {
		return false
	}
	threshold, ok := toInt(spec.Value)
	if !ok {
		return false
	}
	return actual > threshold
}

func checkEquals(ctx *AnalysisContext, spec *MatchSpec) bool {
	metricVal := getMetricValue(ctx, spec.Metric)
	return metricVal == spec.Value
}

func checkNotEquals(ctx *AnalysisContext, spec *MatchSpec) bool {
	metricVal := getMetricValue(ctx, spec.Metric)
	return metricVal != spec.Value
}

func checkContains(ctx *AnalysisContext, spec *MatchSpec) bool {
	content := ctx.Content
	if spec.Metric != "" && spec.Metric != MetricContent {
		content = toString(getMetricValue(ctx, spec.Metric))
	}

	// Check patterns if provided
	if len(spec.Patterns) > 0 {
		for _, pattern := range spec.Patterns {
			if strings.Contains(strings.ToLower(content), strings.ToLower(pattern)) {
				return true
			}
		}
		return false
	}

	// Check single value
	if spec.Value != nil {
		return strings.Contains(strings.ToLower(content), strings.ToLower(toString(spec.Value)))
	}

	return false
}

func checkNotContains(ctx *AnalysisContext, spec *MatchSpec) bool {
	content := ctx.Content
	if spec.Metric != "" && spec.Metric != MetricContent {
		content = toString(getMetricValue(ctx, spec.Metric))
	}

	// Check patterns if provided
	if len(spec.Patterns) > 0 {
		for _, pattern := range spec.Patterns {
			if strings.Contains(strings.ToLower(content), strings.ToLower(pattern)) {
				return false
			}
		}
		return true
	}

	// Check single value
	if spec.Value != nil {
		return !strings.Contains(strings.ToLower(content), strings.ToLower(toString(spec.Value)))
	}

	return true
}

func checkRegexMatch(ctx *AnalysisContext, spec *MatchSpec) bool {
	content := ctx.Content
	if spec.Metric != "" && spec.Metric != MetricContent {
		content = toString(getMetricValue(ctx, spec.Metric))
	}

	// Check patterns if provided
	if len(spec.Patterns) > 0 {
		for _, pattern := range spec.Patterns {
			re, err := regexp.Compile("(?i)" + pattern)
			if err != nil {
				continue
			}
			if re.MatchString(content) {
				return true
			}
		}
		return false
	}

	// Check single value
	if spec.Value != nil {
		re, err := regexp.Compile("(?i)" + toString(spec.Value))
		if err != nil {
			return false
		}
		return re.MatchString(content)
	}

	return false
}

func checkRegexNotMatch(ctx *AnalysisContext, spec *MatchSpec) bool {
	return !checkRegexMatch(ctx, spec)
}

func checkIsPresent(ctx *AnalysisContext, spec *MatchSpec) bool {
	// Check if patterns exist in content
	if len(spec.Patterns) > 0 {
		for _, pattern := range spec.Patterns {
			re, err := regexp.Compile("(?i)" + pattern)
			if err != nil {
				if strings.Contains(strings.ToLower(ctx.Content), strings.ToLower(pattern)) {
					return true
				}
				continue
			}
			if re.MatchString(ctx.Content) {
				return true
			}
		}
		return false
	}

	// Check single value
	if spec.Value != nil {
		return strings.Contains(ctx.Content, toString(spec.Value))
	}

	return false
}

func checkNotPresent(ctx *AnalysisContext, spec *MatchSpec) bool {
	return !checkIsPresent(ctx, spec)
}

func checkListContains(ctx *AnalysisContext, spec *MatchSpec) bool {
	metricVal := getMetricValue(ctx, spec.Metric)
	if metricVal == nil {
		return false
	}
	list, ok := metricVal.([]string)
	if !ok {
		return false
	}
	target := strings.ToLower(toString(spec.Value))
	if target == "" {
		return false
	}
	for _, item := range list {
		if strings.ToLower(item) == target {
			return true
		}
	}
	return false
}

func checkAnd(ctx *AnalysisContext, spec *MatchSpec) bool {
	if len(spec.SubMatch) == 0 {
		return true
	}
	for _, sub := range spec.SubMatch {
		if !EvaluateSpec(ctx, &sub) {
			return false
		}
	}
	return true
}

func checkOr(ctx *AnalysisContext, spec *MatchSpec) bool {
	if len(spec.SubMatch) == 0 {
		return false
	}
	for _, sub := range spec.SubMatch {
		if EvaluateSpec(ctx, &sub) {
			return true
		}
	}
	return false
}

// EvaluateSpec evaluates a MatchSpec against the context
func EvaluateSpec(ctx *AnalysisContext, spec *MatchSpec) bool {
	actionFn, ok := ActionRegistry[spec.Action]
	if !ok {
		return false
	}
	return actionFn(ctx, spec)
}
