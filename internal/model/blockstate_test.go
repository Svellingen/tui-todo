package model

import (
	"testing"

	"github.com/svellingen/md-taco/internal/task"
)

const stateFixture = "## Alpha\n" +
	"- [ ] parent\n" +
	"  a plain note\n" +
	"  - [ ] !! a subtask +work\n"

// onBlockLine puts the cursor on one line of the first task's block.
func onBlockLine(t *testing.T, content string, index int) App {
	t.Helper()
	a := newDeleteApp(t, content)
	a = moveTo(t, a, "task:parent")
	a.list.ToggleExpand()
	for range index + 1 {
		a.list.MoveDown()
	}
	if p, i := a.list.SelectedBlockLine(); p != 0 || i != index {
		t.Fatalf("expected to be on block line %d, got (%d,%d)", index, p, i)
	}
	return a
}

func lineState(a App, index int) int {
	return blockLineState(a.taskFile.Tasks[0].Block[index])
}

// Space walks a block line up the scale and wraps back round to a note.
func TestBlockLineCyclesForward(t *testing.T) {
	a := onBlockLine(t, stateFixture, 0)

	want := []int{blockTodo, blockInProgress, blockDone, blockNote}
	for step, w := range want {
		next, _ := a.cycleStatus()
		a = next.(App)
		if got := lineState(a, 0); got != w {
			t.Fatalf("step %d: got state %d, want %d", step+1, got, w)
		}
	}
}

// ctrl+space walks it back and stops at a note rather than wrapping.
func TestBlockLineCyclesBackAndStopsAtNote(t *testing.T) {
	a := onBlockLine(t, stateFixture, 1)
	// Start from done.
	for lineState(a, 1) != blockDone {
		next, _ := a.cycleStatus()
		a = next.(App)
	}

	want := []int{blockInProgress, blockTodo, blockNote, blockNote}
	for step, w := range want {
		next, _ := a.cycleStatusBack()
		a = next.(App)
		if got := lineState(a, 1); got != w {
			t.Fatalf("step %d: got state %d, want %d", step+1, got, w)
		}
	}
}

// Demoting a subtask keeps everything a note can hold, and promoting it reads
// those markers back.
func TestBlockLineConversionKeepsMetadata(t *testing.T) {
	a := onBlockLine(t, stateFixture, 1)

	next, _ := a.cycleStatusBack()
	a = next.(App)

	line := a.taskFile.Tasks[0].Block[1]
	if line.Subtask != nil {
		t.Fatal("expected a note")
	}
	if line.Note != "!! a subtask +work" {
		t.Errorf("expected the markers kept as text, got %q", line.Note)
	}

	next, _ = a.cycleStatus()
	a = next.(App)

	sub := a.taskFile.Tasks[0].Block[1].Subtask
	if sub == nil {
		t.Fatal("expected a subtask again")
	}
	if sub.Title != "a subtask" {
		t.Errorf("title: got %q", sub.Title)
	}
	if sub.Priority != task.PriorityHigh {
		t.Errorf("expected the priority read back, got %d", sub.Priority)
	}
	if len(sub.Tags) != 1 || sub.Tags[0] != "work" {
		t.Errorf("expected the tag read back, got %v", sub.Tags)
	}
	if sub.Status != task.StatusTodo {
		t.Errorf("a promoted note starts at todo, got %d", sub.Status)
	}
}

// A note that is already a note has nowhere further back to go, and that
// non-move must not reach the undo stack.
func TestBlockLineBackFromNoteIsANoOp(t *testing.T) {
	a := onBlockLine(t, stateFixture, 0)
	before := a.taskFile.Tasks[0].Block[0]

	next, _ := a.cycleStatusBack()
	a = next.(App)

	if a.taskFile.Tasks[0].Block[0] != before {
		t.Error("expected the note unchanged")
	}
	if len(a.undoStack) != 0 {
		t.Errorf("expected no undo entry, got %d", len(a.undoStack))
	}
}

// Cycling a block line leaves its parent alone.
func TestBlockLineCycleLeavesParentAlone(t *testing.T) {
	a := onBlockLine(t, stateFixture, 0)

	next, _ := a.cycleStatus()
	a = next.(App)

	if got := a.taskFile.Tasks[0].Status; got != task.StatusTodo {
		t.Errorf("expected the parent untouched, got %d", got)
	}
}
