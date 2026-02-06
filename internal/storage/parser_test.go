package storage

import (
	"testing"

	"github.com/macone/todo-cli/internal/task"
)

func TestParseEmptyFile(t *testing.T) {
	p := NewParser()
	result, err := p.Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(result.Tasks))
	}
}

func TestParseSingleTask(t *testing.T) {
	input := "## Backlog\n\n- [ ] Write tests\n"
	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}
	if result.Tasks[0].Title != "Write tests" {
		t.Errorf("expected title 'Write tests', got '%s'", result.Tasks[0].Title)
	}
	if result.Tasks[0].Status != task.StatusTodo {
		t.Errorf("expected StatusTodo, got %d", result.Tasks[0].Status)
	}
}

func TestParseAllStatuses(t *testing.T) {
	input := `## Backlog

- [ ] Todo item

## In Progress

- [-] Active item

## Done

- [x] Completed item
`
	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Tasks))
	}
	if result.Tasks[0].Status != task.StatusTodo {
		t.Errorf("task 0: expected StatusTodo")
	}
	if result.Tasks[1].Status != task.StatusInProgress {
		t.Errorf("task 1: expected StatusInProgress")
	}
	if result.Tasks[2].Status != task.StatusDone {
		t.Errorf("task 2: expected StatusDone")
	}
}

func TestParseMetadata(t *testing.T) {
	input := "## Backlog\n\n- [ ] Fix bug priority:high +backend @security due:2026-03-01\n"
	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tk := result.Tasks[0]
	if tk.Title != "Fix bug" {
		t.Errorf("expected title 'Fix bug', got '%s'", tk.Title)
	}
	if tk.Priority != task.PriorityHigh {
		t.Errorf("expected PriorityHigh, got %d", tk.Priority)
	}
	if len(tk.Tags) != 2 || tk.Tags[0] != "backend" || tk.Tags[1] != "security" {
		t.Errorf("expected tags [backend, security], got %v", tk.Tags)
	}
	if tk.DueDate == nil {
		t.Fatal("expected due date to be set")
	}
}

func TestParsePreservesNonTaskLines(t *testing.T) {
	input := "# My Project\n\nSome notes here.\n\n## Backlog\n\n- [ ] A task\n\n## Done\n\n- [x] Finished\n"
	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Lines) < 5 {
		t.Fatalf("expected preserved lines, got %d", len(result.Lines))
	}
}
