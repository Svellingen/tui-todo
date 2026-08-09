package model

import (
	"strings"
	"testing"
)

// cursorGlyph is what marks the selected row in the gutter.
const cursorGlyph = "-"

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
	case itemBlockLine:
		return "block"
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

const levelFixture = "# Title\n" +
	"- [ ] title task\n" +
	"## Alpha\n" +
	"- [ ] alpha task\n" +
	"### Alpha sub\n" +
	"- [ ] sub task\n" +
	"#### Alpha deep\n" +
	"- [ ] deep task\n" +
	"## Beta\n" +
	"- [ ] beta task\n" +
	"### Beta sub\n" +
	"- [ ] beta sub task\n" +
	"## Gamma\n" +
	"- [ ] gamma task\n"

// ctrl+j / ctrl+k visit "##" headings and step over everything deeper.
func TestJumpSectionOfLevelSkipsOtherLevels(t *testing.T) {
	m := newScrollList(t, levelFixture, 40)

	for _, w := range []string{"heading:Alpha", "heading:Beta", "heading:Gamma"} {
		m.JumpNextSectionOfLevel(2)
		if got := m.cursorKind(); got != w {
			t.Fatalf("expected %s, got %s", w, got)
		}
	}
	m.JumpNextSectionOfLevel(2)
	if got := m.cursorKind(); got != "heading:Gamma" {
		t.Errorf("expected to stay on Gamma, got %s", got)
	}

	for _, w := range []string{"heading:Beta", "heading:Alpha"} {
		m.JumpPrevSectionOfLevel(2)
		if got := m.cursorKind(); got != w {
			t.Fatalf("expected %s, got %s", w, got)
		}
	}
}

// The "#" title is a different level, so it is not a stop.
func TestJumpSectionOfLevelIgnoresTheTitle(t *testing.T) {
	m := newScrollList(t, levelFixture, 40)
	m.JumpNextSectionOfLevel(2) // Alpha

	m.JumpPrevSectionOfLevel(2)
	if got := m.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected to stay on Alpha rather than reach the title, got %s", got)
	}
}

// From a task nested under sub-headings, the jump still resolves to the
// enclosing "##" heading.
func TestJumpSectionOfLevelFromNestedTask(t *testing.T) {
	m := newScrollList(t, levelFixture, 40)
	for m.cursorKind() != "task:deep task" {
		before := m.cursor
		m.MoveDown()
		if m.cursor == before {
			t.Fatal("never reached the deep task")
		}
	}

	m.JumpPrevSectionOfLevel(2)
	if got := m.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected heading:Alpha, got %s", got)
	}

	m.JumpNextSectionOfLevel(2)
	if got := m.cursorKind(); got != "heading:Beta" {
		t.Errorf("expected heading:Beta, got %s", got)
	}
}

// ctrl+l descends one heading level at a time and stops at the deepest.
func TestJumpChildSectionDescends(t *testing.T) {
	m := newScrollList(t, levelFixture, 40)
	m.JumpNextSectionOfLevel(2) // Alpha

	for _, w := range []string{"heading:Alpha sub", "heading:Alpha deep"} {
		m.JumpChildSection()
		if got := m.cursorKind(); got != w {
			t.Fatalf("expected %s, got %s", w, got)
		}
	}

	m.JumpChildSection()
	if got := m.cursorKind(); got != "heading:Alpha deep" {
		t.Errorf("expected to stay on the deepest heading, got %s", got)
	}
}

// A heading whose subtree holds no sub-headings has nowhere to descend to.
func TestJumpChildSectionWithoutChildren(t *testing.T) {
	m := newScrollList(t, levelFixture, 40)
	for m.cursorKind() != "heading:Gamma" {
		m.JumpNextSectionOfLevel(2)
	}

	m.JumpChildSection()
	if got := m.cursorKind(); got != "heading:Gamma" {
		t.Errorf("expected Gamma to have no child, got %s", got)
	}
}

// Descending must not escape into the next section's sub-headings.
func TestJumpChildSectionStaysInSubtree(t *testing.T) {
	// Alpha has no sub-heading of its own here, but Beta does.
	content := "## Alpha\n- [ ] alpha task\n## Beta\n### Beta sub\n- [ ] beta sub task\n"
	m := newScrollList(t, content, 40)
	for m.cursorKind() != "heading:Alpha" {
		m.JumpPrevSection()
	}

	m.JumpChildSection()
	if got := m.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected to stay on Alpha, got %s", got)
	}
}

func TestJumpChildSectionDoesNothingOnATask(t *testing.T) {
	m := newScrollList(t, levelFixture, 40)
	before := m.cursorKind()
	m.JumpChildSection()
	if got := m.cursorKind(); got != before {
		t.Errorf("expected no move from a task, went to %s", got)
	}
}

// ctrl+h climbs one level at a time and stops at "##".
func TestJumpParentSectionAscends(t *testing.T) {
	m := newScrollList(t, levelFixture, 40)
	m.JumpNextSectionOfLevel(2)
	m.JumpChildSection()
	m.JumpChildSection() // Alpha deep

	for _, w := range []string{"heading:Alpha sub", "heading:Alpha"} {
		m.JumpParentSection(2)
		if got := m.cursorKind(); got != w {
			t.Fatalf("expected %s, got %s", w, got)
		}
	}

	m.JumpParentSection(2)
	if got := m.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected ## to be the ceiling, got %s", got)
	}
}

// From a task, the parent is the heading the task lives under.
func TestJumpParentSectionFromTask(t *testing.T) {
	m := newScrollList(t, levelFixture, 40)
	for m.cursorKind() != "task:deep task" {
		before := m.cursor
		m.MoveDown()
		if m.cursor == before {
			t.Fatal("never reached the deep task")
		}
	}

	m.JumpParentSection(2)
	if got := m.cursorKind(); got != "heading:Alpha deep" {
		t.Errorf("expected the enclosing heading, got %s", got)
	}
}

// A heading nested under a skipped level still finds its nearest ancestor.
func TestJumpParentSectionWithSkippedLevels(t *testing.T) {
	content := "## Alpha\n- [ ] alpha task\n#### Deep\n- [ ] deep task\n"
	m := newScrollList(t, content, 40)
	for m.cursorKind() != "heading:Deep" {
		m.JumpNextSection()
	}

	m.JumpParentSection(2)
	if got := m.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected Alpha, got %s", got)
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

	if !strings.Contains(row, cursorGlyph) {
		t.Errorf("expected a cursor on the selected heading, got %q", row)
	}
	if !strings.Contains(row, "Alpha") {
		t.Errorf("expected the heading text, got %q", row)
	}

	// Exactly one row carries the cursor.
	count := 0
	for _, l := range lines {
		if strings.Contains(l, cursorGlyph) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one cursor row, got %d", count)
	}
}
