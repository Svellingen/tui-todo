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

// RefreshOrder rebuilds the item list after the underlying task order changed,
// leaving the cursor on the task it was already on.
func (m *TaskListModel) RefreshOrder() {
	selected := m.SelectedTaskIndex()
	m.rebuildItems()
	if selected < 0 {
		return
	}
	for i, it := range m.items {
		if it.kind == itemTask && it.taskIndex == selected {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
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

// MoveToTop puts the cursor on the first task.
func (m *TaskListModel) MoveToTop() {
	for i := range m.items {
		if m.items[i].kind == itemTask {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// MoveToBottom puts the cursor on the last task.
func (m *TaskListModel) MoveToBottom() {
	for i := len(m.items) - 1; i >= 0; i-- {
		if m.items[i].kind == itemTask {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// pageStep is how far ctrl+d and ctrl+u travel: half a viewport, as in vim.
func (m TaskListModel) pageStep() int {
	if m.height < 2 {
		return 1
	}
	return m.height / 2
}

// PageDown moves the cursor to the furthest task within half a viewport below.
func (m *TaskListModel) PageDown() {
	starts := m.itemLineStarts()
	if m.cursor < 0 || m.cursor >= len(starts) {
		return
	}
	target := starts[m.cursor] + m.pageStep()

	next := m.cursor
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].kind != itemTask {
			continue
		}
		if starts[i] > target {
			break
		}
		next = i
	}

	if next == m.cursor {
		// The next task is more than a page away; still advance by one so the
		// key never feels dead.
		m.MoveDown()
		return
	}
	m.cursor = next
	m.adjustScroll()
}

// PageUp moves the cursor to the furthest task within half a viewport above.
func (m *TaskListModel) PageUp() {
	starts := m.itemLineStarts()
	if m.cursor < 0 || m.cursor >= len(starts) {
		return
	}
	target := starts[m.cursor] - m.pageStep()

	prev := m.cursor
	for i := m.cursor - 1; i >= 0; i-- {
		if m.items[i].kind != itemTask {
			continue
		}
		if starts[i] < target {
			break
		}
		prev = i
	}

	if prev == m.cursor {
		m.MoveUp()
		return
	}
	m.cursor = prev
	m.adjustScroll()
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

// itemLineStarts returns, for each item, the rendered line its block begins
// on. A section's block starts at the blank line separating it from the
// previous section, so scrolling to it brings the header into view.
func (m TaskListModel) itemLineStarts() []int {
	_, starts := m.buildLines()
	return starts
}

// cursorAnchorLine returns the first line that must stay visible for the
// cursor to make sense: the section header block introducing it, when the
// cursor is the first task under one, otherwise the cursor's own line.
func (m TaskListModel) cursorAnchorLine(starts []int) int {
	anchor := m.cursor
	for anchor > 0 && m.items[anchor-1].kind != itemTask {
		anchor--
	}
	return starts[anchor]
}

// adjustScroll keeps the cursor visible within the viewport.
func (m *TaskListModel) adjustScroll() {
	if m.height <= 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}

	lines, starts := m.buildLines()
	curLine := starts[m.cursor]
	total := len(lines)

	// Scrolling up: reveal the header block above the cursor rather than
	// parking the cursor on the top row with its section cut off.
	if curLine < m.scrollOffset {
		m.scrollOffset = m.cursorAnchorLine(starts)
	}
	// Scrolling down: pull the cursor onto the last row.
	if curLine >= m.scrollOffset+m.height {
		m.scrollOffset = curLine - m.height + 1
	}

	// On the last task there is nothing below to move onto, so show as much of
	// the document's tail as fits. Without this a trailing header or an empty
	// final section could never be seen.
	if m.isLastTask() {
		target := total - m.height
		if target > curLine {
			target = curLine // never push the cursor off the top
		}
		if target > m.scrollOffset {
			m.scrollOffset = target
		}
	}

	// Never scroll past either end of the content.
	if maxOffset := total - m.height; m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// isLastTask reports whether the cursor is on the final task in the list.
func (m TaskListModel) isLastTask() bool {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].kind == itemTask {
			return false
		}
	}
	return true
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

// renderHeading draws a markdown heading the way the file writes it, coloured
// by level.
func renderHeading(raw string) string {
	trimmed := strings.TrimSpace(raw)
	level, name := storage.ParseHeading(trimmed)
	if level == 0 {
		return ui.HeadingStyle(1).Render(trimmed)
	}
	return ui.HeadingStyle(level).Render(strings.Repeat("#", level) + " " + name)
}

// buildLines renders the list and records, for each item, the line its block
// starts on.
//
// Both come from a single pass on purpose: the scroll maths depends on knowing
// exactly how many lines each item draws, and computing that separately is how
// the two drift apart.
func (m TaskListModel) buildLines() (lines []string, starts []int) {
	starts = make([]int, len(m.items))

	if m.taskFile == nil || len(m.taskFile.Tasks) == 0 {
		return []string{
			"",
			"  " + ui.EmptyState.Render("No tasks yet. Press 'a' to add one."),
			"",
		}, starts
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
		}, starts
	}

	isFirstSection := true

	for i, it := range m.items {
		starts[i] = len(lines)

		switch it.kind {
		case itemSection:
			if !isFirstSection {
				lines = append(lines, "")
			}
			isFirstSection = false
			lines = append(lines, "  "+renderHeading(it.section))

		case itemTask:
			lines = append(lines, m.renderTask(it.taskIndex, i == m.cursor))

		case itemBlank:
			// skip — we manage spacing ourselves
		}
	}

	return lines, starts
}

// buildAllLines builds all rendered lines for the task list.
func (m TaskListModel) buildAllLines() []string {
	lines, _ := m.buildLines()
	return lines
}

// ViewLines returns rendered lines, clipped to maxLines with scrolling.
func (m TaskListModel) ViewLines(innerWidth, maxLines int) []string {
	allLines := m.buildAllLines()

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

	// Priority indicator, prefixed to the title exactly as it appears in the
	// file: no padding, so the list reads like the markdown it came from.
	marker := storage.PriorityMarker(t.Priority)
	prio := marker
	switch t.Priority {
	case task.PriorityHigh:
		prio = ui.PriorityHigh.Render(marker)
	case task.PriorityMedium:
		prio = ui.PriorityMedium.Render(marker)
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
		// Fade the marker along with the rest of a completed task.
		if t.Priority != task.PriorityNone {
			prio = ui.DoneMeta.Render(marker)
		}
		if metaStr != "" {
			metaStr = ui.DoneMeta.Render(metaStr)
		}
	} else if selected {
		title = ui.SelectedTask.Render(title)
	}

	// Assemble
	line := cursor + icon + "  "
	if prio != "" {
		line += prio + " "
	}
	line += title
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

