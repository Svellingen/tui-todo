package model

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// visible strips styling so assertions read against what lands on screen.
func visible(s string) []string {
	lines := strings.Split(ansi.Strip(s), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return lines
}

func TestOverlayKeepsSurroundingText(t *testing.T) {
	base := strings.Join([]string{
		"aaaaaaaaaa",
		"bbbbbbbbbb",
		"cccccccccc",
		"dddddddddd",
	}, "\n")
	box := "XX\nYY"

	got := visible(overlay(base, box, 4, 1))
	want := []string{
		"aaaaaaaaaa",
		"bbbbXXbbbb",
		"ccccYYcccc",
		"dddddddddd",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// Splicing must respect cell positions rather than byte offsets, so a styled
// base keeps both its colours and its layout.
func TestOverlayDoesNotCorruptStyledBase(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	base := red.Render("aaaaaaaaaa") + "\n" + red.Render("bbbbbbbbbb")

	out := overlay(base, "XX", 4, 1)

	got := visible(out)
	if got[0] != "aaaaaaaaaa" {
		t.Errorf("untouched line changed: %q", got[0])
	}
	if got[1] != "bbbbXXbbbb" {
		t.Errorf("spliced line: got %q", got[1])
	}
	// The base's colour must still be present on the parts either side.
	if !strings.Contains(out, "\x1b[") {
		t.Error("expected styling to survive the splice")
	}
}

// A box reaching past the bottom extends the canvas instead of being dropped.
func TestOverlayGrowsForTallBox(t *testing.T) {
	got := visible(overlay("aaaa\nbbbb", "XX\nYY\nZZ", 1, 1))
	want := []string{"aaaa", "bXXb", " YY", " ZZ"}

	if len(got) != len(want) {
		t.Fatalf("expected %d lines, got %d: %q", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// A base line shorter than the overlay position is padded out to it, so the
// box lines up with the rows above instead of sliding into the gap -- and the
// short line's own content survives.
func TestOverlayPadsShortBaseLines(t *testing.T) {
	got := visible(overlay("ab\nabcdefgh", "XX\nXX", 4, 0))
	if got[0] != "ab  XX" {
		t.Errorf("short line: got %q, want %q", got[0], "ab  XX")
	}
	if got[1] != "abcdXXgh" {
		t.Errorf("full line: got %q", got[1])
	}
}

func TestCenterOverlayCentres(t *testing.T) {
	base := strings.Join([]string{
		"..........",
		"..........",
		"..........",
		"..........",
		"..........",
	}, "\n")

	got := visible(centerOverlay(base, "XX\nXX", 10, 5))
	// (10-2)/2 = 4 across, (5-2)/2 = 1 down.
	if got[1] != "....XX...." || got[2] != "....XX...." {
		t.Errorf("got %q", got)
	}
	if got[0] != ".........." || got[3] != ".........." {
		t.Errorf("rows outside the box changed: %q", got)
	}
}

// A box larger than the canvas is pinned to the origin rather than given a
// negative offset.
func TestCenterOverlayClampsOversizedBox(t *testing.T) {
	got := visible(centerOverlay("....", "XXXXXXXX", 4, 1))
	if got[0] != "XXXXXXXX" {
		t.Errorf("got %q", got[0])
	}
}

func TestLineBoundsMeasuresStyledText(t *testing.T) {
	styled := lipgloss.NewStyle().Bold(true).Render("hello")
	w, h := lineBounds(styled + "\n" + "hi")
	if w != 5 || h != 2 {
		t.Errorf("got %dx%d, want 5x2", w, h)
	}
}
