# Project Guidelines

## Issue Tracking

This project uses GitHub Issues for tracking bugs, feature requests, and tasks.

## Go Development

### Build & Test

```bash
make build      # Build the binary
make test       # Run tests
make clean      # Remove build artifacts
```

### Code Style

- Use `gofmt` for formatting
- Run `go vet` before committing
- Keep functions small and focused
- Prefer explicit error handling over panics

### Project Structure

- `main.go` - Entry point, CLI handling, report rendering
- `rules/` - Rule engine and built-in rules

### Key Docs

- `RULES.md` - Complete reference for all built-in rules and custom rule authoring
