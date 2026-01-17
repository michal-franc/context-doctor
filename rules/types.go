package rules

// Severity levels for rules
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// CheckAction defines the type of check to perform
type CheckAction string

const (
	ActionLessThan      CheckAction = "lessThan"
	ActionGreaterThan   CheckAction = "greaterThan"
	ActionEquals        CheckAction = "equals"
	ActionNotEquals     CheckAction = "notEquals"
	ActionContains      CheckAction = "contains"
	ActionNotContains   CheckAction = "notContains"
	ActionRegexMatch    CheckAction = "regexMatch"
	ActionRegexNotMatch CheckAction = "regexNotMatch"
	ActionIsPresent     CheckAction = "isPresent"
	ActionNotPresent    CheckAction = "notPresent"
	ActionAnd           CheckAction = "and"
	ActionOr            CheckAction = "or"
)

// MetricType defines what metric to check
type MetricType string

const (
	MetricLineCount        MetricType = "lineCount"
	MetricInstructionCount MetricType = "instructionCount"
	MetricContent          MetricType = "content"
)

// MatchSpec defines how to match/check a condition
type MatchSpec struct {
	Metric   MetricType  `yaml:"metric,omitempty" json:"metric,omitempty"`
	Action   CheckAction `yaml:"action" json:"action"`
	Value    any         `yaml:"value,omitempty" json:"value,omitempty"`
	Patterns []string    `yaml:"patterns,omitempty" json:"patterns,omitempty"`
	SubMatch []MatchSpec `yaml:"subMatch,omitempty" json:"subMatch,omitempty"`
}

// Rule defines a single check rule
type Rule struct {
	Code         string    `yaml:"code" json:"code"`
	Description  string    `yaml:"description" json:"description"`
	Severity     Severity  `yaml:"severity" json:"severity"`
	Category     string    `yaml:"category,omitempty" json:"category,omitempty"`
	MatchSpec    MatchSpec `yaml:"matchSpec" json:"matchSpec"`
	ErrorMessage string    `yaml:"errorMessage" json:"errorMessage"`
	Suggestion   string    `yaml:"suggestion,omitempty" json:"suggestion,omitempty"`
	Links        []string  `yaml:"links,omitempty" json:"links,omitempty"`
}

// RulesFile represents a file containing rules
type RulesFile struct {
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	Rules   []Rule `yaml:"rules" json:"rules"`
}

// RuleResult represents the result of evaluating a rule
type RuleResult struct {
	Rule    Rule
	Passed  bool
	Message string
	Details map[string]any
}

// AnalysisContext holds all computed metrics for rule evaluation
type AnalysisContext struct {
	FilePath         string
	Content          string
	Lines            []string
	LineCount        int
	InstructionCount int
	Metrics          map[string]any
}
