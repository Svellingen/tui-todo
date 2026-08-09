package model

import (
	"strings"
	"testing"
)

const focusFixture = "# Root\n" +
	"- [ ] root task\n" +
	"## Alpha\n" +
	"- [ ] alpha one\n" +
	"- [ ] alpha two\n" +
	"### Alpha sub\n" +
	"- [ ] sub task\n" +
	"## Beta\n" +
	"- [ ] beta one\n"

// visibleTasks names the tasks currently drawn.
func visibleTasks(m TaskListModel) []string {
	var out []string
	for _, it := range m.items {
		if it.kind == itemTask {
			out = append(out, m.taskFile.Tasks[it.taskIndex].Title)
		}
	}
	return out
}

// Focusing from a task narrows to the heading that task lives under, and takes
// that heading's sub-headings with it.
func TestFocusFromTaskShowsWholeSubtree(t *testing.T) {
	a := newDeleteApp(t, focusFixture)
	a = moveTo(t, a, "task:alpha two")

	if !a.list.FocusHeading() {
		t.Fatal("expected the focus to take")
	}

	got := strings.Join(visibleTasks(a.list), ",")
	if got != "alpha one,alpha two,sub task" {
		t.Errorf("got %q", got)
	}
	if name := a.list.FocusName(); name != "Alpha" {
		t.Errorf("expected Alpha, got %q", name)
	}
	// The cursor stays on the task it was on.
	if k := a.list.cursorKind(); k != "task:alpha two" {
		t.Errorf("expected the cursor kept, got %s", k)
	}
}

func TestFocusFromHeading(t *testing.T) {
	a := newDeleteApp(t, focusFixture)
	for a.list.cursorKind() != "heading:Beta" {
		a.list.JumpNextSection()
	}

	a.list.FocusHeading()
	if got := strings.Join(visibleTasks(a.list), ","); got != "beta one" {
		t.Errorf("got %q", got)
	}
	// The cursor stays on the heading rather than dropping to its first task.
	if k := a.list.cursorKind(); k != "heading:Beta" {
		t.Errorf("expected the cursor kept on the heading, got %s", k)
	}
}

// A sub-heading focuses only its own subtree, not its parent's.
func TestFocusOnSubHeading(t *testing.T) {
	a := newDeleteApp(t, focusFixture)
	for a.list.cursorKind() != "heading:Alpha sub" {
		a.list.JumpNextSection()
	}

	a.list.FocusHeading()
	if got := strings.Join(visibleTasks(a.list), ","); got != "sub task" {
		t.Errorf("got %q", got)
	}
}

func TestClearFocusRestoresEverything(t *testing.T) {
	a := newDeleteApp(t, focusFixture)
	a = moveTo(t, a, "task:alpha two")
	a.list.FocusHeading()

	a.list.ClearFocus()
	if a.list.Focused() {
		t.Error("expected the focus to be cleared")
	}
	got := strings.Join(visibleTasks(a.list), ",")
	if got != "root task,alpha one,alpha two,sub task,beta one" {
		t.Errorf("got %q", got)
	}
	if k := a.list.cursorKind(); k != "task:alpha two" {
		t.Errorf("expected the cursor kept across unfocus, got %s", k)
	}
}

// The indicator names the focused heading so the narrowed view is obvious.
func TestFocusShowsInIndicator(t *testing.T) {
	a := newDeleteApp(t, focusFixture)
	a = moveTo(t, a, "task:alpha one")

	if got := a.list.FilterIndicator(); got != "" {
		t.Errorf("expected no indicator before focusing, got %q", got)
	}
	a.list.FocusHeading()
	if got := a.list.FilterIndicator(); !strings.Contains(got, "focus: Alpha") {
		t.Errorf("got %q", got)
	}
}

// If the focused heading disappears the focus lapses rather than leaving the
// list showing an arbitrary slice of the file.
func TestFocusLapsesWhenHeadingDeleted(t *testing.T) {
	a := newDeleteApp(t, focusFixture)
	for a.list.cursorKind() != "heading:Beta" {
		a.list.JumpNextSection()
	}
	a.list.FocusHeading()
	if !a.list.Focused() {
		t.Fatal("expected to be focused")
	}

	next, _ := a.doDeleteSection()
	a = next.(App)

	if a.list.Focused() {
		t.Error("expected the focus to lapse with its heading")
	}
	if got := strings.Join(visibleTasks(a.list), ","); got != "root task,alpha one,alpha two,sub task" {
		t.Errorf("expected the whole file back, got %q", got)
	}
}

// Focusing a file with no headings at all is refused rather than blanking the
// list.
func TestFocusWithoutHeadings(t *testing.T) {
	a := newDeleteApp(t, "- [ ] lonely\n")
	if a.list.FocusHeading() {
		t.Error("expected focus to be refused with no heading above the cursor")
	}
	if a.list.Focused() {
		t.Error("expected not to be focused")
	}
}

// Status filters still apply inside a focus.
func TestFocusComposesWithStatusFilter(t *testing.T) {
	a := newDeleteApp(t, "## Alpha\n- [ ] open\n- [x] closed\n## Beta\n- [ ] other\n")
	a = moveTo(t, a, "task:open")
	a.list.FocusHeading()

	a.list.SetStatusFilter(filterDone)
	if got := strings.Join(visibleTasks(a.list), ","); got != "closed" {
		t.Errorf("expected only the done task within the focus, got %q", got)
	}
}
