package templates

import (
	"embed"
	"strings"
)

//go:embed *.md
var templateFS embed.FS

// stackTemplateFiles maps stack names to their template file names.
var stackTemplateFiles = map[string]string{
	"go":         "go.md",
	"python":     "python.md",
	"nodejs":     "nodejs.md",
	"typescript": "typescript.md",
	"rust":       "rust.md",
}

// GetTemplate returns the CLAUDE.md template for a given stack name.
// Returns empty string if no template exists for the stack.
func GetTemplate(stack string) string {
	filename, ok := stackTemplateFiles[stack]
	if !ok {
		return ""
	}
	data, err := templateFS.ReadFile(filename)
	if err != nil {
		return ""
	}
	return string(data)
}

// GetCompositeTemplate merges templates for multiple detected stacks.
// The first stack with a template becomes the base; additional stacks
// are appended as extra sections.
func GetCompositeTemplate(stacks []string) string {
	var base string
	var extras []string

	for _, stack := range stacks {
		tmpl := GetTemplate(stack)
		if tmpl == "" {
			continue
		}
		if base == "" {
			base = tmpl
		} else {
			extras = append(extras, tmpl)
		}
	}

	if base == "" {
		return ""
	}

	if len(extras) == 0 {
		return base
	}

	var b strings.Builder
	b.WriteString(base)
	for _, extra := range extras {
		b.WriteString("\n---\n\n")
		b.WriteString(extra)
	}
	return b.String()
}

// AvailableStacks returns the list of stack names that have templates.
func AvailableStacks() []string {
	var stacks []string
	for name := range stackTemplateFiles {
		stacks = append(stacks, name)
	}
	return stacks
}
