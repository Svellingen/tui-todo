// Package storage handles reading and writing task data to markdown files.
package storage

import (
	"strings"
	"time"

	"github.com/macone/todo-cli/internal/task"
)

// LineType identifies the kind of line in the markdown file.
type LineType int

const (
	LineText    LineType = iota // Non-task text (comments, blank lines, etc.)
	LineSection                // Section header (## Backlog, ## Done, etc.)
	LineTask                   // Task checkbox line
)

// Line represents a single line in the markdown file, preserving the raw text
// for roundtrip fidelity.
type Line struct {
	Raw       string
	Type      LineType
	Number    int // 1-based line number
	TaskIndex int // Index into TaskFile.Tasks, or -1 if not a task line
}

// Section represents a status section in the markdown file.
type Section struct {
	Name   string
	Status task.Status
	Line   int // Line number of the section header
}

// TaskFile holds the parsed contents of a todo.md file.
type TaskFile struct {
	Tasks    []task.Task
	Lines    []Line
	Sections []Section
}

// Parser reads markdown content and extracts tasks.
type Parser struct{}

// NewParser creates a new Parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses markdown content and returns a TaskFile.
func (p *Parser) Parse(content string) (*TaskFile, error) {
	tf := &TaskFile{}

	if content == "" {
		return tf, nil
	}

	rawLines := strings.Split(content, "\n")
	// Remove trailing empty string from final newline
	if len(rawLines) > 0 && rawLines[len(rawLines)-1] == "" {
		rawLines = rawLines[:len(rawLines)-1]
	}

	currentStatus := task.StatusTodo // default status if no section header seen

	for i, raw := range rawLines {
		lineNum := i + 1
		line := Line{
			Raw:       raw,
			Number:    lineNum,
			TaskIndex: -1,
		}

		trimmed := strings.TrimSpace(raw)

		// Check for section header: ## SectionName
		if strings.HasPrefix(trimmed, "## ") {
			sectionName := strings.TrimPrefix(trimmed, "## ")
			status := sectionNameToStatus(sectionName)
			currentStatus = status
			line.Type = LineSection
			tf.Sections = append(tf.Sections, Section{
				Name:   sectionName,
				Status: status,
				Line:   lineNum,
			})
			tf.Lines = append(tf.Lines, line)
			continue
		}

		// Check for task checkbox line
		if t, ok := parseTaskLine(trimmed, currentStatus, lineNum, raw); ok {
			line.Type = LineTask
			line.TaskIndex = len(tf.Tasks)
			tf.Tasks = append(tf.Tasks, t)
			tf.Lines = append(tf.Lines, line)
			continue
		}

		// Regular text line
		line.Type = LineText
		tf.Lines = append(tf.Lines, line)
	}

	return tf, nil
}

// sectionNameToStatus maps section header names to task statuses.
func sectionNameToStatus(name string) task.Status {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "in progress":
		return task.StatusInProgress
	case "done":
		return task.StatusDone
	default:
		return task.StatusTodo
	}
}

// parseTaskLine attempts to parse a line as a task checkbox.
// Returns the task and true if successful, zero value and false otherwise.
func parseTaskLine(trimmed string, sectionStatus task.Status, lineNum int, raw string) (task.Task, bool) {
	var status task.Status
	var rest string

	switch {
	case strings.HasPrefix(trimmed, "- [ ] "):
		status = task.StatusTodo
		rest = strings.TrimPrefix(trimmed, "- [ ] ")
	case strings.HasPrefix(trimmed, "- [-] "):
		status = task.StatusInProgress
		rest = strings.TrimPrefix(trimmed, "- [-] ")
	case strings.HasPrefix(trimmed, "- [x] "):
		status = task.StatusDone
		rest = strings.TrimPrefix(trimmed, "- [x] ")
	default:
		return task.Task{}, false
	}

	// If checkbox indicates a status, use it; otherwise fall back to section status
	if status == task.StatusTodo {
		// Only override with section status if the section implies something other than todo
		// Actually, the checkbox is the primary indicator. Section is just grouping.
		// The test expects checkbox to determine status, so keep checkbox-based status.
		_ = sectionStatus
	}

	t := task.Task{
		Status:  status,
		Line:    lineNum,
		RawLine: raw,
	}

	// Parse metadata from rest of line
	t.Title, t.Priority, t.Tags, t.DueDate = parseMetadata(rest)

	return t, true
}

// parseMetadata extracts title, priority, tags, and due date from task text.
// The title is everything before the first metadata token.
// Metadata tokens: priority:X, +tag, @context, due:YYYY-MM-DD, created:YYYY-MM-DD, done:YYYY-MM-DD
func parseMetadata(text string) (title string, priority task.Priority, tags []string, dueDate *time.Time) {
	words := strings.Fields(text)
	var titleWords []string

	for _, word := range words {
		switch {
		case strings.HasPrefix(word, "priority:"):
			val := strings.TrimPrefix(word, "priority:")
			switch strings.ToLower(val) {
			case "high":
				priority = task.PriorityHigh
			case "medium", "med":
				priority = task.PriorityMedium
			case "low":
				priority = task.PriorityLow
			}
		case strings.HasPrefix(word, "+"):
			tag := strings.TrimPrefix(word, "+")
			if tag != "" {
				tags = append(tags, tag)
			}
		case strings.HasPrefix(word, "@"):
			ctx := strings.TrimPrefix(word, "@")
			if ctx != "" {
				tags = append(tags, ctx)
			}
		case strings.HasPrefix(word, "due:"):
			val := strings.TrimPrefix(word, "due:")
			if t, err := time.Parse("2006-01-02", val); err == nil {
				dueDate = &t
			}
		case strings.HasPrefix(word, "created:"):
			// Parsed but not yet stored (CreatedDate field exists but we skip for now)
		case strings.HasPrefix(word, "done:"):
			// Parsed but not yet stored (DoneDate field exists but we skip for now)
		default:
			titleWords = append(titleWords, word)
		}
	}

	title = strings.Join(titleWords, " ")
	return
}
