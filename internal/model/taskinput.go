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
	inputSearch
	inputTag
	inputContext
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
	// No "> " prompt: inline editors sit in a task row that already reads as
	// one, and the search and tag prompts supply their own label.
	ti.Prompt = ""
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

// StartSearch begins the search flow.
func (m *TaskInputModel) StartSearch() {
	m.mode = inputSearch
	m.editIndex = -1
	m.input.SetValue("")
	m.input.Placeholder = "Search..."
	m.input.Focus()
}

// StartTag begins the tag-add flow.
func (m *TaskInputModel) StartTag() {
	m.mode = inputTag
	m.editIndex = -1
	m.input.SetValue("")
	m.input.Placeholder = "Tag name..."
	m.input.Focus()
}

// StartContext begins the add-context flow.
func (m *TaskInputModel) StartContext() {
	m.mode = inputContext
	m.editIndex = -1
	m.input.SetValue("")
	m.input.Placeholder = "Context name..."
	m.input.Focus()
}

// Cancel exits input mode without saving.
func (m *TaskInputModel) Cancel() {
	m.mode = inputNone
	m.input.Placeholder = "Task title..."
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
// Update feeds a message to the text input.
//
// Runes that arrive in a single read come as one message, and bubbles matches
// its key bindings on the message's name -- so text containing "up" or "down"
// would be taken for suggestion navigation and swallowed. Splitting a
// multi-rune message into one message per rune keeps it as text; no binding is
// a single character.
func (m *TaskInputModel) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || keyMsg.Type != tea.KeyRunes || len(keyMsg.Runes) <= 1 {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	}

	var cmds []tea.Cmd
	for _, r := range keyMsg.Runes {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{r},
			Alt:   keyMsg.Alt,
			Paste: keyMsg.Paste,
		})
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// View renders the text input.
func (m TaskInputModel) View() string {
	return m.input.View()
}
