package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	// No config files exist — should return defaults without error
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.File != "tasks.md" {
		t.Errorf("expected File='tasks.md', got %q", cfg.File)
	}
	if cfg.Sections.Todo != "Backlog" {
		t.Errorf("expected Sections.Todo='Backlog', got %q", cfg.Sections.Todo)
	}
	if cfg.Sections.InProgress != "In Progress" {
		t.Errorf("expected Sections.InProgress='In Progress', got %q", cfg.Sections.InProgress)
	}
	if cfg.Sections.Done != "Done" {
		t.Errorf("expected Sections.Done='Done', got %q", cfg.Sections.Done)
	}
	if cfg.ShowDone != true {
		t.Errorf("expected ShowDone=true, got %v", cfg.ShowDone)
	}
}

func TestLocalConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	toml := `file = "tasks.md"

[sections]
todo = "To Do"
in_progress = "Working On"
done = "Completed"

[ui]
show_done = false
`
	os.WriteFile(filepath.Join(dir, ".todo.toml"), []byte(toml), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.File != "tasks.md" {
		t.Errorf("expected File='tasks.md', got %q", cfg.File)
	}
	if cfg.Sections.Todo != "To Do" {
		t.Errorf("expected Sections.Todo='To Do', got %q", cfg.Sections.Todo)
	}
	if cfg.Sections.InProgress != "Working On" {
		t.Errorf("expected Sections.InProgress='Working On', got %q", cfg.Sections.InProgress)
	}
	if cfg.Sections.Done != "Completed" {
		t.Errorf("expected Sections.Done='Completed', got %q", cfg.Sections.Done)
	}
	if cfg.ShowDone != false {
		t.Errorf("expected ShowDone=false, got %v", cfg.ShowDone)
	}
}

func TestGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create a fake global config dir
	globalDir := filepath.Join(dir, "fakehome", ".config", "todo")
	os.MkdirAll(globalDir, 0755)
	toml := `file = "global-tasks.md"

[sections]
todo = "Global Todo"
`
	os.WriteFile(filepath.Join(globalDir, "config.toml"), []byte(toml), 0644)

	cfg, err := loadWithHome(filepath.Join(dir, "fakehome"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.File != "global-tasks.md" {
		t.Errorf("expected File='global-tasks.md', got %q", cfg.File)
	}
	if cfg.Sections.Todo != "Global Todo" {
		t.Errorf("expected Sections.Todo='Global Todo', got %q", cfg.Sections.Todo)
	}
	// Non-overridden fields should keep defaults
	if cfg.Sections.InProgress != "In Progress" {
		t.Errorf("expected Sections.InProgress='In Progress', got %q", cfg.Sections.InProgress)
	}
	if cfg.Sections.Done != "Done" {
		t.Errorf("expected Sections.Done='Done', got %q", cfg.Sections.Done)
	}
	if cfg.ShowDone != true {
		t.Errorf("expected ShowDone=true (default), got %v", cfg.ShowDone)
	}
}

func TestLocalOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Global config
	globalDir := filepath.Join(dir, "fakehome", ".config", "todo")
	os.MkdirAll(globalDir, 0755)
	globalToml := `file = "global-tasks.md"

[sections]
todo = "Global Todo"
in_progress = "Global WIP"
done = "Global Done"

[ui]
show_done = false
`
	os.WriteFile(filepath.Join(globalDir, "config.toml"), []byte(globalToml), 0644)

	// Local config — only overrides file and sections.todo
	localToml := `file = "local-tasks.md"

[sections]
todo = "Local Backlog"
`
	os.WriteFile(filepath.Join(dir, ".todo.toml"), []byte(localToml), 0644)

	cfg, err := loadWithHome(filepath.Join(dir, "fakehome"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Local overrides
	if cfg.File != "local-tasks.md" {
		t.Errorf("expected File='local-tasks.md', got %q", cfg.File)
	}
	if cfg.Sections.Todo != "Local Backlog" {
		t.Errorf("expected Sections.Todo='Local Backlog', got %q", cfg.Sections.Todo)
	}
	// Global values preserved where local doesn't override
	if cfg.Sections.InProgress != "Global WIP" {
		t.Errorf("expected Sections.InProgress='Global WIP', got %q", cfg.Sections.InProgress)
	}
	if cfg.Sections.Done != "Global Done" {
		t.Errorf("expected Sections.Done='Global Done', got %q", cfg.Sections.Done)
	}
	if cfg.ShowDone != false {
		t.Errorf("expected ShowDone=false (from global), got %v", cfg.ShowDone)
	}
}

func TestPartialConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Only set one field
	toml := `[ui]
show_done = false
`
	os.WriteFile(filepath.Join(dir, ".todo.toml"), []byte(toml), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All defaults except show_done
	if cfg.File != "tasks.md" {
		t.Errorf("expected File='tasks.md', got %q", cfg.File)
	}
	if cfg.Sections.Todo != "Backlog" {
		t.Errorf("expected Sections.Todo='Backlog', got %q", cfg.Sections.Todo)
	}
	if cfg.Sections.InProgress != "In Progress" {
		t.Errorf("expected Sections.InProgress='In Progress', got %q", cfg.Sections.InProgress)
	}
	if cfg.Sections.Done != "Done" {
		t.Errorf("expected Sections.Done='Done', got %q", cfg.Sections.Done)
	}
	if cfg.ShowDone != false {
		t.Errorf("expected ShowDone=false, got %v", cfg.ShowDone)
	}
}
