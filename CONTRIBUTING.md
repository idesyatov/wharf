# Contributing to Wharf

Thanks for your interest in contributing to Wharf!

## Development Setup

Go is **not** required locally — all commands run inside Docker.

1. Fork and clone the repository
2. Start example stacks for testing:
   ```bash
   cd examples/simple-web && docker compose up -d && cd ../..
   cd examples/multi-service && docker compose up -d && cd ../..
   ```
3. Build and run:
   ```bash
   make docker-build-linux
   ./bin/wharf-linux-amd64
   ```

## Making Changes

1. Create a branch: `git checkout -b feature/my-feature`
2. Make your changes
3. Verify: `make docker-build && make docker-vet && make docker-test`
4. Commit with a descriptive message
5. Push and open a Pull Request

## Project Structure

```
cmd/wharf/main.go         — entry point
internal/config/          — configuration (YAML)
internal/docker/          — Docker SDK wrapper
internal/tui/             — TUI application (Bubbletea)
internal/tui/views/       — individual view screens
internal/ui/              — styles, keys, clipboard, themes
internal/version/         — version and update check
internal/util/            — utilities (browser open)
```

## Code Style

- Follow standard Go conventions
- Run `make docker-vet` before committing
- Keep functions small and focused
- Add tests for new functionality

## Reporting Issues

- Use GitHub Issues
- Include: steps to reproduce, expected vs actual behavior, wharf version (`wharf --version`)
- Screenshots or terminal output are helpful

## Feature Requests

- Open a GitHub Issue with the "enhancement" label
- Describe the use case and expected behavior
