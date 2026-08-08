package model

import (
	"strings"
	"testing"

	"github.com/macone/todo-cli/internal/storage"
)

// newScrollList builds a list model over content, with a viewport of height
// rows, positioned at the first task.
func newScrollList(t *testing.T, content string, height int) *TaskListModel {
	t.Helper()
	tf, err := storage.NewParser().Parse(content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	m := NewTaskListModel(tf)
	m.SetSize(80, height)
	return &m
}

// scrollFixture has two headers before the first task and enough tasks to
// scroll, so clipping at the top is observable.
func scrollFixture(taskCount int) string {
	var sb strings.Builder
	sb.WriteString("## Alpha\n\n## Beta\n")
	for i := 0; i < taskCount; i++ {
		sb.WriteString("- [ ] task\n")
	}
	sb.WriteString("\n## Omega\n")
	return sb.String()
}

func (m *TaskListModel) toBottom() {
	for range 200 {
		before := m.cursor
		m.MoveDown()
		if m.cursor == before {
			return
		}
	}
}

func (m *TaskListModel) toTop() {
	for range 200 {
		before := m.cursor
		m.MoveUp()
		if m.cursor == before {
			return
		}
	}
}

// Scrolling to the bottom and back must return to the very top of the
// document, headers included -- not park the cursor on the first row with its
// section scrolled off.
func TestScrollBackToTopShowsHeaders(t *testing.T) {
	m := newScrollList(t, scrollFixture(40), 10)

	m.toBottom()
	if m.scrollOffset == 0 {
		t.Fatal("expected the list to have scrolled")
	}

	m.toTop()
	if m.scrollOffset != 0 {
		t.Errorf("expected scrollOffset 0 at the top, got %d", m.scrollOffset)
	}

	lines := m.ViewLines(80, 10)
	if len(lines) == 0 || !strings.Contains(lines[0], "Alpha") {
		t.Errorf("expected the first header on the first line, got %q", lines[0])
	}
}

// Reaching the first task of any section brings that section's header into
// view along with it.
func TestScrollUpRevealsSectionHeader(t *testing.T) {
	content := "## Alpha\n" +
		strings.Repeat("- [ ] alpha task\n", 20) +
		"\n## Beta\n" +
		strings.Repeat("- [ ] beta task\n", 20)

	m := newScrollList(t, content, 8)
	m.toBottom()

	// Walk up to the first task of Beta.
	for range 200 {
		if m.cursor > 0 && m.items[m.cursor-1].kind != itemTask {
			break
		}
		before := m.cursor
		m.MoveUp()
		if m.cursor == before {
			t.Fatal("never reached the first task of a section")
		}
	}

	joined := strings.Join(m.ViewLines(80, 8), "\n")
	if !strings.Contains(joined, "Beta") {
		t.Errorf("expected the Beta header to be visible, got:\n%s", joined)
	}
}

// The cursor stays on screen at every step in both directions.
func TestScrollKeepsCursorVisible(t *testing.T) {
	const height = 9
	m := newScrollList(t, scrollFixture(50), height)

	check := func(step int) {
		t.Helper()
		starts := m.itemLineStarts()
		cur := starts[m.cursor]
		if cur < m.scrollOffset || cur >= m.scrollOffset+height {
			t.Fatalf("step %d: cursor line %d outside viewport [%d,%d)",
				step, cur, m.scrollOffset, m.scrollOffset+height)
		}
	}

	for i := range 60 {
		m.MoveDown()
		check(i)
	}
	for i := range 60 {
		m.MoveUp()
		check(-i)
	}
}

// Scrolling must never run past the end of the content and leave a gap.
func TestScrollDoesNotOverrunTheEnd(t *testing.T) {
	const height = 10
	m := newScrollList(t, scrollFixture(40), height)
	m.toBottom()

	total := len(m.buildAllLines())
	if m.scrollOffset+height > total {
		t.Errorf("scrollOffset %d + height %d exceeds %d rendered lines",
			m.scrollOffset, height, total)
	}
}

// At the end of the list the view shows the document's tail, so a trailing
// header with no tasks under it is still visible.
func TestScrollAtEndShowsTrailingContent(t *testing.T) {
	m := newScrollList(t, scrollFixture(40), 10)
	m.toBottom()

	joined := strings.Join(m.ViewLines(80, 10), "\n")
	if !strings.Contains(joined, "Omega") {
		t.Errorf("expected the trailing Omega header to be visible, got:\n%s", joined)
	}

	total := len(m.buildAllLines())
	if m.scrollOffset+10 != total {
		t.Errorf("expected the last line on screen: offset %d + 10 != %d", m.scrollOffset, total)
	}
}

// A trailing block taller than the viewport cannot be shown in full, and must
// not be shown at the cost of hiding the cursor.
func TestScrollAtEndKeepsCursorVisibleWithLongTail(t *testing.T) {
	const height = 6
	content := "## Alpha\n" + strings.Repeat("- [ ] task\n", 20) +
		strings.Repeat("\n## Empty\n", 8)

	m := newScrollList(t, content, height)
	m.toBottom()

	starts := m.itemLineStarts()
	cur := starts[m.cursor]
	if cur < m.scrollOffset || cur >= m.scrollOffset+height {
		t.Errorf("cursor line %d outside viewport [%d,%d)", cur, m.scrollOffset, m.scrollOffset+height)
	}
}

func TestMoveToTopAndBottom(t *testing.T) {
	m := newScrollList(t, scrollFixture(40), 10)
	first := m.cursor

	m.MoveToBottom()
	if m.isLastTask() != true {
		t.Error("expected the cursor on the last task")
	}
	if m.scrollOffset == 0 {
		t.Error("expected the view to have scrolled to the bottom")
	}

	m.MoveToTop()
	if m.cursor != first {
		t.Errorf("expected the cursor back on the first task %d, got %d", first, m.cursor)
	}
	if m.scrollOffset != 0 {
		t.Errorf("expected scrollOffset 0, got %d", m.scrollOffset)
	}
}

// A page is half a viewport, as in vim, and the two directions are symmetric.
func TestPageDownAndUp(t *testing.T) {
	const height = 10
	m := newScrollList(t, scrollFixture(60), height)

	starts := m.itemLineStarts()
	start := starts[m.cursor]

	m.PageDown()
	starts = m.itemLineStarts()
	moved := starts[m.cursor] - start
	if moved <= 0 || moved > height/2 {
		t.Errorf("expected a move of 1..%d lines, got %d", height/2, moved)
	}

	m.PageUp()
	starts = m.itemLineStarts()
	if got := starts[m.cursor]; got != start {
		t.Errorf("expected to page back to line %d, got %d", start, got)
	}
}

func TestPagingStopsAtTheEnds(t *testing.T) {
	m := newScrollList(t, scrollFixture(30), 8)

	for range 50 {
		m.PageDown()
	}
	if !m.isLastTask() {
		t.Error("expected paging down to settle on the last task")
	}

	for range 50 {
		m.PageUp()
	}
	if m.scrollOffset != 0 {
		t.Errorf("expected paging up to reach the top, got offset %d", m.scrollOffset)
	}
}

// A list shorter than the viewport never scrolls.
func TestScrollStaysAtZeroForShortLists(t *testing.T) {
	m := newScrollList(t, scrollFixture(2), 40)
	m.toBottom()
	if m.scrollOffset != 0 {
		t.Errorf("expected no scrolling, got offset %d", m.scrollOffset)
	}
}
