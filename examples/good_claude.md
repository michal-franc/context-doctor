# MyApp

TypeScript monorepo with React frontend and Node.js API.

## Structure

- `apps/web` - React SPA (Vite)
- `apps/api` - Express REST API
- `packages/shared` - Shared types and utils

## Development

```bash
pnpm install   # Install deps
pnpm dev       # Start all services
pnpm test      # Run tests
```

## Key Docs

For detailed guidance, read the relevant doc before starting:
- `docs/architecture.md` - System design and data flow
- `docs/api-patterns.md` - API conventions and error handling
- `docs/testing.md` - Test structure and mocking approach
