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
	kind    itemType
	section string
	// lineIndex is the item's position in TaskFile.Lines. Headings are
	// addressed by it, since two headings can share the same text.
	lineIndex int
	taskIndex int
}

type statusFilter int

const (
	filterAll statusFilter = iota
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
	for li, line := range m.taskFile.Lines {
		switch line.Type {
		case storage.LineSection:
			m.items = append(m.items, item{kind: itemSection, section: line.Raw, lineIndex: li})
		case storage.LineTask:
			if m.taskVisible(line.TaskIndex) {
				m.items = append(m.items, item{kind: itemTask, lineIndex: li, taskIndex: line.TaskIndex})
			}
		case storage.LineText:
			m.items = append(m.items, item{kind: itemBlank, lineIndex: li})
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

// JumpNextSection selects the next heading. Headings are cursor stops of their
// own, so this lands on the heading rather than the first task beneath it.
func (m *TaskListModel) JumpNextSection() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].kind == itemSection {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// JumpPrevSection selects the nearest heading above the cursor. From a task
// that is the task's own heading; from a heading it is the one before it.
func (m *TaskListModel) JumpPrevSection() {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.items[i].kind == itemSection {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// sectionLevel returns the heading level of an item, or 0 when it is not a
// heading.
func (m TaskListModel) sectionLevel(i int) int {
	if i < 0 || i >= len(m.items) || m.items[i].kind != itemSection {
		return 0
	}
	level, _ := storage.ParseHeading(strings.TrimSpace(m.items[i].section))
	return level
}

// JumpNextSectionOfLevel selects the next heading at exactly the given level,
// stepping over deeper sub-headings.
func (m *TaskListModel) JumpNextSectionOfLevel(level int) {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.sectionLevel(i) == level {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// JumpPrevSectionOfLevel selects the nearest heading above the cursor at
// exactly the given level.
func (m *TaskListModel) JumpPrevSectionOfLevel(level int) {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.sectionLevel(i) == level {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// JumpParentSection moves one step up the heading hierarchy.
//
// From a task that is the heading enclosing it; from a heading it is that
// heading's parent -- the nearest shallower one above it. floor is the level
// the walk stops at, so on a heading at that level nothing happens.
func (m *TaskListModel) JumpParentSection(floor int) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}

	level := m.sectionLevel(m.cursor)
	if level == 0 {
		// On a task: step out to whatever heading it lives under.
		for i := m.cursor - 1; i >= 0; i-- {
			if m.sectionLevel(i) > 0 {
				m.cursor = i
				m.adjustScroll()
				return
			}
		}
		return
	}

	if level <= floor {
		return
	}
	for i := m.cursor - 1; i >= 0; i-- {
		if l := m.sectionLevel(i); l >= floor && l < level {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// JumpChildSection selects the first sub-heading nested under the heading the
// cursor is on. It does nothing on a task, or on a heading that has no
// sub-headings of its own.
func (m *TaskListModel) JumpChildSection() {
	level := m.sectionLevel(m.cursor)
	if level == 0 {
		return
	}

	for i := m.cursor + 1; i < len(m.items); i++ {
		l := m.sectionLevel(i)
		if l == 0 {
			continue
		}
		if l <= level {
			return // left this heading's subtree without finding a child
		}
		m.cursor = i
		m.adjustScroll()
		return
	}
}

// SelectedLineIndex returns the Lines index of whatever the cursor is on,
// heading or task, or -1 when nothing is selected.
func (m TaskListModel) SelectedLineIndex() int {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return -1
	}
	if m.items[m.cursor].kind == itemBlank {
		return -1
	}
	return m.items[m.cursor].lineIndex
}

// SelectedSectionLine returns the Lines index of the heading under the cursor,
// or -1 when the cursor is on a task.
func (m TaskListModel) SelectedSectionLine() int {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return -1
	}
	if m.items[m.cursor].kind != itemSection {
		return -1
	}
	return m.items[m.cursor].lineIndex
}

// SelectTask rebuilds the list and puts the cursor on the given task.
func (m *TaskListModel) SelectTask(taskIndex int) {
	m.rebuildItems()
	for i, it := range m.items {
		if it.kind == itemTask && it.taskIndex == taskIndex {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// SelectPrecedingItem rebuilds the list and puts the cursor on the nearest
// selectable item at or before pos.
//
// After a deletion, callers pass the index just above what was removed, so the
// cursor settles on the previous sibling -- or on the enclosing heading when
// the deleted item was the first thing under it. Only when nothing precedes it
// does the cursor fall back to the top of the list.
func (m *TaskListModel) SelectPrecedingItem(pos int) {
	m.rebuildItems()
	if len(m.items) == 0 {
		return
	}
	if pos >= len(m.items) {
		pos = len(m.items) - 1
	}

	for i := pos; i >= 0; i-- {
		if m.items[i].kind != itemBlank {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
	for i := range m.items {
		if m.items[i].kind != itemBlank {
			m.cursor = i
			m.adjustScroll()
			return
		}
	}
}

// Cursor returns the index of the selected item.
func (m TaskListModel) Cursor() int { return m.cursor }

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
	// A heading is its own anchor; walking further back would drag in the
	// heading above it.
	if m.items[m.cursor].kind == itemSection {
		return starts[m.cursor]
	}
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

// renderHeading draws a markdown heading's text, coloured by level. The "#"
// markers are dropped -- the colour carries the level. The cursor sits in the
// same column as it does on task rows.
func renderHeading(raw string, selected bool) string {
	cursor := "   "
	if selected {
		cursor = ui.CursorStyle.Render(" ▸ ")
	}

	trimmed := strings.TrimSpace(raw)
	level, name := storage.ParseHeading(trimmed)
	if level == 0 {
		return cursor + ui.HeadingStyle(1).Render(trimmed)
	}
	return cursor + ui.HeadingStyle(level).Render(name)
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
			lines = append(lines, renderHeading(it.section, i == m.cursor))

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
