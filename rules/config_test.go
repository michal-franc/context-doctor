package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_MissingFile(t *testing.T) {
	cfg, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Suppress) != 0 || len(cfg.OrphanIgnore) != 0 {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestLoadConfig_ParsesSuppressAndOrphanIgnore(t *testing.T) {
	dir := t.TempDir()
	content := "suppress:\n  - CD052\n  - CD054\norphan-ignore:\n  - \"notes/**\"\n  - \".claude/\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".context-doctor.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Suppress) != 2 || cfg.Suppress[0] != "CD052" || cfg.Suppress[1] != "CD054" {
		t.Errorf("unexpected suppress: %v", cfg.Suppress)
	}
	if len(cfg.OrphanIgnore) != 2 || cfg.OrphanIgnore[0] != "notes/**" || cfg.OrphanIgnore[1] != ".claude/" {
		t.Errorf("unexpected orphan-ignore: %v", cfg.OrphanIgnore)
	}
}

func TestLoadConfig_YamlExtension(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".context-doctor.yaml"), []byte("suppress: [CD060]\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Suppress) != 1 || cfg.Suppress[0] != "CD060" {
		t.Errorf("unexpected suppress: %v", cfg.Suppress)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".context-doctor.yml"), []byte("suppress: [unclosed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadConfig(dir); err == nil {
		t.Error("expected parse error, got nil")
	}
}

func TestRemoveSuppressed(t *testing.T) {
	all := []Rule{{Code: "CD001"}, {Code: "CD052"}, {Code: "CD054"}}

	tests := []struct {
		name     string
		suppress []string
		want     []string
	}{
		{"no suppress list", nil, []string{"CD001", "CD052", "CD054"}},
		{"two codes suppressed", []string{"CD052", "CD054"}, []string{"CD001"}},
		{"lowercase code with spaces", []string{" cd052 "}, []string{"CD001", "CD054"}},
		{"unknown code", []string{"CD999"}, []string{"CD001", "CD052", "CD054"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveSuppressed(all, tt.suppress)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d rules, want %d", len(got), len(tt.want))
			}
			for i, r := range got {
				if r.Code != tt.want[i] {
					t.Errorf("rule %d: got %s, want %s", i, r.Code, tt.want[i])
				}
			}
		})
	}
}

func TestMatchesAnyGlob(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{"double star matches nested file", "sessions/2026/01/log.md", []string{"sessions/**"}, true},
		{"double star does not match sibling dir", "sessionsx/log.md", []string{"sessions/**"}, false},
		{"trailing slash matches nested file", ".claude/skills/foo/SKILL.md", []string{".claude/"}, true},
		{"single star stays within segment", "notes/a/b.md", []string{"notes/*.md"}, false},
		{"single star matches in segment", "notes/b.md", []string{"notes/*.md"}, true},
		{"leading double star matches any depth", "a/b/CHANGELOG.md", []string{"**/CHANGELOG.md"}, true},
		{"leading double star matches root", "CHANGELOG.md", []string{"**/CHANGELOG.md"}, true},
		{"question mark matches one char", "2026-01-05.md", []string{"????-??-??.md"}, true},
		{"leading dot slash ignored", "data-dump/x.md", []string{"./data-dump/**"}, true},
		{"dots in pattern are literal", "READMEXmd", []string{"README.md"}, false},
		{"no patterns", "README.md", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchesAnyGlob(tt.path, tt.patterns); got != tt.want {
				t.Errorf("MatchesAnyGlob(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}
