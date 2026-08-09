package task

import "time"

type Priority int

// Priority levels. There is no separate "low": the scale is none, "!" and
// "!!", so none serves as the bottom of it.
const (
	PriorityNone Priority = iota
	PriorityMedium
	PriorityHigh
)

type Status int

const (
	StatusTodo Status = iota
	StatusInProgress
	StatusDone
)

// BlockLine is one indented line beneath a task: either a subtask or a note.
// Blocks are one level deep -- anything more deeply indented in the source is
// flattened into this level on save.
type BlockLine struct {
	// Subtask is set when the line is a checkbox item; otherwise the line is
	// prose and Note holds its text.
	Subtask *Task
	Note    string
}

// Text returns what the line reads as, whichever kind it is.
func (b BlockLine) Text() string {
	if b.Subtask != nil {
		return b.Subtask.Title
	}
	return b.Note
}

type Task struct {
	Title       string
	Status      Status
	Priority    Priority
	Tags        []string
	Contexts    []string
	DueDate     *time.Time
	CreatedDate time.Time
	DoneDate    *time.Time
	Line        int
	RawLine     string
	// Block holds the indented lines that belong to this task.
	Block []BlockLine
}
