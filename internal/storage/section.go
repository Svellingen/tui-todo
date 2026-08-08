package storage

import (
	"strings"

	"github.com/macone/todo-cli/internal/task"
)

// lineLevel returns the heading level of a line, or 0 when it is not a heading.
func (tf *TaskFile) lineLevel(i int) int {
	if tf.Lines[i].Type != LineSection {
		return 0
	}
	level, _ := ParseHeading(strings.TrimSpace(tf.Lines[i].Raw))
	return level
}

// SectionSpan describes what a heading owns: everything between it and the
// next heading of the same or shallower level.
type SectionSpan struct {
	Name     string
	Level    int
	Start    int // index into Lines of the heading itself
	End      int // exclusive
	Tasks    int // tasks nested anywhere beneath it
	Headings int // sub-headings nested beneath it
}

// Span measures the subtree owned by the heading at lineIdx. ok is false when
// lineIdx is not a heading.
func (tf *TaskFile) Span(lineIdx int) (span SectionSpan, ok bool) {
	if lineIdx < 0 || lineIdx >= len(tf.Lines) {
		return span, false
	}
	level := tf.lineLevel(lineIdx)
	if level == 0 {
		return span, false
	}

	_, name := ParseHeading(strings.TrimSpace(tf.Lines[lineIdx].Raw))
	span = SectionSpan{Name: name, Level: level, Start: lineIdx, End: len(tf.Lines)}

	for i := lineIdx + 1; i < len(tf.Lines); i++ {
		if l := tf.lineLevel(i); l > 0 {
			// A heading at the same or shallower level ends the subtree.
			if l <= level {
				span.End = i
				break
			}
			span.Headings++
			continue
		}
		if tf.Lines[i].Type == LineTask {
			span.Tasks++
		}
	}
	return span, true
}

// DeleteSection removes a heading together with everything nested under it:
// sub-headings, their tasks, and any prose in between.
func (tf *TaskFile) DeleteSection(lineIdx int) bool {
	span, ok := tf.Span(lineIdx)
	if !ok {
		return false
	}

	removed := make(map[int]bool)
	for i := span.Start; i < span.End; i++ {
		if tf.Lines[i].Type == LineTask {
			removed[tf.Lines[i].TaskIndex] = true
		}
	}

	tf.Lines = append(tf.Lines[:span.Start], tf.Lines[span.End:]...)

	if len(removed) > 0 {
		// Renumber the surviving tasks and repoint the lines at them.
		shift := make([]int, len(tf.Tasks))
		kept := make([]task.Task, 0, len(tf.Tasks)-len(removed))
		for i, t := range tf.Tasks {
			if removed[i] {
				shift[i] = -1
				continue
			}
			shift[i] = len(kept)
			kept = append(kept, t)
		}
		tf.Tasks = kept
		for i := range tf.Lines {
			if tf.Lines[i].Type == LineTask {
				tf.Lines[i].TaskIndex = shift[tf.Lines[i].TaskIndex]
			}
		}
	}

	tf.rebuildSections()
	return true
}

// InsertTaskUnder adds a task directly beneath the heading at lineIdx, after
// any blank lines that immediately follow it. It returns the new task's index,
// or -1 when lineIdx is not a heading.
func (tf *TaskFile) InsertTaskUnder(lineIdx int, t task.Task) int {
	if lineIdx < 0 || lineIdx >= len(tf.Lines) || tf.Lines[lineIdx].Type != LineSection {
		return -1
	}

	insertAt := lineIdx + 1
	for insertAt < len(tf.Lines) &&
		tf.Lines[insertAt].Type == LineText &&
		strings.TrimSpace(tf.Lines[insertAt].Raw) == "" {
		insertAt++
	}
	return tf.insertTaskAt(insertAt, t)
}

// InsertTaskAfter adds a task on the line following lineIdx. Adding from a
// task therefore keeps the new one in the same section, next to its sibling,
// rather than sending it to the top of the file.
func (tf *TaskFile) InsertTaskAfter(lineIdx int, t task.Task) int {
	if lineIdx < 0 || lineIdx >= len(tf.Lines) {
		return -1
	}
	return tf.insertTaskAt(lineIdx+1, t)
}

// MoveTaskToSection moves a task into the heading adjacent to its own: the
// next one in the file when dir is +1, the previous when -1. The task is
// placed directly under that heading, and the sort then settles it within its
// rank group there.
//
// It returns the target heading's name and whether anything moved -- there is
// nowhere to go from the first or last section.
func (tf *TaskFile) MoveTaskToSection(taskIndex, dir int) (string, bool) {
	if dir != 1 && dir != -1 {
		return "", false
	}

	from := -1
	for i, l := range tf.Lines {
		if l.Type == LineTask && l.TaskIndex == taskIndex {
			from = i
			break
		}
	}
	if from < 0 {
		return "", false
	}

	// The heading the task currently sits under, if any.
	owner := -1
	for i := from - 1; i >= 0; i-- {
		if tf.Lines[i].Type == LineSection {
			owner = i
			break
		}
	}

	target := -1
	if dir > 0 {
		for i := from + 1; i < len(tf.Lines); i++ {
			if tf.Lines[i].Type == LineSection {
				target = i
				break
			}
		}
	} else {
		if owner < 0 {
			return "", false
		}
		for i := owner - 1; i >= 0; i-- {
			if tf.Lines[i].Type == LineSection {
				target = i
				break
			}
		}
	}
	if target < 0 {
		return "", false
	}

	_, name := ParseHeading(strings.TrimSpace(tf.Lines[target].Raw))

	line := tf.Lines[from]
	tf.Lines = append(tf.Lines[:from], tf.Lines[from+1:]...)
	if target > from {
		target-- // the removal shifted everything after it down
	}

	at := target + 1
	for at < len(tf.Lines) &&
		tf.Lines[at].Type == LineText &&
		strings.TrimSpace(tf.Lines[at].Raw) == "" {
		at++
	}
	tf.Lines = append(tf.Lines[:at], append([]Line{line}, tf.Lines[at:]...)...)

	return name, true
}

// insertTaskAt appends the task and splices a line for it at position at.
func (tf *TaskFile) insertTaskAt(at int, t task.Task) int {
	tf.Tasks = append(tf.Tasks, t)
	idx := len(tf.Tasks) - 1

	line := Line{Type: LineTask, TaskIndex: idx}
	tf.Lines = append(tf.Lines[:at], append([]Line{line}, tf.Lines[at:]...)...)
	return idx
}

// rebuildSections refreshes the section index after the lines changed.
func (tf *TaskFile) rebuildSections() {
	tf.Sections = nil
	for _, line := range tf.Lines {
		if line.Type != LineSection {
			continue
		}
		level, name := ParseHeading(strings.TrimSpace(line.Raw))
		tf.Sections = append(tf.Sections, Section{
			Name:   name,
			Level:  level,
			Status: sectionNameToStatus(name),
			Line:   line.Number,
		})
	}
}
