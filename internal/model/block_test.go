package model

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/macone/todo-cli/internal/task"
)

const blockAppFixture = "## Alpha\n" +
	"- [ ] parent\n" +
	"  a note\n" +
	"  - [ ] first step\n" +
	"  - [x] second step\n" +
	"- [ ] plain\n"

// rows names what the cursor can land on, in order.
func rows(m TaskListModel) []string {
	var out []string
	for i := range m.items {
		if !m.selectable(i) {
			continue
		}
		saved := m.cursor
		m.cursor = i
		out = append(out, m.cursorKind())
		m.cursor = saved
	}
	return out
}

// Blocks start folded, so the list stays scannable.
func TestBlocksStartCollapsed(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)

	if got := strings.Join(rows(a.list), ","); got != "task:parent,task:plain" {
		t.Errorf("expected only top-level tasks, got %q", got)
	}
}

// The fold marker only appears on a task that has something under it.
func TestFoldMarkerOnlyWhereThereIsABlock(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	lines, starts := a.list.buildLines()

	parent := ansi.Strip(lines[starts[indexOfKind(a.list, "task:parent")]])
	if !strings.Contains(parent, "▸") {
		t.Errorf("expected a fold marker on the parent, got %q", parent)
	}

	plain := ansi.Strip(lines[starts[indexOfKind(a.list, "task:plain")]])
	if strings.Contains(plain, "▸") || strings.Contains(plain, "▾") {
		t.Errorf("expected no marker on a task without a block, got %q", plain)
	}
}

func TestToggleExpandRevealsAndHidesTheBlock(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	a = moveTo(t, a, "task:parent")

	if !a.list.ToggleExpand() {
		t.Fatal("expected the toggle to take")
	}
	got := strings.Join(rows(a.list), ",")
	if got != "task:parent,block,block,block,task:plain" {
		t.Errorf("expanded: got %q", got)
	}

	a.list.ToggleExpand()
	if got := strings.Join(rows(a.list), ","); got != "task:parent,task:plain" {
		t.Errorf("collapsed again: got %q", got)
	}
}

// A task with no block has nothing to fold.
func TestToggleExpandDoesNothingWithoutABlock(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	a = moveTo(t, a, "task:plain")

	if a.list.ToggleExpand() {
		t.Error("expected the toggle to be refused")
	}
}

// Folding from inside the block closes the parent, so tab always shuts what
// you are in.
func TestToggleExpandFromInsideFoldsTheParent(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()

	a.list.MoveDown()
	a.list.MoveDown() // onto a subtask
	if a.list.SelectedSubtask() == nil {
		t.Fatal("expected the cursor on a subtask")
	}

	a.list.ToggleExpand()
	if got := strings.Join(rows(a.list), ","); got != "task:parent,task:plain" {
		t.Errorf("expected the block folded, got %q", got)
	}
}

// Status and priority act on the subtask under the cursor, not its parent.
func TestSubtaskStatusAndPriority(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown()
	a.list.MoveDown()

	sub := a.list.SelectedSubtask()
	if sub == nil || sub.Title != "first step" {
		t.Fatalf("expected 'first step', got %+v", sub)
	}

	next, _ := a.cycleStatus()
	a = next.(App)
	if got := a.taskFile.Tasks[0].Block[1].Subtask.Status; got != task.StatusInProgress {
		t.Errorf("subtask status: got %d", got)
	}
	if got := a.taskFile.Tasks[0].Status; got != task.StatusTodo {
		t.Errorf("the parent should be untouched, got %d", got)
	}

	next, _ = a.cyclePriority(1)
	a = next.(App)
	if got := a.taskFile.Tasks[0].Block[1].Subtask.Priority; got != task.PriorityMedium {
		t.Errorf("subtask priority: got %d", got)
	}
}

// Deleting a block line removes just that line.
func TestDeleteBlockLine(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown() // the note

	parent, index := a.list.SelectedBlockLine()
	if parent != 0 || index != 0 {
		t.Fatalf("expected the note at (0,0), got (%d,%d)", parent, index)
	}

	next, _ := a.deleteBlockLine(parent, index)
	a = next.(App)

	block := a.taskFile.Tasks[0].Block
	if len(block) != 2 {
		t.Fatalf("expected 2 block lines left, got %d", len(block))
	}
	if block[0].Subtask == nil || block[0].Subtask.Title != "first step" {
		t.Errorf("expected the note gone and subtasks kept, got %+v", block[0])
	}
	if len(a.taskFile.Tasks) != 2 {
		t.Errorf("the tasks themselves should be untouched, got %d", len(a.taskFile.Tasks))
	}
}

// Operations addressed by task index stay off block lines.
func TestBlockLineIsNotATopLevelTask(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown()
	a.list.MoveDown()

	if got := a.list.SelectedTaskIndex(); got != -1 {
		t.Errorf("expected no task index on a block line, got %d", got)
	}
	// Moving between headings is a top-level operation.
	before := len(a.taskFile.Tasks)
	next, _ := a.moveTaskToSection(1)
	if len(next.(App).taskFile.Tasks) != before {
		t.Error("a block line should not be movable between headings")
	}
}

func indexOfKind(m TaskListModel, kind string) int {
	saved := m.cursor
	defer func() { m.cursor = saved }()
	for i := range m.items {
		m.cursor = i
		if m.cursorKind() == kind {
			return i
		}
	}
	return -1
}

// The fold marker occupies a column of its own, so titles line up whether or
// not a task has a block, and the cursor is the gutter "-".
func TestRowLayoutAlignsTitles(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	a = moveTo(t, a, "task:parent")

	lines, starts := a.list.buildLines()
	parent := ansi.Strip(lines[starts[indexOfKind(a.list, "task:parent")]])
	plain := ansi.Strip(lines[starts[indexOfKind(a.list, "task:plain")]])

	if !strings.HasPrefix(parent, " - ") {
		t.Errorf("expected the selected row to carry the gutter cursor, got %q", parent)
	}
	if !strings.HasPrefix(plain, "   ") {
		t.Errorf("expected an unselected row to have a blank gutter, got %q", plain)
	}

	// Titles start at the same column either way. Compare display columns,
	// not byte offsets: the bullet and marker are multi-byte.
	column := func(row, title string) int {
		return ansi.StringWidth(row[:strings.Index(row, title)])
	}
	if a, b := column(parent, "parent"), column(plain, "plain"); a != b {
		t.Errorf("titles misaligned at columns %d and %d:\n%q\n%q", a, b, parent, plain)
	}

	// The marker sits before the title, not after it.
	if strings.Index(parent, "▸") > strings.Index(parent, "parent") {
		t.Errorf("expected the fold marker before the title, got %q", parent)
	}
}

const multiBlockFixture = "## Alpha\n" +
	"- [ ] first\n" +
	"  note one\n" +
	"  - [ ] sub a\n" +
	"- [ ] second\n" +
	"  note two\n" +
	"- [ ] no block\n"

// countExpanded reports how many tasks currently have their block open.
func countExpanded(m TaskListModel) int {
	n := 0
	for _, i := range m.tasksWithBlocks() {
		if m.expanded[i] {
			n++
		}
	}
	return n
}

// With nothing open, shift+tab opens everything that has a block.
func TestToggleExpandAllOpensWhenNoneOpen(t *testing.T) {
	a := newDeleteApp(t, multiBlockFixture)

	if !a.list.ToggleExpandAll() {
		t.Fatal("expected the toggle to take")
	}
	if got := countExpanded(a.list); got != 2 {
		t.Errorf("expected both blocks open, got %d", got)
	}
}

// With everything open, it shuts everything.
func TestToggleExpandAllClosesWhenAllOpen(t *testing.T) {
	a := newDeleteApp(t, multiBlockFixture)
	a.list.ToggleExpandAll()

	a.list.ToggleExpandAll()
	if got := countExpanded(a.list); got != 0 {
		t.Errorf("expected everything folded, got %d open", got)
	}
}

// The rule is "any open means collapse", so a single open block folds the lot
// rather than opening the rest.
func TestToggleExpandAllCollapsesWhenSomeOpen(t *testing.T) {
	a := newDeleteApp(t, multiBlockFixture)
	a = moveTo(t, a, "task:first")
	a.list.ToggleExpand()
	if got := countExpanded(a.list); got != 1 {
		t.Fatalf("expected exactly one open, got %d", got)
	}

	a.list.ToggleExpandAll()
	if got := countExpanded(a.list); got != 0 {
		t.Errorf("expected everything folded, got %d open", got)
	}
}

// Collapsing out from under the cursor lands on the task the block belonged to.
func TestToggleExpandAllFromInsideABlock(t *testing.T) {
	a := newDeleteApp(t, multiBlockFixture)
	a.list.ToggleExpandAll()

	a = moveTo(t, a, "task:first")
	a.list.MoveDown()
	a.list.MoveDown()
	if a.list.SelectedSubtask() == nil {
		t.Fatal("expected the cursor on a subtask")
	}

	a.list.ToggleExpandAll()
	if got := a.list.cursorKind(); got != "task:first" {
		t.Errorf("expected the cursor on the parent, got %s", got)
	}
}

// A file with no blocks has nothing to fold.
func TestToggleExpandAllWithoutBlocks(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] one\n- [ ] two\n")
	if a.list.ToggleExpandAll() {
		t.Error("expected the toggle to be refused")
	}
}

// A block line's cursor sits two columns further in than a task's, under the
// parent's bullet, while the content stays aligned either way.
func TestBlockLineCursorIsIndented(t *testing.T) {
	a := newDeleteApp(t, blockAppFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown() // the note

	lines, starts := a.list.buildLines()
	selected := ansi.Strip(lines[starts[a.list.cursor]])
	if !strings.HasPrefix(selected, "   -   ") {
		t.Errorf("expected the cursor indented two columns, got %q", selected)
	}

	// An unselected block line keeps the same content column.
	a.list.MoveDown()
	lines, starts = a.list.buildLines()
	other := ansi.Strip(lines[starts[a.list.cursor]-1])
	if !strings.HasPrefix(other, "       ") {
		t.Errorf("expected a blank gutter of the same width, got %q", other)
	}
	if ansi.StringWidth(selected[:7]) != ansi.StringWidth(other[:7]) {
		t.Error("selected and unselected block lines should share a prefix width")
	}
}
