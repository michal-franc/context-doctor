package rules

import (
	"os"
	"path/filepath"
)

// StackMarker defines a marker file or directory that indicates a technology stack.
type StackMarker struct {
	Name    string
	Markers []string
	IsDir   bool
}

// DefaultStackMarkers returns the built-in stack detection markers.
func DefaultStackMarkers() []StackMarker {
	return []StackMarker{
		{Name: "go", Markers: []string{"go.mod", "go.sum"}, IsDir: false},
		{Name: "python", Markers: []string{"requirements.txt", "setup.py", "pyproject.toml", "Pipfile"}, IsDir: false},
		{Name: "nodejs", Markers: []string{"package.json"}, IsDir: false},
		{Name: "typescript", Markers: []string{"tsconfig.json"}, IsDir: false},
		{Name: "rust", Markers: []string{"Cargo.toml"}, IsDir: false},
		{Name: "make", Markers: []string{"Makefile", "makefile", "GNUmakefile"}, IsDir: false},
		{Name: "docker", Markers: []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}, IsDir: false},
		{Name: "github-actions", Markers: []string{".github/workflows"}, IsDir: true},
	}
}

// DetectStacks scans rootDir for known stack markers and returns detected stack names.
func DetectStacks(rootDir string) []string {
	return DetectStacksWithMarkers(rootDir, DefaultStackMarkers())
}

// DetectStacksWithMarkers scans rootDir using the provided markers and returns detected stack names.
func DetectStacksWithMarkers(rootDir string, markers []StackMarker) []string {
	var stacks []string
	for _, sm := range markers {
		for _, marker := range sm.Markers {
			path := filepath.Join(rootDir, marker)
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			if sm.IsDir && info.IsDir() {
				stacks = append(stacks, sm.Name)
				break
			}
			if !sm.IsDir && !info.IsDir() {
				stacks = append(stacks, sm.Name)
				break
			}
		}
	}
	return stacks
}
