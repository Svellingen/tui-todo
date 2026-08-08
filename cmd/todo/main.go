package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/macone/todo-cli/internal/model"
	"github.com/macone/todo-cli/internal/storage"
)

func main() {
	// The root command resolves the file, honouring --file, and hands the
	// store over.
	launchTUI := func(store *storage.Store) error {
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
