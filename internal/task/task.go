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

type Task struct {
	Title       string
	Status      Status
	Priority    Priority
	Tags        []string
	DueDate     *time.Time
	CreatedDate time.Time
	DoneDate    *time.Time
	Line        int
	RawLine     string
}
