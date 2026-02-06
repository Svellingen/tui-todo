package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/macone/todo-cli/internal/model"
	"github.com/macone/todo-cli/internal/storage"
)

func main() {
	launchTUI := func() error {
		store := storage.NewStore(defaultFile)
		m := model.NewApp(store)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	}

	cmd := newRootCmd(os.Stdout, launchTUI)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
