package main

import (
	"errors"
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

// storeResolver hands back the task file to operate on, along with a path
// suitable for messages.
type storeResolver func() (*storage.Store, string, error)

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

// resolveStoreWith honours an explicit --file, falling back to the directory
// walk only when the flag is empty.
//
// An explicit file is taken literally: if it is not there, that is an error
// rather than a reason to go looking for another one.
func resolveStoreWith(file string) (*storage.Store, string, error) {
	if file == "" {
		return resolveStore()
	}

	abs, err := filepath.Abs(file)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("file not found: %s", file)
		}
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("not a file: %s", file)
	}
	return storage.NewStore(abs), displayPath(abs), nil
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
func newRootCmd(out io.Writer, launchTUI func(*storage.Store) error) *cobra.Command {
	// Held per command tree rather than in a package variable, so tests can
	// build independent roots.
	var file string

	resolve := func() (*storage.Store, string, error) { return resolveStoreWith(file) }

	root := &cobra.Command{
		Use:   "todo",
		Short: "A project-local TUI task tracker",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _, err := resolve()
			if err != nil {
				return err
			}
			return launchTUI(store)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&file, "file", "",
		"use this task file instead of searching for one")

	root.AddCommand(newInitCmd(out, &file))
	root.AddCommand(newAddCmd(out, resolve))
	root.AddCommand(newListCmd(out, resolve))
	root.AddCommand(newDoneCmd(out, resolve))

	return root
}

// newInitCmd takes the flag by pointer rather than a resolver: init creates a
// file, so requiring it to already exist would defeat the purpose.
func newInitCmd(out io.Writer, file *string) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create an empty tasks.md in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Without --file, init targets the current directory rather than
			// resolving up the tree, so it can shadow an ancestor's task file.
			path := storage.DefaultName
			if *file != "" {
				path = *file
			}
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists", path)
			}
			if err := os.WriteFile(path, []byte(initTemplate), 0644); err != nil {
				return fmt.Errorf("failed to create %s: %w", path, err)
			}
			fmt.Fprintf(out, "Created %s\n", path)
			return nil
		},
	}
}

func newAddCmd(out io.Writer, resolve storeResolver) *cobra.Command {
	return &cobra.Command{
		Use:   "add [title]",
		Short: "Add a task to the Backlog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.TrimSpace(args[0])
			if title == "" {
				return fmt.Errorf("task title cannot be empty")
			}

			store, path, err := resolve()
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

			tf.Sort()
			if err := store.Save(tf); err != nil {
				return fmt.Errorf("failed to save %s: %w", path, err)
			}

			fmt.Fprintf(out, "Added: %s (%s)\n", title, path)
			return nil
		},
	}
}

func newListCmd(out io.Writer, resolve storeResolver) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, path, err := resolve()
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

				// Priority prints as the same "!" marker used on disk.
				title := t.Title
				if marker := storage.PriorityMarker(t.Priority); marker != "" {
					title = marker + " " + title
				}
				line := fmt.Sprintf("%d. %s %s", i+1, status, title)

				var meta []string
				for _, tag := range t.Tags {
					meta = append(meta, "+"+tag)
				}
				for _, ctx := range t.Contexts {
					meta = append(meta, "@"+ctx)
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

func newDoneCmd(out io.Writer, resolve storeResolver) *cobra.Command {
	return &cobra.Command{
		Use:   "done [number]",
		Short: "Mark a task as done by its number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task number: %s", args[0])
			}

			store, path, err := resolve()
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

			tf.Sort()
			if err := store.Save(tf); err != nil {
				return fmt.Errorf("failed to save %s: %w", path, err)
			}

			fmt.Fprintf(out, "Done: %s\n", tf.Tasks[idx].Title)
			return nil
		},
	}
}
