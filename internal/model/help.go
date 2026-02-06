package model

import "strings"

// helpText returns the full help overlay content.
func helpText() string {
	var sb strings.Builder

	sb.WriteString("Keybinding Reference\n")
	sb.WriteString("====================\n\n")

	sb.WriteString("Navigation\n")
	sb.WriteString("  j/k        Move down/up\n")
	sb.WriteString("  J/K        Jump to next/previous section\n\n")

	sb.WriteString("Actions\n")
	sb.WriteString("  a          Add new task\n")
	sb.WriteString("  e          Edit task title\n")
	sb.WriteString("  d          Toggle done\n")
	sb.WriteString("  x          Delete task (with confirmation)\n")
	sb.WriteString("  s          Cycle status (todo/in-progress/done)\n")
	sb.WriteString("  p          Cycle priority (none/low/medium/high)\n")
	sb.WriteString("  t          Add tag to task\n")
	sb.WriteString("  u          Undo last action\n\n")

	sb.WriteString("Filtering\n")
	sb.WriteString("  /          Search by title\n")
	sb.WriteString("  f          Filter by tag\n")
	sb.WriteString("  1          Show all tasks\n")
	sb.WriteString("  2          Show active only (todo + in-progress)\n")
	sb.WriteString("  3          Show done only\n\n")

	sb.WriteString("General\n")
	sb.WriteString("  ?          Toggle this help\n")
	sb.WriteString("  q          Quit\n")
	sb.WriteString("  Ctrl+C     Quit\n\n")

	sb.WriteString("Press ? or Esc to close")

	return sb.String()
}
