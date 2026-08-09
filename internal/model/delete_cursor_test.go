package model

import (
	"os"
	"testing"

	"github.com/svellingen/md-taco/internal/storage"
)

// newDeleteApp returns an app over content, backed by a real file so saves
// during deletion behave as they do in use.
func newDeleteApp(t *testing.T, content string) App {
	t.Helper()
	path := t.TempDir() + "/tasks.md"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	store := storage.NewStore(path)
	tf, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	a := NewApp(store)
	a.taskFile = tf
	a.list = NewTaskListModel(tf)
	a.width, a.height = 80, 40
	a.updateListSize()
	a.stamp, _ = statFile(path)
	return a
}

// moveTo walks the cursor down until it reaches the named item.
func moveTo(t *testing.T, a App, want string) App {
	t.Helper()
	for range 100 {
		if a.list.cursorKind() == want {
			return a
		}
		before := a.list.cursor
		a.list.MoveDown()
		if a.list.cursor == before {
			break
		}
	}
	t.Fatalf("never reached %s (stopped at %s)", want, a.list.cursorKind())
	return a
}

const deleteFixture = "# Top\n" +
	"- [ ] top task\n" +
	"\n" +
	"## Alpha\n" +
	"- [ ] alpha one\n" +
	"- [ ] alpha two\n" +
	"- [ ] alpha three\n" +
	"\n" +
	"## Beta\n" +
	"- [ ] beta one\n"

// Deleting a task leaves the cursor on the task above it, not back at the top.
func TestDeleteLandsOnPreviousSibling(t *testing.T) {
	a := newDeleteApp(t, deleteFixture)
	a = moveTo(t, a, "task:alpha two")

	next, _ := a.doDelete()
	if got := next.(App).list.cursorKind(); got != "task:alpha one" {
		t.Errorf("expected task:alpha one, got %s", got)
	}
}

// Deleting the first task under a heading leaves the cursor on that heading.
func TestDeleteFirstTaskLandsOnHeading(t *testing.T) {
	a := newDeleteApp(t, deleteFixture)
	a = moveTo(t, a, "task:alpha one")

	next, _ := a.doDelete()
	if got := next.(App).list.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected heading:Alpha, got %s", got)
	}
}

// The first task of the file has a heading above it, so that is where it lands.
func TestDeleteFirstItemLandsOnTitle(t *testing.T) {
	a := newDeleteApp(t, deleteFixture)
	a = moveTo(t, a, "task:top task")

	next, _ := a.doDelete()
	if got := next.(App).list.cursorKind(); got != "heading:Top" {
		t.Errorf("expected heading:Top, got %s", got)
	}
}

// With nothing above at all, the cursor falls back to the top of the list.
func TestDeleteWithNothingAboveFallsBack(t *testing.T) {
	a := newDeleteApp(t, "- [ ] only\n- [ ] second\n")
	a = moveTo(t, a, "task:only")

	next, _ := a.doDelete()
	if got := next.(App).list.cursorKind(); got != "task:second" {
		t.Errorf("expected task:second, got %s", got)
	}
}

func TestDeleteLastTaskLeavesEmptyListIntact(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] only\n")
	a = moveTo(t, a, "task:only")

	next, _ := a.doDelete()
	app := next.(App)
	if len(app.taskFile.Tasks) != 0 {
		t.Errorf("expected no tasks left, got %d", len(app.taskFile.Tasks))
	}
	// The heading is still there and is a valid place to be.
	if got := app.list.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected heading:Alpha, got %s", got)
	}
}

// Deleting a heading lands on whatever preceded it.
func TestDeleteSectionLandsAbove(t *testing.T) {
	a := newDeleteApp(t, deleteFixture)
	for a.list.cursorKind() != "heading:Beta" {
		a.list.JumpNextSection()
	}

	next, _ := a.doDeleteSection()
	if got := next.(App).list.cursorKind(); got != "task:alpha three" {
		t.Errorf("expected task:alpha three, got %s", got)
	}
}
