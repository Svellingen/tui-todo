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

// statusFilter controls which tasks are shown.
type statusFilter int

const (
	filterAll    statusFilter = iota // show everything
	filterActive                     // Todo + InProgress
	filterDone                       // Done only
)

// TaskListModel handles rendering and navigating the task list.
type TaskListModel struct {
	taskFile     *storage.TaskFile
	items        []item       // flattened list of renderable rows
	cursor       int          // index into items (only stops on task rows)
	width        int
	height       int
	statusFilter statusFilter
	searchQuery  string // substring filter on title
	tagFilter    string // filter by tag (empty = no filter)
}

// NewTaskListModel creates a TaskListModel from a TaskFile.
func NewTaskListModel(tf *storage.TaskFile) TaskListModel {
	m := TaskListModel{taskFile: tf}
	m.rebuildItems()
	return m
}

// rebuildItems creates the flat item list from the TaskFile's lines,
// applying any active filters.
func (m *TaskListModel) rebuildItems() {
	m.items = nil
	for _, line := range m.taskFile.Lines {
		switch line.Type {
		case storage.LineSection:
			m.items = append(m.items, item{kind: itemSection, section: line.Raw})
		case storage.LineTask:
			if m.taskVisible(line.TaskIndex) {
				m.items = append(m.items, item{kind: itemTask, taskIndex: line.TaskIndex})
			}
		case storage.LineText:
			m.items = append(m.items, item{kind: itemBlank})
		}
	}
	// Reposition cursor on first visible task
	m.cursor = 0
	m.moveToNextTask(0)
}

// taskVisible returns whether a task passes all active filters.
func (m *TaskListModel) taskVisible(idx int) bool {
	if idx < 0 || idx >= len(m.taskFile.Tasks) {
		return false
	}
	t := m.taskFile.Tasks[idx]

	// Status filter
	switch m.statusFilter {
	case filterActive:
		if t.Status == task.StatusDone {
			return false
		}
	case filterDone:
		if t.Status != task.StatusDone {
			return false
		}
	}

	// Search query filter
	if m.searchQuery != "" {
		if !strings.Contains(strings.ToLower(t.Title), strings.ToLower(m.searchQuery)) {
			return false
		}
	}

	// Tag filter
	if m.tagFilter != "" {
		found := false
		for _, tag := range t.Tags {
			if tag == m.tagFilter {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// SetStatusFilter sets the status filter and rebuilds the item list.
func (m *TaskListModel) SetStatusFilter(f statusFilter) {
	m.statusFilter = f
	m.rebuildItems()
}

// StatusFilter returns the current status filter.
func (m TaskListModel) StatusFilter() statusFilter {
	return m.statusFilter
}

// SetSearchQuery sets the search query and rebuilds the item list.
func (m *TaskListModel) SetSearchQuery(q string) {
	m.searchQuery = q
	m.rebuildItems()
}

// ClearSearch clears the search query and rebuilds.
func (m *TaskListModel) ClearSearch() {
	m.searchQuery = ""
	m.rebuildItems()
}

// SearchQuery returns the current search query.
func (m TaskListModel) SearchQuery() string {
	return m.searchQuery
}

// SetTagFilter sets the tag filter and rebuilds the item list.
func (m *TaskListModel) SetTagFilter(tag string) {
	m.tagFilter = tag
	m.rebuildItems()
}

// ClearTagFilter clears the tag filter and rebuilds.
func (m *TaskListModel) ClearTagFilter() {
	m.tagFilter = ""
	m.rebuildItems()
}

// TagFilter returns the current tag filter.
func (m TaskListModel) TagFilter() string {
	return m.tagFilter
}

// AllTags returns a deduplicated sorted list of all tags across all tasks.
func (m TaskListModel) AllTags() []string {
	seen := make(map[string]bool)
	var tags []string
	for _, t := range m.taskFile.Tasks {
		for _, tag := range t.Tags {
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags
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
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].kind == itemSection {
			m.moveToNextTask(i + 1)
			return
		}
	}
}

// JumpPrevSection moves the cursor to the first task in the previous section.
func (m *TaskListModel) JumpPrevSection() {
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

// FilterIndicator returns a string showing active filters, or empty if none.
func (m TaskListModel) FilterIndicator() string {
	var parts []string
	switch m.statusFilter {
	case filterActive:
		parts = append(parts, "active")
	case filterDone:
		parts = append(parts, "done")
	}
	if m.searchQuery != "" {
		parts = append(parts, fmt.Sprintf("search:%q", m.searchQuery))
	}
	if m.tagFilter != "" {
		parts = append(parts, fmt.Sprintf("tag:+%s", m.tagFilter))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// ProgressString returns a "done/total done" progress string.
func (m TaskListModel) ProgressString() string {
	if m.taskFile == nil {
		return ""
	}
	total := len(m.taskFile.Tasks)
	if total == 0 {
		return ""
	}
	done := 0
	for _, t := range m.taskFile.Tasks {
		if t.Status == task.StatusDone {
			done++
		}
	}
	return fmt.Sprintf("%d/%d done", done, total)
}

// View renders the task list.
func (m TaskListModel) View() string {
	if m.taskFile == nil || len(m.taskFile.Tasks) == 0 {
		return "\n\n" + ui.HelpBar.Render("    No tasks yet. Press 'a' to add one.") + "\n\n"
	}

	// Check if all tasks are filtered out
	hasVisibleTasks := false
	for _, it := range m.items {
		if it.kind == itemTask {
			hasVisibleTasks = true
			break
		}
	}
	if !hasVisibleTasks {
		return "\n" + ui.HelpBar.Render("    No matching tasks.") + "\n"
	}

	var sb strings.Builder

	// Header line: filter indicator + progress
	header := ""
	if indicator := m.FilterIndicator(); indicator != "" {
		header = ui.Tag.Render(indicator)
	}
	if progress := m.ProgressString(); progress != "" {
		if header != "" {
			header += "  "
		}
		header += ui.Progress.Render(progress)
	}
	if header != "" {
		sb.WriteString(header + "\n")
	}

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

// SelectedTaskItem returns the currently selected task, or nil if none.
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

// SelectedTaskIndex returns the index into TaskFile.Tasks for the selected item,
// or -1 if no task is selected.
func (m TaskListModel) SelectedTaskIndex() int {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return -1
	}
	it := m.items[m.cursor]
	if it.kind != itemTask {
		return -1
	}
	return it.taskIndex
}
