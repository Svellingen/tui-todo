package model

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/macone/todo-cli/internal/storage"
)

const inlineFixture = "## Alpha\n" +
	"- [ ] first\n" +
	"- [ ] second\n" +
	"## Beta\n" +
	"- [ ] third\n"

func newInlineApp(t *testing.T) App {
	t.Helper()
	tf, err := storage.NewParser().Parse(inlineFixture)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	a := NewApp(storage.NewStore(t.TempDir() + "/tasks.md"))
	a.taskFile = tf
	a.list = NewTaskListModel(tf)
	a.width, a.height = 80, 24
	a.updateListSize()
	return a
}

// viewRows returns the rendered list rows with styling stripped.
func viewRows(a App) []string {
	lines, starts := a.list.buildLines()
	lines, _ = a.spliceInlineInput(lines, starts)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimRight(ansi.Strip(l), " ")
	}
	return out
}

// Editing replaces the task's own row rather than opening a prompt elsewhere.
func TestInlineEditReplacesTheTaskRow(t *testing.T) {
	a := newInlineApp(t)
	a = moveTo(t, a, "task:second")

	next, _ := a.startEdit()
	a = next.(App)

	rows := viewRows(a)
	// Alpha, first, second(edited), Beta(with blank), third
	if !strings.Contains(rows[2], "second") {
		t.Fatalf("expected the editor on the second task's row, got %q", rows[2])
	}
	if strings.Contains(rows[1], "second") {
		t.Errorf("the row above should be untouched, got %q", rows[1])
	}

	// The row count is unchanged: an edit stands in for a row, it does not add.
	plain := newInlineApp(t)
	if len(rows) != len(viewRows(plain)) {
		t.Errorf("edit changed the row count: %d vs %d", len(rows), len(viewRows(plain)))
	}
}

// Adding from a task opens the editor on the following row.
func TestInlineAddFromTaskUsesNextRow(t *testing.T) {
	a := newInlineApp(t)
	a = moveTo(t, a, "task:first")

	a.mode = modeInput
	a.inputAnchorItem = a.list.Cursor()
	a.input.StartAdd()

	rows := viewRows(a)
	if !strings.Contains(rows[1], "first") {
		t.Fatalf("expected the anchor task to stay, got %q", rows[1])
	}
	// The editor is empty, so its row shows the placeholder.
	if !strings.Contains(rows[2], "Task title") {
		t.Errorf("expected the editor directly below, got %q", rows[2])
	}
	if !strings.Contains(rows[3], "second") {
		t.Errorf("expected the rest to shift down, got %q", rows[3])
	}
}

// Adding from a heading opens the editor directly beneath the heading text.
func TestInlineAddFromHeadingUsesRowBelowHeading(t *testing.T) {
	a := newInlineApp(t)
	for a.list.cursorKind() != "heading:Beta" {
		a.list.JumpNextSection()
	}

	a.mode = modeInput
	a.inputAnchorItem = a.list.Cursor()
	a.input.StartAdd()

	rows := viewRows(a)
	var headingRow int
	for i, r := range rows {
		if strings.Contains(r, "Beta") {
			headingRow = i
		}
	}
	if !strings.Contains(rows[headingRow+1], "Task title") {
		t.Errorf("expected the editor right under Beta, got %q", rows[headingRow+1])
	}
}

// Search and tag keep their prompt below the list.
func TestSearchAndTagAreNotInline(t *testing.T) {
	a := newInlineApp(t)

	a.mode = modeInput
	a.inputAnchorItem = a.list.Cursor()
	a.input.StartSearch()
	if a.inlineInputActive() {
		t.Error("search should not render inline")
	}

	a.input.StartTag()
	if a.inlineInputActive() {
		t.Error("tag should not render inline")
	}

	// Neither adds a row to the list.
	if len(viewRows(a)) != len(viewRows(newInlineApp(t))) {
		t.Error("a non-inline editor changed the row count")
	}
}

// An editor spliced past the bottom of the viewport pulls the view down with
// it rather than falling off the end.
func TestClipAroundKeepsEditorVisible(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "line"
	}
	lines[29] = "EDITOR"

	m := TaskListModel{scrollOffset: 0}
	got := m.clipAround(lines, 10, 29)

	if len(got) != 10 {
		t.Fatalf("expected 10 rows, got %d", len(got))
	}
	if got[9] != "EDITOR" {
		t.Errorf("expected the editor on the last visible row, got %q", got[9])
	}
}

// An offset past the end yields a full window at the end, not a short one.
func TestClipLinesClampsOutOfRangeOffset(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}

	if got := clipLines(lines, 3, 99); len(got) != 3 || got[0] != "c" {
		t.Errorf("offset past the end: got %q", got)
	}
	if got := clipLines(lines, 3, -5); len(got) != 3 || got[0] != "a" {
		t.Errorf("negative offset: got %q", got)
	}
}

// Enter folds the selected task's block; adding is on "a" alone.
func TestEnterFoldsTheBlock(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  a note\n- [ ] plain\n")
	a = moveTo(t, a, "task:parent")

	next, _ := a.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(App)

	if got.mode == modeInput {
		t.Error("Enter should no longer open the add editor")
	}
	if !got.list.expanded[0] {
		t.Error("expected Enter to fold the block open")
	}

	next, _ = got.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(App).list.expanded[0] {
		t.Error("expected Enter again to fold it shut")
	}
}

// The letters "enter" arriving as one message are text, not the Enter key, so
// they must not act as it from the list.
func TestTypingEnterInNormalModeIsInert(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  a note\n")
	a = moveTo(t, a, "task:parent")

	next, _ := a.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})
	if next.(App).list.expanded[0] {
		t.Error("a run of runes spelling \"enter\" should not fold anything")
	}
}

// Esc out of an edit keeps what was typed: it means "stop editing", not
// "undo the edit".
func TestEscKeepsAnEdit(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] original\n")
	a = moveTo(t, a, "task:original")

	next, _ := a.startEdit()
	a = next.(App)
	a = typeRunes(a, " edited")

	next, _ = a.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	a = next.(App)

	if a.mode != modeNormal {
		t.Error("expected Esc to leave input mode")
	}
	if got := a.taskFile.Tasks[0].Title; got != "original edited" {
		t.Errorf("expected the edit kept, got %q", got)
	}
}

// Esc out of an add still discards: there is no prior text to keep.
func TestEscDiscardsAnAdd(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] original\n")
	a = moveTo(t, a, "task:original")

	next, _ := a.startAdd()
	a = next.(App)
	a = typeRunes(a, "should not exist")

	next, _ = a.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	a = next.(App)

	if len(a.taskFile.Tasks) != 1 {
		t.Fatalf("expected the add discarded, got %d tasks", len(a.taskFile.Tasks))
	}
	for _, tk := range a.taskFile.Tasks {
		if tk.Title == "should not exist" {
			t.Error("the abandoned add was kept")
		}
	}
}

// Emptying the title and pressing Esc leaves the task as it was, rather than
// committing a blank one.
func TestEscOnAnEmptiedEditKeepsTheOriginal(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] original\n")
	a = moveTo(t, a, "task:original")

	next, _ := a.startEdit()
	a = next.(App)
	a.input.input.SetValue("")

	next, _ = a.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	a = next.(App)

	if got := a.taskFile.Tasks[0].Title; got != "original" {
		t.Errorf("expected the original title, got %q", got)
	}
}
