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
| `J` / `K` | Jump to next / previous section |

### Actions

| Key | Action |
|-----|--------|
| `a` | Add new task |
| `e` | Edit task title |
| `d` | Toggle done |
| `x` | Delete task (with confirmation) |
| `Space` | Cycle status (todo / in-progress / done) |
| `p` | Cycle priority (none / low / medium / high) |
| `t` | Add tag to task |
| `u` | Undo last action |

### Filtering

| Key | Action |
|-----|--------|
| `/` | Search by title (live filtering) |
| `f` | Filter by tag |
| `1` | Show all tasks |
| `2` | Show active only (todo + in-progress) |
| `3` | Show done only |

### General

| Key | Action |
|-----|--------|
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

## File Format

Tasks are stored in `tasks.md` using GitHub-Flavored Markdown checkbox syntax:

```markdown
# My Project

## Backlog

- [ ] Set up CI pipeline priority:high +devops
- [ ] Write README priority:low +docs

## In Progress

- [-] Design TUI layout priority:high +design

## Done

- [x] Initialize Go module +setup
```

### Task Syntax

```
- [ ] Title priority:high +tag @context due:2026-03-01
```

- `- [ ]` todo, `- [-]` in-progress, `- [x]` done
- `priority:high|medium|low` sets priority
- `+tag` adds a tag
- `@context` adds a context tag
- `due:YYYY-MM-DD` sets a due date

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
