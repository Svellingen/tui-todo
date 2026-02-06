package storage

import (
	"errors"
	"io/fs"
	"os"
)

// Store handles reading and writing TaskFiles to disk.
type Store struct {
	FilePath string
	parser   *Parser
	writer   *Writer
}

// NewStore creates a new Store for the given file path.
func NewStore(path string) *Store {
	return &Store{
		FilePath: path,
		parser:   NewParser(),
		writer:   NewWriter(),
	}
}

// Load reads the markdown file and returns a parsed TaskFile.
// Returns an empty TaskFile (not an error) if the file doesn't exist.
func (s *Store) Load() (*TaskFile, error) {
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &TaskFile{}, nil
		}
		return nil, err
	}
	return s.parser.Parse(string(data))
}

// Save writes the TaskFile to disk, creating the file if it doesn't exist.
func (s *Store) Save(tf *TaskFile) error {
	content := s.writer.Write(tf)
	return os.WriteFile(s.FilePath, []byte(content), 0644)
}
