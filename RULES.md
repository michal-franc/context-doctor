# Rules

context-doctor includes the following built-in rules for analyzing CLAUDE.md files.

## Length Issues

| Code | Severity | Description |
|------|----------|-------------|
| CD001 | error | File has more than 300 lines. Extract task-specific content to separate docs and use progressive disclosure. |
| CD002 | warning | File has more than 100 lines. Consider being more concise. Ideal CLAUDE.md is ~60 lines. |

## Instruction Count

| Code | Severity | Description |
|------|----------|-------------|
| CD003 | error | Too many instructions (>100 detected). LLMs reliably follow 150-200 instructions, and Claude Code adds ~50 of its own. |
| CD004 | warning | High instruction count (~50+ detected). Consider reducing instructions to improve compliance. |

## Linter Abuse

These rules detect formatting/style rules that should be handled by dedicated tools (Prettier, ESLint, etc.) rather than Claude.

| Code | Severity | Description |
|------|----------|-------------|
| CD010 | warning | Indentation rules detected. Use a code formatter instead. |
| CD011 | warning | Line length rules detected. Use a linter for line length enforcement. |
| CD012 | warning | Quote style rules detected. Use a formatter (Prettier, Biome) for quote style. |
| CD013 | warning | Naming convention rules detected. Use a linter for naming conventions. |
| CD014 | warning | Semicolon rules detected. Use a formatter for semicolon style. |
| CD015 | warning | Trailing character rules detected. Use a formatter for trailing characters. |

## Auto-Generated Content

| Code | Severity | Description |
|------|----------|-------------|
| CD020 | warning | File appears to be auto-generated. CLAUDE.md is high-leverage - carefully craft each line manually. |
| CD021 | info | References to /init command found. Avoid using /init and manually craft your CLAUDE.md. |

## Progressive Disclosure

| Code | Severity | Description |
|------|----------|-------------|
| CD030 | info | No progressive disclosure detected in a file over 60 lines. Point to separate docs for task-specific information. |
| CD040 | info | (Good practice) Progressive disclosure pattern detected. |

## Custom Rules

You can create custom rules by adding YAML files to a `.context-doctor/` directory. Rules follow this structure:

```yaml
version: "1.0"
rules:
  - code: CUSTOM001
    description: My custom rule
    severity: warning  # error, warning, or info
    category: my-category
    matchSpec:
      action: regexMatch
      patterns:
        - "pattern to match"
    errorMessage: "Message shown when rule triggers"
    suggestion: "How to fix the issue"
```

### Available Actions

- `greaterThan` - Compare metric against a value
- `regexMatch` - Match against regex patterns
- `regexNotMatch` - Inverse regex match
- `and` - Combine multiple conditions

### Available Metrics

- `lineCount` - Number of lines in the file
- `instructionCount` - Estimated number of instructions
