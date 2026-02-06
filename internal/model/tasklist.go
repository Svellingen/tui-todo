package model

import (
	"fmt"
	"strings"

	"github.com/macone/todo-cli/internal/storage"
	"github.com/macone/todo-cli/internal/task"
	"github.com/macone/todo-cli/internal/ui"
)

// item represents a renderable row in the task list -- either a section header,
// a task, or a blank line.
type itemType int

const (
	itemSection itemType = iota
	itemTask
	itemBlank
)

type item struct {
	kind      itemType
	section   string
	taskIndex int // index into TaskFile.Tasks
}

// TaskListModel handles rendering and navigating the task list.
type TaskListModel struct {
	taskFile *storage.TaskFile
	items    []item       // flattened list of renderable rows
	cursor   int          // index into items (only stops on task rows)
	width    int
	height   int
}

// NewTaskListModel creates a TaskListModel from a TaskFile.
func NewTaskListModel(tf *storage.TaskFile) TaskListModel {
	m := TaskListModel{taskFile: tf}
	m.rebuildItems()
	return m
}

// rebuildItems creates the flat item list from the TaskFile's lines.
func (m *TaskListModel) rebuildItems() {
	m.items = nil
	for _, line := range m.taskFile.Lines {
		switch line.Type {
		case storage.LineSection:
			m.items = append(m.items, item{kind: itemSection, section: line.Raw})
		case storage.LineTask:
			m.items = append(m.items, item{kind: itemTask, taskIndex: line.TaskIndex})
		case storage.LineText:
			m.items = append(m.items, item{kind: itemBlank})
		}
	}
	// Position cursor on first task if possible
	if m.cursor == 0 {
		m.moveToNextTask(0)
	}
}

// moveToNextTask moves the cursor to the next task item at or after pos.
func (m *TaskListModel) moveToNextTask(pos int) {
	for i := pos; i < len(m.items); i++ {
		if m.items[i].kind == itemTask {
			m.cursor = i
			return
		}
	}
}

// moveToPrevTask moves the cursor to the previous task item at or before pos.
func (m *TaskListModel) moveToPrevTask(pos int) {
	for i := pos; i >= 0; i-- {
		if m.items[i].kind == itemTask {
			m.cursor = i
			return
		}
	}
}

// MoveDown moves the cursor to the next task.
func (m *TaskListModel) MoveDown() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].kind == itemTask {
			m.cursor = i
			return
		}
	}
}

// MoveUp moves the cursor to the previous task.
func (m *TaskListModel) MoveUp() {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.items[i].kind == itemTask {
			m.cursor = i
			return
		}
	}
}

// JumpNextSection moves the cursor to the first task in the next section.
func (m *TaskListModel) JumpNextSection() {
	// Find next section header after cursor
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].kind == itemSection {
			// Find first task after this section
			m.moveToNextTask(i + 1)
			return
		}
	}
}

// JumpPrevSection moves the cursor to the first task in the previous section.
func (m *TaskListModel) JumpPrevSection() {
	// Find the section header for the current cursor position
	currentSection := -1
	for i := m.cursor; i >= 0; i-- {
		if m.items[i].kind == itemSection {
			currentSection = i
			break
		}
	}
	if currentSection <= 0 {
		return
	}
	// Find the section before current section
	for i := currentSection - 1; i >= 0; i-- {
		if m.items[i].kind == itemSection {
			m.moveToNextTask(i + 1)
			return
		}
	}
}

// SetSize updates the viewport dimensions.
func (m *TaskListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the task list.
func (m TaskListModel) View() string {
	if m.taskFile == nil || len(m.items) == 0 {
		return ui.HelpBar.Render("No tasks. Press 'a' to add one.")
	}

	var sb strings.Builder

	for i, it := range m.items {
		switch it.kind {
		case itemSection:
			sb.WriteString(ui.SectionHeader.Render(it.section))
		case itemTask:
			line := m.renderTask(it.taskIndex, i == m.cursor)
			sb.WriteString(line)
		case itemBlank:
			// empty line
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// renderTask renders a single task line with styling.
func (m TaskListModel) renderTask(taskIdx int, selected bool) string {
	if taskIdx < 0 || taskIdx >= len(m.taskFile.Tasks) {
		return ""
	}
	t := m.taskFile.Tasks[taskIdx]

	var parts []string

	// Status icon
	switch t.Status {
	case task.StatusTodo:
		parts = append(parts, "\u25cb") // open circle
	case task.StatusInProgress:
		parts = append(parts, "\u25d0") // half circle
	case task.StatusDone:
		parts = append(parts, "\u25cf") // filled circle
	}

	// Title (with priority coloring)
	title := t.Title
	switch t.Priority {
	case task.PriorityHigh:
		title = ui.PriorityHigh.Render(title)
	case task.PriorityMedium:
		title = ui.PriorityMedium.Render(title)
	case task.PriorityLow:
		title = ui.PriorityLow.Render(title)
	}
	parts = append(parts, title)

	// Tags
	for _, tag := range t.Tags {
		parts = append(parts, ui.Tag.Render(fmt.Sprintf("+%s", tag)))
	}

	// Due date
	if t.DueDate != nil {
		parts = append(parts, ui.HelpBar.Render(fmt.Sprintf("due:%s", t.DueDate.Format("2006-01-02"))))
	}

	line := strings.Join(parts, " ")

	// Apply done styling
	if t.Status == task.StatusDone {
		line = ui.DoneTask.Render(line)
	}

	// Apply selection
	if selected {
		line = ui.SelectedTask.Render(line)
	}

	return "  " + line
}

// SelectedTask returns the currently selected task, or nil if none.
func (m TaskListModel) SelectedTaskItem() *task.Task {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	it := m.items[m.cursor]
	if it.kind != itemTask || it.taskIndex < 0 || it.taskIndex >= len(m.taskFile.Tasks) {
		return nil
	}
	return &m.taskFile.Tasks[it.taskIndex]
}
