⏺ Boxy - TUI Package Manager

  Boxy is a cross-platform terminal package manager frontend written in Go using Bubble
  Tea and Lipgloss.

  What it does

  - Provides an interactive TUI for managing system packages
  - Supports Homebrew (macOS) and APT (Linux)
  - Auto-detects which package manager is available

  Key Features

  - Browse installed and bookmarked packages
  - Search packages by name/keyword
  - Install/Uninstall packages with confirmation dialogs
  - Bookmark packages for quick access (persisted in ~/.config/boxy/packages.yaml)
  - View detailed package info (version, description, status)

  Controls
  ┌───────────────┬───────────────────┐
  │      Key      │      Action       │
  ├───────────────┼───────────────────┤
  │ j/k or arrows │ Navigate          │
  ├───────────────┼───────────────────┤
  │ Enter         │ View package info │
  ├───────────────┼───────────────────┤
  │ i             │ Install           │
  ├───────────────┼───────────────────┤
  │ u             │ Uninstall         │
  ├───────────────┼───────────────────┤
  │ b             │ Toggle bookmark   │
  ├───────────────┼───────────────────┤
  │ /             │ Search            │
  ├───────────────┼───────────────────┤
  │ q             │ Quit              │
  └───────────────┴───────────────────┘
  Structure

  cmd/boxy/main.go          # Entry point
  internal/config/          # YAML config management
  internal/manager/         # Package manager abstraction (brew, apt)
  internal/tui/             # Bubble Tea UI (app, keys, styles)
