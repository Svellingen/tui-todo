// Package ui contains lipgloss styles and key bindings for the TUI.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// SectionHeader is bold cyan for section headers.
	SectionHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

	// SelectedTask is reverse video for the cursor.
	SelectedTask = lipgloss.NewStyle().Reverse(true)

	// PriorityHigh is red for high priority.
	PriorityHigh = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	// PriorityMedium is yellow for medium priority.
	PriorityMedium = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	// PriorityLow is dim for low priority.
	PriorityLow = lipgloss.NewStyle().Faint(true)

	// Tag is magenta for tags.
	Tag = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	// DoneTask is dim with strikethrough for completed tasks.
	DoneTask = lipgloss.NewStyle().Faint(true).Strikethrough(true)

	// HelpBar is subtle for the bottom help bar.
	HelpBar = lipgloss.NewStyle().Faint(true)
)
