package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/macone/todo-cli/internal/storage"
	"github.com/macone/todo-cli/internal/task"
	"github.com/spf13/cobra"
)

const initTemplate = `# Todo

## Backlog

## In Progress

## Done
`

// resolveStore locates the task file for the current directory, walking up the
// directory tree, and returns a Store for it plus a path suitable for messages.
func resolveStore() (*storage.Store, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	path, err := storage.ResolveOrDefault(cwd)
	if err != nil {
		return nil, "", err
	}
	return storage.NewStore(path), displayPath(path), nil
}

// displayPath renders an absolute path relative to the current directory when
// that is shorter, so messages read "tasks.md" or "../tasks.md" rather than a
// full absolute path.
func displayPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil || len(rel) >= len(path) {
		return path
	}
	return rel
}

// newRootCmd creates the root cobra command. The out parameter controls where
// output is written (allows testing without capturing os.Stdout).
// The launchTUI function is called when no subcommand is given (allows testing
// without importing bubbletea/lipgloss).
func newRootCmd(out io.Writer, launchTUI func() error) *cobra.Command {
	root := &cobra.Command{
		Use:   "todo",
		Short: "A project-local TUI task tracker",
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchTUI()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newInitCmd(out))
	root.AddCommand(newAddCmd(out))
	root.AddCommand(newListCmd(out))
	root.AddCommand(newDoneCmd(out))

	return root
}

func newInitCmd(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create an empty tasks.md in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// init always targets the current directory rather than resolving
			// up the tree, so it can shadow an ancestor's task file.
			if _, err := os.Stat(storage.DefaultName); err == nil {
				return fmt.Errorf("%s already exists", storage.DefaultName)
			}
			if err := os.WriteFile(storage.DefaultName, []byte(initTemplate), 0644); err != nil {
				return fmt.Errorf("failed to create %s: %w", storage.DefaultName, err)
			}
			fmt.Fprintf(out, "Created %s\n", storage.DefaultName)
			return nil
		},
	}
}

func newAddCmd(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "add [title]",
		Short: "Add a task to the Backlog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.TrimSpace(args[0])
			if title == "" {
				return fmt.Errorf("task title cannot be empty")
			}

			store, path, err := resolveStore()
			if err != nil {
				return err
			}
			tf, err := store.Load()
			if err != nil {
				return fmt.Errorf("failed to load %s: %w", path, err)
			}

			newTask := task.Task{
				Title:  title,
				Status: task.StatusTodo,
			}
			tf.Tasks = append(tf.Tasks, newTask)

			// Find the Backlog section and insert the task line after it
			inserted := false
			newLine := storage.Line{
				Type:      storage.LineTask,
				TaskIndex: len(tf.Tasks) - 1,
			}

			for i, line := range tf.Lines {
				if line.Type == storage.LineSection {
					// Find the section whose status is Todo (Backlog)
					for _, sec := range tf.Sections {
						if sec.Line == line.Number && sec.Status == task.StatusTodo {
							// Insert after this section header and any blank lines following it
							insertIdx := i + 1
							for insertIdx < len(tf.Lines) && tf.Lines[insertIdx].Type == storage.LineText && strings.TrimSpace(tf.Lines[insertIdx].Raw) == "" {
								insertIdx++
							}
							// Insert the new task line
							tf.Lines = append(tf.Lines[:insertIdx], append([]storage.Line{newLine}, tf.Lines[insertIdx:]...)...)
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
				// No Backlog section found, just append
				tf.Lines = append(tf.Lines, newLine)
			}

			if err := store.Save(tf); err != nil {
				return fmt.Errorf("failed to save %s: %w", path, err)
			}

			fmt.Fprintf(out, "Added: %s (%s)\n", title, path)
			return nil
		},
	}
}

func newListCmd(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, path, err := resolveStore()
			if err != nil {
				return err
			}
			tf, err := store.Load()
			if err != nil {
				return fmt.Errorf("failed to load %s: %w", path, err)
			}

			if len(tf.Tasks) == 0 {
				fmt.Fprintln(out, "No tasks found.")
				return nil
			}

			for i, t := range tf.Tasks {
				var status string
				switch t.Status {
				case task.StatusTodo:
					status = "[ ]"
				case task.StatusInProgress:
					status = "[-]"
				case task.StatusDone:
					status = "[x]"
				}

				line := fmt.Sprintf("%d. %s %s", i+1, status, t.Title)

				var meta []string
				switch t.Priority {
				case task.PriorityHigh:
					meta = append(meta, "priority:high")
				case task.PriorityMedium:
					meta = append(meta, "priority:medium")
				case task.PriorityLow:
					meta = append(meta, "priority:low")
				}
				for _, tag := range t.Tags {
					meta = append(meta, "+"+tag)
				}
				if len(meta) > 0 {
					line += " " + strings.Join(meta, " ")
				}

				fmt.Fprintln(out, line)
			}
			return nil
		},
	}
}

func newDoneCmd(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "done [number]",
		Short: "Mark a task as done by its number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task number: %s", args[0])
			}

			store, path, err := resolveStore()
			if err != nil {
				return err
			}
			tf, err := store.Load()
			if err != nil {
				return fmt.Errorf("failed to load %s: %w", path, err)
			}

			if n < 1 || n > len(tf.Tasks) {
				return fmt.Errorf("task %d does not exist (have %d tasks)", n, len(tf.Tasks))
			}

			idx := n - 1
			tf.Tasks[idx].Status = task.StatusDone

			if err := store.Save(tf); err != nil {
				return fmt.Errorf("failed to save %s: %w", path, err)
			}

			fmt.Fprintf(out, "Done: %s\n", tf.Tasks[idx].Title)
			return nil
		},
	}
}
