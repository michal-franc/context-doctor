package main

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"context-doctor/rules"
)

func jsonTestAnalysis() *fileAnalysis {
	problem := func(code string, sev rules.Severity, cat string, detected bool) rules.RuleResult {
		return rules.RuleResult{
			Rule:   rules.Rule{Code: code, Severity: sev, Category: cat, ErrorMessage: code + " message"},
			Passed: detected,
		}
	}
	ctx := rules.BuildContext("CLAUDE.md", "# Title\n- Always run tests\n")
	ctx.Metrics["detected_stacks"] = []string{"go"}
	return &fileAnalysis{
		FilePath: "/repo/CLAUDE.md",
		Ctx:      ctx,
		Results: []rules.RuleResult{
			problem("CD001", rules.SeverityError, "length", true),
			problem("CD002", rules.SeverityWarning, "length", false),
			problem("CD052", rules.SeverityInfo, "content-quality", true),
			problem("CD040", rules.SeverityInfo, "good-practice", true),
			problem("CD041", rules.SeverityInfo, "good-practice", false),
		},
		Refs: []rules.RefInfo{
			{Path: "docs/a.md", Exists: true, DaysSinceUpdate: 3, LastModified: time.Now(), ReferencedBy: "CLAUDE.md"},
			{Path: "docs/missing.md", Exists: false, ReferencedBy: "CLAUDE.md"},
		},
		RefResults: map[string][]rules.RuleResult{
			"docs/a.md": {problem("CD011", rules.SeverityWarning, "linter-abuse", true)},
		},
		DimensionScores: &rules.DimensionScores{
			Scores: map[rules.Dimension]*rules.DimensionScoreResult{
				rules.DimensionCorrectness: {Score: 85},
			},
			Overall: 90,
		},
		FreshnessDays: -1,
		Score:         90,
		Errors:        1,
	}
}

func TestNewFileJSON(t *testing.T) {
	fj := newFileJSON(jsonTestAnalysis(), "CLAUDE.md", rules.FilterOptions{})

	var codes []string
	for _, v := range fj.Violations {
		codes = append(codes, v.Code)
	}
	if !reflect.DeepEqual(codes, []string{"CD001", "CD052"}) {
		t.Errorf("violations = %v, want [CD001 CD052]", codes)
	}
	if !reflect.DeepEqual(fj.PassedRules, []string{"CD002"}) {
		t.Errorf("passedRules = %v, want [CD002]", fj.PassedRules)
	}
	if !reflect.DeepEqual(fj.GoodPractices, []string{"CD040"}) {
		t.Errorf("goodPractices = %v, want [CD040]", fj.GoodPractices)
	}
	if fj.Dimensions["correctness"] != 85 || fj.Score != 90 || fj.Errors != 1 {
		t.Errorf("unexpected scores: %+v %d %d", fj.Dimensions, fj.Score, fj.Errors)
	}
	if fj.Metrics.DaysSinceUpdate != nil {
		t.Errorf("unknown freshness should be null, got %d", *fj.Metrics.DaysSinceUpdate)
	}
	if !reflect.DeepEqual(fj.Metrics.DetectedStacks, []string{"go"}) {
		t.Errorf("detectedStacks = %v", fj.Metrics.DetectedStacks)
	}

	if len(fj.References) != 2 {
		t.Fatalf("expected 2 references, got %d", len(fj.References))
	}
	a, missing := fj.References[0], fj.References[1]
	if a.DaysSinceUpdate == nil || *a.DaysSinceUpdate != 3 || len(a.Violations) != 1 || a.Violations[0].Code != "CD011" {
		t.Errorf("unexpected existing ref: %+v", a)
	}
	if missing.Exists || missing.DaysSinceUpdate != nil || len(missing.Violations) != 0 {
		t.Errorf("unexpected missing ref: %+v", missing)
	}
}

func TestNewFileJSON_SeverityFilter(t *testing.T) {
	opts := rules.FilterOptions{Severities: []rules.Severity{rules.SeverityError}}
	fj := newFileJSON(jsonTestAnalysis(), "CLAUDE.md", opts)
	if len(fj.Violations) != 1 || fj.Violations[0].Code != "CD001" {
		t.Errorf("violations = %+v, want only CD001", fj.Violations)
	}
}

func TestNewFileJSON_EmptyListsEncodeAsArrays(t *testing.T) {
	fa := &fileAnalysis{Ctx: rules.BuildContext("CLAUDE.md", "# Empty"), FreshnessDays: 5}
	out, err := json.Marshal(newFileJSON(fa, "CLAUDE.md", rules.FilterOptions{}))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"violations", "passedRules", "goodPractices", "references", "duplicates"} {
		if _, ok := decoded[key].([]any); !ok {
			t.Errorf("%s should encode as an array, got %v", key, decoded[key])
		}
	}
}

func TestNewRepoJSON(t *testing.T) {
	root := jsonTestAnalysis()
	agent := jsonTestAnalysis()
	agent.FilePath = "/repo/agent-a/CLAUDE.md"
	docs := jsonTestAnalysis()
	docs.FilePath = "/repo/docs/CLAUDE.md"

	files := []*fileAnalysis{root, agent, docs}
	ra := &repoAnalysis{
		Dir:           "/repo",
		Files:         files,
		AgentRoots:    map[string]bool{agent.FilePath: true},
		MultipleFiles: true,
		Orphans:       []string{"notes.md"},
		Summary:       summarizeRepo(files, true),
	}

	rj := newRepoJSON(ra)
	if len(rj.Files) != 3 || rj.Files[1].File != "agent-a/CLAUDE.md" || !rj.Files[1].AgentRoot {
		t.Errorf("unexpected files: %+v", rj.Files)
	}
	if len(rj.StructureIssues) != 1 {
		t.Fatalf("expected CD060 structure issue, got %+v", rj.StructureIssues)
	}
	issue := rj.StructureIssues[0]
	if issue.Code != "CD060" || issue.Penalty != 30 || !reflect.DeepEqual(issue.Files, []string{"CLAUDE.md", "docs/CLAUDE.md"}) {
		t.Errorf("unexpected structure issue: %+v", issue)
	}
	if rj.Summary.AvgFileScore != 90 || rj.Summary.RepoScore != 60 || rj.Summary.AgentRoots != 1 {
		t.Errorf("unexpected summary: %+v", rj.Summary)
	}
	if !reflect.DeepEqual(rj.Orphans, []string{"notes.md"}) {
		t.Errorf("orphans = %v", rj.Orphans)
	}
}
