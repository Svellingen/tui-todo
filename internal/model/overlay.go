package model

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// reset ends any styling carried over from the text a box was pasted into, so
// the box's own colours start clean and the tail after it does not inherit the
// box's.
const reset = "\x1b[0m"

// overlay draws box on top of base at (x, y), leaving the rest of base
// visible. Both are rendered, styled strings.
//
// Lines are spliced with cell-accurate, escape-aware truncation: slicing them
// by byte or rune index would cut through ANSI sequences and corrupt the
// colours.
func overlay(base, box string, x, y int) string {
	baseLines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")

	// Grow the base if the box reaches past the bottom, so nothing is lost.
	for len(baseLines) < y+len(boxLines) {
		baseLines = append(baseLines, "")
	}

	for i, boxLine := range boxLines {
		row := y + i
		if row < 0 {
			continue
		}

		baseLine := baseLines[row]
		boxWidth := ansi.StringWidth(boxLine)

		// Pad a short base line out to the box's left edge, otherwise the box
		// would slide left into the gap.
		if pad := x - ansi.StringWidth(baseLine); pad > 0 {
			baseLine += strings.Repeat(" ", pad)
		}

		left := ansi.Truncate(baseLine, x, "")
		right := ansi.TruncateLeft(baseLine, x+boxWidth, "")

		baseLines[row] = left + reset + boxLine + reset + right
	}

	return strings.Join(baseLines, "\n")
}

// centerOverlay draws box centred over base.
func centerOverlay(base, box string, width, height int) string {
	boxWidth, boxHeight := lineBounds(box)

	x := (width - boxWidth) / 2
	if x < 0 {
		x = 0
	}
	y := (height - boxHeight) / 2
	if y < 0 {
		y = 0
	}
	return overlay(base, box, x, y)
}

// lineBounds measures a rendered block in cells.
func lineBounds(s string) (width, height int) {
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > width {
			width = w
		}
	}
	return width, len(lines)
}
