// Package model contains the bubbletea models for the TUI.
package model

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/macone/todo-cli/internal/storage"
	"github.com/macone/todo-cli/internal/ui"
)

// App is the top-level bubbletea model.
type App struct {
	store    *storage.Store
	taskFile *storage.TaskFile
	list     TaskListModel
	width    int
	height   int
	err      error
}

// NewApp creates a new App model with the given store.
func NewApp(store *storage.Store) App {
	return App{store: store}
}

// Init loads tasks from the store.
func (a App) Init() tea.Cmd {
	return func() tea.Msg {
		tf, err := a.store.Load()
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg{tf}
	}
}

type loadedMsg struct{ tf *storage.TaskFile }
type errMsg struct{ err error }

// Update handles messages.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg:
		a.taskFile = msg.tf
		a.list = NewTaskListModel(a.taskFile)
		a.list.SetSize(a.width, a.height-1) // reserve 1 line for help bar
		return a, nil

	case errMsg:
		a.err = msg.err
		return a, nil

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.list.SetSize(a.width, a.height-1)
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case ui.KeyQuit, "ctrl+c":
			return a, tea.Quit
		case ui.KeyDown:
			a.list.MoveDown()
		case ui.KeyUp:
			a.list.MoveUp()
		case ui.KeySectionDown:
			a.list.JumpNextSection()
		case ui.KeySectionUp:
			a.list.JumpPrevSection()
		}
	}

	return a, nil
}

// View renders the TUI.
func (a App) View() string {
	if a.err != nil {
		return "Error: " + a.err.Error()
	}
	if a.taskFile == nil {
		return "Loading..."
	}

	content := a.list.View()
	helpText := ui.HelpBar.Render("j/k: navigate  J/K: sections  q: quit  ?: help")
	return content + "\n" + helpText
}
