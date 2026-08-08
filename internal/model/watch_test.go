package model

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// waitTimeout runs w.wait in the background and reports whether a change was
// seen before the deadline.
func waitTimeout(t *testing.T, w *fileWatcher, d time.Duration) bool {
	t.Helper()
	seen := make(chan bool, 1)
	go func() { seen <- w.wait() }()
	select {
	case ok := <-seen:
		return ok
	case <-time.After(d):
		return false
	}
}

func TestWatcherSeesDirectWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(path, []byte("## Backlog\n"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newFileWatcher(path)
	if err != nil {
		t.Skipf("filesystem watching unavailable: %v", err)
	}
	t.Cleanup(w.close)

	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(path, []byte("## Backlog\n\n- [ ] New\n"), 0644)
	}()

	if !waitTimeout(t, w, 3*time.Second) {
		t.Error("expected the write to be reported")
	}
}

// Editors that save atomically write a temp file and rename it over the
// original. A watch on the file itself would go stale after the first such
// save, so this must keep working across several.
func TestWatcherSurvivesAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(path, []byte("## Backlog\n"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newFileWatcher(path)
	if err != nil {
		t.Skipf("filesystem watching unavailable: %v", err)
	}
	t.Cleanup(w.close)

	for i := range 3 {
		tmp := filepath.Join(dir, ".tasks.md.tmp")
		if err := os.WriteFile(tmp, []byte("## Backlog\n\n- [ ] Edit\n"), 0644); err != nil {
			t.Fatal(err)
		}
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.Rename(tmp, path)
		}()

		if !waitTimeout(t, w, 3*time.Second) {
			t.Fatalf("atomic replace %d was not reported", i+1)
		}
	}
}

func TestWatcherIgnoresSiblingFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(path, []byte("## Backlog\n"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newFileWatcher(path)
	if err != nil {
		t.Skipf("filesystem watching unavailable: %v", err)
	}
	t.Cleanup(w.close)

	// The watch is on the directory, so an unrelated file in it must not
	// register as a change to the task file.
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("hi\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if waitTimeout(t, w, 500*time.Millisecond) {
		t.Error("a sibling file should not count as a change")
	}
}

func TestWatcherSeesRemoveAndRecreate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(path, []byte("## Backlog\n"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newFileWatcher(path)
	if err != nil {
		t.Skipf("filesystem watching unavailable: %v", err)
	}
	t.Cleanup(w.close)

	go func() {
		time.Sleep(50 * time.Millisecond)
		os.Remove(path)
	}()
	if !waitTimeout(t, w, 3*time.Second) {
		t.Fatal("expected the removal to be reported")
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(path, []byte("## Backlog\n\n- [ ] Back\n"), 0644)
	}()
	if !waitTimeout(t, w, 3*time.Second) {
		t.Error("expected the recreate to be reported")
	}
}

// A burst of writes for one logical save should surface as a single change.
func TestWatcherCoalescesBurst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(path, []byte("## Backlog\n"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newFileWatcher(path)
	if err != nil {
		t.Skipf("filesystem watching unavailable: %v", err)
	}
	t.Cleanup(w.close)

	go func() {
		time.Sleep(50 * time.Millisecond)
		for i := range 5 {
			os.WriteFile(path, []byte("## Backlog\n\n- [ ] Burst\n"), 0644)
			_ = i
			time.Sleep(5 * time.Millisecond)
		}
	}()

	if !waitTimeout(t, w, 3*time.Second) {
		t.Fatal("expected the burst to be reported")
	}
	// The whole burst is one save, so nothing should be left queued.
	if waitTimeout(t, w, 500*time.Millisecond) {
		t.Error("burst produced more than one change")
	}
}
