package rules

import (
	"strings"
)

// DuplicateInfo represents an instruction found in multiple files
type DuplicateInfo struct {
	Instruction string   // the normalized instruction text
	Files       []string // which files contain it
}

// AggregateMetrics holds combined metrics across all context files
type AggregateMetrics struct {
	TotalInstructionCount int
	TotalLineCount        int
	FileCount             int
	Duplicates            []DuplicateInfo
}

// FindDuplicateInstructions finds instructions that appear in multiple files
func FindDuplicateInstructions(primary *AnalysisContext, refs []RefInfo) []DuplicateInfo {
	// Map from normalized instruction -> list of files containing it
	instructionFiles := make(map[string][]string)

	// Extract instructions from primary file
	primaryInstructions := extractInstructionLines(primary.Lines)
	for _, instr := range primaryInstructions {
		normalized := normalizeInstruction(instr)
		if normalized != "" {
			instructionFiles[normalized] = append(instructionFiles[normalized], primary.FilePath)
		}
	}

	// Extract instructions from each referenced file
	for _, ref := range refs {
		if !ref.Exists || ref.Context == nil {
			continue
		}
		refInstructions := extractInstructionLines(ref.Context.Lines)
		for _, instr := range refInstructions {
			normalized := normalizeInstruction(instr)
			if normalized == "" {
				continue
			}
			// Only add the file once per instruction
			files := instructionFiles[normalized]
			alreadyHasFile := false
			for _, f := range files {
				if f == ref.Path {
					alreadyHasFile = true
					break
				}
			}
			if !alreadyHasFile {
				instructionFiles[normalized] = append(instructionFiles[normalized], ref.Path)
			}
		}
	}

	// Collect duplicates (instructions in 2+ files)
	var duplicates []DuplicateInfo
	for instr, files := range instructionFiles {
		if len(files) >= 2 {
			duplicates = append(duplicates, DuplicateInfo{
				Instruction: instr,
				Files:       files,
			})
		}
	}

	return duplicates
}

// ComputeAggregateMetrics computes combined metrics across primary + referenced files
func ComputeAggregateMetrics(primary *AnalysisContext, refs []RefInfo) AggregateMetrics {
	agg := AggregateMetrics{
		TotalInstructionCount: primary.InstructionCount,
		TotalLineCount:        primary.LineCount,
		FileCount:             1,
	}

	for _, ref := range refs {
		if !ref.Exists || ref.Context == nil {
			continue
		}
		agg.TotalInstructionCount += ref.Context.InstructionCount
		agg.TotalLineCount += ref.Context.LineCount
		agg.FileCount++
	}

	agg.Duplicates = FindDuplicateInstructions(primary, refs)

	return agg
}

// extractInstructionLines returns lines that look like instructions
// Reuses the same imperative verb pattern from CountInstructions
func extractInstructionLines(lines []string) []string {
	var instructions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if imperativeVerbPattern.MatchString(line) {
			instructions = append(instructions, line)
		}
	}
	return instructions
}

// normalizeInstruction normalizes an instruction for comparison
func normalizeInstruction(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	// Strip leading list markers
	s = strings.TrimLeft(s, "-*")
	s = strings.TrimSpace(s)
	// Skip very short instructions (likely false positives)
	if len(s) < 15 {
		return ""
	}
	return s
}
