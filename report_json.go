package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"context-doctor/rules"
)

// JSON output (-format json). Field names are part of the CLI contract:
// add fields freely, but don't rename or remove them.

type violationJSON struct {
	Code       string   `json:"code"`
	Severity   string   `json:"severity"`
	Category   string   `json:"category"`
	Dimension  string   `json:"dimension"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion,omitempty"`
	Links      []string `json:"links,omitempty"`
}

type metricsJSON struct {
	LineCount               int      `json:"lineCount"`
	InstructionCount        int      `json:"instructionCount"`
	InstructionDensityPct   int      `json:"instructionDensityPct"`
	ProgressiveDisclosure   bool     `json:"progressiveDisclosure"`
	TotalLineCount          int      `json:"totalLineCount"`
	TotalInstructionCount   int      `json:"totalInstructionCount"`
	ReferencedFileCount     int      `json:"referencedFileCount"`
	BrokenReferences        int      `json:"brokenReferences"`
	StaleReferences         int      `json:"staleReferences"`
	ScopeCommitsSinceUpdate int      `json:"scopeCommitsSinceUpdate"`
	DaysSinceUpdate         *int     `json:"daysSinceUpdate"`
	DetectedStacks          []string `json:"detectedStacks"`
}

type refJSON struct {
	Path            string          `json:"path"`
	ReferencedBy    string          `json:"referencedBy"`
	Depth           int             `json:"depth"`
	Exists          bool            `json:"exists"`
	DaysSinceUpdate *int            `json:"daysSinceUpdate"`
	Stale           bool            `json:"stale"`
	Violations      []violationJSON `json:"violations"`
}

type duplicateJSON struct {
	Instruction string   `json:"instruction"`
	Files       []string `json:"files"`
}

type fileJSON struct {
	File          string          `json:"file"`
	AgentRoot     bool            `json:"agentRoot,omitempty"`
	Score         int             `json:"score"`
	Dimensions    map[string]int  `json:"dimensions"`
	Errors        int             `json:"errors"`
	Warnings      int             `json:"warnings"`
	Metrics       metricsJSON     `json:"metrics"`
	Violations    []violationJSON `json:"violations"`
	PassedRules   []string        `json:"passedRules"`
	GoodPractices []string        `json:"goodPractices"`
	References    []refJSON       `json:"references"`
	Duplicates    []duplicateJSON `json:"duplicates"`
}

type structureIssueJSON struct {
	Code     string   `json:"code"`
	Severity string   `json:"severity"`
	Message  string   `json:"message"`
	Penalty  int      `json:"penalty"`
	Files    []string `json:"files"`
}

type repoSummaryJSON struct {
	Files             int `json:"files"`
	AgentRoots        int `json:"agentRoots"`
	TotalLines        int `json:"totalLines"`
	TotalInstructions int `json:"totalInstructions"`
	Errors            int `json:"errors"`
	Warnings          int `json:"warnings"`
	AvgFileScore      int `json:"avgFileScore"`
	RepoScore         int `json:"repoScore"`
}

type repoJSON struct {
	Directory       string               `json:"directory"`
	Files           []fileJSON           `json:"files"`
	StructureIssues []structureIssueJSON `json:"structureIssues"`
	Orphans         []string             `json:"orphans"`
	Summary         repoSummaryJSON      `json:"summary"`
}

// daysOrNull maps the -1 "unknown" sentinel to JSON null.
func daysOrNull(days int) *int {
	if days < 0 {
		return nil
	}
	return &days
}

// violationsJSON returns the detected problems in results, honoring the
// -categories and -severities filters.
func violationsJSON(results []rules.RuleResult, opts rules.FilterOptions) []violationJSON {
	opts.FailuresOnly = true
	opts.HideGoodPractice = true
	out := []violationJSON{}
	for _, r := range rules.FilterResults(results, opts) {
		out = append(out, violationJSON{
			Code:       r.Rule.Code,
			Severity:   string(r.Rule.Severity),
			Category:   r.Rule.Category,
			Dimension:  string(rules.ResolveDimension(r.Rule)),
			Message:    r.Rule.ErrorMessage,
			Suggestion: r.Rule.Suggestion,
			Links:      r.Rule.Links,
		})
	}
	return out
}

// newFileJSON converts a file analysis to its JSON form. path is the file
// path to report (relative to the scanned directory in repo mode).
func newFileJSON(fa *fileAnalysis, path string, opts rules.FilterOptions) fileJSON {
	ctx := fa.Ctx
	intMetric := func(key string) int {
		v, _ := ctx.Metrics[key].(int)
		return v
	}
	stacks, _ := ctx.Metrics["detected_stacks"].([]string)
	if stacks == nil {
		stacks = []string{}
	}
	progDisc, _ := ctx.Metrics["hasProgressiveDisclosure"].(bool)

	fj := fileJSON{
		File:       path,
		Score:      fa.Score,
		Dimensions: map[string]int{},
		Errors:     fa.Errors,
		Warnings:   fa.Warnings,
		Metrics: metricsJSON{
			LineCount:               ctx.LineCount,
			InstructionCount:        ctx.InstructionCount,
			InstructionDensityPct:   intMetric("instruction_density_pct"),
			ProgressiveDisclosure:   progDisc,
			TotalLineCount:          fa.AggMetrics.TotalLineCount,
			TotalInstructionCount:   fa.AggMetrics.TotalInstructionCount,
			ReferencedFileCount:     len(rules.FlattenRefs(fa.Refs)),
			BrokenReferences:        intMetric("broken_references_count"),
			StaleReferences:         intMetric("stale_references_count"),
			ScopeCommitsSinceUpdate: intMetric("scope_commits_since_update"),
			DaysSinceUpdate:         daysOrNull(fa.FreshnessDays),
			DetectedStacks:          stacks,
		},
		Violations:    violationsJSON(fa.Results, opts),
		PassedRules:   []string{},
		GoodPractices: []string{},
		References:    []refJSON{},
		Duplicates:    []duplicateJSON{},
	}

	if fa.DimensionScores != nil {
		for dim, ds := range fa.DimensionScores.Scores {
			fj.Dimensions[string(dim)] = ds.Score
		}
	}

	for _, r := range fa.Results {
		switch {
		case r.Rule.Category == "good-practice" && r.Passed:
			fj.GoodPractices = append(fj.GoodPractices, r.Rule.Code)
		case r.Rule.Category != "good-practice" && !r.Passed:
			fj.PassedRules = append(fj.PassedRules, r.Rule.Code)
		}
	}

	for _, ref := range rules.FlattenRefs(fa.Refs) {
		rj := refJSON{
			Path:         ref.Path,
			ReferencedBy: ref.ReferencedBy,
			Depth:        ref.Depth,
			Exists:       ref.Exists,
			Stale:        ref.IsStale,
			Violations:   violationsJSON(fa.RefResults[ref.Path], opts),
		}
		if ref.Exists && !ref.LastModified.IsZero() {
			rj.DaysSinceUpdate = daysOrNull(ref.DaysSinceUpdate)
		}
		fj.References = append(fj.References, rj)
	}

	for _, d := range fa.AggMetrics.Duplicates {
		fj.Duplicates = append(fj.Duplicates, duplicateJSON{Instruction: d.Instruction, Files: d.Files})
	}

	return fj
}

func newRepoJSON(ra *repoAnalysis) repoJSON {
	opts := buildFilterOpts()
	rel := func(p string) string {
		if r, err := filepath.Rel(ra.Dir, p); err == nil {
			return r
		}
		return p
	}

	rj := repoJSON{
		Directory:       ra.Dir,
		Files:           []fileJSON{},
		StructureIssues: []structureIssueJSON{},
		Orphans:         []string{},
		Summary: repoSummaryJSON{
			Files:             len(ra.Files),
			AgentRoots:        len(ra.AgentRoots),
			TotalLines:        ra.Summary.TotalLines,
			TotalInstructions: ra.Summary.TotalInstructions,
			Errors:            ra.Summary.Errors,
			Warnings:          ra.Summary.Warnings,
			AvgFileScore:      ra.Summary.AvgFileScore,
			RepoScore:         ra.Summary.RepoScore,
		},
	}
	rj.Orphans = append(rj.Orphans, ra.Orphans...)

	var counted []string
	for _, fa := range ra.Files {
		fj := newFileJSON(fa, rel(fa.FilePath), opts)
		fj.AgentRoot = ra.AgentRoots[fa.FilePath]
		rj.Files = append(rj.Files, fj)
		if !fj.AgentRoot {
			counted = append(counted, fj.File)
		}
	}

	if ra.MultipleFiles {
		rj.StructureIssues = append(rj.StructureIssues, structureIssueJSON{
			Code:     "CD060",
			Severity: string(rules.SeverityError),
			Message:  "Multiple context files detected",
			Penalty:  multipleFilesPenalty,
			Files:    counted,
		})
	}

	return rj
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
