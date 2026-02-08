package rules

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveReferences_NoRefs(t *testing.T) {
	ctx := &AnalysisContext{
		Metrics: map[string]any{
			"progressiveDisclosureRefs": []string{},
		},
	}

	refs := ResolveReferences(ctx, "/tmp", 90)
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestResolveReferences_MissingFile(t *testing.T) {
	ctx := &AnalysisContext{
		Metrics: map[string]any{
			"progressiveDisclosureRefs": []string{"nonexistent.md"},
		},
	}

	refs := ResolveReferences(ctx, "/tmp/definitely-not-a-real-dir-12345", 90)
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].Exists {
		t.Error("expected Exists=false for missing file")
	}
	if refs[0].Path != "nonexistent.md" {
		t.Errorf("expected path 'nonexistent.md', got %q", refs[0].Path)
	}
}

func TestResolveReferences_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a referenced doc
	docPath := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docPath, 0755); err != nil {
		t.Fatal(err)
	}
	docFile := filepath.Join(docPath, "guide.md")
	if err := os.WriteFile(docFile, []byte("# Guide\n\nSome content here.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &AnalysisContext{
		Metrics: map[string]any{
			"progressiveDisclosureRefs": []string{"docs/guide.md"},
		},
	}

	refs := ResolveReferences(ctx, tmpDir, 90)
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}

	ref := refs[0]
	if !ref.Exists {
		t.Error("expected Exists=true for existing file")
	}
	if ref.Context == nil {
		t.Error("expected Context to be built for existing file")
	}
	if ref.IsStale {
		t.Error("expected IsStale=false for freshly created file")
	}
}

func TestResolveReferences_Deduplication(t *testing.T) {
	ctx := &AnalysisContext{
		Metrics: map[string]any{
			"progressiveDisclosureRefs": []string{"docs/a.md", "docs/a.md", "docs/b.md"},
		},
	}

	refs := ResolveReferences(ctx, "/tmp/not-real-dir-99999", 90)
	if len(refs) != 2 {
		t.Errorf("expected 2 unique refs, got %d", len(refs))
	}
}

func TestResolveReferences_StaleDetection(t *testing.T) {
	tmpDir := t.TempDir()

	docFile := filepath.Join(tmpDir, "old.md")
	if err := os.WriteFile(docFile, []byte("# Old doc\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set modification time to 100 days ago
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	if err := os.Chtimes(docFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	ctx := &AnalysisContext{
		Metrics: map[string]any{
			"progressiveDisclosureRefs": []string{"old.md"},
		},
	}

	refs := ResolveReferences(ctx, tmpDir, 90)
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}

	ref := refs[0]
	if !ref.Exists {
		t.Fatal("expected file to exist")
	}
	if !ref.IsStale {
		t.Errorf("expected IsStale=true, DaysSinceUpdate=%d", ref.DaysSinceUpdate)
	}
	if ref.DaysSinceUpdate < 99 {
		t.Errorf("expected DaysSinceUpdate >= 99, got %d", ref.DaysSinceUpdate)
	}
}

func TestEnrichContextWithRefMetrics(t *testing.T) {
	ctx := &AnalysisContext{
		Metrics: make(map[string]any),
	}

	refs := []RefInfo{
		{Path: "docs/a.md", Exists: true, IsStale: false},
		{Path: "docs/b.md", Exists: false},
		{Path: "docs/c.md", Exists: true, IsStale: true},
		{Path: "docs/d.md", Exists: false},
	}

	EnrichContextWithRefMetrics(ctx, refs)

	brokenCount, ok := ctx.Metrics["broken_references_count"].(int)
	if !ok || brokenCount != 2 {
		t.Errorf("expected broken_references_count=2, got %v", ctx.Metrics["broken_references_count"])
	}

	staleCount, ok := ctx.Metrics["stale_references_count"].(int)
	if !ok || staleCount != 1 {
		t.Errorf("expected stale_references_count=1, got %v", ctx.Metrics["stale_references_count"])
	}

	refFiles, ok := ctx.Metrics["referenced_files"].([]string)
	if !ok || len(refFiles) != 4 {
		t.Errorf("expected 4 referenced_files, got %v", ctx.Metrics["referenced_files"])
	}
}

func TestEnrichContextWithRefMetrics_Empty(t *testing.T) {
	ctx := &AnalysisContext{
		Metrics: make(map[string]any),
	}

	EnrichContextWithRefMetrics(ctx, nil)

	brokenCount, ok := ctx.Metrics["broken_references_count"].(int)
	if !ok || brokenCount != 0 {
		t.Errorf("expected broken_references_count=0, got %v", ctx.Metrics["broken_references_count"])
	}
}
