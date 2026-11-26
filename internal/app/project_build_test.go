package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestProjectBuildCorrectFailRebuild(t *testing.T) {
	m := inputEditor(t)
	root := m.project.Root
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the generated starter offline against the API in this checkout.
	cmd := exec.Command("go", "mod", "edit", "-replace=github.com/mirstar13/go-3d-graphics="+repo)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	source := filepath.Join(root, "main.go")
	good, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	build := func() projectBuildResult {
		t.Helper()
		var run tea.Cmd
		m, run = editorKey(m, tea.KeyMsg{Type: tea.KeyCtrlB})
		if run == nil || m.modal != "build" {
			t.Fatal("missing build command")
		}
		result := run().(projectBuildResult)
		next, _ := m.Update(result)
		m = next.(projectEditorModel)
		return result
	}
	first := build()
	if first.err != nil {
		t.Fatal(first.err)
	}
	binary := filepath.Join(root, first.path)
	before, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(binary)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("starter failed: %v: %s", err, out)
	}
	if err = os.WriteFile(source, []byte("package main\nfunc main(){missingFunction()}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	failed := build()
	if failed.err == nil || !strings.Contains(m.View(), "main.go:2") || !strings.Contains(m.View(), "missingFunction") {
		t.Fatal(m.View())
	}
	after, err := os.ReadFile(binary)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed build replaced executable", err)
	}
	if err = os.WriteFile(source, good, 0644); err != nil {
		t.Fatal(err)
	}
	if fixed := build(); fixed.err != nil {
		t.Fatal(fixed.err)
	}
}
