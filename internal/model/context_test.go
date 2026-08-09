package model

import (
	"strings"
	"testing"
)

const contextFixture = "## Alpha\n" +
	"- [ ] write docs +work @home\n" +
	"- [ ] call bank @errands\n" +
	"- [ ] refactor +work\n" +
	"- [ ] plain one\n"

// Filtering by context narrows to tasks carrying it, leaving tags alone.
func TestContextFilter(t *testing.T) {
	a := newDeleteApp(t, contextFixture)

	a.list.SetContextFilter("home")
	if got := strings.Join(visibleTasks(a.list), ","); got != "write docs" {
		t.Errorf("got %q", got)
	}

	a.list.ClearContextFilter()
	if got := len(visibleTasks(a.list)); got != 4 {
		t.Errorf("expected all 4 back, got %d", got)
	}
}

// Tag and context filters are separate axes and apply together.
func TestTagAndContextFiltersCompose(t *testing.T) {
	a := newDeleteApp(t, contextFixture)

	a.list.SetTagFilter("work")
	if got := strings.Join(visibleTasks(a.list), ","); got != "write docs,refactor" {
		t.Errorf("tag only: got %q", got)
	}

	a.list.SetContextFilter("home")
	if got := strings.Join(visibleTasks(a.list), ","); got != "write docs" {
		t.Errorf("tag and context: got %q", got)
	}

	// A context that no tagged task has leaves nothing.
	a.list.SetContextFilter("errands")
	if got := visibleTasks(a.list); len(got) != 0 {
		t.Errorf("expected no matches, got %v", got)
	}
}

func TestAllContextsListsEachOnce(t *testing.T) {
	a := newDeleteApp(t, contextFixture+"- [ ] another @home\n")

	got := strings.Join(a.list.AllContexts(), ",")
	if got != "home,errands" {
		t.Errorf("expected contexts in first-seen order without duplicates, got %q", got)
	}
	// Tags stay out of it.
	if strings.Contains(got, "work") {
		t.Error("a tag leaked into the context list")
	}
}

func TestContextShowsInIndicator(t *testing.T) {
	a := newDeleteApp(t, contextFixture)

	a.list.SetContextFilter("home")
	if got := a.list.FilterIndicator(); !strings.Contains(got, "@home") {
		t.Errorf("got %q", got)
	}

	a.list.SetTagFilter("work")
	got := a.list.FilterIndicator()
	if !strings.Contains(got, "+work") || !strings.Contains(got, "@home") {
		t.Errorf("expected both axes named, got %q", got)
	}
}

// Adding a context through the prompt attaches it to the selected task and
// leaves its tags untouched.
func TestAddContextThroughPrompt(t *testing.T) {
	a := newDeleteApp(t, contextFixture)
	a = moveTo(t, a, "task:refactor")

	next, _ := a.addLabel(labelContext)
	a = next.(App)
	if a.input.Mode() != inputContext {
		t.Fatalf("expected the context prompt, got mode %d", a.input.Mode())
	}

	a = typeRunes(a, "office")
	next, _ = a.commitInput()
	a = next.(App)

	var found *string
	for i := range a.taskFile.Tasks {
		if a.taskFile.Tasks[i].Title == "refactor" {
			found = &a.taskFile.Tasks[i].Contexts[0]
			if got := strings.Join(a.taskFile.Tasks[i].Tags, ","); got != "work" {
				t.Errorf("expected tags untouched, got %q", got)
			}
		}
	}
	if found == nil || *found != "office" {
		t.Error("expected the context to be attached")
	}
}

func TestLabelKindNaming(t *testing.T) {
	if labelTag.sigil() != "+" || labelTag.noun() != "tag" {
		t.Errorf("tag: %q %q", labelTag.sigil(), labelTag.noun())
	}
	if labelContext.sigil() != "@" || labelContext.noun() != "context" {
		t.Errorf("context: %q %q", labelContext.sigil(), labelContext.noun())
	}
}
