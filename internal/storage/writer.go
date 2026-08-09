package storage

import (
	"fmt"
	"strings"

	"github.com/svellingen/md-taco/internal/task"
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

// formatTask renders a task and its block: the checkbox line, then each block
// line indented by two spaces. Whatever depth the source used, the block is
// written back flat at one level.
func formatTask(t task.Task) string {
	var sb strings.Builder
	sb.WriteString(formatTaskLine(t))

	for _, b := range t.Block {
		sb.WriteString("\n  ")
		if b.Subtask != nil {
			sb.WriteString(formatTaskLine(*b.Subtask))
			continue
		}
		sb.WriteString(b.Note)
	}
	return sb.String()
}

// TaskText renders everything a task line carries after its checkbox: the
// priority marker, title and metadata. Turning a subtask into a note keeps
// this much of it.
func TaskText(t task.Task) string {
	var sb strings.Builder

	sb.WriteString(PriorityMarker(t.Priority))
	if t.Priority != task.PriorityNone {
		sb.WriteByte(' ')
	}
	sb.WriteString(t.Title)

	for _, tag := range t.Tags {
		fmt.Fprintf(&sb, " +%s", tag)
	}
	for _, ctx := range t.Contexts {
		fmt.Fprintf(&sb, " @%s", ctx)
	}
	if t.DueDate != nil {
		fmt.Fprintf(&sb, " due:%s", t.DueDate.Format("2006-01-02"))
	}
	return sb.String()
}

// formatTaskLine renders just the checkbox line for a task.
func formatTaskLine(t task.Task) string {
	var sb strings.Builder

	switch t.Status {
	case task.StatusTodo:
		sb.WriteString("- [ ] ")
	case task.StatusInProgress:
		sb.WriteString("- [-] ")
	case task.StatusDone:
		sb.WriteString("- [x] ")
	}
	sb.WriteString(TaskText(t))

	return sb.String()
}
