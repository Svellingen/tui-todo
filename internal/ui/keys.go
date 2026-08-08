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

	// Reorder the selected task within its sort group.
	KeyMoveUp   = "alt+k"
	KeyMoveDown = "alt+j"

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

	// Toggle the task file's path in the title bar.
	KeyToggleFilename = "i"

	KeyFilterAll    = "1"
	KeyFilterActive = "2"
	KeyFilterDone   = "3"
	KeySearch       = "/"
	KeyTag          = "t"
	KeyFilterTag    = "f"
)
