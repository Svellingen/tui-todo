# todo-cli

A project-local TUI task tracker (Go, bubbletea) that keeps its state in a
markdown file. `README.md` documents every keybinding and behaviour — read it
for *what* the app does. This file covers what you can't infer from the code.

## Commands

`just` is the entrypoint, not make. `just build` `test` `lint` `run` `clean`
`release`.

`just lint` currently fails: `go vet` objects to the unkeyed `ui.HelpItem`
literals in `internal/model/help.go`. That is pre-existing — don't assume you
broke it.

## The one idea that explains the design

**The markdown file is the data model, not a serialisation of it.**

`storage.TaskFile` is `{Tasks, Lines, Sections}`, where `Lines` reference tasks
by index. Reordering means permuting `Line.TaskIndex`; the task values don't
move. Most of `internal/storage` is easier to read once you see that.

Consequences worth knowing before you change anything:

- **Sort order is derived, not stored.** `Sort()` runs on every save (done →
  `!!` → `!` → rest, per section, stable). Hand-ordering a file gets rewritten
  on open. `MoveTask` deliberately refuses moves that cross a rank or a
  section.
- **A task owns its indented lines.** They live in `task.Task.Block` as
  `[]task.BlockLine` (each either a `Subtask *Task` or a `Note string`). This
  is *why* sort, move-to-heading, delete and undo carry notes and subtasks
  along without special cases. Prefer solutions that keep that property.
- **Blocks are one level deep.** Deeper indentation is read as an ordinary
  block line and written back at two spaces. Saving normalises the file.
- **`save()` will not clobber an outside edit.** If the file changed underneath
  since the app last read it, the disk version wins and the in-app change is
  dropped with a message. Undo history is cleared on reload.
- **The fsnotify watch is on the directory**, not the file, so editors that
  save by rename keep working.

## Verifying changes

Unit tests are necessary but have not been sufficient here. Every UI change in
this repo gets driven for real in tmux against a scratch `tasks.md`:

```bash
just build && T=$(mktemp -d) && printf '## Alpha\n- [ ] a task\n' > $T/tasks.md
tmux new-session -d -s todo -x 80 -y 20 -c "$T" "$PWD/bin/todo"
tmux send-keys -t todo j; sleep 0.5; tmux capture-pane -t todo -p
```

Use `capture-pane -e` when colour matters. Sleep after `send-keys` — the app
needs a frame. Bugs found this way that the tests missed: an inline editor
rendering inside the wrong block, a note editor showing a task bullet, cursor
landing on the wrong item after undo. Check the file on disk too, not just the
screen.

## Traps

- **Terminal key encoding.** Shift+Space is indistinguishable from Space,
  ctrl+m from Enter, and ctrl+h is BS(8) while Backspace is DEL(127). Don't
  propose the first two as bindings; they were tried and reverted. `ctrl+space`
  arrives as `ctrl+@`.
- **bubbletea coalesces runes.** A fast-typed word arrives as one multi-rune
  `KeyRunes` message. Match `msg.Type` for Enter/Esc rather than the string,
  and see `TaskInputModel.Update`, which splits such messages rune by rune.
- **Measure display columns with `ansi.StringWidth`, never `len`.** Bullets,
  fold markers and tree guides are multi-byte. Overlay compositing uses
  `ansi.Truncate`/`TruncateLeft`.
- **`chromeHeight()` is the single source for layout overhead.** Two copies of
  that number drifted once already.

## Loose ends

- `internal/config` reads `.todo.toml` and `~/.config/todo/config.toml`, but
  **nothing imports it**. Configuration does not work. Wire it up or delete it;
  don't document it as working.
- The module is `github.com/macone/todo-cli` while the repo is
  `svellingen/tui-todo`, so the `go install` line in the README can't work.
- `docs/plans/2026-02-06-todo-cli.md` is the **original plan and is stale** —
  it says `todo.md`, `Makefile`, and bubbletea v2, all of which are wrong now.
  Don't treat it as a description of the current code.
- The `?` help overlay is ~38 rows and clips on short terminals. Known,
  deliberately deferred.

## Conventions

- Comments explain *why*, in prose, and are sparse. Match the surrounding
  density — don't annotate the obvious.
- Tests read as statements about behaviour: a sentence-long comment above each,
  named for what it asserts. See `internal/model/block_test.go`.
- No flash/toast messages for routine actions; the list itself is the feedback.
- `gofmt` before finishing.
