package model

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

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
			{"j / k, ↓ / ↑", "Move down / up"},
			{"tab / S-tab", "Select next / prev heading (or J / K)"},
			{"ctrl+j/ctrl+k", "Select next / prev ## heading only"},
			{"ctrl+h/ctrl+l", "Select parent / child heading"},
			{"gg / G", "Jump to top / bottom"},
			{"ctrl+d/ctrl+u", "Page down / up"},
			{"alt+j / alt+k", "Move task down / up in its group"},
			{"alt+J / alt+K", "Move task to next / prev heading"},
		}},
		{"Actions", []ui.HelpItem{
			{"a", "Add task below the selection"},
			{"e", "Edit task title"},
			{"d", "Toggle done"},
			{"x", "Delete task, or heading + contents"},
			{"Space", "Cycle status"},
			{"p / P", "Raise / lower priority"},
			{"t", "Add tag"},
			{"u", "Undo"},
			{"o", "Open file in editor"},
		}},
		{"Filtering", []ui.HelpItem{
			{"/", "Search by title"},
			{"f", "Filter by tag"},
			{"1", "Show all"},
			{"2", "Active only"},
			{"3", "Done only"},
		}},
		{"General", []ui.HelpItem{
			{"i", "Toggle the file path in the title"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
			{"Ctrl+C", "Quit"},
		}},
	}

	for i, g := range groups {
		sb.WriteString(ui.SectionHeader.Render(g.name) + "\n")
		for _, item := range g.items {
			key := ui.HelpKey.Render(padRight(item.Key, 14))
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

// padRight pads s to width using its rendered width, so multi-byte glyphs like
// the arrow keys stay aligned with the ASCII rows.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
