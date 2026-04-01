# ⚓ Wharf

[![CI](https://github.com/idesyatov/wharf/actions/workflows/ci.yml/badge.svg)](https://github.com/idesyatov/wharf/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/idesyatov/wharf)](https://github.com/idesyatov/wharf/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/idesyatov/wharf)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/idesyatov/wharf)](https://goreportcard.com/report/github.com/idesyatov/wharf)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A terminal UI for managing Docker Compose stacks. Inspired by [k9s](https://github.com/derailed/k9s).

<p align="center">
  <img src="demo.gif" alt="Wharf Demo" width="800">
</p>

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

### Homebrew (macOS/Linux)

```bash
brew tap idesyatov/tap
brew install wharf
```

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

### From source

```bash
git clone https://github.com/idesyatov/wharf.git
cd wharf
make docker-build-all    # Go not required — builds via Docker
```

## Usage

```bash
wharf            # start TUI
wharf --version  # show version
wharf --config   # show config path and current settings
```

## Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `j`/`k`/`↑`/`↓` | Navigate up/down | All |
| `h`/`←`/`Esc` | Go back | All |
| `l`/`→`/`Enter` | Select / drill down | All |
| `gg` / `G` | Jump to top / bottom | All |
| `1`–`6` | Sort by column | Tables |
| `/` | Filter / search | Projects, Services, Logs |
| `?` | Help | All |
| `q` / `:q` | Quit | All |
| | | |
| `s` / `S` / `r` | Start / Stop / Restart service | Services |
| `e` | Exec into container | Services, Detail |
| `L` | View logs | Services, Detail |
| `u` | Compose up | Projects |
| `d` | Compose stop | Projects |
| `X` | Compose down (removes containers) | Projects |
| `R` | Compose restart | Projects |
| `b` / `B` | Build service / all | Services |
| | | |
| `c` | Compose file preview | Services |
| `v` / `n` | Volumes / Networks | Services |
| `i` | Images | Projects |
| `E` | Docker events | Projects |
| `D` | System disk usage | Projects |
| `*` | Toggle bookmark | Projects |
| `y` / `Y` | Copy ID / full info | Services, Detail |
| `f` | Toggle log follow | Logs |
| `n` / `N` | Next / prev search match | Logs |
| `P` | Prune | Volumes, Images, System |
| `Space` | Toggle select (bulk) | Projects |

<details>
<summary>Command Mode</summary>

Press `:` then type a command, `Enter` to execute:

| Command | Action |
|---------|--------|
| `:q` / `:q!` | Quit |
| `:host` | Show current Docker host |
| `:theme dark` / `:theme light` | Switch theme |
| `:version` | Show version info |
| `:save [path]` | Save logs to file (Logs view) |
| `:help` | Show help |

</details>

## Configuration

```yaml
# ~/.config/wharf/config.yaml
poll_interval: 2s
log_tail: 100
max_log_lines: 1000
theme: dark              # dark, light, or custom theme name
docker_host: ""          # e.g. tcp://192.168.1.10:2375 or ssh://user@host
mouse: false             # enable mouse support
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
make docker-vet          # go vet
```

### Testing with example stacks

```bash
cd examples/multi-service && docker compose up -d && cd ../..
./bin/wharf-linux-amd64
```

## Tech Stack

**Go** + [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) + [Docker SDK](https://pkg.go.dev/github.com/docker/docker/client)

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

MIT — see [LICENSE](LICENSE).
