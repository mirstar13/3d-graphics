package app

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/audio"
	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

const soundButtons = "[Import] [Audition] [Stop]"

func TestEditorAudioClickAssignAndReopen(t *testing.T) {
	p, err := project.Create(filepath.Join(t.TempDir(), "audio"))
	if err != nil {
		t.Fatal(err)
	}
	model := newProjectEditor(config.Default())
	model.audio = audio.New(nil)
	t.Cleanup(model.audio.Close)
	if err := model.install(p); err != nil {
		t.Fatal(err)
	}
	model.selectID("floor")
	for _, edit := range [][2]string{{"sensor", "true"}, {"plate_signal", "gate"}} {
		if err := model.document.Set("floor", edit[0], edit[1]); err != nil {
			t.Fatal(err)
		}
	}
	model, _ = editorKey(model, tea.KeyMsg{Type: tea.KeyCtrlK})
	click := func(button string) {
		t.Helper()
		lines := strings.Split(model.View(), "\n")
		y := slices.IndexFunc(lines, func(line string) bool { return strings.HasPrefix(line, soundButtons) })
		if y < 0 {
			t.Fatalf("audio buttons missing from %q", model.View())
		}
		next, _ := model.Update(tea.MouseMsg{X: strings.Index(soundButtons, button), Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
		model = next.(projectEditorModel)
	}
	click("[Import]")
	if model.modal != "sound_import" {
		t.Fatal(model.modal)
	}
	model.draft = "../../examples/switch-room/assets/plate_down.wav"
	model, _ = editorKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.modal != "sound_assets" || len(model.soundCues()) != 1 {
		t.Fatal(model.message)
	}
	click("[Audition]")
	if model.audition == nil {
		t.Fatal(model.message)
	}
	click("[Stop]")
	block := make([]byte, 4096)
	_, _ = model.audio.Read(block)
	for _, b := range block {
		if b != 0 {
			t.Fatal("stopped audition still emits PCM")
		}
	}
	model, _ = editorKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	model.save()
	reopened, err := project.Open(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Scene.Entities[0].PlateSound != "plate_down" {
		t.Fatal(model.message, reopened.Scene.Entities[0].PlateSound)
	}
}
