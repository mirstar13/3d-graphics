package app

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

type ambientFields struct {
	Color     engine.Color
	Intensity float64
}

func sceneRig(s project.SceneSettings) project.LightingSettings {
	if s.Lighting != nil {
		return *s.Lighting
	}
	return project.DefaultLighting()
}

func sceneRow(s project.SceneSettings, row int) (string, any) {
	rig := sceneRig(s)
	switch row {
	case 0:
		if s.Camera != nil {
			return "Camera", s.Camera
		}
		c := project.DefaultCamera()
		return "Camera", &c
	case 1:
		return "Ambient", &ambientFields{rig.Ambient, rig.Intensity}
	case 2:
		if s.Simulation != nil {
			return "Simulation", s.Simulation
		}
		return "Simulation", &project.SimulationSettings{}
	default:
		p := rig.Lights[row-3]
		return p.Name, &p
	}
}

func (m *projectEditorModel) scenePanel() string {
	s := m.document.Document().SceneSettings
	title, v := sceneRow(s, m.sceneField)
	value, _ := json.Marshal(v)
	if m.modal == "scene_edit" {
		return "Edit " + title + " (JSON)\n" + m.draft + "\nEnter apply | Ctrl+U clear | Esc cancel\n" + m.message
	}
	return fmt.Sprintf("Scene settings\nUp/Down select | Enter edit | L add light | D delete\nC reset camera | R reset lighting | S inherit simulation | Esc close\n> %s\n%s\n%s", title, value, m.message)
}

func (m *projectEditorModel) sceneKey(msg tea.KeyMsg) {
	key := msg.String()
	s := m.document.Document().SceneSettings
	rig := sceneRig(s)
	if m.modal == "scene_edit" {
		switch key {
		case "esc":
			m.modal = "scene"
			return
		case "ctrl+u":
			m.draft = ""
			return
		case "backspace", "ctrl+h":
			r := []rune(m.draft)
			if len(r) > 0 {
				m.draft = string(r[:len(r)-1])
			}
			return
		case "enter":
			_, v := sceneRow(s, m.sceneField)
			dec := json.NewDecoder(strings.NewReader(m.draft))
			dec.DisallowUnknownFields()
			if !strings.HasPrefix(strings.TrimSpace(m.draft), "{") || !json.Valid([]byte(m.draft)) {
				m.message = "Invalid JSON"
				return
			}
			if err := dec.Decode(v); err != nil {
				m.message = err.Error()
				return
			}
			switch p := v.(type) {
			case *project.CameraSettings:
				s.Camera = p
			case *project.SimulationSettings:
				s.Simulation = p
			case *project.PointLight:
				if p.ID != rig.Lights[m.sceneField-3].ID {
					m.message = "Point light ID is fixed"
					return
				}
				rig.Lights[m.sceneField-3] = *p
				s.Lighting = &rig
			case *ambientFields:
				rig.Ambient, rig.Intensity = p.Color, p.Intensity
				s.Lighting = &rig
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.draft += string(msg.Runes)
			}
			return
		}
	} else {
		switch strings.ToLower(key) {
		case "esc":
			m.modal = ""
			return
		case "up":
			m.sceneField = max(0, m.sceneField-1)
			return
		case "down":
			m.sceneField = min(2+len(rig.Lights), m.sceneField+1)
			return
		case "enter":
			_, v := sceneRow(s, m.sceneField)
			b, _ := json.Marshal(v)
			m.draft = string(b)
			m.modal = "scene_edit"
			return
		case "c":
			s.Camera = nil
		case "s":
			s.Simulation = nil
		case "r":
			s.Lighting = nil
			m.sceneField = 1
		case "l":
			if len(rig.Lights) >= 8 {
				m.message = "Eight point lights maximum"
				return
			}
			ids := map[string]bool{}
			for _, p := range rig.Lights {
				ids[p.ID] = true
			}
			for i := 1; ; i++ {
				id := fmt.Sprintf("light%d", i)
				if !ids[id] {
					rig.Lights = append(rig.Lights, project.PointLight{ID: id, Name: fmt.Sprintf("Light %d", i), Enabled: true, Color: engine.ColorWhite, Intensity: 1})
					break
				}
			}
			s.Lighting = &rig
			m.sceneField = 2 + len(rig.Lights)
		case "d":
			if m.sceneField < 3 {
				return
			}
			i := m.sceneField - 3
			rig.Lights = append(rig.Lights[:i], rig.Lights[i+1:]...)
			s.Lighting = &rig
			m.sceneField = min(m.sceneField, 2+len(rig.Lights))
		default:
			return
		}
	}
	if err := m.document.SetSceneSettings(s); err != nil {
		m.message = err.Error()
		return
	}
	m.modal = "scene"
	m.message = "Scene settings updated"
	m.refresh()
}
