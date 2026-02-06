package task

import "time"

type Priority int

const (
	PriorityNone Priority = iota
	PriorityLow
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
