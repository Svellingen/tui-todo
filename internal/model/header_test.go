package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeaderPathCollapsesHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home directory available")
	}

	path := filepath.Join(home, "work", "proj", "tasks.md")
	want := filepath.Join("~", "work", "proj", "tasks.md")
	if got := headerPath(path, 80); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHeaderPathLeavesPathsOutsideHomeAlone(t *testing.T) {
	path := filepath.Join(string(filepath.Separator), "srv", "proj", "tasks.md")
	if got := headerPath(path, 80); got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

// When the path does not fit it is cut from the left, so the filename -- the
// part that identifies which file you are looking at -- always survives.
func TestHeaderPathElidesFromTheLeft(t *testing.T) {
	path := "/a/very/deeply/nested/directory/structure/tasks.md"

	got := headerPath(path, 20)
	if len([]rune(got)) > 20 {
		t.Errorf("expected at most 20 cells, got %d: %q", len([]rune(got)), got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Errorf("expected a leading ellipsis, got %q", got)
	}
	if !strings.HasSuffix(got, "tasks.md") {
		t.Errorf("expected the filename to survive, got %q", got)
	}
}

func TestHeaderPathHandlesNoRoom(t *testing.T) {
	for _, width := range []int{1, 0, -5} {
		got := headerPath("/some/where/tasks.md", width)
		if width <= 0 {
			// No budget known: return the path rather than mangling it.
			if got != "/some/where/tasks.md" {
				t.Errorf("width %d: got %q", width, got)
			}
			continue
		}
		if got != "…" {
			t.Errorf("width %d: got %q, want %q", width, got, "…")
		}
	}
}
