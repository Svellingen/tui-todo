package model

import (
	"testing"
)

const undoFixture = "## Alpha\n" +
	"- [ ] one\n" +
	"- [ ] two\n" +
	"- [ ] three\n" +
	"- [ ] four\n" +
	"## Beta\n" +
	"- [ ] five\n"

// Undo puts the cursor back where it was when the change was made, rather
// than at the top of the list.
func TestUndoRestoresCursorPosition(t *testing.T) {
	a := newDeleteApp(t, undoFixture)
	a = moveTo(t, a, "task:three")
	before := a.list.cursorKind()

	next, _ := a.doDelete()
	a = next.(App)
	if a.list.cursorKind() == before {
		t.Fatal("expected the delete to move the cursor off the deleted task")
	}

	next, _ = a.undo()
	a = next.(App)

	if got := a.list.cursorKind(); got != before {
		t.Errorf("expected the cursor back on %s, got %s", before, got)
	}
}

func TestRedoReappliesTheChange(t *testing.T) {
	a := newDeleteApp(t, undoFixture)
	a = moveTo(t, a, "task:three")

	next, _ := a.doDelete()
	a = next.(App)
	afterDelete := len(a.taskFile.Tasks)

	next, _ = a.undo()
	a = next.(App)
	if len(a.taskFile.Tasks) != afterDelete+1 {
		t.Fatalf("undo did not restore the task: %d tasks", len(a.taskFile.Tasks))
	}

	next, _ = a.redo()
	a = next.(App)
	if len(a.taskFile.Tasks) != afterDelete {
		t.Errorf("expected redo to remove it again, got %d tasks", len(a.taskFile.Tasks))
	}
	for _, tk := range a.taskFile.Tasks {
		if tk.Title == "three" {
			t.Error("expected 'three' to be gone after redo")
		}
	}
}

// Several changes unwind and rewind in order.
func TestUndoRedoWalksTheHistory(t *testing.T) {
	a := newDeleteApp(t, undoFixture)
	a = moveTo(t, a, "task:one")

	for range 3 {
		next, _ := a.cyclePriority(1)
		a = next.(App)
	}
	// none -> ! -> !! -> !! (clamped), so only two entries land on the stack.
	if len(a.undoStack) != 2 {
		t.Fatalf("expected 2 undo entries, got %d", len(a.undoStack))
	}

	for range 2 {
		next, _ := a.undo()
		a = next.(App)
	}
	if len(a.undoStack) != 0 || len(a.redoStack) != 2 {
		t.Fatalf("after unwinding: undo=%d redo=%d", len(a.undoStack), len(a.redoStack))
	}

	for range 2 {
		next, _ := a.redo()
		a = next.(App)
	}
	if len(a.undoStack) != 2 || len(a.redoStack) != 0 {
		t.Errorf("after rewinding: undo=%d redo=%d", len(a.undoStack), len(a.redoStack))
	}
}

// A fresh change after an undo drops the redo history, as editors do.
func TestNewChangeClearsRedo(t *testing.T) {
	a := newDeleteApp(t, undoFixture)
	a = moveTo(t, a, "task:one")

	next, _ := a.cyclePriority(1)
	a = next.(App)
	next, _ = a.undo()
	a = next.(App)
	if len(a.redoStack) != 1 {
		t.Fatalf("expected a redo entry, got %d", len(a.redoStack))
	}

	next, _ = a.toggleDone()
	a = next.(App)
	if len(a.redoStack) != 0 {
		t.Errorf("expected the redo stack to be dropped, got %d", len(a.redoStack))
	}
}

func TestUndoRedoOnEmptyStacks(t *testing.T) {
	a := newDeleteApp(t, undoFixture)

	next, _ := a.undo()
	if got := next.(App).statusMsg; got != "Nothing to undo" {
		t.Errorf("got %q", got)
	}

	next, _ = a.redo()
	if got := next.(App).statusMsg; got != "Nothing to redo" {
		t.Errorf("got %q", got)
	}
}

// A snapshot whose cursor no longer lands on a selectable item degrades to the
// nearest one instead of going out of range.
func TestRestoreClampsAnOutOfRangeCursor(t *testing.T) {
	a := newDeleteApp(t, undoFixture)

	err := a.restore(undoEntry{content: "## Alpha\n- [ ] only\n", cursor: 999})
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if got := a.list.cursorKind(); got != "task:only" {
		t.Errorf("expected the cursor clamped onto the task, got %s", got)
	}
}
