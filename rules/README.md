# Rules

Context Doctor ships with built-in rules that validate `CLAUDE.md` files. Rules are defined in `builtin.yaml` and evaluated by the rules engine.

## Scoring

Each finding deducts points from a starting score of 100:

| Severity | Deduction |
|----------|-----------|
| Error    | -15       |
| Warning  | -5        |
| Info     | -2        |

## Rules Reference

### Length

| Code | Severity | Description |
|------|----------|-------------|
| CD001 | Error | File has more than 300 lines. Extract content to separate docs and use progressive disclosure. |
| CD002 | Warning | File has more than 100 lines. Ideal CLAUDE.md is ~60 lines. |

### Instructions

| Code | Severity | Description |
|------|----------|-------------|
| CD003 | Error | Too many instructions (>100 detected, +50 from Claude Code). LLMs reliably follow 150-200 instructions. |
| CD004 | Warning | High instruction count (~50+ detected, +50 from Claude Code = ~100+). |

### Linter Abuse

Detects code style rules that should be handled by formatters/linters, not CLAUDE.md.

| Code | Severity | Description |
|------|----------|-------------|
| CD010 | Warning | Indentation rules found. Use a code formatter instead. |
| CD011 | Warning | Line length rules found. Use a linter instead. |
| CD012 | Warning | Quote style rules found. Use a formatter (Prettier, Biome). |
| CD013 | Warning | Naming convention rules found. Use a linter instead. |
| CD014 | Warning | Semicolon rules found. Use a formatter instead. |
| CD015 | Warning | Trailing character rules found. Use a formatter instead. |

### Auto-Generated

| Code | Severity | Description |
|------|----------|-------------|
| CD020 | Warning | File appears to be auto-generated. Carefully craft each line manually instead of using `/init`. |
| CD021 | Info | References to `/init` command found. Manually craft your CLAUDE.md for best results. |

### Progressive Disclosure

| Code | Severity | Description |
|------|----------|-------------|
| CD030 | Info | No progressive disclosure in a file over 60 lines. Point to separate docs for task-specific information. |

### Generic Advice

| Code | Severity | Description |
|------|----------|-------------|
| CD050 | Warning | Generic advice found (e.g. "write clean code", "follow best practices"). Replace with project-specific instructions. |

### Content Quality

| Code | Severity | Description |
|------|----------|-------------|
| CD051 | Info | No project description or context found. Start with what the project is. |
| CD052 | Info | No build/test/lint commands found. Include commands Claude needs to verify changes. |
| CD053 | Info | No negative instructions in a file over 30 lines. Specify what NOT to do. |
| CD054 | Info | No code examples in a file over 50 lines. Add concrete examples. |

### Good Practices

These rules detect positive patterns and are only shown in verbose mode.

| Code | Severity | Description |
|------|----------|-------------|
| CD040 | Info | Progressive disclosure pattern detected. |
| CD041 | Info | Negative instructions detected (what NOT to do). |
| CD042 | Info | Code examples detected. |
