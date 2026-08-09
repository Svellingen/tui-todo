package model

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/svellingen/md-taco/internal/storage"
)

// newInputApp returns an app sitting in the add prompt.
func newInputApp(t *testing.T) App {
	t.Helper()
	tf, err := storage.NewParser().Parse("## Alpha\n- [ ] existing\n")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	a := NewApp(storage.NewStore(t.TempDir() + "/tasks.md"))
	a.taskFile = tf
	a.list = NewTaskListModel(tf)
	a.width, a.height = 80, 24
	a.updateListSize()
	a.mode = modeInput
	a.input.StartAdd()
	return a
}

// typeRunes delivers a word the way a terminal does when the bytes arrive in a
// single read: one message carrying every rune.
func typeRunes(a App, word string) App {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(word)}
	next, _ := a.handleKey(msg)
	return next.(App)
}

// A run of runes that happens to spell a key name is text, not that key.
// bubbletea folds runes arriving in one read into a single message whose
// String() is the whole word, so matching on names would commit the input.
func TestTypingKeyNamesDoesNotTriggerThoseKeys(t *testing.T) {
	for _, word := range []string{"enter", "esc", "tab", "space", "up", "down"} {
		a := newInputApp(t)
		a = typeRunes(a, word)

		if a.mode != modeInput {
			t.Errorf("typing %q left input mode", word)
		}
		if got := a.input.Value(); got != word {
			t.Errorf("typing %q produced %q", word, got)
		}
	}
}

// The real Enter and Esc keys still work, since they arrive with their own
// key types rather than as runes.
func TestEnterAndEscStillWorkInInput(t *testing.T) {
	a := newInputApp(t)
	a = typeRunes(a, "a new task")
	next, _ := a.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	committed := next.(App)
	if committed.mode != modeNormal {
		t.Error("expected Enter to commit and leave input mode")
	}
	found := false
	for _, tk := range committed.taskFile.Tasks {
		if tk.Title == "a new task" {
			found = true
		}
	}
	if !found {
		t.Error("expected the typed task to be added")
	}

	b := newInputApp(t)
	b = typeRunes(b, "abandoned")
	next, _ = b.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	cancelled := next.(App)
	if cancelled.mode != modeNormal {
		t.Error("expected Esc to leave input mode")
	}
	for _, tk := range cancelled.taskFile.Tasks {
		if tk.Title == "abandoned" {
			t.Error("expected Esc to discard the input")
		}
	}
}

// The help overlay closes on the real Esc, not on the letters "esc".
func TestHelpOverlayEscHandling(t *testing.T) {
	a := newInputApp(t)
	a.mode = modeHelp

	next, _ := a.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")})
	if next.(App).mode != modeHelp {
		t.Error("typing 'esc' should not close the overlay")
	}

	next, _ = a.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(App).mode != modeNormal {
		t.Error("expected the Esc key to close the overlay")
	}
}
