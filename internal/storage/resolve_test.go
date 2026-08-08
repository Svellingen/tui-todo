package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates a file with placeholder content, making parent dirs first.
func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("## Backlog\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePrefersTasksOverIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.md"))
	writeFile(t, filepath.Join(dir, "tasks.md"))

	got, err := Resolve(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(dir, "tasks.md"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveFallsBackToIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.md"))

	got, err := Resolve(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(dir, "index.md"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveWalksUpToAncestor(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tasks.md"))
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(root, "tasks.md"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A nearer index.md beats a further-away tasks.md: the whole candidate list is
// tried in one directory before moving to its parent.
func TestResolveNearerIndexBeatsAncestorTasks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tasks.md"))
	nested := filepath.Join(root, "sub")
	writeFile(t, filepath.Join(nested, "index.md"))

	got, err := Resolve(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(nested, "index.md"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveStopsAtNearestAncestor(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tasks.md"))
	mid := filepath.Join(root, "mid")
	writeFile(t, filepath.Join(mid, "tasks.md"))
	nested := filepath.Join(mid, "deep")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(mid, "tasks.md"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveIgnoresDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "tasks.md"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "index.md"))

	got, err := Resolve(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(dir, "index.md"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// $HOME is the last directory checked; anything above it is out of reach.
func TestResolveStopsAboveHome(t *testing.T) {
	above := t.TempDir()
	writeFile(t, filepath.Join(above, "tasks.md"))
	home := filepath.Join(above, "home")
	nested := filepath.Join(home, "projects", "app")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := resolveFrom(nested, home); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestResolveChecksHomeItself(t *testing.T) {
	above := t.TempDir()
	home := filepath.Join(above, "home")
	writeFile(t, filepath.Join(home, "tasks.md"))
	nested := filepath.Join(home, "projects", "app")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := resolveFrom(nested, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(home, "tasks.md"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A directory outside $HOME never crosses the boundary, so it walks to the root.
func TestResolveOutsideHomeWalksToRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tasks.md"))
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := resolveFrom(nested, filepath.Join(t.TempDir(), "elsewhere"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(root, "tasks.md"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveNotFoundReachesRoot(t *testing.T) {
	// Walking up from a temp dir eventually hits the filesystem root. Skip if
	// the host happens to have a task file in an ancestor of the temp dir.
	dir := t.TempDir()
	_, err := Resolve(dir)
	if err == nil {
		t.Skip("host filesystem has a task file above the temp dir")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestResolveOrDefaultFallsBackToCurrentDir(t *testing.T) {
	dir := t.TempDir()
	got, err := ResolveOrDefault(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "tasks.md")
	if got != want {
		// Only meaningful when nothing was found above the temp dir.
		if _, rerr := Resolve(dir); rerr == nil {
			t.Skip("host filesystem has a task file above the temp dir")
		}
		t.Errorf("got %q, want %q", got, want)
	}
}
