package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// DetectStacksWithMarkers
// =============================================================================

func TestDetectStacksWithMarkers(t *testing.T) {
	tests := []struct {
		name    string
		files   []string // files to create
		dirs    []string // directories to create
		markers []StackMarker
		want    []string
	}{
		{
			name:    "go project",
			files:   []string{"go.mod"},
			markers: DefaultStackMarkers(),
			want:    []string{"go"},
		},
		{
			name:    "python project with pyproject.toml",
			files:   []string{"pyproject.toml"},
			markers: DefaultStackMarkers(),
			want:    []string{"python"},
		},
		{
			name:    "nodejs project",
			files:   []string{"package.json"},
			markers: DefaultStackMarkers(),
			want:    []string{"nodejs"},
		},
		{
			name:    "typescript project",
			files:   []string{"tsconfig.json"},
			markers: DefaultStackMarkers(),
			want:    []string{"typescript"},
		},
		{
			name:    "rust project",
			files:   []string{"Cargo.toml"},
			markers: DefaultStackMarkers(),
			want:    []string{"rust"},
		},
		{
			name:    "make project",
			files:   []string{"Makefile"},
			markers: DefaultStackMarkers(),
			want:    []string{"make"},
		},
		{
			name:    "docker project",
			files:   []string{"Dockerfile"},
			markers: DefaultStackMarkers(),
			want:    []string{"docker"},
		},
		{
			name:    "github-actions project",
			dirs:    []string{".github/workflows"},
			markers: DefaultStackMarkers(),
			want:    []string{"github-actions"},
		},
		{
			name:    "multiple stacks detected",
			files:   []string{"go.mod", "Makefile", "Dockerfile"},
			dirs:    []string{".github/workflows"},
			markers: DefaultStackMarkers(),
			want:    []string{"go", "make", "docker", "github-actions"},
		},
		{
			name:    "empty directory",
			markers: DefaultStackMarkers(),
			want:    nil,
		},
		{
			name:    "dir marker does not match file",
			files:   []string{".github/workflows"}, // file, not dir
			markers: DefaultStackMarkers(),
			want:    nil,
		},
		{
			name:    "file marker does not match directory",
			dirs:    []string{"go.mod"}, // dir, not file
			markers: DefaultStackMarkers(),
			want:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for _, d := range tc.dirs {
				if err := os.MkdirAll(filepath.Join(tmpDir, d), 0755); err != nil {
					t.Fatal(err)
				}
			}
			for _, f := range tc.files {
				dir := filepath.Dir(filepath.Join(tmpDir, f))
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, f), []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := DetectStacksWithMarkers(tmpDir, tc.markers)
			if !slicesEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetectStacksWithMarkers_nonexistentDir(t *testing.T) {
	got := DetectStacksWithMarkers("/nonexistent/path/12345", DefaultStackMarkers())
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

// =============================================================================
// DetectStacks (integration, uses DefaultStackMarkers)
// =============================================================================

func TestDetectStacks(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}
	got := DetectStacks(tmpDir)
	if len(got) != 1 || got[0] != "go" {
		t.Errorf("expected [go], got %v", got)
	}
}

// slicesEqual compares two string slices for equality.
func slicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
