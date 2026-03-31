# Архитектура — Wharf

## Обзор
Wharf — TUI-утилита для управления Docker Compose стеками. Предоставляет интерактивный терминальный интерфейс с vim-like навигацией для мониторинга и управления контейнерами.

## Стек технологий

| Компонент | Технология |
|-----------|-----------|
| Язык | Go 1.24 |
| TUI | Bubbletea (Charm) |
| Стилизация | Lipgloss |
| Docker API | docker/docker/client v25 |
| Конфигурация | gopkg.in/yaml.v3 |
| Compose CLI | os/exec → docker compose |

## Среда разработки

Два режима: через Docker (префикс `docker-`) и локально (требует Go).
Полный список: `make help`

## Архитектура приложения

```
┌─────────────────────────────────────────────────────────┐
│                       TUI Layer                         │
│  ┌─────────┐ ┌─────────┐ ┌────────┐ ┌────┐ ┌────────┐   │
│  │Projects │ │Services │ │ Detail │ │Logs│ │  Help  │   │
│  └─────────┘ └─────────┘ └────────┘ └────┘ └────────┘   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐                    │
│  │Compose  │ │Volumes  │ │Networks │                    │
│  │Preview  │ │         │ │         │                    │
│  └─────────┘ └─────────┘ └─────────┘                    │
├─────────────────────────────────────────────────────────┤
│                  Navigation Layer                       │
│  Vim keybindings, command mode (:q), filter (/), gg/G   │
├─────────────────────────────────────────────────────────┤
│                   Domain Layer                          │
│  Project/Service/Container management, Stats, Actions   │
├─────────────────────────────────────────────────────────┤
│                Infrastructure Layer                     │
│  Docker SDK Client, Compose CLI (os/exec), Config       │
└─────────────────────────────────────────────────────────┘
```

## Структура кода

```
wharf/
├── cmd/wharf/
│   └── main.go              # Точка входа, --config флаг
├── internal/
│   ├── config/
│   │   └── config.go         # Загрузка ~/.config/wharf/config.yaml
│   ├── tui/
│   │   ├── app.go            # Корневая model, view switching, :q
│   │   └── views/
│   │       ├── projects.go   # Список compose-проектов
│   │       ├── services.go   # Сервисы + CPU/RAM + actions
│   │       ├── detail.go     # Детали контейнера
│   │       ├── logs.go       # Стриминг логов
│   │       ├── help.go       # Help screen
│   │       ├── compose.go    # Compose file preview
│   │       ├── volumes.go    # Управление volumes
│   │       └── networks.go   # Управление networks
│   ├── ui/
│   │   ├── keys.go           # KeyMap (20+ bindings)
│   │   └── styles.go         # Lipgloss палитра
│   └── docker/
│       ├── client.go         # Docker SDK, Stats, Inspect, Volumes, Networks
│       ├── compose.go        # ListProjects, ComposeUp/Down
│       └── types.go          # Container, Service, Project, Stats, Volume, Network
├── .github/workflows/
│   └── release.yml           # GitHub Actions — кросс-компиляция + релиз
├── Dockerfile.dev
├── docker-compose.dev.yml
├── Makefile                  # docker-* и локальные команды
├── LICENSE
└── README.md
```

## Граф навигации

```
Projects ──→ Services ──→ Detail ──→ Logs
                │              │
                ├──→ Logs      └──→ Logs
                ├──→ Compose Preview
                ├──→ Volumes
                └──→ Networks

Help (?) доступен из любого view
```

## Деплой / Дистрибуция
- Один статический бинарник
- Кросс-компиляция: Linux, macOS (Intel + Apple Silicon), Windows
- GitHub Actions: автосборка при тегах v*
- Релиз: `make release VERSION=v0.2.0`
