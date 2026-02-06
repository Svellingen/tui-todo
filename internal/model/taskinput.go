package model

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// inputMode describes what the text input is being used for.
type inputMode int

const (
	inputNone inputMode = iota
	inputAdd
	inputEdit
)

// TaskInputModel wraps a bubbles textinput for add/edit operations.
type TaskInputModel struct {
	input textinput.Model
	mode  inputMode
	// editIndex is the task index being edited (only valid when mode == inputEdit).
	editIndex int
}

// NewTaskInput creates a new TaskInputModel.
func NewTaskInput() TaskInputModel {
	ti := textinput.New()
	ti.Placeholder = "Task title..."
	ti.CharLimit = 256
	return TaskInputModel{input: ti}
}

// StartAdd begins the add-task flow.
func (m *TaskInputModel) StartAdd() {
	m.mode = inputAdd
	m.editIndex = -1
	m.input.SetValue("")
	m.input.Focus()
}

// StartEdit begins the edit-task flow with the given title pre-filled.
func (m *TaskInputModel) StartEdit(taskIndex int, currentTitle string) {
	m.mode = inputEdit
	m.editIndex = taskIndex
	m.input.SetValue(currentTitle)
	m.input.CursorEnd()
	m.input.Focus()
}

// Cancel exits input mode without saving.
func (m *TaskInputModel) Cancel() {
	m.mode = inputNone
	m.input.Blur()
}

// Active returns true if the input is active.
func (m TaskInputModel) Active() bool {
	return m.mode != inputNone
}

// Mode returns the current input mode.
func (m TaskInputModel) Mode() inputMode {
	return m.mode
}

// EditIndex returns the task index being edited.
func (m TaskInputModel) EditIndex() int {
	return m.editIndex
}

// Value returns the current text input value.
func (m TaskInputModel) Value() string {
	return m.input.Value()
}

// Update forwards messages to the textinput model.
func (m *TaskInputModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

// View renders the text input.
func (m TaskInputModel) View() string {
	return m.input.View()
}
