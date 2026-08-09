# md-taco

A zero-config, project-local TUI task tracker that stores tasks in a markdown file.

<!-- TODO: Add screenshot/GIF here -->

## Install

```bash
go install github.com/svellingen/md-taco/cmd/taco@latest
```

Or build from source:

```bash
git clone https://github.com/svellingen/md-taco.git
cd md-taco
just build
# Binary at ./bin/taco
```

## Quick Start

```bash
# Create a tasks.md in your project
taco init

# Add tasks
taco add "Set up CI pipeline"
taco add "Write tests"

# Launch the TUI
taco
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `taco` | Launch the TUI (default) |
| `--file <path>` | Use this file instead of searching for one (works with every command) |
| `taco init` | Create an empty `tasks.md` |
| `taco add "title"` | Add a task to the Backlog |
| `taco list` | List all tasks |
| `taco done <n>` | Mark task #n as done |

## TUI Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `↓` / `↑` | Move down / up |
| `ctrl+j` / `ctrl+k` | Select the next / previous `##` heading, skipping deeper ones |
| `ctrl+h` / `ctrl+l` | Select the parent / first child heading |
| `ctrl+e` | Heading popup — filter and jump to any heading |
| `m` | Heading popup — move the selected task to any heading |
| `J` / `K` | Select the next / previous heading, any level |
| `Enter` | Fold the selected task's block open or shut |
| `tab` | Fold every block — shuts them all if any is open, else opens them all |
| `gg` / `G` | Jump to the first / last task |
| `ctrl+d` / `ctrl+u` | Page down / up |
| `alt+j` / `alt+k` | Move the selected task down / up within its sort group — or a subtask or note within its block |
| `alt+shift+j` / `alt+shift+k` | Move the selected task to the next / previous heading |

### Actions

| Key | Action |
|-----|--------|
| `a` | Add a task inline below the selection — or a sibling subtask when on a block line |
| `A` | Add a subtask to the selected task |
| `n` | Add a note to the selected task — or directly below the selected block line |
| `e` | Edit the task, subtask or note inline, on its own row |
| `d` | Toggle done |
| `x` | Delete the task, or the selected heading and everything under it |
| `Space` | Cycle status forward (todo → in-progress → done, wrapping) |
| `ctrl+space` | Step status back (done → in-progress → todo, stopping at todo) |
| `p` / `P` | Raise / lower priority (none ↔ `!` ↔ `!!`, stopping at each end) |
| `t` / `c` | Add a tag / a context to the task |
| `u` / `ctrl+r` | Undo / redo |
| `o` | Open the task file in an editor |

### Filtering

| Key | Action |
|-----|--------|
| `/` | Search by title (live filtering) |
| `T` / `C` | Filter by tag / by context (press again to clear) |
| `1` | Show all tasks |
| `2` | Show active only (todo + in-progress) |
| `3` | Show done only |
| `f` | Focus the current heading — show only its tasks (toggle) |

Switching between these keeps the cursor on the task it was on. If the change
hides that task, the cursor lands near where it was rather than at the top of
the list.

### General

| Key | Action |
|-----|--------|
| `i` | Toggle the task file's path in the title bar (off by default) |
| `?` | Toggle help overlay |
| `q` / `ctrl+c` | Quit |

## File Resolution

Commands locate the task file by checking each directory from the current one
upwards, stopping at the first match:

1. `tasks.md` in the current directory
2. `index.md` in the current directory
3. the same two checks in the parent directory, and so on

`tasks.md` only wins over `index.md` within the *same* directory — an `index.md`
next to you takes precedence over a `tasks.md` further up the tree. This means
you can run `taco` from anywhere inside a project and hit the same file.

The walk stops after checking `$HOME`, so a stray `tasks.md` or `index.md`
above your home directory can't leak into unrelated projects. `$HOME` itself
*is* checked, so a `~/tasks.md` acts as a personal catch-all for work outside
any project. Directories outside `$HOME` never cross that boundary, so there
the walk runs to the filesystem root.

If nothing is found, commands that write (`add`) create `tasks.md` in the
current directory. `taco init` always creates `tasks.md` in the current
directory, which is how you shadow an ancestor's task file for a subproject.

### Overriding the file

`--file <path>` skips the search entirely and uses exactly the file named:

```bash
taco --file notes/plan.md          # open the TUI on that file
taco --file notes/plan.md list
taco --file notes/plan.md add "something"
```

There is no fallback. If the file is missing the command exits with
`file not found: <path>` rather than reaching for another one, so a typo can
never quietly write to the wrong file.

`taco init --file <path>` is the exception: it creates that file, and refuses
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

`J` and `K` select headings themselves; `j` and `k` step over them between
tasks. `ctrl+j` and `ctrl+k` do the same as `J` and `K`, but stop only at
`##` headings — handy for moving between major sections in a file with a lot
of sub-headings. A `#` title is a different level, so it is not one of their
stops; `J` and `K` still reach it.

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
  sub-headings, their tasks, and their notes, up to the next heading
  of the same or shallower level. This one asks for confirmation first, naming
  what will go: `Delete "Alpha" with 2 sub-headings and 4 tasks? y/n`. It is
  still undoable with `u`.

Actions that need a task — `d`, `Space`, `p`, `e`, `t`, `c`, `A`, `n`,
`alt+j`/`alt+k` — do nothing while a heading is selected.

## Editing in Place

`a` and `e` open the editor inside the list rather than in a prompt at the
bottom. `e` takes over the selected task's own row, keeping its status bullet;
`a` opens a fresh row directly below the selection — under the current task, or
under the heading when one is selected.

`Enter` commits. `Esc` behaves differently either side: leaving an *edit* keeps
what you typed — it means "stop editing", not "undo the edit" — while leaving
an *add* discards it, since there is no earlier text to keep. Clearing a title
entirely and pressing either key leaves the task as it was rather than saving a
blank one.

Search (`/`) and the label prompts (`t`, `c`) still appear below the list,
since none of them belongs to a particular row.

## The Heading Popup

In a file with many headings, `ctrl+e` opens a filterable list of them all:

```
╭──────────────────── Jump to heading ─────────────────────╮
│ > filter…                                            8/8 │
│ ──────────────────────────────────────────────────────── │
│ repo / github / svellingen / md-taco                      │
│ ├─ shamone                                               │
│ ├─ tasks                                                 │
│ ├─ test header 4                                         │
│ ├─ test header 5                                         │
│ │  ╰─ sub header 55                                      │
│ │     ╰─ sub header 555                                  │
│ ╰─ test header 6                                         │
╰──────────────────────────────────────────────────────────╯
```

Type to filter by name, `ctrl+j` / `ctrl+k` (or the arrows) to move, `Enter` to
jump there, `Esc` to close. The tree guides are drawn over whatever is
currently visible, so a filtered list still reads as a tree rather than
sprouting branches to hidden parents.

`m` opens the same popup in move mode: choosing a heading moves the selected
task under it, and the cursor follows the task to its new home. It only applies
to tasks — pressing it on a heading does nothing.

## Focus View

`f` narrows the list to one heading: the one the cursor is on, or the one the
selected task lives under. Its sub-headings and their tasks come along, so
focusing a `##` shows the whole branch beneath it.

```
✦ taco  [focus: Alpha]
 - Alpha
   ○   alpha one
   ○   alpha two
   Alpha sub
   ○   sub task
```

Press `f` again to widen back out. The cursor is kept across both, whether it
is on a task or on the heading itself.

Focus composes with the status filters, so `f` then `2` shows just the active
tasks in that branch. It is a scope rather than a filter, so `1` does not clear
it — `f` does.

If the focused heading disappears — deleted, or gone after an outside edit —
the focus lapses and the whole file comes back, rather than leaving the list
showing an arbitrary slice.

While focused, the `ctrl+e` popup lists only the headings inside the focus.
Widen with `f` to jump elsewhere.

## Tags and Contexts

Tags (`+tag`) and contexts (`@context`) are separate axes, each with its own
key and colour — tags in mauve, contexts in rose:

```markdown
- [ ] write docs +work @home
- [ ] call bank @errands
```

`t` and `c` attach one to the selected task; `T` and `C` filter by one, and
pressing the same key again clears that filter. The two compose, so `+work`
and `@home` together shows only tasks carrying both. Either can also be typed
inline while adding or editing a task.

## Task Blocks

Lines indented beneath a task belong to that task:

```markdown
- [ ] deploy the thing
  check the logs first
  - [ ] tag the release
  - [x] update the changelog
```

In the list a task with a block carries a marker in its own column, between
the status bullet and the title, so titles stay aligned either way. The cursor
is the `-` in the left gutter:

```
   ○   a task with nothing under it
 - ○ ▸ a task with a block, folded
   ○ ▾ a task with a block, open
   -   a note
       ○  a subtask
```

On a block line the cursor sits two columns further in, under the parent's
bullet.

The block travels with its task — sorting, moving between headings, deleting
and undo all carry it. `Enter` folds it open and shut. `tab` folds the whole
file: if any block is open it shuts them all, otherwise it opens them all.
Blocks start folded.

While expanded, `j` and `k` walk into the block. A subtask behaves like a task
for the things that act in place: `Space`, `ctrl+space`, `d`, `p` and `P` all
apply to the subtask under the cursor rather than its parent.

Inside a block the status scale has a fourth step below todo — being a note at
all:

```
note  →  todo  →  in progress  →  done
```

`Space` walks it forward, wrapping from done back to a note; `ctrl+space` walks
it back and stops at a note. So `Space` on a note turns it into a subtask, and
`ctrl+space` off todo turns a subtask back into a note.

Converting keeps what the other form can hold: a demoted subtask leaves its
marker and labels in the note text (`- [ ] !! ship it +work` becomes
`!! ship it +work`), and promoting it reads them back as priority and tags. The
status is the one thing a note cannot carry, so a promoted note starts at todo.

`x` on a block line removes just that line. Subtasks keep the order you wrote
them in — only top-level tasks are sorted.

Adding and editing work at both levels:

- `A` on a task appends a subtask to its block, unfolding it first.
- `n` appends a note to the selected task's block, or inserts one directly
  after the selected block line.
- `a` on a block line inserts a sibling directly after it.
- `e` edits whatever is under the cursor — task, subtask or note — prefilled
  with its text. A note is edited without a bullet, since it is not a task. A
  subtask keeps its status and any metadata the new text does not mention,
  exactly as editing a task does. A note retyped as `- [ ] …` becomes a
  subtask.

Metadata can be given inline when adding a subtask, so `!! ship it +proj @desk`
works there too.

Blocks are one level deep. Anything more deeply indented in the source is read
as an ordinary block line and written back at two spaces, so saving normalises
the file:

```markdown
- [ ] task            - [ ] task
      a stray note      a stray note
  - [ ] sub       →     - [ ] sub
        - [ ] deep      - [ ] deep
```

A blank line ends a block; what follows belongs to the file rather than the
task.

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

Block lines are not sorted — they keep the order they are written in — so on a
subtask or a note the same `alt+j` and `alt+k` reorder freely within the block,
stopping at its ends. A block line never leaves its own task this way.

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
- `@context` adds a context
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

There is none yet. Section names (`Backlog`, `In Progress`, `Done`) are
recognised by name in the parser, and the task file is found by the resolution
rules above or named outright with `--file`.

An `internal/config` package exists that reads `.todo.toml` and
`~/.config/todo/config.toml`, but nothing imports it, so those files are
ignored if present.

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
