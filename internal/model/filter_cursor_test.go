package model

import "testing"

const filterCursorFixture = "## Alpha\n" +
	"- [ ] one\n" +
	"- [x] two done\n" +
	"- [ ] three\n" +
	"- [-] four active\n" +
	"- [ ] five\n" +
	"- [x] six done\n"

// Switching status filters keeps the cursor on the task it was on, rather than
// dumping you back at the top.
func TestStatusFilterKeepsCursorOnVisibleTask(t *testing.T) {
	m := newScrollList(t, filterCursorFixture, 30)
	for m.cursorKind() != "task:five" {
		before := m.cursor
		m.MoveDown()
		if m.cursor == before {
			t.Fatal("never reached 'five'")
		}
	}

	m.SetStatusFilter(filterActive)
	if got := m.cursorKind(); got != "task:five" {
		t.Errorf("expected the cursor kept, got %s", got)
	}

	m.SetStatusFilter(filterAll)
	if got := m.cursorKind(); got != "task:five" {
		t.Errorf("expected the cursor kept on the way back, got %s", got)
	}
}

// When the filter hides the selected task, the cursor lands near where it was
// rather than at the very top.
func TestStatusFilterFallsBackNearOldPosition(t *testing.T) {
	m := newScrollList(t, filterCursorFixture, 30)
	for m.cursorKind() != "task:six done" {
		before := m.cursor
		m.MoveDown()
		if m.cursor == before {
			t.Fatal("never reached 'six done'")
		}
	}
	pos := m.cursor

	m.SetStatusFilter(filterActive)

	if got := m.cursorKind(); got == "task:six done" {
		t.Fatal("the done task should be hidden by the active filter")
	}
	// Near where it was, not reset to the first item.
	if m.cursor != pos && m.cursor != len(m.items)-1 {
		t.Errorf("expected the cursor near %d, got %d", pos, m.cursor)
	}
	if m.cursor == 0 {
		t.Error("expected not to jump to the top of the list")
	}
}

// The tag and context filters preserve it too.
func TestLabelFiltersKeepCursor(t *testing.T) {
	content := "## Alpha\n" +
		"- [ ] one +work\n" +
		"- [ ] two +work @home\n" +
		"- [ ] three +work\n"

	m := newScrollList(t, content, 30)
	for m.cursorKind() != "task:three" {
		before := m.cursor
		m.MoveDown()
		if m.cursor == before {
			t.Fatal("never reached 'three'")
		}
	}

	m.SetTagFilter("work")
	if got := m.cursorKind(); got != "task:three" {
		t.Errorf("tag filter: expected the cursor kept, got %s", got)
	}

	m.ClearTagFilter()
	if got := m.cursorKind(); got != "task:three" {
		t.Errorf("clearing the tag filter: got %s", got)
	}

	// A context filter that hides it still lands somewhere sensible.
	m.SetContextFilter("home")
	if got := m.cursorKind(); got != "task:two" {
		t.Errorf("context filter: expected the only match, got %s", got)
	}
}

// A heading under the cursor is preserved across a filter change too.
func TestFilterKeepsCursorOnHeading(t *testing.T) {
	m := newScrollList(t, filterCursorFixture, 30)
	for m.cursorKind() != "heading:Alpha" {
		m.JumpPrevSection()
	}

	m.SetStatusFilter(filterDone)
	if got := m.cursorKind(); got != "heading:Alpha" {
		t.Errorf("expected the heading kept, got %s", got)
	}
}
