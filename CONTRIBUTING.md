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
   make docker-build
   ./bin/wharf-linux-amd64
   ```

## Making Changes

1. Create a branch: `git checkout -b feature/my-feature`
2. Make your changes
3. Verify: `make docker-check` (runs build + vet + test + lint)
4. Commit with a descriptive message
5. Push and open a Pull Request

If `make` is not in your PATH, use docker compose directly:
```bash
docker compose -f docker-compose.dev.yml run --rm dev go vet ./...
docker compose -f docker-compose.dev.yml run --rm dev go test -buildvcs=false ./...
```

## Project Structure

```
cmd/wharf/main.go         — entry point
internal/config/          — configuration (YAML)
internal/docker/          — Docker SDK wrapper (client.go, compose.go, types.go)
internal/tui/             — TUI application (app.go, app_routing.go, app_render.go, app_commands.go)
internal/tui/views/       — individual view screens (15+ views)
internal/ui/              — styles, keys, clipboard, themes, braille charts
internal/version/         — version and update check
internal/util/            — utilities (browser open, editor detect)
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
