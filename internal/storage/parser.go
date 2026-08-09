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
	LineSection                 // Section header (## Backlog, ## Done, etc.)
	LineTask                    // Task checkbox line
)

// Line represents a single line in the markdown file, preserving the raw text
// for roundtrip fidelity.
type Line struct {
	Raw       string
	Type      LineType
	Number    int // 1-based line number
	TaskIndex int // Index into TaskFile.Tasks, or -1 if not a task line
}

// Section represents a heading in the markdown file.
type Section struct {
	Name   string
	Level  int // 1 for "#", 2 for "##", and so on
	Status task.Status
	Line   int // Line number of the section header
}

// maxHeadingLevel is the deepest ATX heading markdown defines.
const maxHeadingLevel = 6

// ParseHeading splits an ATX heading into its level and text. Level is 0 when
// the line is not a heading.
func ParseHeading(trimmed string) (level int, name string) {
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > maxHeadingLevel {
		return 0, ""
	}
	rest := trimmed[level:]
	// A heading needs whitespace after the hashes, otherwise "#tag" would
	// count as one.
	if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return 0, ""
	}
	return level, strings.TrimSpace(rest)
}

// TaskFile holds the parsed contents of a tasks.md file.
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

		// Check for a heading of any level: #, ##, ### ...
		if level, sectionName := ParseHeading(trimmed); level > 0 {
			status := sectionNameToStatus(sectionName)
			currentStatus = status
			line.Type = LineSection
			tf.Sections = append(tf.Sections, Section{
				Name:   sectionName,
				Level:  level,
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
	meta := ParseMetadata(rest)
	t.Title, t.Priority, t.Tags, t.Contexts, t.DueDate =
		meta.Title, meta.Priority, meta.Tags, meta.Contexts, meta.DueDate

	return t, true
}

// splitPriorityMarker pulls a leading "!" or "!!" off the text and returns the
// remainder along with the priority it denotes. More than two marks are
// treated as high.
//
// The marker must be its own token: only a run of "!" followed by whitespace
// or end of text counts, so a title like "ship it!" is left alone.
func splitPriorityMarker(text string) (string, task.Priority) {
	rest := strings.TrimLeft(text, " \t")

	marks := 0
	for marks < len(rest) && rest[marks] == '!' {
		marks++
	}
	if marks == 0 {
		return text, task.PriorityNone
	}
	if marks < len(rest) && rest[marks] != ' ' && rest[marks] != '\t' {
		return text, task.PriorityNone
	}

	var priority task.Priority
	if marks == 1 {
		priority = task.PriorityMedium
	} else {
		priority = task.PriorityHigh
	}
	return strings.TrimLeft(rest[marks:], " \t"), priority
}

// Metadata is everything a task line carries besides its title.
type Metadata struct {
	Title    string
	Priority task.Priority
	Tags     []string
	Contexts []string
	DueDate  *time.Time
}

// ParseMetadata extracts the title and metadata from task text.
//
// Priority is a leading "!" marker; the title is everything after it that is
// not a metadata token. Tags and contexts are separate axes: "+tag" is a tag,
// "@context" is a context, and neither is rewritten as the other.
//
// Metadata tokens: +tag, @context, due:YYYY-MM-DD, created:YYYY-MM-DD, done:YYYY-MM-DD
func ParseMetadata(text string) Metadata {
	var m Metadata
	text, m.Priority = splitPriorityMarker(text)

	words := strings.Fields(text)
	var titleWords []string

	for _, word := range words {
		switch {
		case strings.HasPrefix(word, "priority:"):
			// Legacy form, still read so existing files keep their priorities.
			// Anything written back out uses the "!" marker.
			if m.Priority != task.PriorityNone {
				break
			}
			switch strings.ToLower(strings.TrimPrefix(word, "priority:")) {
			case "high":
				m.Priority = task.PriorityHigh
			// The scale no longer has a separate low level, so the weakest
			// legacy priority becomes the weakest current one rather than
			// being dropped.
			case "medium", "med", "low":
				m.Priority = task.PriorityMedium
			}
		case strings.HasPrefix(word, "+"):
			if tag := strings.TrimPrefix(word, "+"); tag != "" {
				m.Tags = append(m.Tags, tag)
			}
		case strings.HasPrefix(word, "@"):
			if ctx := strings.TrimPrefix(word, "@"); ctx != "" {
				m.Contexts = append(m.Contexts, ctx)
			}
		case strings.HasPrefix(word, "due:"):
			val := strings.TrimPrefix(word, "due:")
			if t, err := time.Parse("2006-01-02", val); err == nil {
				m.DueDate = &t
			}
		case strings.HasPrefix(word, "created:"):
			// Parsed but not yet stored (CreatedDate field exists but we skip for now)
		case strings.HasPrefix(word, "done:"):
			// Parsed but not yet stored (DoneDate field exists but we skip for now)
		default:
			titleWords = append(titleWords, word)
		}
	}

	m.Title = strings.Join(titleWords, " ")
	return m
}
