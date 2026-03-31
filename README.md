# ⚓ Wharf

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A terminal UI for managing Docker Compose stacks. Inspired by [k9s](https://github.com/derailed/k9s).

<!-- TODO: add screenshot/gif -->

## Features

- Browse Docker Compose projects and services
- Start, stop, restart services with a single keystroke
- Docker Compose up/down with confirmation
- Exec into containers (shell) directly from TUI
- Docker Compose build (service or project)
- Real-time CPU and memory monitoring
- Stream container logs with follow/pause mode
- Inspect container details — ports, volumes, environment, networks
- View compose file with syntax highlighting
- Manage volumes and networks
- Image management — list, pull, prune
- Bookmark favorite projects (★)
- Copy container ID to clipboard (OSC 52)
- Vim-style navigation (hjkl, gg/G, /, :q, arrows)
- Inline search/filter
- Configurable via `~/.config/wharf/config.yaml`
- Single binary, zero dependencies

## Installation

### From releases

Download the latest archive from [GitHub Releases](https://github.com/idesyatov/wharf/releases):

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

### Navigation
| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move cursor up / down |
| `h` / `←` / `Esc` | Go back |
| `l` / `→` / `Enter` | Select / drill down |
| `gg` / `G` | Jump to top / bottom |
| `/` | Filter / search |
| `?` | Help |
| `q` / `:q` | Quit |

### Actions (Services view)
| Key | Action |
|-----|--------|
| `s` | Start service |
| `S` | Stop service |
| `r` | Restart service |
| `e` | Exec into container (shell) |
| `b` | Build service |
| `B` | Build all services |
| `u` | Docker Compose up |
| `d` | Docker Compose down |
| `L` | View logs |
| `c` | View compose file |
| `v` | View volumes |
| `n` | View networks |

### Other
| Key | Action |
|-----|--------|
| `i` | Images (from Projects view) |
| `*` | Toggle bookmark |
| `y` | Copy container ID |
| `f` | Toggle log follow |
| `x` | Remove volume |
| `P` | Prune (volumes/images) |

## Configuration

Wharf looks for config at `~/.config/wharf/config.yaml`:

```yaml
poll_interval: 2s
log_tail: 100
max_log_lines: 1000
bookmarks:
  - my-project
keybindings:
  quit: "ctrl+q"
```

## Development

Go is **not** required locally. All commands run inside Docker.

```bash
make help            # show all commands
make docker-run      # run TUI via Docker
make docker-test     # run tests
make docker-vet      # go vet
make docker-build-all  # cross-compile all platforms
```

If you have Go installed locally:

```bash
make run             # run TUI
make build           # build for current platform
```

### Testing with example stacks

```bash
cd examples/simple-web && docker compose up -d && cd ../..
cd examples/multi-service && docker compose up -d && cd ../..
cd examples/with-volumes && docker compose up -d && cd ../..
./bin/wharf-linux-amd64
```

### Releasing

```bash
make release VERSION=v0.3.0
```

## Tech Stack

- **Go** + [Bubbletea](https://github.com/charmbracelet/bubbletea) (TUI framework)
- **Lipgloss** (styling)
- **Docker SDK** for Go

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
