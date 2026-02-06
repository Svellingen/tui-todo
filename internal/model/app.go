// Package model contains the bubbletea models for the TUI.
package model

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/macone/todo-cli/internal/storage"
	"github.com/macone/todo-cli/internal/task"
	"github.com/macone/todo-cli/internal/ui"
)

const maxUndoStack = 20

// appMode tracks the current interaction mode.
type appMode int

const (
	modeNormal appMode = iota
	modeInput          // text input active (add/edit)
	modeConfirmDelete  // waiting for y/n on delete
)

// undoEntry stores a snapshot of the task file for undo.
type undoEntry struct {
	content string // raw markdown content for restoring
	desc    string // human-readable description
}

// flashMsg clears the status message after a delay.
type flashMsg struct{}

// App is the top-level bubbletea model.
type App struct {
	store    *storage.Store
	taskFile *storage.TaskFile
	list     TaskListModel
	input    TaskInputModel
	mode     appMode
	width    int
	height   int
	err      error

	undoStack []undoEntry
	statusMsg string // temporary message shown in help bar
}

// NewApp creates a new App model with the given store.
func NewApp(store *storage.Store) App {
	return App{
		store: store,
		input: NewTaskInput(),
	}
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

// pushUndo saves the current state for undo.
func (a *App) pushUndo(desc string) {
	w := storage.NewWriter()
	content := w.Write(a.taskFile)
	a.undoStack = append(a.undoStack, undoEntry{content: content, desc: desc})
	if len(a.undoStack) > maxUndoStack {
		a.undoStack = a.undoStack[1:]
	}
}

// save writes the task file to disk.
func (a *App) save() tea.Cmd {
	tf := a.taskFile
	s := a.store
	return func() tea.Msg {
		if err := s.Save(tf); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// flash sets a temporary status message that clears after 2 seconds.
func flash(msg string) (string, tea.Cmd) {
	return msg, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return flashMsg{}
	})
}

// Update handles messages.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg:
		a.taskFile = msg.tf
		a.list = NewTaskListModel(a.taskFile)
		a.list.SetSize(a.width, a.height-1)
		return a, nil

	case errMsg:
		a.err = msg.err
		return a, nil

	case flashMsg:
		a.statusMsg = ""
		return a, nil

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.list.SetSize(a.width, a.height-1)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle confirm-delete mode
	if a.mode == modeConfirmDelete {
		switch key {
		case "y":
			return a.doDelete()
		default:
			a.mode = modeNormal
			a.statusMsg = ""
			return a, nil
		}
	}

	// Handle text input mode
	if a.mode == modeInput {
		switch key {
		case "enter":
			return a.commitInput()
		case "esc":
			a.input.Cancel()
			a.mode = modeNormal
			return a, nil
		default:
			cmd := a.input.Update(msg)
			return a, cmd
		}
	}

	// Normal mode
	switch key {
	case "ctrl+c":
		return a, tea.Quit
	case ui.KeyQuit:
		return a, tea.Quit
	case ui.KeyDown:
		a.list.MoveDown()
	case ui.KeyUp:
		a.list.MoveUp()
	case ui.KeySectionDown:
		a.list.JumpNextSection()
	case ui.KeySectionUp:
		a.list.JumpPrevSection()
	case ui.KeyDone:
		return a.toggleDone()
	case ui.KeyAdd:
		a.mode = modeInput
		a.input.StartAdd()
		return a, nil
	case ui.KeyEdit:
		return a.startEdit()
	case ui.KeyDelete:
		return a.confirmDelete()
	case ui.KeyStatus:
		return a.cycleStatus()
	case ui.KeyPrio:
		return a.cyclePriority()
	case ui.KeyUndo:
		return a.undo()
	}

	return a, nil
}

func (a App) toggleDone() (tea.Model, tea.Cmd) {
	t := a.list.SelectedTaskItem()
	if t == nil {
		return a, nil
	}
	a.pushUndo("toggle done")
	if t.Status == task.StatusDone {
		t.Status = task.StatusTodo
	} else {
		t.Status = task.StatusDone
	}
	a.statusMsg, _ = flash("Done!")
	cmd := a.save()
	return a, tea.Batch(cmd, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return flashMsg{} }))
}

func (a App) startEdit() (tea.Model, tea.Cmd) {
	idx := a.list.SelectedTaskIndex()
	if idx < 0 {
		return a, nil
	}
	a.mode = modeInput
	a.input.StartEdit(idx, a.taskFile.Tasks[idx].Title)
	return a, nil
}

func (a App) commitInput() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(a.input.Value())
	if value == "" {
		a.input.Cancel()
		a.mode = modeNormal
		return a, nil
	}

	a.pushUndo("input")

	switch a.input.Mode() {
	case inputAdd:
		a.addTask(value)
		var msg string
		var cmd tea.Cmd
		msg, cmd = flash("Added: " + value)
		a.statusMsg = msg
		a.input.Cancel()
		a.mode = modeNormal
		return a, tea.Batch(a.save(), cmd)

	case inputEdit:
		idx := a.input.EditIndex()
		if idx >= 0 && idx < len(a.taskFile.Tasks) {
			// Parse metadata from the edited value
			title, priority, tags, dueDate := storage.ParseMetadata(value)
			a.taskFile.Tasks[idx].Title = title
			if priority != task.PriorityNone {
				a.taskFile.Tasks[idx].Priority = priority
			}
			if len(tags) > 0 {
				a.taskFile.Tasks[idx].Tags = tags
			}
			if dueDate != nil {
				a.taskFile.Tasks[idx].DueDate = dueDate
			}
		}
		a.input.Cancel()
		a.mode = modeNormal
		return a, a.save()
	}

	a.input.Cancel()
	a.mode = modeNormal
	return a, nil
}

func (a *App) addTask(value string) {
	title, priority, tags, dueDate := storage.ParseMetadata(value)
	newTask := task.Task{
		Title:    title,
		Status:   task.StatusTodo,
		Priority: priority,
		Tags:     tags,
		DueDate:  dueDate,
	}
	a.taskFile.Tasks = append(a.taskFile.Tasks, newTask)

	// Add task line to the Backlog section
	newLine := storage.Line{
		Type:      storage.LineTask,
		TaskIndex: len(a.taskFile.Tasks) - 1,
	}

	inserted := false
	for i, line := range a.taskFile.Lines {
		if line.Type == storage.LineSection {
			for _, sec := range a.taskFile.Sections {
				if sec.Line == line.Number && sec.Status == task.StatusTodo {
					insertIdx := i + 1
					for insertIdx < len(a.taskFile.Lines) &&
						a.taskFile.Lines[insertIdx].Type == storage.LineText &&
						strings.TrimSpace(a.taskFile.Lines[insertIdx].Raw) == "" {
						insertIdx++
					}
					a.taskFile.Lines = append(a.taskFile.Lines[:insertIdx],
						append([]storage.Line{newLine}, a.taskFile.Lines[insertIdx:]...)...)
					inserted = true
					break
				}
			}
			if inserted {
				break
			}
		}
	}
	if !inserted {
		a.taskFile.Lines = append(a.taskFile.Lines, newLine)
	}

	a.list.rebuildItems()
}

func (a App) confirmDelete() (tea.Model, tea.Cmd) {
	if a.list.SelectedTaskItem() == nil {
		return a, nil
	}
	a.mode = modeConfirmDelete
	a.statusMsg = "Delete? y/n"
	return a, nil
}

func (a App) doDelete() (tea.Model, tea.Cmd) {
	idx := a.list.SelectedTaskIndex()
	if idx < 0 {
		a.mode = modeNormal
		a.statusMsg = ""
		return a, nil
	}

	a.pushUndo("delete")
	title := a.taskFile.Tasks[idx].Title

	// Remove task from Tasks slice
	a.taskFile.Tasks = append(a.taskFile.Tasks[:idx], a.taskFile.Tasks[idx+1:]...)

	// Remove the corresponding line and fix task indices
	newLines := make([]storage.Line, 0, len(a.taskFile.Lines))
	for _, line := range a.taskFile.Lines {
		if line.Type == storage.LineTask && line.TaskIndex == idx {
			continue // skip the deleted task's line
		}
		if line.Type == storage.LineTask && line.TaskIndex > idx {
			line.TaskIndex-- // adjust indices
		}
		newLines = append(newLines, line)
	}
	a.taskFile.Lines = newLines

	a.list.rebuildItems()
	a.mode = modeNormal

	var msg string
	var cmd tea.Cmd
	msg, cmd = flash("Deleted: " + title)
	a.statusMsg = msg
	return a, tea.Batch(a.save(), cmd)
}

func (a App) cycleStatus() (tea.Model, tea.Cmd) {
	t := a.list.SelectedTaskItem()
	if t == nil {
		return a, nil
	}
	a.pushUndo("cycle status")
	switch t.Status {
	case task.StatusTodo:
		t.Status = task.StatusInProgress
	case task.StatusInProgress:
		t.Status = task.StatusDone
	case task.StatusDone:
		t.Status = task.StatusTodo
	}
	return a, a.save()
}

func (a App) cyclePriority() (tea.Model, tea.Cmd) {
	t := a.list.SelectedTaskItem()
	if t == nil {
		return a, nil
	}
	a.pushUndo("cycle priority")
	switch t.Priority {
	case task.PriorityNone:
		t.Priority = task.PriorityLow
	case task.PriorityLow:
		t.Priority = task.PriorityMedium
	case task.PriorityMedium:
		t.Priority = task.PriorityHigh
	case task.PriorityHigh:
		t.Priority = task.PriorityNone
	}
	return a, a.save()
}

func (a App) undo() (tea.Model, tea.Cmd) {
	if len(a.undoStack) == 0 {
		a.statusMsg = "Nothing to undo"
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return flashMsg{} })
	}

	entry := a.undoStack[len(a.undoStack)-1]
	a.undoStack = a.undoStack[:len(a.undoStack)-1]

	p := storage.NewParser()
	tf, err := p.Parse(entry.content)
	if err != nil {
		a.err = err
		return a, nil
	}
	a.taskFile = tf
	a.list = NewTaskListModel(a.taskFile)
	a.list.SetSize(a.width, a.height-1)

	var msg string
	var cmd tea.Cmd
	msg, cmd = flash("Undone: " + entry.desc)
	a.statusMsg = msg
	return a, tea.Batch(a.save(), cmd)
}

// View renders the TUI.
func (a App) View() string {
	if a.err != nil {
		return "Error: " + a.err.Error()
	}
	if a.taskFile == nil {
		return "Loading..."
	}

	var sb strings.Builder
	sb.WriteString(a.list.View())

	// Show input if active
	if a.mode == modeInput {
		prefix := "Add: "
		if a.input.Mode() == inputEdit {
			prefix = "Edit: "
		}
		sb.WriteString(prefix + a.input.View() + "\n")
	}

	// Help/status bar
	var helpText string
	if a.statusMsg != "" {
		helpText = a.statusMsg
	} else if a.mode == modeConfirmDelete {
		helpText = "Delete? y/n"
	} else {
		helpText = "j/k: navigate  d: done  a: add  e: edit  x: delete  s: status  p: priority  u: undo  q: quit"
	}
	sb.WriteString(ui.HelpBar.Render(helpText))

	return sb.String()
}
