package storage

import (
	"slices"

	"github.com/svellingen/md-taco/internal/task"
)

// sortRank orders tasks within a section: done first, then high priority,
// then medium, then everything else.
func sortRank(t task.Task) int {
	switch {
	case t.Status == task.StatusDone:
		return 0
	case t.Priority == task.PriorityHigh:
		return 1
	case t.Priority == task.PriorityMedium:
		return 2
	default:
		return 3
	}
}

// taskRegions returns the indices of task lines, grouped by the section they
// fall in. Tasks appearing before the first section header form their own
// group.
func (tf *TaskFile) taskRegions() [][]int {
	var regions [][]int
	var current []int

	for i, line := range tf.Lines {
		switch line.Type {
		case LineSection:
			if len(current) > 0 {
				regions = append(regions, current)
				current = nil
			}
		case LineTask:
			current = append(current, i)
		}
	}
	if len(current) > 0 {
		regions = append(regions, current)
	}
	return regions
}

// Sort orders the tasks in each section by rank, reporting whether anything
// moved.
//
// Only task lines are permuted, and only among the slots they already occupy,
// so headers, blank lines and prose stay exactly where they are. The sort is
// stable, which is what lets a manual reordering within a rank group survive
// the next save.
func (tf *TaskFile) Sort() bool {
	changed := false

	for _, region := range tf.taskRegions() {
		order := make([]int, len(region))
		for i, lineIdx := range region {
			order[i] = tf.Lines[lineIdx].TaskIndex
		}

		sorted := slices.Clone(order)
		slices.SortStableFunc(sorted, func(a, b int) int {
			return sortRank(tf.Tasks[a]) - sortRank(tf.Tasks[b])
		})

		for i, lineIdx := range region {
			if tf.Lines[lineIdx].TaskIndex != sorted[i] {
				changed = true
			}
			tf.Lines[lineIdx].TaskIndex = sorted[i]
		}
	}

	return changed
}

// MoveTask shifts a task one slot within its rank group, swapping it with the
// neighbouring task of the same rank in the same section. delta is -1 to move
// up and +1 to move down.
//
// Moving past either end of the group, out of the section, or into a
// different rank does nothing. It reports whether anything moved.
func (tf *TaskFile) MoveTask(taskIndex, delta int) bool {
	if taskIndex < 0 || taskIndex >= len(tf.Tasks) {
		return false
	}

	for _, region := range tf.taskRegions() {
		for pos, lineIdx := range region {
			if tf.Lines[lineIdx].TaskIndex != taskIndex {
				continue
			}

			target := pos + delta
			if target < 0 || target >= len(region) {
				return false
			}
			neighbour := tf.Lines[region[target]].TaskIndex
			if sortRank(tf.Tasks[neighbour]) != sortRank(tf.Tasks[taskIndex]) {
				return false
			}

			tf.Lines[lineIdx].TaskIndex = neighbour
			tf.Lines[region[target]].TaskIndex = taskIndex
			return true
		}
	}

	return false
}
