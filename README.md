# ⚓ Wharf

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A terminal UI for managing Docker Compose stacks.

<!-- TODO: add screenshot/gif -->

## Features

- Browse running Docker Compose projects and services
- Start, stop, restart services with a single keystroke
- Docker Compose up/down with confirmation
- Real-time CPU and memory monitoring
- Stream container logs with follow/pause mode
- Inspect container details — ports, volumes, environment, networks
- View compose file with syntax highlighting
- Manage volumes and networks
- Vim-style navigation (hjkl, gg/G, /, :q)
- Inline search/filter
- Configurable via `~/.config/wharf/config.yaml`
- Single binary, zero dependencies

## Installation

### From releases

Download the latest archive for your platform from [GitHub Releases](https://github.com/idesyatov/wharf/releases), extract and run:

```bash
tar xzf wharf-v*.tar.gz
./wharf
```

| Platform | Archive |
|----------|---------|
| Linux amd64 | `wharf-vX.X.X-linux-amd64.tar.gz` |
| macOS Intel | `wharf-vX.X.X-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `wharf-vX.X.X-darwin-arm64.tar.gz` |
| Windows amd64 | `wharf-vX.X.X-windows-amd64.zip` |

### From source (requires Docker)

```bash
git clone https://github.com/idesyatov/wharf.git
cd wharf
make docker-build-all
```

## Usage

```bash
wharf            # start TUI
wharf --config   # show config path and current settings
```

Make sure the Docker daemon is running and accessible.

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate up / down |
| `h` / `←` / `Esc` | Go back |
| `l` / `→` / `Enter` | Select / drill down |
| `gg` / `G` | Jump to top / bottom |
| `/` | Filter / search |
| `s` | Start service |
| `S` | Stop service |
| `r` | Restart service |
| `u` | Docker Compose up |
| `d` | Docker Compose down (with confirmation) |
| `L` | View logs |
| `c` | View compose file |
| `v` | View volumes |
| `n` | View networks |
| `f` | Toggle log follow mode |
| `?` | Help |
| `q` / `:q` | Quit |

## Configuration

Wharf looks for config at `~/.config/wharf/config.yaml`:

```yaml
poll_interval: 2s
log_tail: 100
max_log_lines: 1000
keybindings:
  quit: "ctrl+q"
```

## Development

Go is **not** required locally. All build, test, and lint commands run inside Docker.

```bash
make help        # show all commands
make docker-run  # run TUI via Docker
make docker-test # run tests
make docker-vet  # go vet
```

If you have Go installed locally:

```bash
make run         # run TUI
make test        # run tests
make build       # build for current platform
```

### Releasing

```bash
make release VERSION=v0.2.0
```

GitHub Actions will build binaries for all platforms, package them into archives with README and LICENSE, and create a GitHub Release.

## Tech Stack

- **Go** + [Bubbletea](https://github.com/charmbracelet/bubbletea) (TUI framework)
- **Lipgloss** (styling)
- **Docker SDK** for Go

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
