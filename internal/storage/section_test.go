package storage

import (
	"strings"
	"testing"

	"github.com/macone/todo-cli/internal/task"
)

const nested = `# Top
- [ ] top task

## Alpha
- [ ] alpha one
- [ ] alpha two

### Alpha sub
- [ ] alpha sub task

#### Alpha deep
- [ ] alpha deep task

## Beta
- [ ] beta task
`

// headingLine finds the Lines index of the heading with the given text.
func headingLine(t *testing.T, tf *TaskFile, name string) int {
	t.Helper()
	for i, line := range tf.Lines {
		if line.Type != LineSection {
			continue
		}
		if _, n := ParseHeading(line.Raw); n == name {
			return i
		}
	}
	t.Fatalf("heading %q not found", name)
	return -1
}

func TestSpanCoversNestedSubtree(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	span, ok := tf.Span(headingLine(t, tf, "Alpha"))
	if !ok {
		t.Fatal("expected Alpha to be a heading")
	}
	if span.Level != 2 {
		t.Errorf("expected level 2, got %d", span.Level)
	}
	// alpha one, alpha two, alpha sub task, alpha deep task
	if span.Tasks != 4 {
		t.Errorf("expected 4 nested tasks, got %d", span.Tasks)
	}
	// Alpha sub, Alpha deep
	if span.Headings != 2 {
		t.Errorf("expected 2 nested headings, got %d", span.Headings)
	}
}

// A span stops at the next heading of the same or shallower level.
func TestSpanStopsAtSameLevel(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	span, _ := tf.Span(headingLine(t, tf, "Beta"))
	if span.Tasks != 1 || span.Headings != 0 {
		t.Errorf("expected Beta to own 1 task and no headings, got %d and %d",
			span.Tasks, span.Headings)
	}

	deep, _ := tf.Span(headingLine(t, tf, "Alpha sub"))
	if deep.Tasks != 2 || deep.Headings != 1 {
		t.Errorf("expected Alpha sub to own 2 tasks and 1 heading, got %d and %d",
			deep.Tasks, deep.Headings)
	}
}

func TestSpanRejectsNonHeadings(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	for i, line := range tf.Lines {
		if line.Type == LineSection {
			continue
		}
		if _, ok := tf.Span(i); ok {
			t.Errorf("line %d is not a heading but Span accepted it", i)
		}
	}
	if _, ok := tf.Span(-1); ok {
		t.Error("expected out-of-range index to be rejected")
	}
}

func TestDeleteSectionRemovesWholeSubtree(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !tf.DeleteSection(headingLine(t, tf, "Alpha")) {
		t.Fatal("expected the delete to succeed")
	}

	want := "# Top\n- [ ] top task\n\n## Beta\n- [ ] beta task\n"
	if got := NewWriter().Write(tf); got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
	if len(tf.Tasks) != 2 {
		t.Errorf("expected 2 surviving tasks, got %d", len(tf.Tasks))
	}
}

// Surviving task lines must still point at the right tasks after the
// renumbering that deletion forces.
func TestDeleteSectionRepointsSurvivingTasks(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	tf.DeleteSection(headingLine(t, tf, "Alpha"))

	for _, line := range tf.Lines {
		if line.Type != LineTask {
			continue
		}
		if line.TaskIndex < 0 || line.TaskIndex >= len(tf.Tasks) {
			t.Fatalf("task line points at index %d, out of %d tasks",
				line.TaskIndex, len(tf.Tasks))
		}
	}

	titles := map[string]bool{}
	for _, line := range tf.Lines {
		if line.Type == LineTask {
			titles[tf.Tasks[line.TaskIndex].Title] = true
		}
	}
	for _, want := range []string{"top task", "beta task"} {
		if !titles[want] {
			t.Errorf("expected %q to survive, got %v", want, titles)
		}
	}
}

// Deleting the last heading takes everything to the end of the file.
func TestDeleteSectionAtEndOfFile(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	tf.DeleteSection(headingLine(t, tf, "Beta"))

	if got := NewWriter().Write(tf); got == "" {
		t.Fatal("expected content to remain")
	}
	for _, line := range tf.Lines {
		if line.Type == LineSection {
			if _, n := ParseHeading(line.Raw); n == "Beta" {
				t.Error("Beta should be gone")
			}
		}
	}
}

func TestDeleteSectionRefreshesSectionIndex(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	tf.DeleteSection(headingLine(t, tf, "Alpha"))

	if len(tf.Sections) != 2 {
		t.Fatalf("expected 2 sections left, got %d", len(tf.Sections))
	}
	if tf.Sections[0].Name != "Top" || tf.Sections[1].Name != "Beta" {
		t.Errorf("unexpected sections: %+v", tf.Sections)
	}
}

func TestInsertTaskUnderHeading(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	idx := tf.InsertTaskUnder(headingLine(t, tf, "Beta"), task.Task{
		Title:  "fresh",
		Status: task.StatusTodo,
	})
	if idx < 0 {
		t.Fatal("expected the insert to succeed")
	}

	want := "## Beta\n- [ ] fresh\n- [ ] beta task\n"
	got := NewWriter().Write(tf)
	if !strings.Contains(got, want) {
		t.Errorf("expected output to contain:\n%s\ngot:\n%s", want, got)
	}
}

// The new task goes after any blank lines that pad the heading, not before.
func TestInsertTaskUnderHeadingSkipsBlankLines(t *testing.T) {
	tf, err := NewParser().Parse("## Alpha\n\n\n- [ ] existing\n")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	tf.InsertTaskUnder(headingLine(t, tf, "Alpha"), task.Task{Title: "fresh"})

	want := "## Alpha\n\n\n- [ ] fresh\n- [ ] existing\n"
	if got := NewWriter().Write(tf); got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

// Adding from a task keeps the new one beside it rather than sending it to
// the top of the file.
func TestInsertTaskAfterTask(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// The line holding "alpha one".
	var line int
	for i, l := range tf.Lines {
		if l.Type == LineTask && tf.Tasks[l.TaskIndex].Title == "alpha one" {
			line = i
			break
		}
	}

	if idx := tf.InsertTaskAfter(line, task.Task{Title: "fresh"}); idx < 0 {
		t.Fatal("expected the insert to succeed")
	}

	want := "## Alpha\n- [ ] alpha one\n- [ ] fresh\n- [ ] alpha two\n"
	if got := NewWriter().Write(tf); !strings.Contains(got, want) {
		t.Errorf("expected output to contain:\n%s\ngot:\n%s", want, got)
	}
}

// Adding from the last task of a section must not spill into the next one.
func TestInsertTaskAfterLastTaskInSection(t *testing.T) {
	tf, err := NewParser().Parse("## Alpha\n- [ ] alpha one\n\n## Beta\n- [ ] beta task\n")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	var line int
	for i, l := range tf.Lines {
		if l.Type == LineTask && tf.Tasks[l.TaskIndex].Title == "alpha one" {
			line = i
			break
		}
	}
	tf.InsertTaskAfter(line, task.Task{Title: "fresh"})

	want := "## Alpha\n- [ ] alpha one\n- [ ] fresh\n\n## Beta\n- [ ] beta task\n"
	if got := NewWriter().Write(tf); got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestInsertTaskAfterRejectsOutOfRange(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tf.InsertTaskAfter(-1, task.Task{}) >= 0 || tf.InsertTaskAfter(999, task.Task{}) >= 0 {
		t.Error("expected out-of-range indices to be rejected")
	}
}

// taskLine finds the Lines index of the task with the given title.
func taskLine(t *testing.T, tf *TaskFile, title string) (line, index int) {
	t.Helper()
	for i, l := range tf.Lines {
		if l.Type == LineTask && tf.Tasks[l.TaskIndex].Title == title {
			return i, l.TaskIndex
		}
	}
	t.Fatalf("task %q not found", title)
	return -1, -1
}

func TestMoveTaskToNextSection(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, idx := taskLine(t, tf, "alpha one")

	name, ok := tf.MoveTaskToSection(idx, 1)
	if !ok {
		t.Fatal("expected the move to succeed")
	}
	if name != "Alpha sub" {
		t.Errorf("expected to land in Alpha sub, got %q", name)
	}

	want := "### Alpha sub\n- [ ] alpha one\n- [ ] alpha sub task\n"
	if got := NewWriter().Write(tf); !strings.Contains(got, want) {
		t.Errorf("expected output to contain:\n%s\ngot:\n%s", want, got)
	}
}

func TestMoveTaskToPrevSection(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, idx := taskLine(t, tf, "beta task")

	name, ok := tf.MoveTaskToSection(idx, -1)
	if !ok {
		t.Fatal("expected the move to succeed")
	}
	if name != "Alpha deep" {
		t.Errorf("expected to land in Alpha deep, got %q", name)
	}
}

// There is nowhere to go past either end of the file.
func TestMoveTaskToSectionStopsAtEnds(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	_, last := taskLine(t, tf, "beta task")
	if _, ok := tf.MoveTaskToSection(last, 1); ok {
		t.Error("expected no section after the last one")
	}

	_, first := taskLine(t, tf, "top task")
	if _, ok := tf.MoveTaskToSection(first, -1); ok {
		t.Error("expected no section before the first one")
	}

	before := NewWriter().Write(tf)
	tf.MoveTaskToSection(last, 1)
	tf.MoveTaskToSection(first, -1)
	if after := NewWriter().Write(tf); after != before {
		t.Errorf("refused moves must not change the file:\n%s", after)
	}
}

// Moving a task only relocates its line; the other tasks keep their indices.
func TestMoveTaskToSectionKeepsOtherTasksIntact(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	wantTasks := len(tf.Tasks)
	_, idx := taskLine(t, tf, "alpha one")

	tf.MoveTaskToSection(idx, 1)

	if len(tf.Tasks) != wantTasks {
		t.Errorf("expected %d tasks, got %d", wantTasks, len(tf.Tasks))
	}
	if tf.Tasks[idx].Title != "alpha one" {
		t.Errorf("expected index %d to still be 'alpha one', got %q", idx, tf.Tasks[idx].Title)
	}
	seen := map[string]bool{}
	for _, l := range tf.Lines {
		if l.Type == LineTask {
			if l.TaskIndex < 0 || l.TaskIndex >= len(tf.Tasks) {
				t.Fatalf("dangling task index %d", l.TaskIndex)
			}
			seen[tf.Tasks[l.TaskIndex].Title] = true
		}
	}
	if len(seen) != wantTasks {
		t.Errorf("expected every task to still have a line, got %v", seen)
	}
}

func TestMoveTaskToSectionRejectsBadInput(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, ok := tf.MoveTaskToSection(0, 0); ok {
		t.Error("expected a zero direction to be rejected")
	}
	if _, ok := tf.MoveTaskToSection(999, 1); ok {
		t.Error("expected an unknown task index to be rejected")
	}
}

func TestInsertTaskUnderRejectsNonHeadings(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tf.InsertTaskUnder(1, task.Task{Title: "nope"}) >= 0 {
		t.Error("expected insert under a task line to be rejected")
	}
}

// MoveTaskUnder relocates a task to a heading chosen directly, rather than by
// stepping to an adjacent one.
func TestMoveTaskUnderNamedHeading(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, idx := taskLine(t, tf, "alpha one")

	name, ok := tf.MoveTaskUnder(idx, headingLine(t, tf, "Beta"))
	if !ok {
		t.Fatal("expected the move to succeed")
	}
	if name != "Beta" {
		t.Errorf("expected the target name, got %q", name)
	}

	want := "## Beta\n- [ ] alpha one\n- [ ] beta task\n"
	if got := NewWriter().Write(tf); !strings.Contains(got, want) {
		t.Errorf("expected output to contain:\n%s\ngot:\n%s", want, got)
	}
}

// Moving backwards in the file works too, and every task keeps a line.
func TestMoveTaskUnderEarlierHeading(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	total := len(tf.Tasks)
	_, idx := taskLine(t, tf, "beta task")

	if _, ok := tf.MoveTaskUnder(idx, headingLine(t, tf, "Top")); !ok {
		t.Fatal("expected the move to succeed")
	}

	want := "# Top\n- [ ] beta task\n- [ ] top task\n"
	if got := NewWriter().Write(tf); !strings.Contains(got, want) {
		t.Errorf("expected output to contain:\n%s\ngot:\n%s", want, got)
	}

	seen := 0
	for _, l := range tf.Lines {
		if l.Type == LineTask {
			if l.TaskIndex < 0 || l.TaskIndex >= len(tf.Tasks) {
				t.Fatalf("dangling task index %d", l.TaskIndex)
			}
			seen++
		}
	}
	if seen != total {
		t.Errorf("expected %d task lines, got %d", total, seen)
	}
}

func TestMoveTaskUnderRejectsNonHeadings(t *testing.T) {
	tf, err := NewParser().Parse(nested)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, idx := taskLine(t, tf, "alpha one")

	if _, ok := tf.MoveTaskUnder(idx, 1); ok {
		t.Error("expected a task line to be rejected as a target")
	}
	if _, ok := tf.MoveTaskUnder(idx, 999); ok {
		t.Error("expected an out-of-range target to be rejected")
	}
	if _, ok := tf.MoveTaskUnder(999, headingLine(t, tf, "Beta")); ok {
		t.Error("expected an unknown task to be rejected")
	}
}
