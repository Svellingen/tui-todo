package storage

import (
	"testing"

	"github.com/svellingen/md-taco/internal/task"
)

func TestParseEmptyFile(t *testing.T) {
	p := NewParser()
	result, err := p.Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(result.Tasks))
	}
}

func TestParseSingleTask(t *testing.T) {
	input := "## Backlog\n\n- [ ] Write tests\n"
	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}
	if result.Tasks[0].Title != "Write tests" {
		t.Errorf("expected title 'Write tests', got '%s'", result.Tasks[0].Title)
	}
	if result.Tasks[0].Status != task.StatusTodo {
		t.Errorf("expected StatusTodo, got %d", result.Tasks[0].Status)
	}
}

func TestParseAllStatuses(t *testing.T) {
	input := `## Backlog

- [ ] Todo item

## In Progress

- [-] Active item

## Done

- [x] Completed item
`
	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Tasks))
	}
	if result.Tasks[0].Status != task.StatusTodo {
		t.Errorf("task 0: expected StatusTodo")
	}
	if result.Tasks[1].Status != task.StatusInProgress {
		t.Errorf("task 1: expected StatusInProgress")
	}
	if result.Tasks[2].Status != task.StatusDone {
		t.Errorf("task 2: expected StatusDone")
	}
}

func TestParseMetadata(t *testing.T) {
	input := "## Backlog\n\n- [ ] !! Fix bug +backend @security due:2026-03-01\n"
	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tk := result.Tasks[0]
	if tk.Title != "Fix bug" {
		t.Errorf("expected title 'Fix bug', got '%s'", tk.Title)
	}
	if tk.Priority != task.PriorityHigh {
		t.Errorf("expected PriorityHigh, got %d", tk.Priority)
	}
	if len(tk.Tags) != 1 || tk.Tags[0] != "backend" {
		t.Errorf("expected tags [backend], got %v", tk.Tags)
	}
	if len(tk.Contexts) != 1 || tk.Contexts[0] != "security" {
		t.Errorf("expected contexts [security], got %v", tk.Contexts)
	}
	if tk.DueDate == nil {
		t.Fatal("expected due date to be set")
	}
}

func TestParsePriorityMarkers(t *testing.T) {
	input := "## Backlog\n" +
		"- [ ] no pri\n" +
		"- [ ] ! medium priority task\n" +
		"- [ ] !! high priority task\n"

	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []struct {
		title    string
		priority task.Priority
	}{
		{"no pri", task.PriorityNone},
		{"medium priority task", task.PriorityMedium},
		{"high priority task", task.PriorityHigh},
	}

	if len(result.Tasks) != len(want) {
		t.Fatalf("expected %d tasks, got %d", len(want), len(result.Tasks))
	}
	for i, w := range want {
		got := result.Tasks[i]
		if got.Title != w.title {
			t.Errorf("task %d: expected title %q, got %q", i, w.title, got.Title)
		}
		if got.Priority != w.priority {
			t.Errorf("task %d: expected priority %d, got %d", i, w.priority, got.Priority)
		}
	}
}

// A "!" that is part of a word is text, not a priority marker.
func TestParseExclamationInTitleIsNotPriority(t *testing.T) {
	cases := []struct {
		line  string
		title string
	}{
		{"- [ ] ship it!", "ship it!"},
		{"- [ ] !important-looking", "!important-looking"},
		{"- [ ] wow!!! that broke", "wow!!! that broke"},
	}

	p := NewParser()
	for _, c := range cases {
		result, err := p.Parse("## Backlog\n" + c.line + "\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Tasks) != 1 {
			t.Fatalf("%q: expected 1 task, got %d", c.line, len(result.Tasks))
		}
		got := result.Tasks[0]
		if got.Title != c.title {
			t.Errorf("%q: expected title %q, got %q", c.line, c.title, got.Title)
		}
		if got.Priority != task.PriorityNone {
			t.Errorf("%q: expected no priority, got %d", c.line, got.Priority)
		}
	}
}

// More marks than the scale defines clamp to high rather than being dropped.
func TestParseExcessMarksClampToHigh(t *testing.T) {
	p := NewParser()
	result, err := p.Parse("## Backlog\n- [ ] !!! very urgent\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.Tasks[0]
	if got.Priority != task.PriorityHigh {
		t.Errorf("expected PriorityHigh, got %d", got.Priority)
	}
	if got.Title != "very urgent" {
		t.Errorf("expected title 'very urgent', got %q", got.Title)
	}
}

// Files written by earlier versions still load with their priorities intact.
// The scale has no separate low level any more, so legacy low lands on medium
// rather than being dropped.
func TestParseLegacyPriorityToken(t *testing.T) {
	cases := []struct {
		line string
		want task.Priority
	}{
		{"- [ ] Old style priority:high +backend", task.PriorityHigh},
		{"- [ ] Old style priority:medium +backend", task.PriorityMedium},
		{"- [ ] Old style priority:low +backend", task.PriorityMedium},
	}

	p := NewParser()
	for _, c := range cases {
		result, err := p.Parse("## Backlog\n" + c.line + "\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := result.Tasks[0]
		if got.Priority != c.want {
			t.Errorf("%q: expected priority %d, got %d", c.line, c.want, got.Priority)
		}
		if got.Title != "Old style" {
			t.Errorf("%q: expected title 'Old style', got %q", c.line, got.Title)
		}
	}
}

func TestParseHeadingLevels(t *testing.T) {
	cases := []struct {
		line  string
		level int
		name  string
	}{
		{"# top", 1, "top"},
		{"## second", 2, "second"},
		{"### third", 3, "third"},
		{"#### fourth", 4, "fourth"},
		{"###### sixth", 6, "sixth"},
		{"##   extra spaces  ", 2, "extra spaces"},
		// Not headings:
		{"####### too deep", 0, ""},
		{"#no-space", 0, ""},
		{"#", 0, ""},
		{"plain text", 0, ""},
		{"- [ ] a task", 0, ""},
	}

	for _, c := range cases {
		level, name := ParseHeading(c.line)
		if level != c.level || name != c.name {
			t.Errorf("%q: got (%d, %q), want (%d, %q)", c.line, level, name, c.level, c.name)
		}
	}
}

// Every heading level becomes a section, so tasks group under sub-headers too.
func TestParseAllHeadingLevelsBecomeSections(t *testing.T) {
	input := "# Top\n" +
		"- [ ] top task\n" +
		"## Second\n" +
		"- [ ] second task\n" +
		"### Third\n" +
		"- [ ] third task\n" +
		"#### Fourth\n" +
		"- [ ] fourth task\n"

	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []struct {
		name  string
		level int
	}{
		{"Top", 1}, {"Second", 2}, {"Third", 3}, {"Fourth", 4},
	}
	if len(tf.Sections) != len(want) {
		t.Fatalf("expected %d sections, got %d", len(want), len(tf.Sections))
	}
	for i, w := range want {
		if tf.Sections[i].Name != w.name || tf.Sections[i].Level != w.level {
			t.Errorf("section %d: got (%q, %d), want (%q, %d)",
				i, tf.Sections[i].Name, tf.Sections[i].Level, w.name, w.level)
		}
	}
	if len(tf.Tasks) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(tf.Tasks))
	}
}

// Headings of any level round-trip untouched.
func TestWriteRoundtripAllHeadingLevels(t *testing.T) {
	input := "# Top\n\n## Second\n\n- [ ] !! a task\n\n### Third\n\n#### Fourth\n\n- [ ] another\n"
	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := NewWriter().Write(tf); got != input {
		t.Errorf("expected:\n%s\ngot:\n%s", input, got)
	}
}

func TestParsePreservesNonTaskLines(t *testing.T) {
	input := "# My Project\n\nSome notes here.\n\n## Backlog\n\n- [ ] A task\n\n## Done\n\n- [x] Finished\n"
	p := NewParser()
	result, err := p.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Lines) < 5 {
		t.Fatalf("expected preserved lines, got %d", len(result.Lines))
	}
}

// Tags and contexts are separate axes: "+x" is a tag, "@x" is a context, and
// a task can carry both.
func TestParseSeparatesTagsAndContexts(t *testing.T) {
	tf, err := NewParser().Parse("## Backlog\n- [ ] task +work +urgent @home @errands\n")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	got := tf.Tasks[0]

	if len(got.Tags) != 2 || got.Tags[0] != "work" || got.Tags[1] != "urgent" {
		t.Errorf("tags: got %v", got.Tags)
	}
	if len(got.Contexts) != 2 || got.Contexts[0] != "home" || got.Contexts[1] != "errands" {
		t.Errorf("contexts: got %v", got.Contexts)
	}
	if got.Title != "task" {
		t.Errorf("title: got %q", got.Title)
	}
}

// A context survives a round-trip as "@x" rather than being rewritten as a tag.
func TestContextRoundTrips(t *testing.T) {
	input := "## Backlog\n- [ ] task +work @home\n"

	tf, err := NewParser().Parse(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if got := NewWriter().Write(tf); got != input {
		t.Errorf("expected:\n%s\ngot:\n%s", input, got)
	}
}

// A bare sigil is text, not an empty label.
func TestParseIgnoresBareSigils(t *testing.T) {
	tf, err := NewParser().Parse("## Backlog\n- [ ] email me @ noon +\n")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	got := tf.Tasks[0]
	if len(got.Contexts) != 0 {
		t.Errorf("expected no contexts, got %v", got.Contexts)
	}
	if len(got.Tags) != 0 {
		t.Errorf("expected no tags, got %v", got.Tags)
	}
}
