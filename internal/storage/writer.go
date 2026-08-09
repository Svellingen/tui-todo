package storage

import (
	"fmt"
	"strings"

	"github.com/macone/todo-cli/internal/task"
)

// Writer serializes a TaskFile back to markdown format.
type Writer struct{}

// NewWriter creates a new Writer.
func NewWriter() *Writer {
	return &Writer{}
}

// Write serializes a TaskFile to markdown string.
// Unmodified lines are passed through unchanged. Task lines are regenerated
// from the task data to reflect any changes.
func (w *Writer) Write(tf *TaskFile) string {
	if tf == nil || len(tf.Lines) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, line := range tf.Lines {
		if line.Type == LineTask && line.TaskIndex >= 0 && line.TaskIndex < len(tf.Tasks) {
			sb.WriteString(formatTask(tf.Tasks[line.TaskIndex]))
		} else {
			sb.WriteString(line.Raw)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// PriorityMarker returns the "!" run denoting a priority, or "" for none.
// This is the canonical on-disk form and is also what the CLI prints.
func PriorityMarker(p task.Priority) string {
	switch p {
	case task.PriorityMedium:
		return "!"
	case task.PriorityHigh:
		return "!!"
	default:
		return ""
	}
}

// formatTask renders a task as a markdown checkbox line.
func formatTask(t task.Task) string {
	var sb strings.Builder

	// Checkbox
	switch t.Status {
	case task.StatusTodo:
		sb.WriteString("- [ ] ")
	case task.StatusInProgress:
		sb.WriteString("- [-] ")
	case task.StatusDone:
		sb.WriteString("- [x] ")
	}

	// Priority marker, prefixed to the title
	sb.WriteString(PriorityMarker(t.Priority))
	if t.Priority != task.PriorityNone {
		sb.WriteByte(' ')
	}

	// Title
	sb.WriteString(t.Title)

	// Tags
	for _, tag := range t.Tags {
		fmt.Fprintf(&sb, " +%s", tag)
	}

	// Contexts keep their own sigil rather than being folded into tags.
	for _, ctx := range t.Contexts {
		fmt.Fprintf(&sb, " @%s", ctx)
	}

	// Due date
	if t.DueDate != nil {
		fmt.Fprintf(&sb, " due:%s", t.DueDate.Format("2006-01-02"))
	}

	return sb.String()
}
