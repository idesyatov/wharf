# Архитектура — Wharf

## Обзор
Wharf — TUI-утилита для управления Docker-контейнерами и Compose стеками. Интерактивный терминальный интерфейс с vim-like навигацией для мониторинга и управления контейнерами. Поддерживает как Compose-проекты, так и standalone-контейнеры (запущенные через `docker run`).

## Стек технологий

| Компонент | Технология |
|-----------|-----------|
| Язык | Go 1.24 |
| TUI | Bubbletea (Charm) |
| Стилизация | Lipgloss |
| Docker API | docker/docker/client v25 |
| Конфигурация | gopkg.in/yaml.v3 |
| Compose CLI | os/exec -> docker compose |
| Релизы | goreleaser v2 |
| CI/CD | GitHub Actions (go vet, go test, golangci-lint) |
| Дистрибуция | GitHub Releases, Homebrew tap, Docker (ghcr.io) |

## Архитектура приложения

### Слои

```
+-------------------------------------------------------------+
|                         TUI Layer                           |
|  Projects | Services | Detail | Logs | Compose | Top | Help |
|  Volumes | Networks | Images | Events | System | EnvFile    |
|  FileBrowser | Hosts                                        |
+-------------------------------------------------------------+
|                    Navigation Layer                         |
|  Vim keys, Command mode (:q, :go, :exec, :theme, :validate) |
|  Filter (/), Sort (1-6), Bulk (Space), Mouse, Tab-complete  |
+-------------------------------------------------------------+
|                      Domain Layer                           |
|  Project/Service/Container management, Stats, Health checks |
|  Custom commands, Compose operations, File browser          |
+-------------------------------------------------------------+
|                  Infrastructure Layer                       |
|  Docker SDK Client, Compose CLI, Config, Version, Themes    |
+-------------------------------------------------------------+
```

### Граф навигации

```
Projects --> Services --> Detail --> Logs
   |            |            |
   |            +---> Logs   +---> Exec (shell)
   |            +---> Compose Preview ---> Edit ($EDITOR)
   |            +---> Volumes
   |            +---> Networks
   |            +---> Top (container)
   |            +---> Env Preview
   |            +---> File Browser
   |
   +---> Top (project)
   +---> Images
   +---> Events
   +---> System (disk usage)
   +---> Hosts (Host Switcher)

Help (?) и Command mode (:) доступны из любого view
```

## Структура кода

```
wharf/
+-- cmd/wharf/
|   +-- main.go                  # Точка входа, --version, --config, загрузка темы
+-- internal/
|   +-- config/
|   |   +-- config.go            # YAML конфиг, bookmarks, theme, mouse, docker_host, custom_commands
|   +-- version/
|   |   +-- version.go           # Version/Commit/BuildDate через ldflags
|   |   +-- update.go            # CheckUpdate() через GitHub Releases API
|   +-- tui/
|   |   +-- app.go               # Корневая model, Init, state, layout
|   |   +-- app_routing.go       # Обработка всех Msg (navigation, data, ticks, events)
|   |   +-- app_render.go        # View() и menu bar рендеринг
|   |   +-- app_commands.go      # Command mode обработчики (:up, :down, :go, :exec, :validate)
|   |   +-- cmdmode.go           # Command mode (:q, :theme, :go, :exec, :validate, Tab-complete)
|   |   +-- views/
|   |       +-- projects.go      # Список compose-проектов, bulk select, bookmarks
|   |       +-- services.go      # Сервисы + CPU/RAM + health + actions + custom commands
|   |       +-- detail.go        # Детали контейнера, ports, env, networks, health log
|   |       +-- logs.go          # Стриминг логов, поиск (/pattern, n/N), export (w)
|   |       +-- compose.go       # Compose file preview + edit ($EDITOR)
|   |       +-- top.go           # Resource Monitor — braille charts CPU/MEM, Network I/O
|   |       +-- volumes.go       # Управление volumes, remove, prune
|   |       +-- networks.go      # Управление networks
|   |       +-- images.go        # Images: pull, prune
|   |       +-- events.go        # Live Docker events stream
|   |       +-- system.go        # docker system df, prune
|   |       +-- envfile.go       # .env file preview с подсветкой
|   |       +-- filebrowser.go   # Просмотр файлов внутри контейнера (ls -la, cat)
|   |       +-- hosts.go         # Host Switcher — список хостов, add/delete, connect
|   |       +-- helpers.go       # Shared helpers для views
|   |       +-- help.go          # Help screen с поиском (/, n/N)
|   +-- ui/
|   |   +-- keys.go              # 35+ keybindings, ApplyKeyBindings из конфига
|   |   +-- styles.go            # Мягкая тёмная палитра, FormatMenuItem, Separator
|   |   +-- theme.go             # LoadTheme/ApplyTheme, встроенные dark/light, кастомные из YAML
|   |   +-- clipboard.go         # CopyToClipboard через OSC 52
|   |   +-- sparkline.go         # Unicode sparkline графики, ColoredSparkline
|   |   +-- barchart.go          # Braille line chart (BrailleChart, Bresenham)
|   +-- docker/
|   |   +-- client.go            # Docker SDK обёртка: Stats, Inspect, Volumes, Networks, Images,
|   |   |                        # Events, SystemDf, ExecOutput, DetectShell, SubscribeEvents
|   |   +-- compose.go           # ListProjects (compose + standalone), composeExec helper, compose operations
|   |   +-- types.go             # Container, Service, Project, Stats, Image, Event,
|   |                            # Volume, Network, HealthCheck, SystemDf
|   +-- util/
|       +-- browser.go           # OpenBrowser кросс-платформа
|       +-- editor.go            # DetectEditor ($VISUAL -> $EDITOR -> vi)
+-- examples/                    # 4 тестовых compose-стека
+-- .goreleaser.yaml             # goreleaser v2, 5 платформ, homebrew tap
+-- .github/workflows/
|   +-- release.yml              # goreleaser action
|   +-- ci.yml                   # go vet + go test + golangci-lint
+-- .golangci.yml                # linter конфиг
+-- CONTRIBUTING.md              # гайд для контрибьюторов
+-- .github/ISSUE_TEMPLATE/      # bug_report.md, feature_request.md
+-- Makefile                     # docker-*/local команды, -buildvcs=false, ldflags
```

## Layout экрана

```
+--------------------------------------------------+
| Wharf                     Docker: * local v0.9   |  <- Info bar
+--------------------------------------------------+
| > multi-service                                  |  <- Breadcrumbs
+--------------------------------------------------+
|                                                  |
|  SERVICE  CONTAINER  STATUS  H  CPU  MEM  ...    |  <- Content area
|  api      ms-api-1   running -  2%   128Mi ...   |
|  redis    ms-redis   running -  0%   7Mi  ...    |
|                                                  |
+--------------------------------------------------+
| <s>tart <S>top <r>estart <e>xec <L>ogs           |  <- Menu bar (2 строки)
| <t>op <F>iles <b>uild <c>ompose <v>ol </>filter  |
+--------------------------------------------------+
| Loading stats...                                 |  <- Status line
+--------------------------------------------------+
```

## Braille Charts (internal/ui/barchart.go)

Container Top View использует braille line charts для визуализации CPU и MEM.

- `BrailleChart()` — рендерит braille line chart (plain text, без ANSI)
- Braille-символы (U+2800–U+28FF) — матрица 2×4 точки, удвоенное разрешение
- Алгоритм Брезенхема для плавных диагональных линий
- Автошкала по максимальному значению из истории
- CPU-график окрашен по зонам высоты (зелёный/жёлтый/красный)
- MEM-график одноцветный по текущему уровню
- Рамки через lipgloss.Border() (RoundedBorder)

## Smart Polling

Wharf использует двухуровневый polling для снижения нагрузки на Docker daemon:

- **FastTick** (poll_interval, по умолчанию 2с) — обновление метрик (CPU/MEM/health/top)
- **SlowTick** (10с) — обновление списка проектов и контейнеров
- **Event-driven** — немедленное обновление при container start/stop/die/create/destroy

Это снижает количество вызовов Docker API при большом числе контейнеров.

## Деплой / Дистрибуция
- Один статический бинарник, zero dependencies
- Кросс-компиляция: Linux amd64/arm64, macOS amd64/arm64, Windows amd64
- goreleaser: автосборка + Homebrew formula + Docker image при тегах v*
- Docker: `docker pull ghcr.io/idesyatov/wharf:latest` (multi-arch: amd64/arm64)
- Homebrew: `brew tap idesyatov/tap && brew install wharf`
- Install-скрипт: `curl -sL .../install.sh | bash`
- Ротация Docker-пакетов: хранение 5 последних версий
