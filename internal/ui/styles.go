package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette — cohesive purple/blue theme with semantic colors.
var (
	ColorPurple    = lipgloss.AdaptiveColor{Light: "#5B21B6", Dark: "#A78BFA"}
	ColorBlue      = lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#60A5FA"}
	ColorCyan      = lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#67E8F9"}
	ColorGreen     = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#86EFAC"}
	ColorRed       = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}
	ColorYellow    = lipgloss.AdaptiveColor{Light: "#A16207", Dark: "#FDE68A"}
	ColorMauve     = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#C4B5FD"}
	ColorGray      = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	ColorDimGray   = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}
	ColorFaintGray = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}
	ColorBorder    = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#7C3AED"}
	ColorWhite     = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F9FAFB"}
)

// UI element styles.
var (
	TitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(ColorPurple)
	SectionHeader  = lipgloss.NewStyle().Bold(true).Foreground(ColorBlue)
	SectionSep     = lipgloss.NewStyle().Foreground(ColorDimGray)
	CursorStyle    = lipgloss.NewStyle().Foreground(ColorPurple).Bold(true)
	SelectedTask   = lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	PriorityHigh   = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	PriorityMedium = lipgloss.NewStyle().Foreground(ColorYellow)
	Tag            = lipgloss.NewStyle().Foreground(ColorMauve)
	DueStyle       = lipgloss.NewStyle().Foreground(ColorGray).Italic(true)
	DoneTask       = lipgloss.NewStyle().Foreground(ColorFaintGray).Strikethrough(true)
	DoneMeta       = lipgloss.NewStyle().Foreground(ColorFaintGray)
	StatusTodo     = lipgloss.NewStyle().Foreground(ColorGray)
	StatusActive   = lipgloss.NewStyle().Foreground(ColorCyan)
	StatusDone     = lipgloss.NewStyle().Foreground(ColorGreen)
	HelpBar        = lipgloss.NewStyle().Foreground(ColorFaintGray)
	HelpKey        = lipgloss.NewStyle().Foreground(ColorGray).Bold(true)
	HelpOverlay    = lipgloss.NewStyle().Padding(1, 2)
	Progress       = lipgloss.NewStyle().Foreground(ColorGreen)
	ProgressFull   = lipgloss.NewStyle().Foreground(ColorGreen)
	ProgressEmpty  = lipgloss.NewStyle().Foreground(ColorDimGray)
	FlashStyle     = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	InputLabel     = lipgloss.NewStyle().Foreground(ColorPurple).Bold(true)
	EmptyState     = lipgloss.NewStyle().Foreground(ColorGray).Italic(true)
	FilterBadge    = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
)

// AlignLR left-right aligns two strings within a given width.
func AlignLR(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// HelpItem represents a key-action pair for the help bar.
type HelpItem struct {
	Key  string
	Desc string
}

// RenderProgressBar renders: 3/8 ████░░░░
func RenderProgressBar(done, total, barWidth int) string {
	if total == 0 {
		return ""
	}
	label := Progress.Render(fmt.Sprintf("%d/%d", done, total))
	ratio := float64(done) / float64(total)
	filled := int(ratio * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled
	bar := ProgressFull.Render(strings.Repeat("█", filled)) +
		ProgressEmpty.Render(strings.Repeat("░", empty))
	return label + " " + bar
}
