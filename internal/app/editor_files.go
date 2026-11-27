package app

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

type editorFiles struct {
	directory, returnModal string
	names                  []string
}

func (m *projectEditorModel) browseFiles() {
	dir := "."
	if m.project != nil {
		dir = m.project.Root
	}
	if path, err := filepath.Abs(strings.TrimSpace(m.draft)); err == nil && m.draft != "" {
		parent := filepath.Dir(path)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			dir = path
		} else if info, err := os.Stat(parent); err == nil && info.IsDir() {
			dir = parent
		}
	}
	m.files.returnModal = m.modal
	if m.readDirectory(dir) {
		m.modal = "files"
	}
}

func (m *projectEditorModel) readDirectory(dir string) bool {
	dir, err := filepath.Abs(dir)
	if err != nil {
		m.message = err.Error()
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.message = err.Error()
		return false
	}
	names := []string{".."}
	want := map[string]string{"mesh_import": ".glb", "sound_import": ".wav"}[m.files.returnModal]
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			names = append(names, name+string(filepath.Separator))
		} else if want != "" && strings.EqualFold(filepath.Ext(name), want) {
			files = append(files, name)
		}
	}
	m.files.directory = dir
	m.files.names = append(names, files...)
	m.fileSelection = 0
	m.message = ""
	return true
}

func (m projectEditorModel) fileDialog() editorDialog {
	var body strings.Builder
	body.WriteString("Browse " + printableEditorText(m.files.directory) + "\n")
	visible := max(1, m.height-6)
	first := max(0, m.fileSelection-visible+1)
	end := min(len(m.files.names), first+visible)
	for i := first; i < end; i++ {
		writeChoice(&body, printableEditorText(m.files.names[i]), m.fileSelection == i)
	}
	buttons := []editorButton{{"Cancel", "esc"}, {"Open selection", "enter"}, {"Parent", "ui.up_dir"}, {"Previous", "up"}, {"Next", "down"}}
	if m.files.returnModal == "new" || m.files.returnModal == "open" {
		buttons = append([]editorButton{{"Use folder", "ui.use_dir"}}, buttons...)
	}
	return editorDialog{body: body.String(), buttons: buttons, listRow: 1, first: first, end: end}
}

func (m projectEditorModel) fileKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "ctrl+c":
		m.modal = m.files.returnModal
	case "up":
		m.fileSelection = max(0, m.fileSelection-1)
	case "down", "tab":
		m.fileSelection = min(len(m.files.names)-1, m.fileSelection+1)
	case "ui.up_dir":
		m.readDirectory(filepath.Dir(m.files.directory))
	case "ui.use_dir":
		m.modal, m.draft = m.files.returnModal, m.files.directory
		if m.modal == "new" && !strings.HasSuffix(m.draft, string(filepath.Separator)) {
			m.draft += string(filepath.Separator)
		}
	case "enter":
		if len(m.files.names) == 0 {
			return m, nil
		}
		name := m.files.names[m.fileSelection]
		next := filepath.Join(m.files.directory, name)
		if name == ".." || strings.HasSuffix(name, string(filepath.Separator)) {
			m.readDirectory(next)
		} else {
			m.modal, m.draft = m.files.returnModal, next
		}
	}
	return m, nil
}

func printableEditorText(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '�'
		}
		return r
	}, name)
}
