package app

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/demo"
)

type settingField struct {
	label   string
	value   string
	choices []string
	help    string
}

type settingsModel struct {
	settings     config.Settings
	fields       []settingField
	path         string
	selected     int
	height       int
	width        int
	editing      bool
	draft        string
	message      string
	start        bool
	returnToMenu bool
}

func newSettingsModel(settings config.Settings, path string) settingsModel {
	number := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	toggle := []string{"true", "false"}
	return settingsModel{
		settings: settings, path: path, height: 24, width: 80,
		fields: []settingField{
			{fmt.Sprintf("Demo (1-%d)", len(demo.Presets)), strconv.Itoa(settings.Demo), nil, "Select a curated scene; New scene shows descriptions."},
			{"CPU mode", settings.RenderMode, config.RenderModes, "single, tiles, jobs, or batched"},
			{"Anti-aliasing", settings.AntiAliasing, config.AAModes, "MSAA samples triangle edges at 2 or 4 positions; SSAA renders 2x each dimension."},
			{"Workers", strconv.Itoa(settings.Workers), nil, "Worker count for parallel CPU modes."},
			{"Tile size", strconv.Itoa(settings.TileSize), nil, "Tile width and height in terminal cells."},
			{"Target FPS", number(settings.FPS), nil, "1-240 frames per second."},
			{"Width", strconv.Itoa(settings.Width), nil, "0 fits the terminal; explicit sizes are capped to the window."},
			{"Height", strconv.Itoa(settings.Height), nil, "0 fits the terminal; one row is reserved for status."},
			{"Color", settings.Color, config.ColorModes, "auto follows TERM, COLORTERM, and NO_COLOR."},
			{"Horizontal FOV", number(settings.FOVX), nil, "Field of view in degrees (1-179). Larger angles show more of the scene."},
			{"Vertical FOV", number(settings.FOVY), nil, "Field of view in degrees (1-179). Larger angles show more of the scene."},
			{"Auto orbit", strconv.FormatBool(settings.AutoOrbit), toggle, "Orbit the scene automatically until movement input."},
			{"Debug display", strconv.FormatBool(settings.ShowDebug), toggle, "Show the FPS and camera status line."},
			{"Profiling", strconv.FormatBool(settings.Profiling), toggle, "Collect internal frame timing statistics."},
			{"Gravity X (m/s²)", number(settings.Physics.Gravity.X), nil, "Physics changes restart the scene. Fixed simulation remains 60 Hz."},
			{"Gravity Y (m/s²)", number(settings.Physics.Gravity.Y), nil, "Default -9.81; +Y is up."},
			{"Gravity Z (m/s²)", number(settings.Physics.Gravity.Z), nil, "Gravity acceleration along Z."},
			{"Velocity iterations", strconv.Itoa(settings.Physics.VelocityIterations), nil, "1-100 solver iterations; more cost more CPU time."},
			{"Position iterations", strconv.Itoa(settings.Physics.PositionIterations), nil, "1-100 penetration correction iterations."},
			{"Sleeping", strconv.FormatBool(settings.Physics.Sleeping), toggle, "Let settled contact islands sleep; impacts and support changes wake them."},
			{"Sphere CCD", strconv.FormatBool(settings.Physics.CCD), toggle, "Default sphere sweeps against static planes, boxes, and meshes. Other pairs remain discrete."},
			{"Interpolation", strconv.FormatBool(settings.Physics.Interpolation), toggle, "Smooth running body poses with up to one tick of visual latency; debug views stay exact."},
		},
	}
}

func (m settingsModel) Init() tea.Cmd { return nil }

func (m settingsModel) values() (config.Settings, error) {
	s := m.settings
	ints := []struct {
		index  int
		target *int
	}{
		{0, &s.Demo}, {3, &s.Workers}, {4, &s.TileSize}, {6, &s.Width}, {7, &s.Height}, {17, &s.Physics.VelocityIterations}, {18, &s.Physics.PositionIterations},
	}
	for _, field := range ints {
		value, err := strconv.Atoi(m.fields[field.index].value)
		if err != nil {
			return s, fmt.Errorf("%s must be an integer", m.fields[field.index].label)
		}
		*field.target = value
	}
	floats := []struct {
		index  int
		target *float64
	}{{5, &s.FPS}, {9, &s.FOVX}, {10, &s.FOVY}, {14, &s.Physics.Gravity.X}, {15, &s.Physics.Gravity.Y}, {16, &s.Physics.Gravity.Z}}
	for _, field := range floats {
		value, err := strconv.ParseFloat(m.fields[field.index].value, 64)
		if err != nil {
			return s, fmt.Errorf("%s must be a number", m.fields[field.index].label)
		}
		*field.target = value
	}
	s.RenderMode, s.AntiAliasing, s.Color = m.fields[1].value, m.fields[2].value, m.fields[8].value
	s.AutoOrbit = m.fields[11].value == "true"
	s.ShowDebug = m.fields[12].value == "true"
	s.Profiling = m.fields[13].value == "true"
	s.Physics.Sleeping = m.fields[19].value == "true"
	s.Physics.CCD = m.fields[20].value == "true"
	s.Physics.Interpolation = m.fields[21].value == "true"
	return s, s.Validate()
}

func (m settingsModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.editing {
			switch key {
			case "esc":
				m.editing = false
			case "enter":
				m.fields[m.selected].value = m.draft
				m.editing = false
			case "backspace", "ctrl+h":
				if len(m.draft) > 0 {
					m.draft = m.draft[:len(m.draft)-1]
				}
			default:
				if msg.Type == tea.KeyRunes && len(m.draft) < 12 {
					for _, r := range msg.Runes {
						if strings.ContainsRune("0123456789.-", r) {
							m.draft += string(r)
						}
					}
				}
			}
			return m, nil
		}
		switch key {
		case "q", "esc":
			return m, tea.Quit
		case "up", "k":
			m.selected = (m.selected + len(m.fields) - 1) % len(m.fields)
		case "down", "j", "tab":
			m.selected = (m.selected + 1) % len(m.fields)
		case "e":
			if len(m.fields[m.selected].choices) == 0 {
				m.editing, m.draft = true, m.fields[m.selected].value
			}
		case "left", "right", "h", "l", " ":
			step := 1
			if key == "left" || key == "h" {
				step = -1
			}
			field := &m.fields[m.selected]
			if len(field.choices) > 0 {
				index := 0
				for i, value := range field.choices {
					if value == field.value {
						index = i
						break
					}
				}
				field.value = field.choices[(index+step+len(field.choices))%len(field.choices)]
			} else {
				value, err := strconv.ParseFloat(field.value, 64)
				if err == nil {
					field.value = strconv.FormatFloat(value+float64(step), 'f', -1, 64)
				}
			}
			m.message = ""
		case "s", "enter":
			settings, err := m.values()
			if err != nil {
				m.message = err.Error()
				return m, nil
			}
			if key == "s" {
				if err := config.Save(m.path, settings); err != nil {
					m.message = "Save failed: " + err.Error()
				} else {
					m.settings = settings
					m.message = "Saved " + m.path
				}
			} else {
				m.settings, m.start = settings, true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m settingsModel) View() string {
	var out strings.Builder
	out.WriteString("Renderer settings\n\n")
	visible := max(1, m.height-9)
	first := max(0, m.selected-visible+1)
	last := min(len(m.fields), first+visible)
	for index := first; index < last; index++ {
		field := m.fields[index]
		prefix := "  "
		value := field.value
		if index == m.selected {
			prefix = "> "
			if m.editing {
				value = m.draft + "_"
			}
		}
		fmt.Fprintf(&out, "%s%-18s %s\n", prefix, field.label, value)
	}
	help := m.fields[m.selected].help
	if m.selected == 0 {
		index, err := strconv.Atoi(m.fields[0].value)
		names := demo.Names
		if err == nil && index >= 1 && index <= len(names) {
			help = names[index-1]
		}
	}
	out.WriteString("\n" + help + "\n")
	if m.editing {
		out.WriteString("Type value | Backspace delete | Enter accept | Esc cancel\n")
	} else {
		out.WriteString("Up/Down select | Left/Right change | E edit number\n")
		if m.returnToMenu {
			out.WriteString("Enter apply | S save YAML | Esc back\n")
		} else {
			out.WriteString("Enter run (P returns) | S save YAML | Esc menu\n")
		}
	}
	out.WriteString(m.message)
	return fitView(out.String(), m.width, m.height)
}

func editSettings(settings config.Settings, path string) (config.Settings, bool, error) {
	result, err := tea.NewProgram(newSettingsModel(settings, path), tea.WithAltScreen()).Run()
	if err != nil {
		return settings, false, err
	}
	model := result.(settingsModel)
	return model.settings, model.start, nil
}
