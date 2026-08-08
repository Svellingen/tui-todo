package storage

import (
	"testing"
	"time"

	"github.com/macone/todo-cli/internal/task"
)

func TestWriteEmptyTaskFile(t *testing.T) {
	w := NewWriter()
	tf := &TaskFile{}
	result := w.Write(tf)
	if result != "" {
		t.Errorf("expected empty string for empty task file, got %q", result)
	}
}

func TestWriteTasksGroupedBySections(t *testing.T) {
	w := NewWriter()
	tf := &TaskFile{
		Tasks: []task.Task{
			{Title: "Todo item", Status: task.StatusTodo},
			{Title: "Active item", Status: task.StatusInProgress},
			{Title: "Done item", Status: task.StatusDone},
		},
		Sections: []Section{
			{Name: "Backlog", Status: task.StatusTodo, Line: 1},
			{Name: "In Progress", Status: task.StatusInProgress, Line: 5},
			{Name: "Done", Status: task.StatusDone, Line: 9},
		},
		Lines: []Line{
			{Raw: "## Backlog", Type: LineSection, Number: 1, TaskIndex: -1},
			{Raw: "", Type: LineText, Number: 2, TaskIndex: -1},
			{Raw: "- [ ] Todo item", Type: LineTask, Number: 3, TaskIndex: 0},
			{Raw: "", Type: LineText, Number: 4, TaskIndex: -1},
			{Raw: "## In Progress", Type: LineSection, Number: 5, TaskIndex: -1},
			{Raw: "", Type: LineText, Number: 6, TaskIndex: -1},
			{Raw: "- [-] Active item", Type: LineTask, Number: 7, TaskIndex: 1},
			{Raw: "", Type: LineText, Number: 8, TaskIndex: -1},
			{Raw: "## Done", Type: LineSection, Number: 9, TaskIndex: -1},
			{Raw: "", Type: LineText, Number: 10, TaskIndex: -1},
			{Raw: "- [x] Done item", Type: LineTask, Number: 11, TaskIndex: 2},
		},
	}

	result := w.Write(tf)

	expected := "## Backlog\n\n- [ ] Todo item\n\n## In Progress\n\n- [-] Active item\n\n## Done\n\n- [x] Done item\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestWriteMetadata(t *testing.T) {
	due := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	w := NewWriter()
	tf := &TaskFile{
		Tasks: []task.Task{
			{
				Title:    "Fix bug",
				Status:   task.StatusTodo,
				Priority: task.PriorityHigh,
				Tags:     []string{"backend", "security"},
				DueDate:  &due,
			},
		},
		Sections: []Section{
			{Name: "Backlog", Status: task.StatusTodo, Line: 1},
		},
		Lines: []Line{
			{Raw: "## Backlog", Type: LineSection, Number: 1, TaskIndex: -1},
			{Raw: "", Type: LineText, Number: 2, TaskIndex: -1},
			{Raw: "- [ ] !! Fix bug +backend +security due:2026-03-01", Type: LineTask, Number: 3, TaskIndex: 0},
		},
	}

	result := w.Write(tf)
	expected := "## Backlog\n\n- [ ] !! Fix bug +backend +security due:2026-03-01\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestWritePriorityMarkers(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] no pri\n" +
		"- [ ] ! medium priority task\n" +
		"- [ ] !! high priority task\n"

	p := NewParser()
	tf, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := NewWriter().Write(tf); got != input {
		t.Errorf("expected:\n%s\ngot:\n%s", input, got)
	}
}

// The legacy priority:X token is read but never written back; saving a file
// migrates it to the "!" marker.
func TestWriteMigratesLegacyPriority(t *testing.T) {
	p := NewParser()
	tf, err := p.Parse("## Backlog\n- [ ] Old style priority:medium +backend\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "## Backlog\n- [ ] ! Old style +backend\n"
	if got := NewWriter().Write(tf); got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestWriteRoundtrip(t *testing.T) {
	input := "# My Project\n\n## Backlog\n\n- [ ] Write tests\n- [ ] !! Fix bug +backend due:2026-03-01\n\n## In Progress\n\n- [-] Design TUI\n\n## Done\n\n- [x] Init project\n"

	p := NewParser()
	w := NewWriter()

	tf1, err := p.Parse(input)
	if err != nil {
		t.Fatalf("parse 1 failed: %v", err)
	}

	output := w.Write(tf1)

	tf2, err := p.Parse(output)
	if err != nil {
		t.Fatalf("parse 2 failed: %v", err)
	}

	// Verify same number of tasks
	if len(tf1.Tasks) != len(tf2.Tasks) {
		t.Fatalf("roundtrip task count mismatch: %d vs %d", len(tf1.Tasks), len(tf2.Tasks))
	}

	// Verify each task matches
	for i := range tf1.Tasks {
		t1 := tf1.Tasks[i]
		t2 := tf2.Tasks[i]
		if t1.Title != t2.Title {
			t.Errorf("task %d title mismatch: %q vs %q", i, t1.Title, t2.Title)
		}
		if t1.Status != t2.Status {
			t.Errorf("task %d status mismatch: %d vs %d", i, t1.Status, t2.Status)
		}
		if t1.Priority != t2.Priority {
			t.Errorf("task %d priority mismatch: %d vs %d", i, t1.Priority, t2.Priority)
		}
	}
}

func TestWritePreservesNonTaskLines(t *testing.T) {
	input := "# My Project\n\nSome notes here.\n\n## Backlog\n\n- [ ] A task\n"

	p := NewParser()
	w := NewWriter()

	tf, err := p.Parse(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	result := w.Write(tf)

	if result != input {
		t.Errorf("expected non-task lines preserved.\nexpected:\n%s\ngot:\n%s", input, result)
	}
}
