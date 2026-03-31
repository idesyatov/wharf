# ⚓ Wharf

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A terminal UI for managing Docker Compose stacks. Inspired by [k9s](https://github.com/derailed/k9s).

<!-- TODO: add screenshot/gif -->

## Features

- Browse Docker Compose projects and services
- Start, stop, restart services with a single keystroke
- Docker Compose up/down/build
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

### Actions
| Key | Action |
|-----|--------|
| `s` / `S` / `r` | Start / Stop / Restart service |
| `e` | Exec into container |
| `b` / `B` | Build service / all |
| `u` / `d` | Compose up / down |
| `L` | View logs |
| `o` | Open in browser |
| `c` | View compose file |
| `v` / `n` | Volumes / Networks |
| `i` | Images |
| `E` | Docker events |
| `D` | System disk usage |
| `*` | Toggle bookmark |
| `y` / `Y` | Copy ID / full info |
| `f` | Toggle log follow |
| `P` | Prune (context-dependent) |

### Logs search
| Key | Action |
|-----|--------|
| `/` | Search in logs |
| `n` / `N` | Next / previous match |
| `Esc` | Clear search |

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
make release VERSION=v0.4.0
```

## Tech Stack

- **Go** + [Bubbletea](https://github.com/charmbracelet/bubbletea) (TUI)
- **Lipgloss** (styling)
- **Docker SDK** for Go

## License

MIT — see [LICENSE](LICENSE).
