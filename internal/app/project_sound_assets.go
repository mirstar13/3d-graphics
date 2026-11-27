package app

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/audio"
)

type audioFailure struct{ err error }

func waitAudio(manager *audio.Manager) tea.Cmd {
	if manager == nil {
		return nil
	}
	return func() tea.Msg {
		<-manager.Finished()
		return audioFailure{manager.Err()}
	}
}

func (m *projectEditorModel) toggleAudioMute() {
	m.message = "Audio on"
	if m.audio.ToggleMute() {
		m.message = "Audio muted"
	}
}

func (m *projectEditorModel) soundCues() []string {
	return slices.Sorted(maps.Keys(m.project.Manifest.Sounds))
}

func (m *projectEditorModel) soundBrowser() string {
	var out strings.Builder
	out.WriteString("\nAudio cues\n")
	cues := m.soundCues()
	if len(cues) == 0 {
		out.WriteString("No project WAV cues\n")
	}
	start := max(0, m.soundSelection-1)
	for i := start; i < min(len(cues), start+4); i++ {
		prefix := "  "
		if i == m.soundSelection {
			prefix = "> "
		}
		label := cues[i]
		if label == m.project.Manifest.WinSound {
			label += " · Win"
		}
		if cues[i] == m.project.Manifest.LossSound {
			label += " · Loss"
		}
		fmt.Fprintln(&out, prefix+label)
	}
	out.WriteString("Arrows choose | Enter assign to plate | I import | A audition | S stop | Esc close\n")
	return out.String()
}

func (m *projectEditorModel) setResultCue(won bool, cue string) {
	if err := m.project.SaveResultSound(won, cue); err != nil {
		m.message = err.Error()
		return
	}
	kind := "Loss"
	if won {
		kind = "Win"
	}
	if cue == "" {
		m.message = kind + " cue cleared"
	} else {
		m.message = kind + " cue set to " + cue
	}
}

func (m *projectEditorModel) updateSoundBrowser(key string) {
	cues := m.soundCues()
	switch key {
	case "i":
		m.audition.Close()
		m.modal, m.draft = "sound_import", ""
		return
	case "s":
		m.audition.Close()
		m.message = "Audition stopped"
		return
	case "clear-win", "clear-loss":
		m.setResultCue(key == "clear-win", "")
		return
	case "ctrl+y":
		m.toggleAudioMute()
		return
	}
	if len(cues) == 0 {
		return
	}
	cue := cues[m.soundSelection]
	switch key {
	case "win", "loss":
		m.setResultCue(key == "win", cue)
	case "up":
		m.soundSelection = (m.soundSelection + len(cues) - 1) % len(cues)
	case "down", "tab":
		m.soundSelection = (m.soundSelection + 1) % len(cues)
	case "a":
		if err := m.audio.Err(); err != nil {
			m.message = err.Error()
			return
		}
		clip, err := m.project.LoadSound(cue)
		if err != nil {
			m.message = err.Error()
			return
		}
		m.audition = m.audio.Scope()
		m.audition.Play(clip)
		m.message = "Auditioning " + cue
	case "enter":
		entity, ok := m.selectedEntity()
		if !ok || entity.PlateSignal == "" {
			m.message = "Select an entity with a Plate signal"
			return
		}
		if err := m.document.Set(entity.ID, "plate_sound", cue); err != nil {
			m.message = err.Error()
			return
		}
		m.audition.Close()
		m.modal = ""
		m.refresh()
		m.message = "Assigned " + cue
	}
}
