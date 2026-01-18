# context-doctor

A CLI tool that analyzes CLAUDE.md files and provides feedback on best practices for Claude Code context management.

## Installation

### From source

```bash
make build
```

### Install to system (requires sudo)

```bash
sudo make install
```

### Install to user directory (no sudo)

```bash
make install-user
```

Make sure `~/go/bin` is in your PATH.

## Usage

```bash
context-doctor [options] <path-to-CLAUDE.md>
```

### Options

| Flag | Description |
|------|-------------|
| `-rules-dir` | Directory containing custom rules (default: same directory as CLAUDE.md) |
| `-no-builtin` | Disable built-in rules |
| `-verbose` | Show detailed output including passed checks |
| `-score` | Show overall score (default: true) |
| `-categories` | Filter by categories (comma-separated) |
| `-severities` | Filter by severities: error, warning, info (comma-separated) |
| `-version` | Show version information |

### Example

```bash
# Analyze a CLAUDE.md file
context-doctor ./CLAUDE.md

# Verbose output with all checks
context-doctor -verbose ./CLAUDE.md

# Only show errors
context-doctor -severities error ./CLAUDE.md
```

### Example Output

Analyzing a CLAUDE.md with issues:

```
============================================================
  CLAUDE.md Analysis Report
============================================================

File: examples/bad_claude.md

METRICS
----------------------------------------
  Lines:        60 (OK)
  Instructions: ~37 (+50 Claude = ~87) (OK)
  Progressive Disclosure: NO

LINTER ABUSE DETECTED
----------------------------------------
  ⚠ [CD011] Line length rules found
     → Use a linter for line length enforcement
  ⚠ [CD012] Quote style rules found
     → Use a formatter (Prettier, Biome) for quote style
  ⚠ [CD013] Naming convention rules found
     → Use a linter for naming conventions
  ⚠ [CD014] Semicolon rules found
     → Use a formatter for semicolon style
  ⚠ [CD015] Trailing character rules found
     → Use a formatter for trailing characters

AUTO-GENERATED CONTENT
----------------------------------------
  ⚠ [CD020] File appears to be auto-generated
     → CLAUDE.md is high-leverage. Carefully craft each line manually instead of using /init

OVERALL SCORE
----------------------------------------
  65/100
```

Analyzing a well-crafted CLAUDE.md:

```
============================================================
  CLAUDE.md Analysis Report
============================================================

File: examples/good_claude.md

METRICS
----------------------------------------
  Lines:        25 (OK)
  Instructions: ~6 (+50 Claude = ~56) (OK)
  Progressive Disclosure: YES

GOOD PRACTICES DETECTED
----------------------------------------
  ✓ [CD040] Progressive disclosure pattern detected

PROGRESSIVE DISCLOSURE REFERENCES
----------------------------------------
  - docs/architecture.md
  - docs/api-patterns.md
  - docs/testing.md

OVERALL SCORE
----------------------------------------
  95/100

  ✓ Excellent! Your CLAUDE.md follows best practices.
```

### Using with Claude Code

You can ask Claude to analyze your CLAUDE.md and suggest improvements:

```
> Run context-doctor on my CLAUDE.md and suggest improvements

Running context-doctor...

Based on the analysis, here are my recommendations:

**Issues Found:**
1. [CD011-CD015] Linter abuse - Your CLAUDE.md contains formatting rules like
   "use 2 spaces for indentation" and "always use single quotes". These should
   be handled by tools like Prettier or ESLint, not Claude.

2. [CD020] Auto-generated content - The file appears to be from /init. Consider
   rewriting it to focus on your project's unique context.

**Suggested Improvements:**

1. Remove formatting rules and add a `.prettierrc` instead:
   ```json
   { "singleQuote": true, "tabWidth": 2 }
   ```

2. Replace generic instructions with project-specific context:
   - What makes this codebase unique?
   - What patterns should Claude follow?
   - What mistakes should Claude avoid?

3. Add progressive disclosure for complex topics:
   - "For API patterns, see docs/api-patterns.md"
   - "For testing guidelines, see docs/testing.md"

Would you like me to help rewrite your CLAUDE.md?
```

## What it checks

- **Length issues** - File and line count thresholds
- **Instruction count** - Too many instructions reduce LLM compliance
- **Linter abuse** - Rules that should be handled by formatters/linters
- **Auto-generated content** - Detects `/init` generated files
- **Progressive disclosure** - Encourages linking to separate docs

See [RULES.md](RULES.md) for the complete list of rules.

## Custom Rules

Place custom rule YAML files in a `.context-doctor/` directory or specify a custom directory with `-rules-dir`.

## License

MIT
