package model

import (
	"fmt"
	"strings"

	"github.com/macone/todo-cli/internal/storage"
	"github.com/macone/todo-cli/internal/task"
	"github.com/macone/todo-cli/internal/ui"
)

type itemType int

const (
	itemSection itemType = iota
	itemTask
	itemBlank
)

type item struct {
	kind      itemType
	section   string
	taskIndex int
}

type statusFilter int

const (
	filterAll    statusFilter = iota
	filterActive
	filterDone
)

// TaskListModel handles rendering and navigating the task list.
type TaskListModel struct {
	taskFile     *storage.TaskFile
	items        []item
	cursor       int
	width        int
	height       int
	scrollOffset int
	statusFilter statusFilter
	searchQuery  string
	tagFilter    string
}

func NewTaskListModel(tf *storage.TaskFile) TaskListModel {
	m := TaskListModel{taskFile: tf}
	m.rebuildItems()
	return m
}

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
	m.cursor = 0
	m.moveToNextTask(0)
	m.scrollOffset = 0
}

// Reload swaps in freshly parsed content, keeping the active filters and
// leaving the cursor on the same task where that task still exists.
func (m *TaskListModel) Reload(tf *storage.TaskFile) {
	var selected string
	if t := m.SelectedTaskItem(); t != nil {
		selected = t.Title
	}

	m.taskFile = tf
	m.rebuildItems()

	if selected == "" {
		return
	}
	for i, it := range m.items {
		if it.kind != itemTask || it.taskIndex >= len(tf.Tasks) {
			continue
		}
		if tf.Tasks[it.taskIndex].Title == selected {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

func (m *TaskListModel) taskVisible(idx int) bool {
	if idx < 0 || idx >= len(m.taskFile.Tasks) {
		return false
	}
	t := m.taskFile.Tasks[idx]

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

	if m.searchQuery != "" {
		if !strings.Contains(strings.ToLower(t.Title), strings.ToLower(m.searchQuery)) {
			return false
		}
	}

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

func (m *TaskListModel) SetStatusFilter(f statusFilter) {
	m.statusFilter = f
	m.rebuildItems()
}

func (m TaskListModel) StatusFilter() statusFilter { return m.statusFilter }

func (m *TaskListModel) SetSearchQuery(q string) {
	m.searchQuery = q
	m.rebuildItems()
}

func (m *TaskListModel) ClearSearch() {
	m.searchQuery = ""
	m.rebuildItems()
}

func (m TaskListModel) SearchQuery() string { return m.searchQuery }

func (m *TaskListModel) SetTagFilter(tag string) {
	m.tagFilter = tag
	m.rebuildItems()
}

func (m *TaskListModel) ClearTagFilter() {
	m.tagFilter = ""
	m.rebuildItems()
}

func (m TaskListModel) TagFilter() string { return m.tagFilter }

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

func (m *TaskListModel) moveToNextTask(pos int) {
	for i := pos; i < len(m.items); i++ {
		if m.items[i].kind == itemTask {
			m.cursor = i
			return
		}
	}
}

func (m *TaskListModel) moveToPrevTask(pos int) {
	for i := pos; i >= 0; i-- {
		if m.items[i].kind == itemTask {
			m.cursor = i
			return
		}
	}
}

func (m *TaskListModel) MoveDown() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].kind == itemTask {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

func (m *TaskListModel) MoveUp() {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.items[i].kind == itemTask {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

func (m *TaskListModel) JumpNextSection() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].kind == itemSection {
			m.moveToNextTask(i + 1)
			m.adjustScroll()
			return
		}
	}
}

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
			m.adjustScroll()
			return
		}
	}
}

func (m *TaskListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// cursorLineIndex returns the line index of the cursor in the rendered output.
func (m TaskListModel) cursorLineIndex() int {
	idx := 0
	isFirst := true
	for i := 0; i < len(m.items); i++ {
		if i == m.cursor {
			return idx
		}
		switch m.items[i].kind {
		case itemSection:
			if !isFirst {
				idx++ // blank line between sections
			}
			isFirst = false
			idx += 2 // header + separator
		case itemTask:
			idx++
		}
	}
	return idx
}

// adjustScroll keeps the cursor visible within the viewport.
func (m *TaskListModel) adjustScroll() {
	if m.height <= 0 {
		return
	}
	curLine := m.cursorLineIndex()
	if curLine < m.scrollOffset {
		m.scrollOffset = curLine
	}
	if curLine >= m.scrollOffset+m.height {
		m.scrollOffset = curLine - m.height + 1
	}
}

// FilterIndicator returns a string showing active filters.
func (m TaskListModel) FilterIndicator() string {
	var parts []string
	switch m.statusFilter {
	case filterActive:
		parts = append(parts, "active")
	case filterDone:
		parts = append(parts, "done")
	}
	if m.searchQuery != "" {
		parts = append(parts, fmt.Sprintf("/%s", m.searchQuery))
	}
	if m.tagFilter != "" {
		parts = append(parts, fmt.Sprintf("+%s", m.tagFilter))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// ProgressString returns "3/8 done".
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

// ProgressCounts returns the done and total task counts.
func (m TaskListModel) ProgressCounts() (done, total int) {
	if m.taskFile == nil {
		return
	}
	total = len(m.taskFile.Tasks)
	for _, t := range m.taskFile.Tasks {
		if t.Status == task.StatusDone {
			done++
		}
	}
	return
}

// buildAllLines builds all rendered lines for the task list.
func (m TaskListModel) buildAllLines(innerWidth int) []string {
	if m.taskFile == nil || len(m.taskFile.Tasks) == 0 {
		return []string{
			"",
			"  " + ui.EmptyState.Render("No tasks yet. Press 'a' to add one."),
			"",
		}
	}

	hasVisible := false
	for _, it := range m.items {
		if it.kind == itemTask {
			hasVisible = true
			break
		}
	}
	if !hasVisible {
		return []string{
			"",
			"  " + ui.EmptyState.Render("No matching tasks."),
			"",
		}
	}

	var lines []string
	isFirstSection := true

	for i, it := range m.items {
		switch it.kind {
		case itemSection:
			if !isFirstSection {
				lines = append(lines, "")
			}
			isFirstSection = false

			name := extractSectionName(it.section)
			lines = append(lines, "  "+ui.SectionHeader.Render(strings.ToUpper(name)))

			sepWidth := innerWidth - 4
			if sepWidth < 10 {
				sepWidth = 10
			}
			lines = append(lines, "  "+ui.SectionSep.Render(strings.Repeat("─", sepWidth)))

		case itemTask:
			lines = append(lines, m.renderTask(it.taskIndex, i == m.cursor))

		case itemBlank:
			// skip — we manage spacing ourselves
		}
	}

	return lines
}

// ViewLines returns rendered lines, clipped to maxLines with scrolling.
func (m TaskListModel) ViewLines(innerWidth, maxLines int) []string {
	allLines := m.buildAllLines(innerWidth)

	if maxLines <= 0 || len(allLines) <= maxLines {
		return allLines
	}

	start := m.scrollOffset
	end := start + maxLines
	if start < 0 {
		start = 0
	}
	if end > len(allLines) {
		end = len(allLines)
	}
	if start >= len(allLines) {
		start = 0
		end = maxLines
		if end > len(allLines) {
			end = len(allLines)
		}
	}

	return allLines[start:end]
}

// View renders the task list as a single string.
func (m TaskListModel) View() string {
	lines := m.ViewLines(m.width, m.height)
	return strings.Join(lines, "\n")
}

func (m TaskListModel) renderTask(taskIdx int, selected bool) string {
	if taskIdx < 0 || taskIdx >= len(m.taskFile.Tasks) {
		return ""
	}
	t := m.taskFile.Tasks[taskIdx]

	// Cursor arrow
	cursor := "   "
	if selected {
		cursor = ui.CursorStyle.Render(" ▸ ")
	}

	// Status icon
	var icon string
	switch t.Status {
	case task.StatusTodo:
		icon = ui.StatusTodo.Render("○")
	case task.StatusInProgress:
		icon = ui.StatusActive.Render("◐")
	case task.StatusDone:
		icon = ui.StatusDone.Render("●")
	}

	// Title
	title := t.Title

	// Priority indicator
	var prio string
	switch t.Priority {
	case task.PriorityHigh:
		prio = ui.PriorityHigh.Render("!!")
	case task.PriorityMedium:
		prio = ui.PriorityMedium.Render("!")
	}

	// Tags
	var tagParts []string
	for _, tag := range t.Tags {
		tagParts = append(tagParts, ui.Tag.Render("+"+tag))
	}
	tagStr := strings.Join(tagParts, " ")

	// Due date
	var due string
	if t.DueDate != nil {
		due = ui.DueStyle.Render("due:" + t.DueDate.Format("2006-01-02"))
	}

	// Build metadata
	var meta []string
	if prio != "" {
		meta = append(meta, prio)
	}
	if tagStr != "" {
		meta = append(meta, tagStr)
	}
	if due != "" {
		meta = append(meta, due)
	}
	metaStr := strings.Join(meta, " ")

	// Apply styling based on state
	if t.Status == task.StatusDone {
		title = ui.DoneTask.Render(title)
		if metaStr != "" {
			metaStr = ui.DoneMeta.Render(strings.Join(meta, " "))
		}
	} else if selected {
		title = ui.SelectedTask.Render(title)
	}

	// Assemble
	line := cursor + icon + "  " + title
	if metaStr != "" {
		line += "  " + metaStr
	}

	return line
}

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

func extractSectionName(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "## ")
	s = strings.TrimPrefix(s, "# ")
	return strings.TrimSpace(s)
}
