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

func TestResolveReferences_Recursive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create docs/a.md which references b.md (relative to its own dir)
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// "see b.md" will be extracted as a ref, resolved relative to docs/
	if err := os.WriteFile(filepath.Join(docsDir, "a.md"), []byte("# A\n\nFor details see b.md\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "b.md"), []byte("# B\n\nLeaf node content.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &AnalysisContext{
		FilePath: filepath.Join(tmpDir, "CLAUDE.md"),
		Metrics: map[string]any{
			"progressiveDisclosureRefs": []string{"docs/a.md"},
		},
	}

	refs := ResolveReferences(ctx, tmpDir, 90)
	if len(refs) != 1 {
		t.Fatalf("expected 1 top-level ref, got %d", len(refs))
	}

	a := refs[0]
	if !a.Exists {
		t.Fatal("expected docs/a.md to exist")
	}
	if len(a.Children) != 1 {
		t.Fatalf("expected docs/a.md to have 1 child, got %d", len(a.Children))
	}

	b := a.Children[0]
	if !b.Exists {
		t.Error("expected b.md to exist")
	}
	if b.Depth != 1 {
		t.Errorf("expected depth 1, got %d", b.Depth)
	}
	if b.ReferencedBy != "docs/a.md" {
		t.Errorf("expected ReferencedBy='docs/a.md', got %q", b.ReferencedBy)
	}

	// FlattenRefs should return both
	flat := FlattenRefs(refs)
	if len(flat) != 2 {
		t.Errorf("expected 2 flattened refs, got %d", len(flat))
	}
}

func TestResolveReferences_CycleDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// a.md references b.md, b.md references a.md — cycle
	if err := os.WriteFile(filepath.Join(tmpDir, "a.md"), []byte("# A\n\nSee b.md for more.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.md"), []byte("# B\n\nSee a.md for more.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &AnalysisContext{
		FilePath: filepath.Join(tmpDir, "CLAUDE.md"),
		Metrics: map[string]any{
			"progressiveDisclosureRefs": []string{"a.md"},
		},
	}

	refs := ResolveReferences(ctx, tmpDir, 90)

	// Should resolve a.md -> b.md but NOT b.md -> a.md again
	flat := FlattenRefs(refs)
	if len(flat) != 2 {
		t.Errorf("expected 2 refs (cycle broken), got %d", len(flat))
	}

	// Verify no infinite recursion happened (test completing is proof enough)
}

func TestFlattenRefs(t *testing.T) {
	refs := []RefInfo{
		{
			Path: "a.md",
			Children: []RefInfo{
				{
					Path: "b.md",
					Children: []RefInfo{
						{Path: "c.md"},
					},
				},
			},
		},
		{Path: "d.md"},
	}

	flat := FlattenRefs(refs)
	if len(flat) != 4 {
		t.Errorf("expected 4 flattened refs, got %d", len(flat))
	}

	expected := []string{"a.md", "b.md", "c.md", "d.md"}
	for i, ref := range flat {
		if ref.Path != expected[i] {
			t.Errorf("flat[%d] = %q, want %q", i, ref.Path, expected[i])
		}
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
