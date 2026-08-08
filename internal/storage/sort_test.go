package storage

import "testing"

// parseWrite runs content through the parser and writer, applying fn in
// between, and returns the resulting markdown.
func parseWrite(t *testing.T, content string, fn func(*TaskFile)) string {
	t.Helper()
	tf, err := NewParser().Parse(content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if fn != nil {
		fn(tf)
	}
	return NewWriter().Write(tf)
}

func TestSortRanksDoneThenPriority(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] plain one\n" +
		"- [ ] ! medium one\n" +
		"- [x] finished one\n" +
		"- [ ] !! high one\n"

	want := "## Backlog\n" +
		"- [x] finished one\n" +
		"- [ ] !! high one\n" +
		"- [ ] ! medium one\n" +
		"- [ ] plain one\n"

	got := parseWrite(t, input, func(tf *TaskFile) {
		if !tf.Sort() {
			t.Error("expected Sort to report a change")
		}
	})
	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

// Each section sorts on its own; tasks never migrate across a header.
func TestSortIsPerSection(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] plain backlog\n" +
		"- [ ] !! high backlog\n" +
		"\n" +
		"## In Progress\n" +
		"- [ ] plain active\n" +
		"- [ ] ! medium active\n"

	want := "## Backlog\n" +
		"- [ ] !! high backlog\n" +
		"- [ ] plain backlog\n" +
		"\n" +
		"## In Progress\n" +
		"- [ ] ! medium active\n" +
		"- [ ] plain active\n"

	got := parseWrite(t, input, func(tf *TaskFile) { tf.Sort() })
	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

// Blank lines and prose keep their positions; only task lines are permuted.
func TestSortPreservesNonTaskLines(t *testing.T) {
	input := "# My Project\n" +
		"\n" +
		"Some notes here.\n" +
		"\n" +
		"## Backlog\n" +
		"\n" +
		"- [ ] plain one\n" +
		"- [ ] !! high one\n"

	want := "# My Project\n" +
		"\n" +
		"Some notes here.\n" +
		"\n" +
		"## Backlog\n" +
		"\n" +
		"- [ ] !! high one\n" +
		"- [ ] plain one\n"

	got := parseWrite(t, input, func(tf *TaskFile) { tf.Sort() })
	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

// Equally ranked tasks keep their relative order, which is what makes a manual
// reordering survive later sorts.
func TestSortIsStableWithinRank(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] !! bravo\n" +
		"- [ ] !! alpha\n" +
		"- [ ] !! charlie\n"

	got := parseWrite(t, input, func(tf *TaskFile) {
		if tf.Sort() {
			t.Error("expected Sort to report no change for an already-sorted section")
		}
	})
	if got != input {
		t.Errorf("expected order preserved:\n%s\ngot:\n%s", input, got)
	}
}

func TestSortReportsNoChangeWhenAlreadySorted(t *testing.T) {
	input := "## Backlog\n- [x] done\n- [ ] !! high\n- [ ] ! medium\n- [ ] plain\n"
	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tf.Sort() {
		t.Error("expected no change")
	}
}

func TestMoveTaskWithinRankGroup(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] !! alpha\n" +
		"- [ ] !! bravo\n" +
		"- [ ] plain\n"

	want := "## Backlog\n" +
		"- [ ] !! bravo\n" +
		"- [ ] !! alpha\n" +
		"- [ ] plain\n"

	got := parseWrite(t, input, func(tf *TaskFile) {
		// alpha is task 0; move it down one slot.
		if !tf.MoveTask(0, 1) {
			t.Error("expected the move to succeed")
		}
	})
	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

// A move that would cross into a different rank is refused, so priority
// ordering cannot be broken by hand.
func TestMoveTaskStopsAtRankBoundary(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] !! high\n" +
		"- [ ] ! medium\n" +
		"- [ ] plain\n"

	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// The only high task cannot move down into the medium group.
	if tf.MoveTask(0, 1) {
		t.Error("expected the move into another rank to be refused")
	}
	// Nor can the medium task move up into the high group.
	if tf.MoveTask(1, -1) {
		t.Error("expected the move into another rank to be refused")
	}

	if got := NewWriter().Write(tf); got != input {
		t.Errorf("expected file unchanged:\n%s\ngot:\n%s", input, got)
	}
}

func TestMoveTaskStopsAtSectionBoundary(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] alpha\n" +
		"\n" +
		"## In Progress\n" +
		"- [ ] bravo\n"

	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if tf.MoveTask(0, 1) {
		t.Error("expected the move across a section to be refused")
	}
	if tf.MoveTask(1, -1) {
		t.Error("expected the move across a section to be refused")
	}
}

func TestMoveTaskRejectsOutOfRange(t *testing.T) {
	tf, err := NewParser().Parse("## Backlog\n- [ ] only\n")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tf.MoveTask(-1, 1) || tf.MoveTask(5, 1) {
		t.Error("expected out-of-range task indices to be refused")
	}
	if tf.MoveTask(0, -1) || tf.MoveTask(0, 1) {
		t.Error("expected a lone task to have nowhere to go")
	}
}

// A manual reordering must not be undone by the next save.
func TestMoveThenSortKeepsManualOrder(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] !! alpha\n" +
		"- [ ] !! bravo\n"

	want := "## Backlog\n" +
		"- [ ] !! bravo\n" +
		"- [ ] !! alpha\n"

	got := parseWrite(t, input, func(tf *TaskFile) {
		if !tf.MoveTask(0, 1) {
			t.Fatal("expected the move to succeed")
		}
		if tf.Sort() {
			t.Error("expected the sort to leave the manual order alone")
		}
	})
	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}
