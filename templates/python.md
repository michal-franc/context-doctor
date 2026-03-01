# Project Guidelines

## Setup

```bash
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

## Build & Test

```bash
pytest
pytest --cov=src  # with coverage
```

## Code Style

- Format with `ruff format` or `black`
- Lint with `ruff check` or `flake8`
- Type check with `mypy`

## Project Structure

- Describe your package layout here
- Document entry points and key modules

## Testing

- Use `pytest` for all tests
- Place tests in `tests/` directory mirroring source structure
- Use fixtures for shared test setup
