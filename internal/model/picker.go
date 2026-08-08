package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/macone/todo-cli/internal/ui"
)

// pickerRows is how many headings the popup shows before it scrolls.
const pickerRows = 12

// pickerAction is what choosing an entry does.
type pickerAction int

const (
	pickerJump pickerAction = iota
	pickerMove
)

// headerEntry is one heading offered by the picker.
type headerEntry struct {
	itemIndex int // position in the list's items, for jumping
	lineIndex int // position in TaskFile.Lines, for moving a task under it
	level     int
	name      string
}

// headerPicker is the ctrl+e / M popup: every heading in the file, filterable
// by typing, navigable with ctrl+j and ctrl+k.
type headerPicker struct {
	action  pickerAction
	entries []headerEntry
	// matches indexes into entries, narrowed by the query.
	matches []int
	cursor  int
	offset  int
	input   textinput.Model
	// taskIndex is the task being moved, for the move action.
	taskIndex int
}

func newHeaderPicker(entries []headerEntry, action pickerAction, taskIndex int) headerPicker {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "filter…"
	ti.CharLimit = 128
	ti.Focus()

	p := headerPicker{
		action:    action,
		entries:   entries,
		input:     ti,
		taskIndex: taskIndex,
	}
	p.refilter()
	return p
}

// refilter narrows the entries to those matching the query, keeping the cursor
// in range.
func (p *headerPicker) refilter() {
	query := strings.ToLower(strings.TrimSpace(p.input.Value()))

	p.matches = p.matches[:0]
	for i, e := range p.entries {
		if query == "" || strings.Contains(strings.ToLower(e.name), query) {
			p.matches = append(p.matches, i)
		}
	}

	if p.cursor >= len(p.matches) {
		p.cursor = len(p.matches) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	p.scrollToCursor()
}

func (p *headerPicker) scrollToCursor() {
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+pickerRows {
		p.offset = p.cursor - pickerRows + 1
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

func (p *headerPicker) move(delta int) {
	if len(p.matches) == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.matches) {
		p.cursor = len(p.matches) - 1
	}
	p.scrollToCursor()
}

// selected returns the highlighted entry, if there is one.
func (p headerPicker) selected() (headerEntry, bool) {
	if p.cursor < 0 || p.cursor >= len(p.matches) {
		return headerEntry{}, false
	}
	return p.entries[p.matches[p.cursor]], true
}

// pickerResult says how a key was handled.
type pickerResult int

const (
	pickerContinue pickerResult = iota
	pickerCancelled
	pickerChosen
)

// handleKey feeds a key to the picker. Navigation is matched on key type where
// a name could be spelled by ordinary text.
func (p *headerPicker) handleKey(msg tea.KeyMsg) pickerResult {
	switch msg.Type {
	case tea.KeyEnter:
		if _, ok := p.selected(); ok {
			return pickerChosen
		}
		return pickerContinue
	case tea.KeyEsc:
		return pickerCancelled
	}

	switch msg.String() {
	case ui.KeyMajorSectionNext, "down":
		p.move(1)
		return pickerContinue
	case ui.KeyMajorSectionPrev, "up":
		p.move(-1)
		return pickerContinue
	}

	before := p.input.Value()
	p.input, _ = p.input.Update(msg)
	if p.input.Value() != before {
		p.refilter()
	}
	return pickerContinue
}

// depths converts heading levels into nesting depths, so a file that jumps
// from "##" straight to "####" still draws as one step in.
func depths(levels []int) []int {
	out := make([]int, len(levels))
	var stack []int // levels of the current ancestors

	for i, level := range levels {
		for len(stack) > 0 && stack[len(stack)-1] >= level {
			stack = stack[:len(stack)-1]
		}
		out[i] = len(stack)
		stack = append(stack, level)
	}
	return out
}

// treePrefix draws the indent guides leading to row i.
func treePrefix(d []int, i int) string {
	if d[i] == 0 {
		return ""
	}

	// Does the ancestor at depth level have another child after row i?
	continues := func(level int) bool {
		for j := i + 1; j < len(d); j++ {
			if d[j] == level {
				return true
			}
			if d[j] < level {
				return false
			}
		}
		return false
	}

	var sb strings.Builder
	for level := range d[i] - 1 {
		if continues(level + 1) {
			sb.WriteString("│  ")
		} else {
			sb.WriteString("   ")
		}
	}
	if continues(d[i]) {
		sb.WriteString("├─ ")
	} else {
		sb.WriteString("╰─ ")
	}
	return sb.String()
}

// View renders the popup.
func (p headerPicker) View() string {
	const inner = 56

	title := "Jump to heading"
	if p.action == pickerMove {
		title = "Move task to heading"
	}

	accent := lipgloss.NewStyle().Foreground(ui.ColorPickerAccent)
	count := accent.Render(fmt.Sprintf("%d/%d", len(p.matches), len(p.entries)))
	search := accent.Render("> ") + p.input.View()

	var rows []string
	rows = append(rows, ui.AlignLR(search, count, inner))
	rows = append(rows, lipgloss.NewStyle().
		Foreground(ui.ColorDimGray).
		Render(strings.Repeat("─", inner)))

	if len(p.matches) == 0 {
		rows = append(rows, ui.EmptyState.Render("no matching headings"))
	}

	// Guides are computed over the visible set, so filtering yields a tree of
	// what is actually on screen rather than dangling branches.
	levels := make([]int, len(p.matches))
	for i, m := range p.matches {
		levels[i] = p.entries[m].level
	}
	d := depths(levels)

	end := min(p.offset+pickerRows, len(p.matches))
	for i := p.offset; i < end; i++ {
		e := p.entries[p.matches[i]]
		text := treePrefix(d, i) + e.name

		if i == p.cursor {
			// One style for the whole row: nesting a coloured name inside a
			// background would cut the highlight short at its reset.
			rows = append(rows, lipgloss.NewStyle().
				Background(ui.ColorPickerSel).
				Foreground(ui.ColorPickerAccent).
				Bold(true).
				Width(inner).
				Render(text))
			continue
		}
		rows = append(rows, lipgloss.NewStyle().
			Foreground(ui.ColorDimGray).Render(treePrefix(d, i))+
			ui.HeadingStyle(e.level).Render(e.name))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPickerAccent).
		Padding(0, 1).
		Render(strings.Join(rows, "\n"))

	// Sit the title in the top border, as the reference does.
	label := accent.Render(" " + title + " ")
	x := (lipgloss.Width(box) - lipgloss.Width(label)) / 2
	return overlay(box, label, x, 0)
}
