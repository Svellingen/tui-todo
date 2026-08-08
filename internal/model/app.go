package model

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/macone/todo-cli/internal/storage"
	"github.com/macone/todo-cli/internal/task"
	"github.com/macone/todo-cli/internal/ui"
)

const maxUndoStack = 20

// watchInterval is how often the task file is polled for outside edits. This
// is only the fallback for when a filesystem watch cannot be established.
const watchInterval = time.Second

// fileStamp is a cheap fingerprint of the task file, used to skip parsing when
// nothing has changed. The zero value stands for "file does not exist".
type fileStamp struct {
	modUnixNano int64
	size        int64
}

func statFile(path string) (fileStamp, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileStamp{}, err
	}
	return fileStamp{modUnixNano: info.ModTime().UnixNano(), size: info.Size()}, nil
}

type appMode int

const (
	modeNormal appMode = iota
	modeInput
	modeConfirmDelete
	modeTagSelect
	modeHelp
)

type undoEntry struct {
	content string
	desc    string
}

type flashMsg struct{}

type App struct {
	store    *storage.Store
	taskFile *storage.TaskFile
	list     TaskListModel
	input    TaskInputModel
	mode     appMode
	width    int
	height   int
	err      error

	undoStack  []undoEntry
	statusMsg  string
	tagOptions []string
	tagCursor  int

	// stamp is the fingerprint of the task file as the app last saw it, so an
	// outside edit can be told from the app's own writes.
	stamp fileStamp

	// watcher is nil when a filesystem watch could not be established, in
	// which case the app falls back to polling.
	watcher *fileWatcher

	// pendingReload records a change that arrived while a modal was open, to
	// be applied once the app is back at the list.
	pendingReload bool

	// showFilename swaps the title bar between the app name and the task
	// file's path. Off by default; toggled with "i" and not persisted.
	showFilename bool
}

func NewApp(store *storage.Store) App {
	return App{
		store: store,
		input: NewTaskInput(),
	}
}

func (a App) Init() tea.Cmd {
	return func() tea.Msg {
		tf, err := a.store.Load()
		if err != nil {
			return errMsg{err}
		}
		stamp, _ := statFile(a.store.FilePath)
		return loadedMsg{tf: tf, stamp: stamp}
	}
}

type loadedMsg struct {
	tf    *storage.TaskFile
	stamp fileStamp
}
type errMsg struct{ err error }

// watchTickMsg drives the fallback poll for outside edits.
type watchTickMsg struct{}

// fileChangedMsg reports that the filesystem watch saw the task file change.
type fileChangedMsg struct{}

// startWatch establishes a filesystem watch, falling back to polling when one
// is unavailable -- an exhausted inotify limit, or a filesystem that does not
// support notifications.
func (a *App) startWatch() tea.Cmd {
	w, err := newFileWatcher(a.store.FilePath)
	if err != nil {
		return watchTick()
	}
	a.watcher = w
	return waitForChange(w)
}

// waitForChange blocks in a command until the next change to the task file.
// Each change re-arms it, so exactly one of these is in flight at a time.
func waitForChange(w *fileWatcher) tea.Cmd {
	return func() tea.Msg {
		if !w.wait() {
			return nil
		}
		return fileChangedMsg{}
	}
}

// reloadMsg carries the result of that check. A nil tf means the file changed
// in a way that needs no new content -- only the stamp is refreshed.
type reloadMsg struct {
	tf    *storage.TaskFile
	stamp fileStamp
}

func watchTick() tea.Cmd {
	return tea.Tick(watchInterval, func(time.Time) tea.Msg { return watchTickMsg{} })
}

// checkReload compares the file on disk against the stamp the app last saw and
// parses it when they differ. The app's own saves also move the stamp, so the
// freshly parsed content is compared against what the app would write; if they
// match, this was our own write and the list is left untouched.
func (a App) checkReload() tea.Cmd {
	s := a.store
	known := a.stamp
	current := storage.NewWriter().Write(a.taskFile)

	return func() tea.Msg {
		stamp, err := statFile(s.FilePath)
		if err != nil {
			// Gone or unreadable: keep what is in memory rather than blanking
			// the list, but record the absence so a recreate is picked up.
			if known == (fileStamp{}) {
				return nil
			}
			return reloadMsg{stamp: fileStamp{}}
		}
		if stamp == known {
			return nil
		}

		tf, err := s.Load()
		if err != nil {
			return errMsg{err}
		}
		if storage.NewWriter().Write(tf) == current {
			return reloadMsg{stamp: stamp}
		}
		return reloadMsg{tf: tf, stamp: stamp}
	}
}

func (a *App) pushUndo(desc string) {
	a.pushUndoContent(storage.NewWriter().Write(a.taskFile), desc)
}

// pushUndoContent records an already-captured snapshot, for callers that must
// decide whether a change happened before committing it to the undo stack.
func (a *App) pushUndoContent(content, desc string) {
	a.undoStack = append(a.undoStack, undoEntry{content: content, desc: desc})
	if len(a.undoStack) > maxUndoStack {
		a.undoStack = a.undoStack[1:]
	}
}

// save writes the in-memory state to disk.
//
// If the file changed underneath since the app last read it, the outside edit
// wins: rather than overwriting someone else's work, the app reloads and
// reports that the in-app change was not applied.
//
// This runs synchronously rather than in a tea.Cmd so the stamp bookkeeping
// cannot interleave with the watcher or with a second save; the task file is
// small enough that the write is not worth deferring.
func (a *App) save() tea.Cmd {
	if stamp, err := statFile(a.store.FilePath); err == nil && stamp != a.stamp {
		disk, loadErr := a.store.Load()
		if loadErr != nil {
			a.err = loadErr
			return nil
		}
		a.stamp = stamp
		a.taskFile = disk
		a.list.Reload(a.taskFile)
		a.undoStack = nil
		msg, cmd := flash("File changed on disk — reloaded, change not applied")
		a.statusMsg = msg
		return cmd
	}

	// Every write leaves the file in sorted order, so no mutation path can
	// forget to re-sort after changing a task's status or priority.
	if a.taskFile.Sort() {
		a.list.RefreshOrder()
	}

	if err := a.store.Save(a.taskFile); err != nil {
		a.err = err
		return nil
	}
	a.stamp, _ = statFile(a.store.FilePath)
	return nil
}

// normalizeOrder sorts freshly loaded content and writes it back when the
// stored order differed, keeping the file canonical.
func (a *App) normalizeOrder() tea.Cmd {
	if !a.taskFile.Sort() {
		return nil
	}
	a.list.RefreshOrder()
	return a.save()
}

// moveTask shifts the selected task within its sort group.
func (a App) moveTask(delta int) (tea.Model, tea.Cmd) {
	idx := a.list.SelectedTaskIndex()
	if idx < 0 {
		return a, nil
	}

	// Snapshot before mutating: a blocked move should not land on the undo
	// stack.
	before := storage.NewWriter().Write(a.taskFile)
	if !a.taskFile.MoveTask(idx, delta) {
		return a, nil
	}
	a.pushUndoContent(before, "move task")
	a.list.RefreshOrder()
	return a, a.save()
}

func flash(msg string) (string, tea.Cmd) {
	return msg, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return flashMsg{}
	})
}

// chromeHeight is the number of rows the view spends on everything that is not
// the task list: a leading blank, the title, a blank, a trailing blank and the
// status line, plus whatever the active mode adds.
//
// Both the layout and the scroll maths derive the viewport from this, so they
// cannot drift apart.
func (a App) chromeHeight() int {
	overhead := 5
	if a.mode == modeInput {
		overhead++
	}
	if a.mode == modeTagSelect && len(a.tagOptions) > 0 {
		overhead += len(a.tagOptions) + 1
	}
	return overhead
}

func (a *App) updateListSize() {
	h := a.height - a.chromeHeight()
	if h < 3 {
		h = 3
	}
	w := a.width
	if w < 20 {
		w = 20
	}
	a.list.SetSize(w, h)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg:
		a.taskFile = msg.tf
		a.stamp = msg.stamp
		a.list = NewTaskListModel(a.taskFile)
		a.updateListSize()
		// Start watching only once there is content to compare against.
		return a, tea.Batch(a.normalizeOrder(), a.startWatch())

	case fileChangedMsg:
		next := waitForChange(a.watcher)
		// Skip the reload while a modal is open: an in-flight edit refers to a
		// task index that may not survive it. Remember the change so it lands
		// as soon as the app is back at the list.
		if a.mode != modeNormal {
			a.pendingReload = true
			return a, next
		}
		return a, tea.Batch(next, a.checkReload())

	case watchTickMsg:
		if a.mode != modeNormal {
			a.pendingReload = true
			return a, watchTick()
		}
		return a, tea.Batch(watchTick(), a.checkReload())

	case reloadMsg:
		a.stamp = msg.stamp
		if msg.tf == nil {
			return a, nil
		}
		a.taskFile = msg.tf
		a.list.Reload(a.taskFile)
		// The snapshots describe a file that no longer exists on disk;
		// replaying them would silently discard the outside edit.
		a.undoStack = nil
		a.updateListSize()
		var cmd tea.Cmd
		a.statusMsg, cmd = flash("Reloaded from disk")
		return a, tea.Batch(cmd, a.normalizeOrder())

	case editorFinishedMsg:
		if msg.err != nil {
			a.err = msg.err
			return a, nil
		}
		// The watch may already have reported the editor's save while the app
		// was suspended; checking again is harmless when nothing changed.
		a.pendingReload = false
		return a, a.checkReload()

	case errMsg:
		a.err = msg.err
		return a, nil

	case flashMsg:
		a.statusMsg = ""
		return a, nil

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateListSize()
		return a, nil

	case tea.KeyMsg:
		next, cmd := a.handleKey(msg)
		updated, ok := next.(App)
		if !ok {
			return next, cmd
		}
		// A change seen while a modal was open applies as soon as it closes.
		if updated.pendingReload && updated.mode == modeNormal {
			updated.pendingReload = false
			return updated, tea.Batch(cmd, updated.checkReload())
		}
		return updated, cmd
	}

	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if a.mode == modeConfirmDelete {
		switch key {
		case "y":
			return a.doDelete()
		default:
			a.mode = modeNormal
			a.statusMsg = ""
			a.updateListSize()
			return a, nil
		}
	}

	if a.mode == modeHelp {
		switch key {
		case ui.KeyHelp, "esc":
			a.mode = modeNormal
		}
		return a, nil
	}

	if a.mode == modeTagSelect {
		switch key {
		case ui.KeyDown, ui.KeyArrowDown:
			if a.tagCursor < len(a.tagOptions)-1 {
				a.tagCursor++
			}
		case ui.KeyUp, ui.KeyArrowUp:
			if a.tagCursor > 0 {
				a.tagCursor--
			}
		case "enter":
			if a.tagCursor >= 0 && a.tagCursor < len(a.tagOptions) {
				a.list.SetTagFilter(a.tagOptions[a.tagCursor])
			}
			a.mode = modeNormal
			a.tagOptions = nil
			a.updateListSize()
		case "esc":
			a.mode = modeNormal
			a.tagOptions = nil
			a.updateListSize()
		}
		return a, nil
	}

	if a.mode == modeInput {
		switch key {
		case "enter":
			return a.commitInput()
		case "esc":
			if a.input.Mode() == inputSearch {
				a.list.ClearSearch()
			}
			a.input.Cancel()
			a.mode = modeNormal
			a.updateListSize()
			return a, nil
		default:
			cmd := a.input.Update(msg)
			if a.input.Mode() == inputSearch {
				a.list.SetSearchQuery(a.input.Value())
			}
			return a, cmd
		}
	}

	switch key {
	case "ctrl+c":
		return a, tea.Quit
	case ui.KeyQuit:
		return a, tea.Quit
	case ui.KeyDown, ui.KeyArrowDown:
		a.list.MoveDown()
	case ui.KeyUp, ui.KeyArrowUp:
		a.list.MoveUp()
	case ui.KeySectionDown, ui.KeySectionNext:
		a.list.JumpNextSection()
	case ui.KeySectionUp, ui.KeySectionPrev:
		a.list.JumpPrevSection()
	case ui.KeyMoveDown:
		return a.moveTask(1)
	case ui.KeyMoveUp:
		return a.moveTask(-1)
	case ui.KeyDone:
		return a.toggleDone()
	case ui.KeyAdd:
		a.mode = modeInput
		a.input.StartAdd()
		a.updateListSize()
		return a, nil
	case ui.KeyEdit:
		return a.startEdit()
	case ui.KeyDelete:
		return a.confirmDelete()
	case ui.KeyStatus:
		return a.cycleStatus()
	case ui.KeyPrio:
		return a.cyclePriority()
	case ui.KeyUndo:
		return a.undo()
	case ui.KeyOpen:
		return a.openInEditor()
	case ui.KeyToggleFilename:
		a.showFilename = !a.showFilename
		return a, nil
	case ui.KeyFilterAll:
		a.list.SetStatusFilter(filterAll)
	case ui.KeyFilterActive:
		a.list.SetStatusFilter(filterActive)
	case ui.KeyFilterDone:
		a.list.SetStatusFilter(filterDone)
	case ui.KeySearch:
		a.mode = modeInput
		a.input.StartSearch()
		a.updateListSize()
		return a, nil
	case ui.KeyTag:
		return a.addTag()
	case ui.KeyFilterTag:
		return a.openTagFilter()
	case ui.KeyHelp:
		a.mode = modeHelp
		return a, nil
	}

	return a, nil
}

// headerPath renders the task file's location for the title bar: absolute,
// with $HOME collapsed to "~", elided from the left when it does not fit so
// the filename itself always stays visible.
func headerPath(path string, width int) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, relErr := filepath.Rel(home, path); relErr == nil && !strings.HasPrefix(rel, "..") {
			path = filepath.Join("~", rel)
		}
	}

	if width <= 0 || lipgloss.Width(path) <= width {
		return path
	}

	runes := []rune(path)
	keep := width - 1 // one cell for the ellipsis
	if keep < 1 {
		return "…"
	}
	if keep > len(runes) {
		keep = len(runes)
	}
	return "…" + string(runes[len(runes)-keep:])
}

// editorFinishedMsg reports that the editor subprocess exited.
type editorFinishedMsg struct{ err error }

// editorArgv picks the editor to hand the task file to, following the usual
// convention: $VISUAL, then $EDITOR, then a sensible default. Either variable
// may carry arguments, as in "code -w", so they are split.
func editorArgv() []string {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			if fields := strings.Fields(v); len(fields) > 0 {
				return fields
			}
		}
	}
	if _, err := exec.LookPath("nvim"); err == nil {
		return []string{"nvim"}
	}
	return []string{"vi"}
}

// openInEditor hands the terminal over to an editor for the task file and
// reloads once it exits.
func (a App) openInEditor() (tea.Model, tea.Cmd) {
	argv := append(editorArgv(), a.store.FilePath)
	c := exec.Command(argv[0], argv[1:]...)
	return a, tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

func (a App) toggleDone() (tea.Model, tea.Cmd) {
	t := a.list.SelectedTaskItem()
	if t == nil {
		return a, nil
	}
	a.pushUndo("toggle done")
	if t.Status == task.StatusDone {
		t.Status = task.StatusTodo
	} else {
		t.Status = task.StatusDone
	}
	a.statusMsg, _ = flash("Done!")
	cmd := a.save()
	return a, tea.Batch(cmd, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return flashMsg{} }))
}

func (a App) startEdit() (tea.Model, tea.Cmd) {
	idx := a.list.SelectedTaskIndex()
	if idx < 0 {
		return a, nil
	}
	a.mode = modeInput
	a.input.StartEdit(idx, a.taskFile.Tasks[idx].Title)
	a.updateListSize()
	return a, nil
}

func (a App) commitInput() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(a.input.Value())
	if value == "" {
		a.input.Cancel()
		a.mode = modeNormal
		a.updateListSize()
		return a, nil
	}

	a.pushUndo("input")

	switch a.input.Mode() {
	case inputAdd:
		a.addTask(value)
		var msg string
		var cmd tea.Cmd
		msg, cmd = flash("Added: " + value)
		a.statusMsg = msg
		a.input.Cancel()
		a.mode = modeNormal
		a.updateListSize()
		return a, tea.Batch(a.save(), cmd)

	case inputEdit:
		idx := a.input.EditIndex()
		if idx >= 0 && idx < len(a.taskFile.Tasks) {
			title, priority, tags, dueDate := storage.ParseMetadata(value)
			a.taskFile.Tasks[idx].Title = title
			if priority != task.PriorityNone {
				a.taskFile.Tasks[idx].Priority = priority
			}
			if len(tags) > 0 {
				a.taskFile.Tasks[idx].Tags = tags
			}
			if dueDate != nil {
				a.taskFile.Tasks[idx].DueDate = dueDate
			}
		}
		a.input.Cancel()
		a.mode = modeNormal
		a.updateListSize()
		return a, a.save()

	case inputSearch:
		a.input.Cancel()
		a.mode = modeNormal
		a.updateListSize()
		return a, nil

	case inputTag:
		idx := a.list.SelectedTaskIndex()
		if idx >= 0 && idx < len(a.taskFile.Tasks) {
			a.pushUndo("add tag")
			a.taskFile.Tasks[idx].Tags = append(a.taskFile.Tasks[idx].Tags, value)
		}
		a.input.Cancel()
		a.mode = modeNormal
		a.updateListSize()
		return a, a.save()
	}

	a.input.Cancel()
	a.mode = modeNormal
	a.updateListSize()
	return a, nil
}

func (a *App) addTask(value string) {
	title, priority, tags, dueDate := storage.ParseMetadata(value)
	newTask := task.Task{
		Title:    title,
		Status:   task.StatusTodo,
		Priority: priority,
		Tags:     tags,
		DueDate:  dueDate,
	}
	a.taskFile.Tasks = append(a.taskFile.Tasks, newTask)

	newLine := storage.Line{
		Type:      storage.LineTask,
		TaskIndex: len(a.taskFile.Tasks) - 1,
	}

	inserted := false
	for i, line := range a.taskFile.Lines {
		if line.Type == storage.LineSection {
			for _, sec := range a.taskFile.Sections {
				if sec.Line == line.Number && sec.Status == task.StatusTodo {
					insertIdx := i + 1
					for insertIdx < len(a.taskFile.Lines) &&
						a.taskFile.Lines[insertIdx].Type == storage.LineText &&
						strings.TrimSpace(a.taskFile.Lines[insertIdx].Raw) == "" {
						insertIdx++
					}
					a.taskFile.Lines = append(a.taskFile.Lines[:insertIdx],
						append([]storage.Line{newLine}, a.taskFile.Lines[insertIdx:]...)...)
					inserted = true
					break
				}
			}
			if inserted {
				break
			}
		}
	}
	if !inserted {
		a.taskFile.Lines = append(a.taskFile.Lines, newLine)
	}

	a.list.rebuildItems()
}

func (a App) confirmDelete() (tea.Model, tea.Cmd) {
	if a.list.SelectedTaskItem() == nil {
		return a, nil
	}
	a.mode = modeConfirmDelete
	a.statusMsg = "Delete? y/n"
	return a, nil
}

func (a App) doDelete() (tea.Model, tea.Cmd) {
	idx := a.list.SelectedTaskIndex()
	if idx < 0 {
		a.mode = modeNormal
		a.statusMsg = ""
		return a, nil
	}

	a.pushUndo("delete")
	title := a.taskFile.Tasks[idx].Title

	a.taskFile.Tasks = append(a.taskFile.Tasks[:idx], a.taskFile.Tasks[idx+1:]...)

	newLines := make([]storage.Line, 0, len(a.taskFile.Lines))
	for _, line := range a.taskFile.Lines {
		if line.Type == storage.LineTask && line.TaskIndex == idx {
			continue
		}
		if line.Type == storage.LineTask && line.TaskIndex > idx {
			line.TaskIndex--
		}
		newLines = append(newLines, line)
	}
	a.taskFile.Lines = newLines

	a.list.rebuildItems()
	a.mode = modeNormal

	var msg string
	var cmd tea.Cmd
	msg, cmd = flash("Deleted: " + title)
	a.statusMsg = msg
	return a, tea.Batch(a.save(), cmd)
}

func (a App) addTag() (tea.Model, tea.Cmd) {
	if a.list.SelectedTaskItem() == nil {
		return a, nil
	}
	a.mode = modeInput
	a.input.StartTag()
	a.updateListSize()
	return a, nil
}

func (a App) openTagFilter() (tea.Model, tea.Cmd) {
	if a.list.TagFilter() != "" {
		a.list.ClearTagFilter()
		return a, nil
	}

	tags := a.list.AllTags()
	if len(tags) == 0 {
		a.statusMsg = "No tags found"
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return flashMsg{} })
	}

	a.mode = modeTagSelect
	a.tagOptions = tags
	a.tagCursor = 0
	a.updateListSize()
	return a, nil
}

func (a App) cycleStatus() (tea.Model, tea.Cmd) {
	t := a.list.SelectedTaskItem()
	if t == nil {
		return a, nil
	}
	a.pushUndo("cycle status")
	switch t.Status {
	case task.StatusTodo:
		t.Status = task.StatusInProgress
	case task.StatusInProgress:
		t.Status = task.StatusDone
	case task.StatusDone:
		t.Status = task.StatusTodo
	}
	return a, a.save()
}

func (a App) cyclePriority() (tea.Model, tea.Cmd) {
	t := a.list.SelectedTaskItem()
	if t == nil {
		return a, nil
	}
	a.pushUndo("cycle priority")
	switch t.Priority {
	case task.PriorityNone:
		t.Priority = task.PriorityMedium
	case task.PriorityMedium:
		t.Priority = task.PriorityHigh
	case task.PriorityHigh:
		t.Priority = task.PriorityNone
	}
	return a, a.save()
}

func (a App) undo() (tea.Model, tea.Cmd) {
	if len(a.undoStack) == 0 {
		a.statusMsg = "Nothing to undo"
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return flashMsg{} })
	}

	entry := a.undoStack[len(a.undoStack)-1]
	a.undoStack = a.undoStack[:len(a.undoStack)-1]

	p := storage.NewParser()
	tf, err := p.Parse(entry.content)
	if err != nil {
		a.err = err
		return a, nil
	}
	a.taskFile = tf
	a.list = NewTaskListModel(a.taskFile)
	a.updateListSize()

	var msg string
	var cmd tea.Cmd
	msg, cmd = flash("Undone: " + entry.desc)
	a.statusMsg = msg
	return a, tea.Batch(a.save(), cmd)
}

// ─── View ──────────────────────────────────────────────────────────────────

func (a App) View() string {
	if a.err != nil {
		return "Error: " + a.err.Error()
	}
	if a.taskFile == nil {
		return "Loading..."
	}

	// Help overlay takes over the full screen
	if a.mode == modeHelp {
		return a.renderHelpOverlay()
	}

	w := a.width
	if w < 40 {
		w = 80
	}
	h := a.height
	if h < 10 {
		h = 24
	}

	innerWidth := w

	var lines []string

	// Leading blank line
	lines = append(lines, "")

	// Title bar: ✦ ~/work/proj/tasks.md      [filter] 3/8 ████░░░░
	var titleRight string
	done, total := a.list.ProgressCounts()
	if total > 0 {
		titleRight = ui.RenderProgressBar(done, total, 8)
	}

	var filterPart string
	if fi := a.list.FilterIndicator(); fi != "" {
		filterPart = "  " + ui.FilterBadge.Render(fi)
	}

	label := "todo"
	if a.showFilename {
		// The path gets whatever the progress bar, the filter badge, the "✦ "
		// prefix and a two-cell gap leave behind.
		budget := innerWidth - lipgloss.Width(titleRight) - lipgloss.Width(filterPart) - 4
		label = headerPath(a.store.FilePath, budget)
	}
	titleLeft := ui.TitleStyle.Render("✦ "+label) + filterPart

	lines = append(lines, ui.AlignLR(titleLeft, titleRight, innerWidth))

	// Blank line
	lines = append(lines, "")

	// Content area
	contentHeight := h - a.chromeHeight()
	if contentHeight < 3 {
		contentHeight = 3
	}

	contentLines := a.list.ViewLines(innerWidth, contentHeight)
	lines = append(lines, contentLines...)

	// Trailing blank
	lines = append(lines, "")

	// Input area (if active)
	if a.mode == modeInput {
		var prefix string
		switch a.input.Mode() {
		case inputAdd:
			prefix = ui.InputLabel.Render("  Add: ")
		case inputEdit:
			prefix = ui.InputLabel.Render("  Edit: ")
		case inputSearch:
			prefix = ui.InputLabel.Render("  / ")
		case inputTag:
			prefix = ui.InputLabel.Render("  Tag: ")
		}
		lines = append(lines, prefix+a.input.View())
	}

	// Tag selector (if active)
	if a.mode == modeTagSelect && len(a.tagOptions) > 0 {
		lines = append(lines, "  "+ui.InputLabel.Render("Filter by tag:"))
		for i, tag := range a.tagOptions {
			cursor := "    "
			if i == a.tagCursor {
				cursor = ui.CursorStyle.Render("  ▸ ")
			}
			lines = append(lines, cursor+ui.Tag.Render("+"+tag))
		}
	}

	// Status line. Kept even when empty so the list does not jump as flash
	// messages come and go.
	status := ""
	if a.statusMsg != "" {
		status = "  " + ui.FlashStyle.Render(a.statusMsg)
	} else if a.mode == modeConfirmDelete {
		status = "  " + ui.PriorityHigh.Render("Delete?") + " " + ui.HelpBar.Render("y/n")
	}
	lines = append(lines, status)

	return strings.Join(lines, "\n")
}

func (a App) renderHelpOverlay() string {
	w := a.width
	h := a.height
	if w < 40 {
		w = 80
	}
	if h < 10 {
		h = 24
	}

	content := helpContent()

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBorder).
		Padding(1, 3)

	box := style.Render(content)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
