package templates

import (
	"strings"
	"testing"
)

// =============================================================================
// GetTemplate
// =============================================================================

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name      string
		stack     string
		wantEmpty bool
		contains  string
	}{
		{"go template", "go", false, "go build"},
		{"python template", "python", false, "pytest"},
		{"nodejs template", "nodejs", false, "npm"},
		{"typescript template", "typescript", false, "tsc"},
		{"rust template", "rust", false, "cargo"},
		{"unknown stack returns empty", "java", true, ""},
		{"empty string returns empty", "", true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetTemplate(tc.stack)
			if tc.wantEmpty && got != "" {
				t.Errorf("expected empty, got %q", got[:50])
			}
			if !tc.wantEmpty && got == "" {
				t.Error("expected non-empty template")
			}
			if tc.contains != "" && !strings.Contains(got, tc.contains) {
				t.Errorf("expected template to contain %q", tc.contains)
			}
		})
	}
}

// =============================================================================
// GetCompositeTemplate
// =============================================================================

func TestGetCompositeTemplate(t *testing.T) {
	t.Run("single stack", func(t *testing.T) {
		got := GetCompositeTemplate([]string{"go"})
		if got == "" {
			t.Fatal("expected non-empty")
		}
		if !strings.Contains(got, "go build") {
			t.Error("expected go content")
		}
	})

	t.Run("multiple stacks merged", func(t *testing.T) {
		got := GetCompositeTemplate([]string{"go", "rust"})
		if !strings.Contains(got, "go build") {
			t.Error("expected go content")
		}
		if !strings.Contains(got, "cargo") {
			t.Error("expected rust content")
		}
		if !strings.Contains(got, "---") {
			t.Error("expected separator between templates")
		}
	})

	t.Run("unknown stacks skipped", func(t *testing.T) {
		got := GetCompositeTemplate([]string{"go", "make", "docker"})
		if !strings.Contains(got, "go build") {
			t.Error("expected go content")
		}
		// make and docker have no templates, should not cause errors
	})

	t.Run("all unknown stacks returns empty", func(t *testing.T) {
		got := GetCompositeTemplate([]string{"make", "docker", "github-actions"})
		if got != "" {
			t.Errorf("expected empty, got non-empty")
		}
	})

	t.Run("empty stacks returns empty", func(t *testing.T) {
		got := GetCompositeTemplate(nil)
		if got != "" {
			t.Errorf("expected empty, got non-empty")
		}
	})
}

// =============================================================================
// AvailableStacks
// =============================================================================

func TestAvailableStacks(t *testing.T) {
	stacks := AvailableStacks()
	if len(stacks) < 5 {
		t.Errorf("expected at least 5 available stacks, got %d", len(stacks))
	}
}
