package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigFileNames are the repo-level config file names, in lookup order.
var ConfigFileNames = []string{".context-doctor.yml", ".context-doctor.yaml"}

// Config holds repo-level settings read from .context-doctor.yml
type Config struct {
	// Suppress lists rule codes (e.g. CD052) that should not be evaluated.
	Suppress []string `yaml:"suppress,omitempty"`
	// OrphanIgnore lists glob patterns of .md files excluded from orphan detection.
	OrphanIgnore []string `yaml:"orphan-ignore,omitempty"`
}

// LoadConfig reads the config file from dir. A missing file is not an error
// and yields an empty Config.
func LoadConfig(dir string) (Config, error) {
	var cfg Config
	for _, name := range ConfigFileNames {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return cfg, fmt.Errorf("failed to read %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		return cfg, nil
	}
	return cfg, nil
}

// IsSuppressed reports whether code is in the suppress list (case-insensitive).
func IsSuppressed(code string, suppress []string) bool {
	for _, s := range suppress {
		if strings.EqualFold(strings.TrimSpace(s), code) {
			return true
		}
	}
	return false
}

// RemoveSuppressed returns the rules whose codes are not suppressed.
func RemoveSuppressed(rules []Rule, suppress []string) []Rule {
	if len(suppress) == 0 {
		return rules
	}
	var kept []Rule
	for _, r := range rules {
		if !IsSuppressed(r.Code, suppress) {
			kept = append(kept, r)
		}
	}
	return kept
}

// MatchesAnyGlob reports whether the slash-separated relative path matches
// any of the patterns. Patterns support * (within a path segment),
// ** (across segments) and ?. A trailing slash matches everything below
// that directory ("notes/" is the same as "notes/**").
func MatchesAnyGlob(path string, patterns []string) bool {
	path = filepath.ToSlash(path)
	for _, p := range patterns {
		if globToRegexp(p).MatchString(path) {
			return true
		}
	}
	return false
}

func globToRegexp(pattern string) *regexp.Regexp {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	pattern = strings.TrimPrefix(pattern, "./")
	if strings.HasSuffix(pattern, "/") {
		pattern += "**"
	}

	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case c == '*' && i+1 < len(pattern) && pattern[i+1] == '*':
			i++
			// "**/" also matches zero directories
			if i+1 < len(pattern) && pattern[i+1] == '/' {
				i++
				b.WriteString("(?:.*/)?")
			} else {
				b.WriteString(".*")
			}
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}
