package model

import (
	"strings"
	"testing"
)

const headingFixture = "# Top\n" +
	"- [ ] top task\n" +
	"## Alpha\n" +
	"- [ ] alpha one\n" +
	"- [ ] alpha two\n" +
	"### Alpha sub\n" +
	"- [ ] alpha sub task\n" +
	"## Beta\n" +
	"- [ ] beta task\n"

// kindsAt names what the cursor is sitting on, for readable assertions.
func (m TaskListModel) cursorKind() string {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return "none"
	}
	switch m.items[m.cursor].kind {
	case itemSection:
		_, name := parseSectionName(m.items[m.cursor].section)
		return "heading:" + name
	case itemTask:
		return "task:" + m.taskFile.Tasks[m.items[m.cursor].taskIndex].Title
	default:
		return "blank"
	}
}

func parseSectionName(raw string) (int, string) {
	trimmed := strings.TrimSpace(raw)
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	return level, strings.TrimSpace(trimmed[level:])
}

// tab walks headings themselves, not the first task beneath them.
func TestJumpSectionLandsOnHeadings(t *testing.T) {
	m := newScrollList(t, headingFixture, 30)

	want := []string{"heading:Alpha", "heading:Alpha sub", "heading:Beta"}
	for _, w := range want {
		m.JumpNextSection()
		if got := m.cursorKind(); got != w {
			t.Fatalf("expected %s, got %s", w, got)
		}
	}

	// Past the last heading it stays put.
	m.JumpNextSection()
	if got := m.cursorKind(); got != "heading:Beta" {
		t.Errorf("expected to stay on Beta, got %s", got)
	}

	back := []string{"heading:Alpha sub", "heading:Alpha", "heading:Top"}
	for _, w := range back {
		m.JumpPrevSection()
		if got := m.cursorKind(); got != w {
			t.Fatalf("expected %s, got %s", w, got)
		}
	}
}

// From a task, the previous heading is that task's own heading.
func TestJumpPrevSectionFromTaskSelectsItsHeading(t *testing.T) {
	m := newScrollList(t, headingFixture, 30)
	m.MoveDown() // top task -> alpha one
	m.MoveDown() // -> alpha two

	m.JumpPrevSection()
	if got := m.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected heading:Alpha, got %s", got)
	}
}

// j and k stay on tasks, stepping over the headings between them.
func TestMoveSkipsHeadings(t *testing.T) {
	m := newScrollList(t, headingFixture, 30)

	want := []string{"task:alpha one", "task:alpha two", "task:alpha sub task", "task:beta task"}
	for _, w := range want {
		m.MoveDown()
		if got := m.cursorKind(); got != w {
			t.Fatalf("expected %s, got %s", w, got)
		}
	}
}

// Moving down off a heading lands on the next task, not the next heading.
func TestMoveDownFromHeading(t *testing.T) {
	m := newScrollList(t, headingFixture, 30)
	m.JumpNextSection() // Alpha
	m.MoveDown()
	if got := m.cursorKind(); got != "task:alpha one" {
		t.Errorf("expected task:alpha one, got %s", got)
	}
}

func TestSelectedSectionLine(t *testing.T) {
	m := newScrollList(t, headingFixture, 30)
	if got := m.SelectedSectionLine(); got != -1 {
		t.Errorf("on a task, expected -1, got %d", got)
	}
	if got := m.SelectedTaskIndex(); got < 0 {
		t.Error("on a task, expected a valid task index")
	}

	m.JumpNextSection()
	line := m.SelectedSectionLine()
	if line < 0 {
		t.Fatal("on a heading, expected a line index")
	}
	if m.taskFile.Lines[line].Raw != "## Alpha" {
		t.Errorf("expected the Alpha heading line, got %q", m.taskFile.Lines[line].Raw)
	}
	if got := m.SelectedTaskIndex(); got != -1 {
		t.Errorf("on a heading, expected task index -1, got %d", got)
	}
}

// The cursor is drawn on a selected heading, in the same column as on tasks.
func TestHeadingShowsCursor(t *testing.T) {
	m := newScrollList(t, headingFixture, 30)
	m.JumpNextSection()

	// A heading's block starts at the blank line separating it from the
	// section above, so the heading itself is the first non-blank line of it.
	lines, starts := m.buildLines()
	var row string
	for i := starts[m.cursor]; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			row = lines[i]
			break
		}
	}

	if !strings.Contains(row, "▸") {
		t.Errorf("expected a cursor on the selected heading, got %q", row)
	}
	if !strings.Contains(row, "Alpha") {
		t.Errorf("expected the heading text, got %q", row)
	}

	// Exactly one row carries the cursor.
	count := 0
	for _, l := range lines {
		if strings.Contains(l, "▸") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one cursor row, got %d", count)
	}
}
