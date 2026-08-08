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

	KeyQuit   = "q"
	KeyHelp   = "?"
	KeyDone   = "d"
	KeyAdd    = "a"
	KeyEdit   = "e"
	KeyDelete = "x"
	// KeyStatus is the space bar; bubbletea reports it as a single space.
	KeyStatus = " "
	KeyPrio   = "p"
	KeyUndo   = "u"
	KeyOpen   = "o"

	KeyFilterAll    = "1"
	KeyFilterActive = "2"
	KeyFilterDone   = "3"
	KeySearch       = "/"
	KeyTag          = "t"
	KeyFilterTag    = "f"
)
