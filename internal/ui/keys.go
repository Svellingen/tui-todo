package ui

// Key constants for the TUI.
const (
	KeyUp   = "k"
	KeyDown = "j"

	// Arrow keys are accepted everywhere the vim-style keys move the cursor.
	KeyArrowUp   = "up"
	KeyArrowDown = "down"

	KeySectionUp   = "K"
	KeySectionDown = "J"

	// Same jump, on the keys most people reach for.
	KeySectionNext = "tab"
	KeySectionPrev = "shift+tab"

	// vim-style paging. KeyTop is a prefix: pressed twice it jumps to the top.
	KeyTop      = "g"
	KeyBottom   = "G"
	KeyPageDown = "ctrl+d"
	KeyPageUp   = "ctrl+u"

	// Reorder the selected task within its sort group.
	KeyMoveUp   = "alt+k"
	KeyMoveDown = "alt+j"

	// Move the selected task into the next / previous heading.
	KeyMoveToNextSection = "alt+J"
	KeyMoveToPrevSection = "alt+K"

	// Step between "##" headings only, skipping deeper sub-headings. ctrl+j is
	// LF and ctrl+k is VT, both distinct from Enter (CR), so neither collides.
	KeyMajorSectionNext = "ctrl+j"
	KeyMajorSectionPrev = "ctrl+k"

	// Move between heading levels. ctrl+h is BS (8), distinct from Backspace
	// (DEL 127), so it does not disturb the text input.
	KeyParentSection = "ctrl+h"
	KeyChildSection  = "ctrl+l"

	KeyQuit   = "q"
	KeyHelp   = "?"
	KeyDone   = "d"
	KeyAdd    = "a"
	KeyEdit   = "e"
	KeyDelete = "x"
	// KeyStatus is the space bar; bubbletea reports it as a single space.
	KeyStatus   = " "
	KeyPrio     = "p"
	KeyPrioDown = "P"
	KeyUndo     = "u"
	KeyOpen     = "o"

	// Toggle the task file's path in the title bar.
	KeyToggleFilename = "i"

	KeyFilterAll    = "1"
	KeyFilterActive = "2"
	KeyFilterDone   = "3"
	KeySearch       = "/"
	KeyTag          = "t"
	KeyFilterTag    = "f"
)
