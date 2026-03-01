# Project Guidelines

## Build & Test

```bash
cargo build
cargo test
cargo clippy  # lint
```

## Code Style

- Use `rustfmt` for formatting (`cargo fmt`)
- Run `cargo clippy` before committing
- Prefer `Result` over `unwrap()`/`expect()` in library code
- Use `thiserror` or `anyhow` for error handling

## Project Structure

- Describe your crate/module layout here
- Document public API boundaries

## Testing

- Use `#[cfg(test)]` modules for unit tests
- Place integration tests in `tests/` directory
- Use `cargo test -- --nocapture` to see print output
