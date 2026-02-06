package model

import (
	"strings"

	"github.com/macone/todo-cli/internal/ui"
)

// helpContent returns styled help text for the overlay.
func helpContent() string {
	var sb strings.Builder

	title := ui.TitleStyle.Render("Keybinding Reference")
	sb.WriteString(title + "\n\n")

	groups := []struct {
		name  string
		items []ui.HelpItem
	}{
		{"Navigation", []ui.HelpItem{
			{"j / k", "Move down / up"},
			{"J / K", "Jump next / prev section"},
		}},
		{"Actions", []ui.HelpItem{
			{"a", "Add new task"},
			{"e", "Edit task title"},
			{"d", "Toggle done"},
			{"x", "Delete task"},
			{"s", "Cycle status"},
			{"p", "Cycle priority"},
			{"t", "Add tag"},
			{"u", "Undo"},
		}},
		{"Filtering", []ui.HelpItem{
			{"/", "Search by title"},
			{"f", "Filter by tag"},
			{"1", "Show all"},
			{"2", "Active only"},
			{"3", "Done only"},
		}},
		{"General", []ui.HelpItem{
			{"?", "Toggle this help"},
			{"q", "Quit"},
			{"Ctrl+C", "Quit"},
		}},
	}

	for i, g := range groups {
		sb.WriteString(ui.SectionHeader.Render(g.name) + "\n")
		for _, item := range g.items {
			key := ui.HelpKey.Render(padRight(item.Key, 10))
			desc := ui.HelpBar.Render(item.Desc)
			sb.WriteString("  " + key + " " + desc + "\n")
		}
		if i < len(groups)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n" + ui.HelpBar.Render("Press ? or Esc to close"))

	return sb.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
