package model

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/svellingen/md-taco/internal/ui"
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
			{"Enter / tab", "Fold one block / all blocks"},
			{"J / K", "Select next / prev heading"},
			{"ctrl+j/ctrl+k", "Select next / prev ## heading only"},
			{"ctrl+h/ctrl+l", "Select parent / child heading"},
			{"ctrl+e", "Heading popup: jump to one"},
			{"m", "Heading popup: move task to one"},
			{"gg / G", "Jump to top / bottom"},
			{"ctrl+d/ctrl+u", "Page down / up"},
			{"alt+j / alt+k", "Move task in its group / block line in its block"},
			{"alt+J / alt+K", "Move task to next / prev heading"},
		}},
		{"Actions", []ui.HelpItem{
			{"a / A / n", "Add task / subtask / note"},
			{"e", "Edit task or block line"},
			{"d", "Toggle done"},
			{"x", "Delete task, or heading + contents"},
			{"Space / C-Space", "Cycle status (note↔task in a block)"},
			{"p / P", "Raise / lower priority"},
			{"t / c", "Add tag / context"},
			{"u / ctrl+r", "Undo / redo"},
			{"o", "Open file in editor"},
		}},
		{"Filtering", []ui.HelpItem{
			{"/", "Search by title"},
			{"T / C", "Filter by tag / context"},
			{"1", "Show all"},
			{"2", "Active only"},
			{"3", "Done only"},
			{"f", "Focus current heading"},
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
			key := ui.HelpKey.Render(padRight(item.Key, 16))
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
