package rules

import (
	"testing"
)

func TestFindDuplicateInstructions_NoDuplicates(t *testing.T) {
	primary := &AnalysisContext{
		FilePath: "CLAUDE.md",
		Lines:    []string{"- Always use TypeScript strict mode"},
	}

	refs := []RefInfo{
		{
			Path:   "docs/guide.md",
			Exists: true,
			Context: &AnalysisContext{
				FilePath: "docs/guide.md",
				Lines:    []string{"- Never use var declarations"},
			},
		},
	}

	dups := FindDuplicateInstructions(primary, refs)
	if len(dups) != 0 {
		t.Errorf("expected 0 duplicates, got %d", len(dups))
	}
}

func TestFindDuplicateInstructions_WithDuplicates(t *testing.T) {
	primary := &AnalysisContext{
		FilePath: "CLAUDE.md",
		Lines:    []string{"- Always use TypeScript strict mode"},
	}

	refs := []RefInfo{
		{
			Path:   "docs/guide.md",
			Exists: true,
			Context: &AnalysisContext{
				FilePath: "docs/guide.md",
				Lines:    []string{"- always use typescript strict mode"},
			},
		},
	}

	dups := FindDuplicateInstructions(primary, refs)
	if len(dups) != 1 {
		t.Fatalf("expected 1 duplicate, got %d", len(dups))
	}
	if len(dups[0].Files) != 2 {
		t.Errorf("expected duplicate in 2 files, got %d", len(dups[0].Files))
	}
}

func TestFindDuplicateInstructions_SkipsNonExistent(t *testing.T) {
	primary := &AnalysisContext{
		FilePath: "CLAUDE.md",
		Lines:    []string{"- Always use TypeScript strict mode"},
	}

	refs := []RefInfo{
		{
			Path:   "docs/missing.md",
			Exists: false,
		},
	}

	dups := FindDuplicateInstructions(primary, refs)
	if len(dups) != 0 {
		t.Errorf("expected 0 duplicates, got %d", len(dups))
	}
}

func TestFindDuplicateInstructions_ShortInstructionsIgnored(t *testing.T) {
	primary := &AnalysisContext{
		FilePath: "CLAUDE.md",
		Lines:    []string{"- Use gofmt"},
	}

	refs := []RefInfo{
		{
			Path:   "docs/guide.md",
			Exists: true,
			Context: &AnalysisContext{
				FilePath: "docs/guide.md",
				Lines:    []string{"- Use gofmt"},
			},
		},
	}

	// "use gofmt" is < 15 chars, should be skipped
	dups := FindDuplicateInstructions(primary, refs)
	if len(dups) != 0 {
		t.Errorf("expected 0 duplicates for short instruction, got %d", len(dups))
	}
}

func TestComputeAggregateMetrics_SingleFile(t *testing.T) {
	primary := &AnalysisContext{
		FilePath:         "CLAUDE.md",
		InstructionCount: 20,
		LineCount:        50,
		Lines:            []string{},
	}

	agg := ComputeAggregateMetrics(primary, nil)

	if agg.TotalInstructionCount != 20 {
		t.Errorf("expected TotalInstructionCount=20, got %d", agg.TotalInstructionCount)
	}
	if agg.TotalLineCount != 50 {
		t.Errorf("expected TotalLineCount=50, got %d", agg.TotalLineCount)
	}
	if agg.FileCount != 1 {
		t.Errorf("expected FileCount=1, got %d", agg.FileCount)
	}
}

func TestComputeAggregateMetrics_MultipleFiles(t *testing.T) {
	primary := &AnalysisContext{
		FilePath:         "CLAUDE.md",
		InstructionCount: 20,
		LineCount:        50,
		Lines:            []string{},
	}

	refs := []RefInfo{
		{
			Path:   "docs/a.md",
			Exists: true,
			Context: &AnalysisContext{
				FilePath:         "docs/a.md",
				InstructionCount: 15,
				LineCount:        30,
				Lines:            []string{},
			},
		},
		{
			Path:   "docs/b.md",
			Exists: false,
		},
		{
			Path:   "docs/c.md",
			Exists: true,
			Context: &AnalysisContext{
				FilePath:         "docs/c.md",
				InstructionCount: 10,
				LineCount:        25,
				Lines:            []string{},
			},
		},
	}

	agg := ComputeAggregateMetrics(primary, refs)

	if agg.TotalInstructionCount != 45 {
		t.Errorf("expected TotalInstructionCount=45, got %d", agg.TotalInstructionCount)
	}
	if agg.TotalLineCount != 105 {
		t.Errorf("expected TotalLineCount=105, got %d", agg.TotalLineCount)
	}
	if agg.FileCount != 3 {
		t.Errorf("expected FileCount=3, got %d", agg.FileCount)
	}
}

func TestNormalizeInstruction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"- Always use TypeScript strict mode", "always use typescript strict mode"},
		{"* Never use var declarations", "never use var declarations"},
		{"  - Use explicit error handling  ", "use explicit error handling"},
		{"Short", ""}, // too short
		{"", ""},
	}

	for _, tc := range tests {
		got := normalizeInstruction(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeInstruction(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestExtractInstructionLines(t *testing.T) {
	lines := []string{
		"# Header",
		"",
		"- Always use TypeScript",
		"Some random text",
		"- Never use var",
		"- Run tests before committing",
	}

	instructions := extractInstructionLines(lines)
	if len(instructions) != 3 {
		t.Errorf("expected 3 instructions, got %d: %v", len(instructions), instructions)
	}
}
