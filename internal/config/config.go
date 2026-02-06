// Package config handles loading and merging TOML configuration.
package config

import (
	"errors"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

type Sections struct {
	Todo       string `toml:"todo"`
	InProgress string `toml:"in_progress"`
	Done       string `toml:"done"`
}

type Config struct {
	File     string   `toml:"file"`
	Sections Sections `toml:"sections"`
	ShowDone bool     `toml:"-"`
}

// uiBlock captures the [ui] table from TOML.
type uiBlock struct {
	ShowDone *bool `toml:"show_done"`
}

// fileConfig is the full TOML structure on disk.
type fileConfig struct {
	File     *string  `toml:"file"`
	Sections Sections `toml:"sections"`
	UI       uiBlock  `toml:"ui"`
}

func defaultConfig() Config {
	return Config{
		File: "todo.md",
		Sections: Sections{
			Todo:       "Backlog",
			InProgress: "In Progress",
			Done:       "Done",
		},
		ShowDone: true,
	}
}

// Load reads configuration from the local .todo.toml and global config,
// merging local over global over defaults.
func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		// If we can't determine home, just skip global config
		home = ""
	}
	return loadWithHome(home)
}

// loadWithHome is the internal implementation that accepts a home directory,
// making it testable without touching the real home directory.
func loadWithHome(home string) (Config, error) {
	cfg := defaultConfig()

	// Load global config first
	if home != "" {
		globalPath := filepath.Join(home, ".config", "todo", "config.toml")
		if err := applyFile(&cfg, globalPath); err != nil {
			return cfg, err
		}
	}

	// Load local config (overrides global)
	localPath := filepath.Join(".", ".todo.toml")
	if err := applyFile(&cfg, localPath); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// applyFile reads a TOML file and merges non-zero values into cfg.
// If the file does not exist, it is silently skipped.
func applyFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return err
	}

	if fc.File != nil {
		cfg.File = *fc.File
	}
	if fc.Sections.Todo != "" {
		cfg.Sections.Todo = fc.Sections.Todo
	}
	if fc.Sections.InProgress != "" {
		cfg.Sections.InProgress = fc.Sections.InProgress
	}
	if fc.Sections.Done != "" {
		cfg.Sections.Done = fc.Sections.Done
	}
	if fc.UI.ShowDone != nil {
		cfg.ShowDone = *fc.UI.ShowDone
	}

	return nil
}
