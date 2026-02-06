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
make build
# Binary at ./bin/todo
```

## Quick Start

```bash
# Create a todo.md in your project
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
| `todo init` | Create an empty `todo.md` |
| `todo add "title"` | Add a task to the Backlog |
| `todo list` | List all tasks |
| `todo done <n>` | Mark task #n as done |

## TUI Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `J` / `K` | Jump to next / previous section |

### Actions

| Key | Action |
|-----|--------|
| `a` | Add new task |
| `e` | Edit task title |
| `d` | Toggle done |
| `x` | Delete task (with confirmation) |
| `s` | Cycle status (todo / in-progress / done) |
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

## File Format

Tasks are stored in `todo.md` using GitHub-Flavored Markdown checkbox syntax:

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
# File to use for tasks (default: "todo.md")
file = "todo.md"

[sections]
todo = "Backlog"
in_progress = "In Progress"
done = "Done"

[ui]
show_done = true
```

## Development

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run go vet
make clean    # Remove build artifacts
make release  # Cross-compile for all platforms
```

## License

MIT
