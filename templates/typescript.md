# Project Guidelines

## Build & Test

```bash
npx tsc --noEmit   # type check
npm run build
npm test
```

## Code Style

- Enable `strict` mode in `tsconfig.json`
- Prefer explicit return types on exported functions
- Use `interface` for object shapes, `type` for unions/intersections
- Avoid `any`; use `unknown` when the type is truly unknown

## Linting

```bash
npm run lint
```

## Project Structure

- Describe your directory layout here
- Document key modules and their responsibilities

## Testing

- Write tests with proper type coverage
- Mock external dependencies with typed mocks
