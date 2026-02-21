# Project Guidelines

## Build & Test

```bash
go build ./...
go test ./...
go test ./... -cover  # with coverage
```

## Code Style

- Use `gofmt` for formatting
- Run `go vet` before committing
- Keep functions small and focused
- Prefer explicit error handling over panics
- Return errors instead of using log.Fatal in library code

## Linting

```bash
golangci-lint run
```

## Project Structure

- Describe your package layout here
- Document any non-obvious package responsibilities

## Testing

- Use table-driven tests with `t.Run` subtests
- Name tests after the scenario being tested
- Run `go test ./... -race` to check for data races
