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

	// Title
	sb.WriteString(t.Title)

	// Priority
	switch t.Priority {
	case task.PriorityHigh:
		sb.WriteString(" priority:high")
	case task.PriorityMedium:
		sb.WriteString(" priority:medium")
	case task.PriorityLow:
		sb.WriteString(" priority:low")
	}

	// Tags
	for _, tag := range t.Tags {
		fmt.Fprintf(&sb, " +%s", tag)
	}

	// Due date
	if t.DueDate != nil {
		fmt.Fprintf(&sb, " due:%s", t.DueDate.Format("2006-01-02"))
	}

	return sb.String()
}
