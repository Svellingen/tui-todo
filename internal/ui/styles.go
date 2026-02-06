// Package ui contains lipgloss styles and key bindings for the TUI.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// SectionHeader is bold cyan for section headers.
	SectionHeader = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "4", Dark: "6"})

	// SelectedTask is reverse video for the cursor.
	SelectedTask = lipgloss.NewStyle().Reverse(true)

	// PriorityHigh is red for high priority.
	PriorityHigh = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "1", Dark: "9"})

	// PriorityMedium is yellow for medium priority.
	PriorityMedium = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "3", Dark: "11"})

	// PriorityLow is dim for low priority.
	PriorityLow = lipgloss.NewStyle().Faint(true)

	// Tag is magenta for tags.
	Tag = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "5", Dark: "13"})

	// DoneTask is dim with strikethrough for completed tasks.
	DoneTask = lipgloss.NewStyle().Faint(true).Strikethrough(true)

	// HelpBar is subtle for the bottom help bar.
	HelpBar = lipgloss.NewStyle().Faint(true)

	// HelpOverlay is for the help screen.
	HelpOverlay = lipgloss.NewStyle().Padding(1, 2)

	// Progress is green-tinted for the done counter.
	Progress = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "2", Dark: "10"})
)
