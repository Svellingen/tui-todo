package model

import (
	"slices"
	"testing"

	"github.com/macone/todo-cli/internal/storage"
)

func TestEditorInvocationAddsLineForKnownEditors(t *testing.T) {
	cases := []struct {
		editor []string
		line   int
		want   []string
	}{
		{[]string{"nvim"}, 12, []string{"nvim", "+12", "tasks.md"}},
		{[]string{"vim"}, 1, []string{"vim", "+1", "tasks.md"}},
		{[]string{"/usr/bin/nvim"}, 7, []string{"/usr/bin/nvim", "+7", "tasks.md"}},
		{[]string{"nano"}, 3, []string{"nano", "+3", "tasks.md"}},
		// Arguments already on $EDITOR are preserved, line goes after them.
		{[]string{"vim", "-R"}, 9, []string{"vim", "-R", "+9", "tasks.md"}},
	}

	for _, c := range cases {
		got := editorInvocation(c.editor, "tasks.md", c.line)
		if !slices.Equal(got, c.want) {
			t.Errorf("%v: got %v, want %v", c.editor, got, c.want)
		}
	}
}

// An unrecognised editor would read "+12" as a second file to open, so it is
// left off.
func TestEditorInvocationOmitsLineForUnknownEditors(t *testing.T) {
	for _, editor := range [][]string{{"code"}, {"hx"}, {"my-editor"}, {"code", "-w"}} {
		got := editorInvocation(editor, "tasks.md", 12)
		want := append(slices.Clone(editor), "tasks.md")
		if !slices.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", editor, got, want)
		}
	}
}

func TestEditorInvocationOmitsLineWhenUnknownPosition(t *testing.T) {
	got := editorInvocation([]string{"nvim"}, "tasks.md", 0)
	if !slices.Equal(got, []string{"nvim", "tasks.md"}) {
		t.Errorf("got %v", got)
	}
}

// The editor opens on the file line of whatever the cursor is on.
func TestSelectedFileLineTracksCursor(t *testing.T) {
	content := "# Title\n" + // 1
		"- [ ] first\n" + // 2
		"\n" + // 3
		"## Alpha\n" + // 4
		"- [ ] second\n" + // 5
		"- [ ] third\n" // 6

	tf, err := storage.NewParser().Parse(content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	a := NewApp(storage.NewStore(t.TempDir() + "/tasks.md"))
	a.taskFile = tf
	a.list = NewTaskListModel(tf)
	a.width, a.height = 80, 24
	a.updateListSize()

	if got := a.selectedFileLine(); got != 2 {
		t.Errorf("first task: expected line 2, got %d", got)
	}

	a.list.MoveDown()
	if got := a.selectedFileLine(); got != 5 {
		t.Errorf("second task: expected line 5, got %d", got)
	}

	a.list.JumpPrevSection()
	if got := a.selectedFileLine(); got != 4 {
		t.Errorf("Alpha heading: expected line 4, got %d", got)
	}
}
