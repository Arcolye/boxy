package main

import (
	"fmt"
	"os"

	"boxy/internal/config"
	"boxy/internal/manager"
	"boxy/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	mgr := manager.Detect()
	if mgr == nil {
		fmt.Fprintln(os.Stderr, "Error: No supported package manager found (brew or apt)")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	m := tui.NewModel(mgr, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
