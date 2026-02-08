# Rules

context-doctor includes the following built-in rules for analyzing CLAUDE.md files.

## Rule scoping: `primaryOnly`

Rules can be scoped to only apply to the primary CLAUDE.md file, not to referenced docs. Referenced documentation files (README.md, docs/*.md, etc.) serve humans too, so rules like line count limits or "missing project context" don't apply to them. Only universal rules (like linter abuse detection) run against referenced docs.

Rules marked with **(primary)** below only run against CLAUDE.md. All other rules run against both CLAUDE.md and any referenced docs.

## Length Issues (primary)

| Code | Severity | Description |
|------|----------|-------------|
| CD001 | error | File has more than 300 lines. Extract task-specific content to separate docs and use progressive disclosure. |
| CD002 | warning | File has more than 100 lines. Consider being more concise. Ideal CLAUDE.md is ~60 lines. |

## Instruction Count (primary)

| Code | Severity | Description |
|------|----------|-------------|
| CD003 | error | Too many instructions (>100 detected). LLMs reliably follow 150-200 instructions, and Claude Code adds ~50 of its own. |
| CD004 | warning | High instruction count (~50+ detected). Consider reducing instructions to improve compliance. |

## Linter Abuse

These rules detect formatting/style rules that should be handled by dedicated tools (Prettier, ESLint, etc.) rather than Claude. These run against **all files** including referenced docs, since linter abuse is bad in any context the agent reads.

| Code | Severity | Description |
|------|----------|-------------|
| CD010 | warning | Indentation rules detected. Use a code formatter instead. |
| CD011 | warning | Line length rules detected. Use a linter for line length enforcement. |
| CD012 | warning | Quote style rules detected. Use a formatter (Prettier, Biome) for quote style. |
| CD013 | warning | Naming convention rules detected. Use a linter for naming conventions. |
| CD014 | warning | Semicolon rules detected. Use a formatter for semicolon style. |
| CD015 | warning | Trailing character rules detected. Use a formatter for trailing characters. |

## Auto-Generated Content (primary)

| Code | Severity | Description |
|------|----------|-------------|
| CD020 | warning | File appears to be auto-generated. CLAUDE.md is high-leverage - carefully craft each line manually. |
| CD021 | info | References to /init command found. Avoid using /init and manually craft your CLAUDE.md. |

## Progressive Disclosure (primary)

| Code | Severity | Description |
|------|----------|-------------|
| CD030 | info | No progressive disclosure detected in a file over 60 lines. Point to separate docs for task-specific information. |
| CD040 | info | (Good practice) Progressive disclosure pattern detected. |

## Content Quality (primary)

Based on [The Complete Guide to CLAUDE.md](https://www.builder.io/blog/claude-md-guide).

| Code | Severity | Description |
|------|----------|-------------|
| CD050 | warning | Generic advice detected (e.g., "write clean code", "follow best practices"). Replace with project-specific instructions. |
| CD051 | info | No project description or context found. Start with what the project is. |
| CD052 | info | No build/test/lint commands found. Include commands Claude needs to verify changes. |
| CD053 | info | No negative instructions in a file over 30 lines. Specify what NOT to do, not just what to do. |
| CD054 | info | No code examples in a file over 50 lines. Concrete examples trump abstract rules. |
| CD041 | info | (Good practice) Negative instructions detected (don't, avoid, never). |
| CD042 | info | (Good practice) Code examples detected. |

## Referenced Documentation (primary)

These rules validate files referenced via progressive disclosure (e.g., "see docs/architecture.md"). References are followed **recursively** — if `docs/architecture.md` references `docs/patterns.md`, the full tree is resolved. Circular references are detected and broken automatically.

| Code | Severity | Description |
|------|----------|-------------|
| CD031 | error | Referenced documentation file not found. Remove broken references or create the missing files. |
| CD032 | warning | Referenced documentation file is stale (not updated in 90+ days, configurable with `-stale-threshold`). Review and update or remove. |
| CD033 | warning | Combined instruction count across all context files exceeds 200. Trim instructions. |

## Cross-File Consistency (primary)

| Code | Severity | Description |
|------|----------|-------------|
| CD034 | warning | Same instructions found in multiple context files. Keep each instruction in one place. |

## Repository-Level Rules

These rules only fire when scanning a directory (`context-doctor .`).

| Code | Severity | Description |
|------|----------|-------------|
| CD060 | error | Multiple CLAUDE.md files detected. A repo should have exactly one CLAUDE.md at the root. Use progressive disclosure to reference supporting docs. (-30 score penalty) |

The repo report also lists **orphan docs** — `.md` files in the repo that aren't referenced by any CLAUDE.md. These aren't errors, but help you spot documentation that could be linked or cleaned up.

## Custom Rules

You can create custom rules by adding YAML files to a `.context-doctor/` directory. Rules follow this structure:

```yaml
version: "1.0"
rules:
  - code: CUSTOM001
    description: My custom rule
    severity: warning  # error, warning, or info
    category: my-category
    primaryOnly: true  # optional: only run against CLAUDE.md, not referenced docs
    matchSpec:
      action: regexMatch
      patterns:
        - "pattern to match"
    errorMessage: "Message shown when rule triggers"
    suggestion: "How to fix the issue"
```

### Available Fields

| Field | Required | Description |
|-------|----------|-------------|
| `code` | yes | Unique rule identifier (e.g., `CUSTOM001`) |
| `description` | yes | Short description of the rule |
| `severity` | yes | `error`, `warning`, or `info` |
| `category` | no | Group rules under a category heading |
| `primaryOnly` | no | If `true`, only runs against CLAUDE.md, not referenced docs (default: `false`) |
| `matchSpec` | yes | The condition to check (see below) |
| `errorMessage` | yes | Message shown when the rule triggers |
| `suggestion` | no | How to fix the issue |
| `links` | no | URLs for further reading |

### Available Actions

- `greaterThan` - Compare metric against a value
- `lessThan` - Compare metric against a value
- `equals` / `notEquals` - Exact match
- `contains` / `notContains` - Substring match
- `regexMatch` - Match against regex patterns
- `regexNotMatch` - Inverse regex match
- `isPresent` / `notPresent` - Check for pattern existence
- `and` - All sub-conditions must match
- `or` - Any sub-condition must match

### Available Metrics

- `lineCount` - Number of lines in the file
- `instructionCount` - Estimated number of instructions
- `broken_references_count` - Number of broken references (primary file only)
- `stale_references_count` - Number of stale references (primary file only)
- `total_instruction_count` - Combined instructions across all context files
- `duplicate_instruction_count` - Number of duplicated instructions across files
