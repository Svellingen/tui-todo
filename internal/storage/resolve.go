package storage

import (
	"errors"
	"os"
	"path/filepath"
)

// CandidateNames are the file names searched in each directory, in priority
// order: tasks.md wins over index.md within the same directory, even when an
// ancestor directory has a tasks.md.
var CandidateNames = []string{"tasks.md", "index.md"}

// DefaultName is the file created when no existing task file is found.
const DefaultName = "tasks.md"

// ErrNotFound is returned by Resolve when neither the starting directory nor
// any of its ancestors holds a task file.
var ErrNotFound = errors.New("no task file found")

// Resolve searches dir and then each ancestor directory in turn for a task
// file, returning the absolute path of the first match. The walk stops after
// checking $HOME, so a stray file further up cannot leak into unrelated
// projects; it returns ErrNotFound if nothing matched.
//
// Directories outside $HOME have no such ancestor, so there the walk runs to
// the filesystem root instead.
func Resolve(dir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		// Without a known home directory there is no boundary to apply.
		home = ""
	}
	return resolveFrom(dir, home)
}

// resolveFrom is the testable core of Resolve. It walks up from dir, stopping
// once it has checked boundary or reached the filesystem root, whichever comes
// first. An empty boundary means "root only".
func resolveFrom(dir, boundary string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if boundary != "" {
		if boundary, err = filepath.Abs(boundary); err != nil {
			return "", err
		}
	}

	for {
		for _, name := range CandidateNames {
			path := filepath.Join(dir, name)
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
				return path, nil
			}
		}

		parent := filepath.Dir(dir)
		if dir == boundary || parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// ResolveOrDefault behaves like Resolve, but falls back to DefaultName in dir
// when no task file exists, giving callers that create files a target path.
func ResolveOrDefault(dir string) (string, error) {
	path, err := Resolve(dir)
	switch {
	case err == nil:
		return path, nil
	case errors.Is(err, ErrNotFound):
		return filepath.Abs(filepath.Join(dir, DefaultName))
	default:
		return "", err
	}
}
