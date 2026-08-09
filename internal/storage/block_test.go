package storage

import (
	"strings"
	"testing"

	"github.com/macone/todo-cli/internal/task"
)

const blockFixture = "## Alpha\n" +
	"- [ ] plain task\n" +
	"  some notes here\n" +
	"  - [ ] a subtask\n" +
	"- [x] done task\n"

func TestParseBlockAttachesIndentedLines(t *testing.T) {
	tf, err := NewParser().Parse(blockFixture)
	if err != nil {
		t.Fatal(err)
	}

	// The subtask is part of the block, not a task in its own right.
	if len(tf.Tasks) != 2 {
		t.Fatalf("expected 2 top-level tasks, got %d", len(tf.Tasks))
	}

	block := tf.Tasks[0].Block
	if len(block) != 2 {
		t.Fatalf("expected a block of 2, got %d", len(block))
	}
	if block[0].Subtask != nil || block[0].Note != "some notes here" {
		t.Errorf("expected a note first, got %+v", block[0])
	}
	if block[1].Subtask == nil || block[1].Subtask.Title != "a subtask" {
		t.Errorf("expected a subtask second, got %+v", block[1])
	}
}

// Depth beyond one level is flattened, and the block is written back at two
// spaces however it was indented.
func TestBlockIndentationIsNormalised(t *testing.T) {
	input := "## Alpha\n" +
		"- [ ] task\n" +
		"      deeply indented note\n" +
		"\t- [ ] tab indented sub\n" +
		"          - [x] very deep\n"

	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatal(err)
	}

	want := "## Alpha\n" +
		"- [ ] task\n" +
		"  deeply indented note\n" +
		"  - [ ] tab indented sub\n" +
		"  - [x] very deep\n"
	if got := NewWriter().Write(tf); got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestBlockRoundTrips(t *testing.T) {
	if got := roundTrip(t, blockFixture); got != blockFixture {
		t.Errorf("expected:\n%s\ngot:\n%s", blockFixture, got)
	}
}

// A blank line ends a block: what follows belongs to the file, not the task.
func TestBlankLineEndsABlock(t *testing.T) {
	input := "## Alpha\n- [ ] task\n  owned note\n\n  orphaned note\n"
	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(tf.Tasks[0].Block); got != 1 {
		t.Errorf("expected a block of 1, got %d", got)
	}
}

// The block travels with its task when the sort moves it, which is what stops
// notes being left behind under the wrong task.
func TestBlockFollowsItsTaskThroughSorting(t *testing.T) {
	tf, err := NewParser().Parse(blockFixture)
	if err != nil {
		t.Fatal(err)
	}
	if !tf.Sort() {
		t.Fatal("expected the done task to sort above the plain one")
	}

	want := "## Alpha\n" +
		"- [x] done task\n" +
		"- [ ] plain task\n" +
		"  some notes here\n" +
		"  - [ ] a subtask\n"
	if got := NewWriter().Write(tf); got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

// Likewise when a task is moved to another heading.
func TestBlockFollowsItsTaskWhenMoved(t *testing.T) {
	input := blockFixture + "## Beta\n- [ ] beta task\n"
	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := tf.MoveTaskUnder(0, headingLine(t, tf, "Beta")); !ok {
		t.Fatal("expected the move to succeed")
	}

	want := "## Beta\n" +
		"- [ ] plain task\n" +
		"  some notes here\n" +
		"  - [ ] a subtask\n"
	if got := NewWriter().Write(tf); !strings.Contains(got, want) {
		t.Errorf("expected output to contain:\n%s\ngot:\n%s", want, got)
	}
}

// Subtasks keep the order they were written in; only top-level tasks sort.
func TestSubtasksKeepWrittenOrder(t *testing.T) {
	input := "## Alpha\n" +
		"- [ ] parent\n" +
		"  - [ ] first step\n" +
		"  - [x] already done\n" +
		"  - [ ] last step\n"

	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	tf.Sort()

	if got := roundTrip(t, input); got != input {
		t.Errorf("expected the block order untouched:\n%s\ngot:\n%s", input, got)
	}
	if got := NewWriter().Write(tf); got != input {
		t.Errorf("after sorting, expected:\n%s\ngot:\n%s", input, got)
	}
}

// Subtasks carry their own metadata.
func TestSubtaskMetadata(t *testing.T) {
	tf, err := NewParser().Parse("## Alpha\n- [ ] parent\n  - [x] !! sub +work @home\n")
	if err != nil {
		t.Fatal(err)
	}
	sub := tf.Tasks[0].Block[0].Subtask
	if sub == nil {
		t.Fatal("expected a subtask")
	}
	if sub.Status != task.StatusDone {
		t.Errorf("status: got %d", sub.Status)
	}
	if sub.Priority != task.PriorityHigh {
		t.Errorf("priority: got %d", sub.Priority)
	}
	if sub.Title != "sub" {
		t.Errorf("title: got %q", sub.Title)
	}
	if len(sub.Tags) != 1 || sub.Tags[0] != "work" {
		t.Errorf("tags: got %v", sub.Tags)
	}
	if len(sub.Contexts) != 1 || sub.Contexts[0] != "home" {
		t.Errorf("contexts: got %v", sub.Contexts)
	}
}

func roundTrip(t *testing.T, input string) string {
	t.Helper()
	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	return NewWriter().Write(tf)
}
