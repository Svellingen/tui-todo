# todo-cli

A zero-config, project-local TUI task tracker that stores tasks in a markdown file.

<!-- TODO: Add screenshot/GIF here -->

## Install

```bash
go install github.com/macone/todo-cli/cmd/todo@latest
```

Or build from source:

```bash
git clone https://github.com/macone/todo-cli.git
cd todo-cli
just build
# Binary at ./bin/todo
```

## Quick Start

```bash
# Create a tasks.md in your project
todo init

# Add tasks
todo add "Set up CI pipeline"
todo add "Write tests"

# Launch the TUI
todo
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `todo` | Launch the TUI (default) |
| `--file <path>` | Use this file instead of searching for one (works with every command) |
| `todo init` | Create an empty `tasks.md` |
| `todo add "title"` | Add a task to the Backlog |
| `todo list` | List all tasks |
| `todo done <n>` | Mark task #n as done |

## TUI Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `↓` / `↑` | Move down / up |
| `tab` / `shift+tab` | Select the next / previous heading, any level |
| `ctrl+j` / `ctrl+k` | Select the next / previous `##` heading, skipping deeper ones |
| `ctrl+h` / `ctrl+l` | Select the parent / first child heading |
| `J` / `K` | Same jump, vim-style |
| `gg` / `G` | Jump to the first / last task |
| `ctrl+d` / `ctrl+u` | Page down / up |
| `alt+j` / `alt+k` | Move the selected task down / up within its sort group |
| `alt+shift+j` / `alt+shift+k` | Move the selected task to the next / previous heading |

### Actions

| Key | Action |
|-----|--------|
| `a` / `Enter` | Add a task inline on the next row — below the current task, or under the current heading |
| `e` | Edit the task title inline, on its own row |
| `d` | Toggle done |
| `x` | Delete the task, or the selected heading and everything under it |
| `Space` | Cycle status forward (todo → in-progress → done, wrapping) |
| `ctrl+space` | Step status back (done → in-progress → todo, stopping at todo) |
| `p` / `P` | Raise / lower priority (none ↔ `!` ↔ `!!`, stopping at each end) |
| `t` | Add tag to task |
| `u` / `ctrl+r` | Undo / redo |
| `o` | Open the task file in an editor |

### Filtering

| Key | Action |
|-----|--------|
| `/` | Search by title (live filtering) |
| `T` | Filter by tag |
| `1` | Show all tasks |
| `2` | Show active only (todo + in-progress) |
| `3` | Show done only |

### General

| Key | Action |
|-----|--------|
| `i` | Toggle the task file's path in the title bar (off by default) |
| `?` | Toggle help overlay |
| `q` | Quit |

## File Resolution

Commands locate the task file by checking each directory from the current one
upwards, stopping at the first match:

1. `tasks.md` in the current directory
2. `index.md` in the current directory
3. the same two checks in the parent directory, and so on

`tasks.md` only wins over `index.md` within the *same* directory — an `index.md`
next to you takes precedence over a `tasks.md` further up the tree. This means
you can run `todo` from anywhere inside a project and hit the same file.

The walk stops after checking `$HOME`, so a stray `tasks.md` or `index.md`
above your home directory can't leak into unrelated projects. `$HOME` itself
*is* checked, so a `~/tasks.md` acts as a personal catch-all for work outside
any project. Directories outside `$HOME` never cross that boundary, so there
the walk runs to the filesystem root.

If nothing is found, commands that write (`add`) create `tasks.md` in the
current directory. `todo init` always creates `tasks.md` in the current
directory, which is how you shadow an ancestor's task file for a subproject.

### Overriding the file

`--file <path>` skips the search entirely and uses exactly the file named:

```bash
todo --file notes/plan.md          # open the TUI on that file
todo --file notes/plan.md list
todo --file notes/plan.md add "something"
```

There is no fallback. If the file is missing the command exits with
`file not found: <path>` rather than reaching for another one, so a typo can
never quietly write to the wrong file.

`todo init --file <path>` is the exception: it creates that file, and refuses
if it already exists.

## Headings

Headings of every level (`#` through `######`) are parsed and shown by name —
the `#` markers are dropped, since the colour already carries the level:

| Level | Colour |
|-------|--------|
| `#` | blue |
| `##` | amber |
| `###` | green |
| `####` | teal |

Levels beyond the fourth cycle back through the same four colours. Every
heading starts a new section, so tasks group and sort under sub-headings as
well as top-level ones.

### Selecting headings

`tab` and `shift+tab` select headings themselves; `j` and `k` step over them
between tasks. `ctrl+j` and `ctrl+k` do the same as `tab`, but stop only at
`##` headings — handy for moving between major sections in a file with a lot
of sub-headings. A `#` title is a different level, so it is not one of their
stops; `tab` still reaches it.

`ctrl+h` and `ctrl+l` move across levels rather than along one. `ctrl+l`
descends to the first sub-heading nested under the current one, and `ctrl+h`
climbs back to its parent. Together with `ctrl+j`/`ctrl+k` that gives you the
whole tree: across `##` sections, then down into their sub-headings.

- `ctrl+h` from a task selects the heading the task lives under, so it doubles
  as "where am I?".
- `ctrl+h` stops at `##`; pressing it there does nothing.
- `ctrl+l` does nothing on a task, or on a heading with no sub-headings.
- Levels may be skipped in the file — a `####` directly under a `##` still
  finds that `##` as its parent.

Note that `ctrl+h` acts as backspace inside the add, edit and search prompts,
which is the usual terminal convention.

With a heading selected:

- `a` adds a task directly beneath it rather than in the Backlog section.
  From a task, `a` instead adds directly below that task, keeping it in the
  same section.
- `x` deletes the heading together with everything nested under it — its
  sub-headings, their tasks, and any prose in between, up to the next heading
  of the same or shallower level. This one asks for confirmation first, naming
  what will go: `Delete "Alpha" with 2 sub-headings and 4 tasks? y/n`. It is
  still undoable with `u`.

Actions that need a task — `d`, `Space`, `p`, `e`, `t`, `alt+j`/`alt+k` — do
nothing while a heading is selected.

## Editing in Place

`a` and `e` open the editor inside the list rather than in a prompt at the
bottom. `e` takes over the selected task's own row, keeping its status bullet;
`a` opens a fresh row directly below the selection — under the current task, or
under the heading when one is selected. `Enter` commits and `Esc` discards,
leaving the task as it was.

Search (`/`) and the tag prompt (`t`) still appear below the list, since
neither belongs to a particular row.

## Sorting

Tasks are kept sorted inside each section, top to bottom:

1. done tasks
2. high priority (`!!`)
3. medium priority (`!`)
4. everything else

Sorting is per section — a task never crosses a `##` header. The order is
written back to `tasks.md`, so what the TUI shows is what the file contains.
Opening the app on a hand-ordered file rewrites it into sorted order, as does
any edit made through the CLI.

The sort is stable: tasks of equal rank keep their relative order. `alt+j` and
`alt+k` move the selected task down and up within its own rank group, and that
manual order survives later sorts. A move that would cross into another rank,
or out of the section, does nothing — priority ordering can't be broken by
hand. Changing a task's status or priority re-sorts it immediately, with the
cursor following the task rather than staying at the same screen position.

## Editing in an Editor

Pressing `o` hands the terminal to your editor on the task file, opening on the
line of the task or heading you had selected. Quitting the editor returns to
the TUI, which reloads whatever you saved; the cursor stays on the task it was
on. Quitting without saving changes nothing.

The line is passed as `+N`, which vi, vim, nvim, nano, emacs, micro, joe and
kak understand. Any other editor is opened on the file without a line
argument, since an unrecognised `+N` would be taken for a second file to open.

The editor is `$VISUAL`, else `$EDITOR`, else `nvim` if installed, else `vi`.
Both variables may carry arguments, as in `EDITOR="code -w"`.

## Outside Edits

The TUI watches the task file and reloads it the moment it changes on disk, so
editing `tasks.md` in another editor shows up without restarting. Active
filters and the cursor position are preserved where possible.

The watch is on the file's directory rather than the file itself, so it keeps
working with editors that save atomically by writing a temp file and renaming
it into place. If a filesystem watch cannot be established — an exhausted
inotify limit, or a filesystem without notification support — the app falls
back to polling once a second.

Because a reload replaces the in-memory state, the undo history is cleared —
replaying it would discard the outside edit. If the file changes while you have
a modal open (add, edit, tag filter), the reload waits until you are
back at the list.

Nothing is overwritten silently: if the file changed underneath since the app
last read it, the outside edit wins. The app reloads and reports `File changed
on disk — reloaded, change not applied` rather than clobbering it. Deleting the
file does not blank the list; recreating it is picked up on the next poll.

## File Format

Tasks are stored in `tasks.md` using GitHub-Flavored Markdown checkbox syntax:

```markdown
# My Project

## Backlog

- [ ] !! Set up CI pipeline +devops
- [ ] ! Write README +docs

## In Progress

- [-] !! Design TUI layout +design

## Done

- [x] Initialize Go module +setup
```

### Task Syntax

```
- [ ] !! Title +tag @context due:2026-03-01
```

- `- [ ]` todo, `- [-]` in-progress, `- [x]` done
- `!` medium, `!!` high priority — a prefix before the title
- `+tag` adds a tag
- `@context` adds a context tag
- `due:YYYY-MM-DD` sets a due date

### Priority

Priority is a run of `!` immediately after the checkbox:

```markdown
- [ ] no pri
- [ ] ! medium priority task
- [ ] !! high priority task
```

No marker is the bottom of the scale; there is no separate "low". The marker
must be its own word, so a title like `- [ ] ship it!` is plain text, and more
than two marks count as high.

The older `priority:high|medium|low` token is still read, so existing files
keep their priorities, but it is never written back — saving a file rewrites it
using the `!` marker. Legacy `priority:low` becomes `!`, the weakest level the
scale still has.

## Configuration

Configuration is loaded from `.todo.toml` (project-local) or `~/.config/todo/config.toml` (global). Local settings override global.

```toml
# File to use for tasks (default: "tasks.md")
file = "tasks.md"

[sections]
todo = "Backlog"
in_progress = "In Progress"
done = "Done"

[ui]
show_done = true
```

## Development

```bash
just          # List available recipes
just build    # Build binary
just run      # Build and run
just test     # Run tests
just lint     # Run go vet
just clean    # Remove build artifacts
just release  # Cross-compile for all platforms
```

## License

MIT
