package model

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const pickerFixture = "# Root\n" +
	"- [ ] root task\n" +
	"## Alpha\n" +
	"- [ ] alpha task\n" +
	"### Alpha sub\n" +
	"#### Alpha deep\n" +
	"- [ ] deep task\n" +
	"## Beta\n" +
	"- [ ] beta task\n"

func newPicker(t *testing.T, action pickerAction) (App, headerPicker) {
	t.Helper()
	a := newDeleteApp(t, pickerFixture)
	return a, newHeaderPicker(a.list.HeadingEntries(), action, a.list.SelectedTaskIndex())
}

func typeInto(p *headerPicker, s string) {
	for _, r := range s {
		p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestPickerListsEveryHeading(t *testing.T) {
	_, p := newPicker(t, pickerJump)

	want := []string{"Root", "Alpha", "Alpha sub", "Alpha deep", "Beta"}
	if len(p.matches) != len(want) {
		t.Fatalf("expected %d headings, got %d", len(want), len(p.matches))
	}
	for i, w := range want {
		if got := p.entries[p.matches[i]].name; got != w {
			t.Errorf("entry %d: got %q, want %q", i, got, w)
		}
	}
}

func TestPickerFiltersCaseInsensitively(t *testing.T) {
	_, p := newPicker(t, pickerJump)
	typeInto(&p, "ALPHA")

	if len(p.matches) != 3 {
		t.Fatalf("expected 3 matches for \"ALPHA\", got %d", len(p.matches))
	}
	for _, m := range p.matches {
		if !strings.Contains(strings.ToLower(p.entries[m].name), "alpha") {
			t.Errorf("unexpected match %q", p.entries[m].name)
		}
	}
}

// Narrowing must not leave the cursor pointing past the end.
func TestPickerKeepsCursorInRangeWhileFiltering(t *testing.T) {
	_, p := newPicker(t, pickerJump)
	p.move(4) // last entry

	typeInto(&p, "beta")
	if p.cursor >= len(p.matches) {
		t.Fatalf("cursor %d out of range for %d matches", p.cursor, len(p.matches))
	}
	if e, ok := p.selected(); !ok || e.name != "Beta" {
		t.Errorf("expected Beta selected, got %+v", e)
	}
}

func TestPickerNavigationKeys(t *testing.T) {
	_, p := newPicker(t, pickerJump)

	p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\n'}}) // not a key we bind
	if p.cursor != 0 {
		t.Fatalf("expected to start at 0, got %d", p.cursor)
	}

	// ctrl+j and ctrl+k arrive by name.
	p.handleKey(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if p.cursor != 1 {
		t.Errorf("ctrl+j: expected 1, got %d", p.cursor)
	}
	p.handleKey(tea.KeyMsg{Type: tea.KeyCtrlK})
	if p.cursor != 0 {
		t.Errorf("ctrl+k: expected 0, got %d", p.cursor)
	}
	// Stops at the top rather than wrapping.
	p.handleKey(tea.KeyMsg{Type: tea.KeyCtrlK})
	if p.cursor != 0 {
		t.Errorf("expected to stay at 0, got %d", p.cursor)
	}
}

func TestPickerEnterAndEsc(t *testing.T) {
	_, p := newPicker(t, pickerJump)

	if got := p.handleKey(tea.KeyMsg{Type: tea.KeyEnter}); got != pickerChosen {
		t.Errorf("expected Enter to choose, got %v", got)
	}
	if got := p.handleKey(tea.KeyMsg{Type: tea.KeyEsc}); got != pickerCancelled {
		t.Errorf("expected Esc to cancel, got %v", got)
	}

	// With nothing matching, Enter does nothing.
	typeInto(&p, "zzzz")
	if got := p.handleKey(tea.KeyMsg{Type: tea.KeyEnter}); got != pickerContinue {
		t.Errorf("expected Enter with no match to be inert, got %v", got)
	}
}

// Depths collapse skipped heading levels, so "##" followed by "####" reads as
// one step in rather than two.
func TestDepthsCollapseSkippedLevels(t *testing.T) {
	got := depths([]int{1, 2, 3, 4, 2})
	want := []int{0, 1, 2, 3, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}

	got = depths([]int{2, 4, 4, 2})
	want = []int{0, 1, 1, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("skipped levels: got %v, want %v", got, want)
		}
	}
}

// The last child of a branch gets the corner glyph; earlier ones get a tee,
// and ancestors with more children keep a vertical guide.
func TestTreePrefixGuides(t *testing.T) {
	d := depths([]int{1, 2, 3, 2})

	if got := treePrefix(d, 0); got != "" {
		t.Errorf("root: got %q", got)
	}
	if got := treePrefix(d, 1); got != "├─ " {
		t.Errorf("first child with a sibling later: got %q", got)
	}
	if got := treePrefix(d, 2); got != "│  ╰─ " {
		t.Errorf("last grandchild under a continuing parent: got %q", got)
	}
	if got := treePrefix(d, 3); got != "╰─ " {
		t.Errorf("last child: got %q", got)
	}
}

// The popup draws its title in the top border and shows a match count.
func TestPickerViewShowsTitleAndCount(t *testing.T) {
	_, jump := newPicker(t, pickerJump)
	plain := ansi.Strip(jump.View())
	if !strings.Contains(plain, "Jump to heading") {
		t.Errorf("expected the jump title, got:\n%s", plain)
	}
	if !strings.Contains(plain, "5/5") {
		t.Errorf("expected a 5/5 count, got:\n%s", plain)
	}

	_, move := newPicker(t, pickerMove)
	if !strings.Contains(ansi.Strip(move.View()), "Move task to heading") {
		t.Error("expected the move title")
	}
}
