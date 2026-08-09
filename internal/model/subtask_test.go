package model

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/svellingen/md-taco/internal/task"
)

const subtaskFixture = "## Alpha\n" +
	"- [ ] parent\n" +
	"  a note\n" +
	"  - [x] done sub +work\n" +
	"- [ ] no block\n"

// blockTexts names each line of a task's block.
func blockTexts(a App, parent int) []string {
	var out []string
	for _, b := range a.taskFile.Tasks[parent].Block {
		if b.Subtask != nil {
			out = append(out, "sub:"+b.Subtask.Title)
			continue
		}
		out = append(out, "note:"+b.Note)
	}
	return out
}

// "A" on a task appends a subtask and unfolds the block to show it.
func TestAddSubtaskFromTask(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")

	next, _ := a.startAddSubtask(0, len(a.taskFile.Tasks[0].Block))
	a = next.(App)
	if !a.list.expanded[0] {
		t.Error("expected the block unfolded so the editor is visible")
	}

	a = typeRunes(a, "brand new")
	next, _ = a.commitInput()
	a = next.(App)

	got := strings.Join(blockTexts(a, 0), ",")
	if got != "note:a note,sub:done sub,sub:brand new" {
		t.Errorf("got %q", got)
	}
}

// "a" on a block line inserts a sibling directly after it.
func TestAddSubtaskAfterCurrentBlockLine(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown() // the note, at index 0

	parent, index := a.list.SelectedBlockLine()
	next, _ := a.startAddSubtask(parent, index+1)
	a = next.(App)
	a = typeRunes(a, "between")
	next, _ = a.commitInput()
	a = next.(App)

	got := strings.Join(blockTexts(a, 0), ",")
	if got != "note:a note,sub:between,sub:done sub" {
		t.Errorf("got %q", got)
	}
}

// Metadata can be given inline when adding, as it can for a task.
func TestAddSubtaskParsesMetadata(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")

	next, _ := a.startAddSubtask(0, 0)
	a = next.(App)
	a = typeRunes(a, "!! urgent thing +proj @desk")
	next, _ = a.commitInput()
	a = next.(App)

	sub := a.taskFile.Tasks[0].Block[0].Subtask
	if sub == nil {
		t.Fatal("expected a subtask")
	}
	if sub.Title != "urgent thing" || sub.Priority != task.PriorityHigh {
		t.Errorf("got %q priority %d", sub.Title, sub.Priority)
	}
	if len(sub.Tags) != 1 || sub.Tags[0] != "proj" {
		t.Errorf("tags: %v", sub.Tags)
	}
	if len(sub.Contexts) != 1 || sub.Contexts[0] != "desk" {
		t.Errorf("contexts: %v", sub.Contexts)
	}
}

// Editing a subtask prefills its title and keeps what the new text does not
// mention -- the same contract as editing a top-level task.
func TestEditSubtaskKeepsUnmentionedMetadata(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown()
	a.list.MoveDown() // the subtask

	next, _ := a.startEdit()
	a = next.(App)
	if got := a.input.Value(); got != "done sub" {
		t.Errorf("expected the title prefilled, got %q", got)
	}

	a = typeRunes(a, " renamed")
	next, _ = a.commitInput()
	a = next.(App)

	sub := a.taskFile.Tasks[0].Block[1].Subtask
	if sub.Title != "done sub renamed" {
		t.Errorf("title: got %q", sub.Title)
	}
	if sub.Status != task.StatusDone {
		t.Errorf("expected the status kept, got %d", sub.Status)
	}
	if len(sub.Tags) != 1 || sub.Tags[0] != "work" {
		t.Errorf("expected the tag kept, got %v", sub.Tags)
	}
}

// A note is editable too, and stays a note.
func TestEditNote(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown() // the note

	next, _ := a.startEdit()
	a = next.(App)
	if got := a.input.Value(); got != "a note" {
		t.Errorf("expected the note prefilled, got %q", got)
	}

	a = typeRunes(a, " updated")
	next, _ = a.commitInput()
	a = next.(App)

	line := a.taskFile.Tasks[0].Block[0]
	if line.Subtask != nil || line.Note != "a note updated" {
		t.Errorf("expected a note, got %+v", line)
	}
}

// Retyping a note as a checkbox item promotes it to a subtask.
func TestEditNoteIntoSubtask(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown()

	next, _ := a.startEditBlockLine(0, 0)
	a = next.(App)
	a.input.input.SetValue("- [ ] now a subtask")
	next, _ = a.commitInput()
	a = next.(App)

	line := a.taskFile.Tasks[0].Block[0]
	if line.Subtask == nil || line.Subtask.Title != "now a subtask" {
		t.Errorf("expected a subtask, got %+v", line)
	}
}

// Esc keeps a subtask edit, as it does for a task.
func TestEscKeepsASubtaskEdit(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown()
	a.list.MoveDown()

	next, _ := a.startEdit()
	a = next.(App)
	a = typeRunes(a, " kept")

	next, _ = a.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	a = next.(App)

	if got := a.taskFile.Tasks[0].Block[1].Subtask.Title; got != "done sub kept" {
		t.Errorf("got %q", got)
	}
}

// "A" needs a top-level task; it does nothing from a block line.
func TestAddSubtaskNeedsATask(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown()

	if idx := a.list.SelectedTaskIndex(); idx >= 0 {
		t.Fatalf("expected no task index on a block line, got %d", idx)
	}
}

// The add editor is drawn where the new task will actually land: after the
// anchor's whole block, not wedged between the task and its own subtasks.
func TestAddEditorSitsBelowAnExpandedBlock(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  a note\n  - [ ] sub\n- [ ] next\n")
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()

	next, _ := a.startAdd()
	a = next.(App)

	lines, starts := a.list.buildLines()
	_, editorRow := a.spliceInlineInput(lines, starts)

	blockRows := []int{
		starts[a.list.ItemForBlockLine(0, 0)],
		starts[a.list.ItemForBlockLine(0, 1)],
	}
	for _, r := range blockRows {
		if editorRow <= r {
			t.Errorf("editor at row %d should be below block row %d", editorRow, r)
		}
	}
	if editorRow > starts[a.list.ItemForTask(1)] {
		t.Error("the editor should still come before the following task")
	}
}

// Folded, there is no block on screen, so the editor sits directly below.
func TestAddEditorDirectlyBelowACollapsedTask(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  a note\n- [ ] next\n")
	a = moveTo(t, a, "task:parent")

	next, _ := a.startAdd()
	a = next.(App)

	lines, starts := a.list.buildLines()
	_, editorRow := a.spliceInlineInput(lines, starts)

	if want := starts[a.list.ItemForTask(0)] + 1; editorRow != want {
		t.Errorf("expected the editor at row %d, got %d", want, editorRow)
	}
}

// A subtask add still anchors inside the block, right after its sibling.
func TestSubtaskEditorStaysInsideTheBlock(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  a note\n  - [ ] sub\n- [ ] next\n")
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()

	next, _ := a.startAddSubtask(0, 1)
	a = next.(App)

	lines, starts := a.list.buildLines()
	_, editorRow := a.spliceInlineInput(lines, starts)

	if editorRow > starts[a.list.ItemForBlockLine(0, 1)] {
		t.Error("expected the editor above the sibling it precedes")
	}
	if editorRow <= starts[a.list.ItemForBlockLine(0, 0)] {
		t.Error("expected the editor below the line it follows")
	}
}

// A note is not a task, so editing one shows no bullet -- otherwise it reads
// as a subtask being created.
func TestNoteEditorHasNoBullet(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown() // the note

	next, _ := a.startEdit()
	a = next.(App)

	row := ansi.Strip(a.renderInlineInput())
	for _, bullet := range []string{"○", "◐", "●"} {
		if strings.Contains(row, bullet) {
			t.Errorf("expected no bullet editing a note, got %q", row)
		}
	}
	if !strings.Contains(row, "a note") {
		t.Errorf("expected the note text, got %q", row)
	}
}

// A subtask editor keeps its bullet, and the bullet reflects its status.
func TestSubtaskEditorKeepsItsBullet(t *testing.T) {
	a := newDeleteApp(t, subtaskFixture)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown()
	a.list.MoveDown() // the done subtask

	next, _ := a.startEdit()
	a = next.(App)

	row := ansi.Strip(a.renderInlineInput())
	if !strings.Contains(row, "●") {
		t.Errorf("expected the done bullet, got %q", row)
	}
}

// "n" adds a note to the selected task, appended to its block.
func TestAddNoteToTask(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  - [ ] a subtask\n- [ ] other\n")
	a = moveTo(t, a, "task:parent")

	next, _ := a.startAddNote(0, len(a.taskFile.Tasks[0].Block))
	a = next.(App)
	if !a.list.expanded[0] {
		t.Error("expected the block unfolded so the editor is visible")
	}

	a = typeRunes(a, "remember the milk")
	next, _ = a.commitInput()
	a = next.(App)

	got := strings.Join(blockTexts(a, 0), ",")
	if got != "sub:a subtask,note:remember the milk" {
		t.Errorf("got %q", got)
	}
}

// On a block line it inserts directly after it, as "a" does for a subtask.
func TestAddNoteAfterBlockLine(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  - [ ] a subtask\n")
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	a.list.MoveDown()

	parent, index := a.list.SelectedBlockLine()
	next, _ := a.startAddNote(parent, index+1)
	a = next.(App)
	a = typeRunes(a, "inserted")
	next, _ = a.commitInput()
	a = next.(App)

	got := strings.Join(blockTexts(a, 0), ",")
	if got != "sub:a subtask,note:inserted" {
		t.Errorf("got %q", got)
	}
}

// A note added this way is a real note, not a subtask whose text looks like
// one, so it takes part in the four-state cycle.
func TestAddedNoteIsANote(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  - [ ] a subtask\n")
	a = moveTo(t, a, "task:parent")

	next, _ := a.startAddNote(0, 0)
	a = next.(App)
	a = typeRunes(a, "- [ ] looks like a task")
	next, _ = a.commitInput()
	a = next.(App)

	line := a.taskFile.Tasks[0].Block[0]
	if line.Subtask != nil {
		t.Errorf("expected a note even when the text looks like a task, got %+v", line)
	}
	if line.Note != "- [ ] looks like a task" {
		t.Errorf("expected the text verbatim, got %q", line.Note)
	}
}

// The note editor shows no bullet while adding, matching how a note renders.
func TestAddNoteEditorHasNoBullet(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] parent\n  - [ ] a subtask\n")
	a = moveTo(t, a, "task:parent")

	next, _ := a.startAddNote(0, 0)
	a = next.(App)

	row := ansi.Strip(a.renderInlineInput())
	for _, bullet := range []string{"○", "◐", "●"} {
		if strings.Contains(row, bullet) {
			t.Errorf("expected no bullet, got %q", row)
		}
	}
}
