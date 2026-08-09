package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/svellingen/md-taco/internal/task"
)

func TestLoadMissingFile(t *testing.T) {
	s := NewStore("/nonexistent/path/tasks.md")
	tf, err := s.Load()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(tf.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tf.Tasks))
	}
}

func TestLoadExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")

	content := "## Backlog\n\n- [ ] Test task\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	s := NewStore(path)
	tf, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tf.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tf.Tasks))
	}
	if tf.Tasks[0].Title != "Test task" {
		t.Errorf("expected 'Test task', got %q", tf.Tasks[0].Title)
	}
}

func TestSaveCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")

	s := NewStore(path)
	tf := &TaskFile{
		Tasks: []task.Task{
			{Title: "New task", Status: task.StatusTodo},
		},
		Sections: []Section{
			{Name: "Backlog", Status: task.StatusTodo, Line: 1},
		},
		Lines: []Line{
			{Raw: "## Backlog", Type: LineSection, Number: 1, TaskIndex: -1},
			{Raw: "", Type: LineText, Number: 2, TaskIndex: -1},
			{Raw: "- [ ] New task", Type: LineTask, Number: 3, TaskIndex: 0},
		},
	}

	if err := s.Save(tf); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	expected := "## Backlog\n\n- [ ] New task\n"
	if string(data) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(data))
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")

	s := NewStore(path)

	// Save
	tf := &TaskFile{
		Tasks: []task.Task{
			{Title: "Task one", Status: task.StatusTodo, Priority: task.PriorityHigh},
			{Title: "Task two", Status: task.StatusDone},
		},
		Sections: []Section{
			{Name: "Backlog", Status: task.StatusTodo, Line: 1},
			{Name: "Done", Status: task.StatusDone, Line: 4},
		},
		Lines: []Line{
			{Raw: "## Backlog", Type: LineSection, Number: 1, TaskIndex: -1},
			{Raw: "", Type: LineText, Number: 2, TaskIndex: -1},
			{Raw: "- [ ] Task one priority:high", Type: LineTask, Number: 3, TaskIndex: 0},
			{Raw: "## Done", Type: LineSection, Number: 4, TaskIndex: -1},
			{Raw: "", Type: LineText, Number: 5, TaskIndex: -1},
			{Raw: "- [x] Task two", Type: LineTask, Number: 6, TaskIndex: 1},
		},
	}

	if err := s.Save(tf); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Load
	tf2, err := s.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(tf2.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tf2.Tasks))
	}
	if tf2.Tasks[0].Title != "Task one" {
		t.Errorf("task 0 title: expected 'Task one', got %q", tf2.Tasks[0].Title)
	}
	if tf2.Tasks[0].Priority != task.PriorityHigh {
		t.Errorf("task 0 priority: expected High, got %d", tf2.Tasks[0].Priority)
	}
	if tf2.Tasks[1].Status != task.StatusDone {
		t.Errorf("task 1 status: expected Done, got %d", tf2.Tasks[1].Status)
	}
}
