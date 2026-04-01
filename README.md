# ⚓ Wharf

[![CI](https://github.com/idesyatov/wharf/actions/workflows/ci.yml/badge.svg)](https://github.com/idesyatov/wharf/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/idesyatov/wharf)](https://github.com/idesyatov/wharf/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/idesyatov/wharf)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/idesyatov/wharf)](https://goreportcard.com/report/github.com/idesyatov/wharf)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A terminal UI for managing Docker Compose stacks. Inspired by [k9s](https://github.com/derailed/k9s).

<!-- TODO: add screenshot/gif -->

## Features

- Browse Docker Compose projects and services
- Start, stop, restart services and compose projects
- Docker Compose up / stop / down / restart / build
- Exec into containers (shell) directly from TUI
- Real-time CPU and memory monitoring
- Stream container logs with follow/pause and search
- Inspect container details — ports, volumes, environment, networks
- View compose file with syntax highlighting
- Manage volumes, networks, and images
- Docker System overview with disk usage
- Live Docker events monitoring
- Remote Docker host support
- Bookmark favorite projects (★)
- Copy container ID to clipboard (OSC 52)
- Sort tables by any column
- Vim-style navigation (hjkl, gg/G, /, :q, arrows)
- Customizable themes (dark/light/custom)
- Auto-update check via GitHub Releases
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
wharf --version  # show version
wharf --config   # show config path and current settings
```

## Keybindings

### Navigation
| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move up / down |
| `h` / `←` / `Esc` | Go back |
| `l` / `→` / `Enter` | Select / drill down |
| `gg` / `G` | Top / bottom |
| `1`–`6` | Sort by column (repeat to reverse) |
| `/` | Filter / search |
| `?` | Help |
| `q` / `:q` | Quit |

### Service actions
| Key | Action |
|-----|--------|
| `s` | Start service |
| `S` | Stop service |
| `r` | Restart service |
| `e` | Exec into container (shell) |
| `o` | Open in browser |
| `L` | View logs |

### Compose actions
| Key | Action |
|-----|--------|
| `u` | Compose up (start project) |
| `d` | Compose stop (stop, keep containers) |
| `X` | Compose down (stop and REMOVE containers) |
| `R` | Compose restart |
| `b` / `B` | Build service / all |

### Views
| Key | Action |
|-----|--------|
| `c` | View compose file |
| `v` | Volumes |
| `n` | Networks |
| `i` | Images |
| `E` | Docker events |
| `D` | System disk usage |

### Other
| Key | Action |
|-----|--------|
| `*` | Toggle bookmark |
| `y` / `Y` | Copy ID / full info |
| `f` | Toggle log follow |
| `n` / `N` | Next / previous log search match |
| `P` | Prune (context-dependent) |

## Configuration

```yaml
# ~/.config/wharf/config.yaml
poll_interval: 2s
log_tail: 100
max_log_lines: 1000
theme: dark              # dark, light, or custom theme name
docker_host: ""          # e.g. tcp://192.168.1.10:2375 or ssh://user@host
bookmarks:
  - my-project
keybindings:
  quit: "ctrl+q"
```

Custom themes: `~/.config/wharf/themes/<name>.yaml`

## Development

```bash
make help                # show all commands
make docker-build-all    # cross-compile all platforms
make docker-run          # run TUI via Docker
make docker-test         # run tests
```

### Testing with example stacks

```bash
cd examples/simple-web && docker compose up -d && cd ../..
cd examples/multi-service && docker compose up -d && cd ../..
./bin/wharf-linux-amd64
```

### Releasing

```bash
make release VERSION=v0.4.2
```

## Tech Stack

- **Go** + [Bubbletea](https://github.com/charmbracelet/bubbletea) (TUI)
- **Lipgloss** (styling)
- **Docker SDK** for Go

## License

MIT — see [LICENSE](LICENSE).
