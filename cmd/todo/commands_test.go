package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper to run a command in a temp directory with an optional existing todo.md
func runCmd(t *testing.T, existingContent string, args ...string) (string, error) {
	t.Helper()

	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if existingContent != "" {
		if err := os.WriteFile(filepath.Join(dir, defaultFile), []byte(existingContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var buf bytes.Buffer
	cmd := newRootCmd(&buf, func() error { return nil })
	cmd.SetArgs(args)
	execErr := cmd.Execute()
	return buf.String(), execErr
}

// readFile reads todo.md from the current directory
func readFile(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(defaultFile)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRootCommandHasSubcommands(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd(&buf, func() error { return nil })

	// Verify expected subcommands exist
	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}

	for _, name := range []string{"init", "add", "list", "done"} {
		if !subcommands[name] {
			t.Errorf("expected subcommand %q to exist", name)
		}
	}
}

func TestInitCreatesFile(t *testing.T) {
	out, err := runCmd(t, "", "init")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created") {
		t.Errorf("expected Created message, got: %s", out)
	}

	content := readFile(t)
	if !strings.Contains(content, "## Backlog") {
		t.Errorf("expected Backlog section in file, got:\n%s", content)
	}
	if !strings.Contains(content, "## In Progress") {
		t.Errorf("expected In Progress section in file, got:\n%s", content)
	}
	if !strings.Contains(content, "## Done") {
		t.Errorf("expected Done section in file, got:\n%s", content)
	}
}

func TestInitFailsIfFileExists(t *testing.T) {
	_, err := runCmd(t, "existing content", "init")
	if err == nil {
		t.Fatal("expected error when file already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestAddTask(t *testing.T) {
	template := "# Todo\n\n## Backlog\n\n## In Progress\n\n## Done\n"
	out, err := runCmd(t, template, "add", "Write tests")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Added: Write tests") {
		t.Errorf("expected Added message, got: %s", out)
	}

	content := readFile(t)
	if !strings.Contains(content, "- [ ] Write tests") {
		t.Errorf("expected task in file, got:\n%s", content)
	}
}

func TestAddRequiresTitle(t *testing.T) {
	_, err := runCmd(t, "", "add")
	if err == nil {
		t.Fatal("expected error when no title provided")
	}
}

func TestListTasks(t *testing.T) {
	content := "## Backlog\n\n- [ ] Task one priority:high +backend\n- [ ] Task two\n\n## Done\n\n- [x] Task three\n"
	out, err := runCmd(t, content, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1. [ ] Task one priority:high +backend") {
		t.Errorf("expected formatted task one, got:\n%s", out)
	}
	if !strings.Contains(out, "2. [ ] Task two") {
		t.Errorf("expected formatted task two, got:\n%s", out)
	}
	if !strings.Contains(out, "3. [x] Task three") {
		t.Errorf("expected formatted task three, got:\n%s", out)
	}
}

func TestListEmpty(t *testing.T) {
	out, err := runCmd(t, "## Backlog\n", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No tasks found") {
		t.Errorf("expected 'No tasks found', got: %s", out)
	}
}

func TestDoneMarksTask(t *testing.T) {
	content := "## Backlog\n\n- [ ] First task\n- [ ] Second task\n\n## Done\n"
	out, err := runCmd(t, content, "done", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Done: First task") {
		t.Errorf("expected Done message, got: %s", out)
	}

	fileContent := readFile(t)
	if !strings.Contains(fileContent, "- [x] First task") {
		t.Errorf("expected task marked done in file, got:\n%s", fileContent)
	}
	// Second task should remain unchanged
	if !strings.Contains(fileContent, "- [ ] Second task") {
		t.Errorf("expected second task unchanged, got:\n%s", fileContent)
	}
}

func TestDoneInvalidNumber(t *testing.T) {
	content := "## Backlog\n\n- [ ] Only task\n"
	_, err := runCmd(t, content, "done", "5")
	if err == nil {
		t.Fatal("expected error for out-of-range task number")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %v", err)
	}
}

func TestDoneNonNumeric(t *testing.T) {
	_, err := runCmd(t, "## Backlog\n\n- [ ] A task\n", "done", "abc")
	if err == nil {
		t.Fatal("expected error for non-numeric task number")
	}
	if !strings.Contains(err.Error(), "invalid task number") {
		t.Errorf("expected 'invalid task number' error, got: %v", err)
	}
}
