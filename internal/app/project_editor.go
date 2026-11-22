package app

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

type editorTick time.Time

func nextEditorTick() tea.Cmd {
	return tea.Tick(time.Second/20, func(t time.Time) tea.Msg { return editorTick(t) })
}

type projectEditorModel struct {
	settings                                       config.Settings
	project                                        *project.Project
	document                                       *project.Editor
	preview, play                                  *game.Session
	width, height, selected, property, focus, view int
	zoom                                           float64
	modal, draft, pendingAction, message           string
	frame                                          []string
	lastTick                                       time.Time
	timerScheduled                                 bool
	parentChoice                                   int
}

func newProjectEditor(settings config.Settings) projectEditorModel {
	return projectEditorModel{settings: settings, width: 80, height: 30, zoom: 1}
}
func (m projectEditorModel) Init() tea.Cmd { return nil }
func (m *projectEditorModel) install(p *project.Project) error {
	ed, err := project.NewEditor(p.Scene)
	if err != nil {
		return err
	}
	preview, err := game.NewSession(p.Scene, m.settings.Physics.WorldOptions())
	if err != nil {
		return err
	}
	m.project = p
	m.document = ed
	m.preview = preview
	m.play = nil
	m.selected = 0
	m.property = 0
	m.modal = ""
	m.message = "Opened " + p.Manifest.Name
	m.renderPreview()
	return nil
}
func (m *projectEditorModel) selectedEntity() (project.Entity, bool) {
	if m.document == nil {
		return project.Entity{}, false
	}
	entities := m.document.Document().Hierarchy()
	if len(entities) == 0 {
		return project.Entity{}, false
	}
	m.selected = max(0, min(m.selected, len(entities)-1))
	return entities[m.selected].Entity, true
}
func (m *projectEditorModel) refresh() {
	m.selectedEntity()
	preview, err := game.NewSession(m.document.Document(), m.settings.Physics.WorldOptions())
	if err != nil {
		m.message = err.Error()
		return
	}
	m.preview = preview
	m.renderPreview()
}
func (m *projectEditorModel) renderPreview() {
	session := m.preview
	if m.play != nil {
		session = m.play
	}
	if session == nil {
		m.frame = nil
		return
	}
	w, h := max(12, min(70, (m.width-3)/2)), max(4, min(26, m.height-9))
	if m.width < 65 {
		w = max(12, m.width-2)
		h = 4
	}
	renderer := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), h, w)
	renderer.UseColor = false
	renderer.ShadowRenderer = nil
	renderer.SetLightingSystem(session.Lighting)
	eye := engine.Point{X: 14, Y: 11, Z: -20}
	if m.view == 1 {
		eye = engine.Point{Y: 8, Z: -24}
	}
	if m.view == 2 {
		eye = engine.Point{X: 24, Y: 8}
	}
	session.Scene.Camera.SetPosition(eye.X*m.zoom, eye.Y*m.zoom, eye.Z*m.zoom)
	session.Scene.Camera.LookAt(engine.Point{Y: 1})
	renderer.RenderScene(session.Scene)
	if e, ok := m.selectedEntity(); ok {
		if id, ok := session.Find(e.ID); ok {
			renderer.RenderPhysicsDebug(session.SelectionLines(id), session.Scene.Camera)
		}
	}
	m.frame = make([]string, h)
	for y, row := range renderer.Surface {
		m.frame[y] = string(row)
	}
}
func (m *projectEditorModel) save() {
	if m.document == nil {
		return
	}
	if err := m.project.Save(m.document.Document()); err != nil {
		m.message = err.Error()
	} else {
		m.document.MarkSaved()
		m.message = "Saved " + m.project.Manifest.EntryScene
	}
}
func (m *projectEditorModel) begin(action string) tea.Cmd {
	if m.play != nil {
		m.message = "Stop Play before changing projects"
		return nil
	}
	if m.document != nil && m.document.Dirty() {
		m.modal = "discard"
		m.pendingAction = action
		return nil
	}
	if action == "quit" {
		return tea.Quit
	}
	m.modal = action
	m.draft = ""
	return nil
}
func (m *projectEditorModel) startPlay() tea.Cmd {
	if m.document == nil {
		return nil
	}
	session, err := game.NewSession(m.document.Document(), m.settings.Physics.WorldOptions())
	if err != nil {
		m.message = err.Error()
		return nil
	}
	m.play = session
	m.lastTick = time.Now()
	m.message = "Playing a copy; Stop preserves authored edits"
	m.renderPreview()
	if !m.timerScheduled {
		m.timerScheduled = true
		return nextEditorTick()
	}
	return nil
}
func (m projectEditorModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.renderPreview()
		return m, nil
	case editorTick:
		if m.play == nil {
			m.timerScheduled = false
			return m, nil
		}
		now := time.Now()
		elapsed := now.Sub(m.lastTick)
		m.lastTick = now
		if err := m.play.Advance(elapsed); err != nil {
			m.message = err.Error()
		}
		m.renderPreview()
		return m, nextEditorTick()
	case tea.KeyMsg:
		key := msg.String()
		if m.modal != "" {
			if key == "esc" || key == "ctrl+c" {
				m.modal = ""
				return m, nil
			}
			if m.modal == "parent" {
				m.updateParentPicker(key)
				return m, nil
			}
			if m.modal == "discard" {
				if key == "d" {
					action := m.pendingAction
					m.modal = ""
					if action == "quit" {
						return m, tea.Quit
					}
					m.modal = action
					m.draft = ""
				}
				return m, nil
			}
			switch key {
			case "enter":
				var err error
				switch m.modal {
				case "new", "open":
					var p *project.Project
					path := strings.TrimSpace(m.draft)
					if path == "" {
						err = fmt.Errorf("enter a project directory")
					} else if m.modal == "new" {
						p, err = project.Create(path)
					} else {
						p, err = project.Open(path)
					}
					if err == nil {
						err = m.install(p)
					}
				case "property":
					if e, ok := m.selectedEntity(); ok {
						err = m.document.Set(e.ID, project.Properties[m.property].Key, m.draft)
					}
					if err == nil {
						m.modal = ""
						m.refresh()
						m.message = "Property updated"
					}
				}
				if err != nil {
					m.message = err.Error()
				}
			case "backspace", "ctrl+h":
				runes := []rune(m.draft)
				if len(runes) > 0 {
					m.draft = string(runes[:len(runes)-1])
				}
			default:
				if msg.Type == tea.KeyRunes && len([]rune(m.draft)) < 512 {
					runes := append([]rune(m.draft), msg.Runes...)
					m.draft = string(runes[:min(512, len(runes))])
				}
			}
			return m, nil
		}
		if key == "ctrl+c" {
			if m.play != nil {
				m.play = nil
				m.renderPreview()
				m.message = "Stopped; authored scene preserved"
				return m, nil
			}
			cmd := m.begin("quit")
			return m, cmd
		}
		switch key {
		case "ctrl+n":
			cmd := m.begin("new")
			return m, cmd
		case "ctrl+o":
			cmd := m.begin("open")
			return m, cmd
		case "ctrl+s":
			m.save()
			return m, nil
		case "esc", "q":
			if m.play != nil {
				m.play = nil
				m.renderPreview()
				m.message = "Stopped; authored scene preserved"
				return m, nil
			}
			cmd := m.begin("quit")
			return m, cmd
		}
		if m.document == nil {
			if key == "n" {
				m.modal = "new"
				m.draft = ""
			}
			if key == "o" {
				m.modal = "open"
				m.draft = ""
			}
			return m, nil
		}
		switch key {
		case "tab":
			m.focus = 1 - m.focus
		case "up", "down":
			delta := 1
			if key == "up" {
				delta = -1
			}
			if m.focus == 0 {
				count := len(m.document.Document().Entities)
				if count > 0 {
					m.selected = (m.selected + count + delta) % count
					m.renderPreview()
				}
			} else {
				m.property = (m.property + len(project.Properties) + delta) % len(project.Properties)
			}
		case "1", "2", "3":
			m.view = 0
			if key == "1" {
				m.view = 1
			}
			if key == "2" {
				m.view = 2
			}
			m.renderPreview()
		case "+", "=":
			m.zoom = max(.25, m.zoom*.8)
			m.renderPreview()
		case "-":
			m.zoom = min(5, m.zoom*1.25)
			m.renderPreview()
		case "f5", "p":
			if m.play == nil {
				cmd := m.startPlay()
				return m, cmd
			}
		case "f8":
			m.play = nil
			m.renderPreview()
			m.message = "Stopped; authored scene preserved"
		case "f6", " ":
			if m.play != nil {
				m.play.World.SetPaused(!m.play.World.Stats().Paused)
				m.lastTick = time.Now()
			}
		case "f7", "n":
			if m.play != nil && m.play.World.Stats().Paused {
				if err := m.play.Step(); err != nil {
					m.message = err.Error()
				}
				m.renderPreview()
			}
		case "i":
			if m.play != nil {
				if e, ok := m.selectedEntity(); ok {
					if id, ok := m.play.Find(e.ID); ok {
						if err := m.play.QueueImpulse(id, engine.Point{Y: 5}); err != nil {
							m.message = err.Error()
						} else {
							m.message = "Upward impulse queued for next tick"
						}
					}
				}
			}
		default:
			if m.play != nil {
				m.message = "Stop Play to edit authored properties"
				return m, nil
			}
			switch key {
			case "h":
				if e, ok := m.selectedEntity(); ok {
					m.modal = "parent"
					m.message = "Choose a parent; local pose stays unchanged"
					if e.Body != "none" {
						m.message = "Physics roots: set Body to none before parenting"
					}
					m.parentChoice = 0
					for i, p := range m.document.Document().ParentChoices(e.ID) {
						if p.ID == e.Parent {
							m.parentChoice = i + 1
						}
					}
				}
			case "enter", "e":
				if e, ok := m.selectedEntity(); ok {
					m.focus = 1
					m.modal = "property"
					m.draft = project.Properties[m.property].Get(e)
				}
			case "b", "g":
				shape := "box"
				if key == "g" {
					shape = "sphere"
				}
				if id, err := m.document.Add(shape); err != nil {
					m.message = err.Error()
				} else {
					m.selectID(id)
					m.refresh()
					m.message = "Added " + shape
				}
			case "delete":
				if e, ok := m.selectedEntity(); ok {
					if err := m.document.Delete(e.ID); err != nil {
						m.message = err.Error()
					} else {
						m.refresh()
						m.message = "Deleted subtree; Ctrl+Z undoes"
					}
				}
			case "ctrl+d":
				if e, ok := m.selectedEntity(); ok {
					if id, err := m.document.Duplicate(e.ID); err != nil {
						m.message = err.Error()
					} else {
						m.selectID(id)
						m.refresh()
					}
				}
			case "ctrl+z":
				e, _ := m.selectedEntity()
				if m.document.Undo() {
					m.selectID(e.ID)
					m.refresh()
					m.message = "Undo"
				}
			case "ctrl+y":
				e, _ := m.selectedEntity()
				if m.document.Redo() {
					m.selectID(e.ID)
					m.refresh()
					m.message = "Redo"
				}
			}
		}
	}
	return m, nil
}
func (m projectEditorModel) View() string {
	var out strings.Builder
	if m.project == nil {
		out.WriteString("Project editor\n\nN New project | O Open project\n\nCreate a new directory or open one containing project.yaml.\nEsc returns to launcher.\n")
	} else {
		dirty := ""
		if m.document.Dirty() {
			dirty = " *"
		}
		mode := "EDIT"
		if m.play != nil {
			mode = "PLAY"
			if m.play.World.Stats().Paused {
				mode = "PAUSED"
			}
		}
		runtimeStatus := ""
		if m.play != nil {
			runtimeStatus = fmt.Sprintf(" Tick:%d Pending:%d", m.play.World.Stats().Tick, m.play.PendingCommands())
		}
		fmt.Fprintf(&out, "Project editor - %s%s [%s]%s\nResource: %s\n", m.project.Manifest.Name, dirty, mode, runtimeStatus, m.project.Manifest.EntryScene)
		side := []string{"Hierarchy (Tab switches focus)"}
		entities := m.document.Document().Hierarchy()
		first := max(0, m.selected-2)
		for i := first; i < min(len(entities), first+4); i++ {
			prefix := "  "
			if i == m.selected {
				prefix = "> "
			}
			side = append(side, prefix+strings.Repeat("  ", min(entities[i].Depth, 8))+entities[i].Name)
		}
		if len(entities) == 0 {
			side = append(side, "(empty: B box / G sphere)")
		}
		focus := "Hierarchy"
		if m.focus == 1 {
			focus = "Inspector"
		}
		side = append(side, "", "Inspector (local) - focus: "+focus)
		if e, ok := m.selectedEntity(); ok {
			first = max(0, m.property-1)
			visible := max(2, len(m.frame)-len(side))
			for i := first; i < min(len(project.Properties), first+visible); i++ {
				p := project.Properties[i]
				prefix := "  "
				if i == m.property {
					prefix = "> "
				}
				side = append(side, prefix+p.Label+": "+p.Get(e)+" "+p.Units)
			}
			if m.play != nil {
				if id, ok := m.play.Find(e.ID); ok {
					if state, err := m.play.State(id); err == nil {
						side = append(side, fmt.Sprintf("Live Y:%.2f Tick:%d Pending:%d", state.Position.Y, m.play.World.Stats().Tick, m.play.PendingCommands()))
					}
				}
			}
		}
		contentHeight := max(1, m.height-9)
		if m.width < 65 || contentHeight < 10 {
			side = nil
			if m.focus == 0 {
				side = append(side, "Hierarchy (Tab: Inspector)")
				first := max(0, m.selected-contentHeight+2)
				for i := first; i < min(len(entities), first+contentHeight-1); i++ {
					prefix := "  "
					if i == m.selected {
						prefix = "> "
					}
					side = append(side, prefix+strings.Repeat("  ", min(entities[i].Depth, 8))+entities[i].Name)
				}
			} else if e, ok := m.selectedEntity(); ok {
				side = append(side, "Inspector (local; Tab: Hierarchy)")
				first := max(0, m.property-contentHeight+2)
				for i := first; i < min(len(project.Properties), first+contentHeight-1); i++ {
					p := project.Properties[i]
					prefix := "  "
					if i == m.property {
						prefix = "> "
					}
					side = append(side, prefix+p.Label+": "+p.Get(e))
				}
			}
		}
		if m.width >= 65 {
			w := max(12, min(70, (m.width-3)/2))
			for i := 0; i < min(contentHeight, max(len(m.frame), len(side))); i++ {
				a, b := "", ""
				if i < len(m.frame) {
					a = m.frame[i]
				}
				if i < len(side) {
					b = side[i]
				}
				fmt.Fprintf(&out, "%-*s | %s\n", w, a, b)
			}
		} else {
			for _, line := range side[:min(contentHeight, len(side))] {
				out.WriteString(line + "\n")
			}
		}
		out.WriteString("B box / G sphere | E edit | Del delete | Ctrl+D duplicate\nCtrl+S save | Ctrl+Z/Y undo/redo | Ctrl+N/O new/open\nP/F5 Play | Space pause | N step | I impulse | F8/Esc Stop\nH parent | 1/2/3 camera | +/- zoom | Q exit editor\n")
	}
	if m.modal != "" {
		if m.modal == "discard" {
			out.WriteString("\nUnsaved edits: D discard | Esc cancel\n")
		} else if m.modal == "parent" {
			e, ok := m.selectedEntity()
			if ok {
				choices := m.document.Document().ParentChoices(e.ID)
				out.WriteString("\nParent picker (keeps LOCAL pose)\n")
				first := max(0, m.parentChoice-1)
				for i := first; i < min(len(choices)+1, first+3); i++ {
					label := "Scene root"
					if i > 0 {
						label = choices[i-1].Name + " [" + choices[i-1].ID + "]"
					}
					prefix := "  "
					if i == m.parentChoice {
						prefix = "> "
					}
					out.WriteString(prefix + label + "\n")
				}
				out.WriteString("Up/Down choose | Enter apply | Esc cancel\n")
			}
		} else if m.modal == "property" {
			p := project.Properties[m.property]
			fmt.Fprintf(&out, "\nEdit %s %s %v: %s_\nEnter apply | Esc cancel\n", p.Label, p.Units, p.Choices, m.draft)
		} else {
			fmt.Fprintf(&out, "\n%s project directory: %s_\nEnter confirm | Esc cancel\n", m.modal, m.draft)
		}
	}
	out.WriteString("\n" + m.message)
	// Keep dialogs and errors visible even in a short terminal.
	view := out.String()
	if m.modal != "" {
		lines := strings.Split(view, "\n")
		if len(lines) > m.height {
			view = strings.Join(lines[len(lines)-m.height:], "\n")
		}
	}
	return fitView(view, m.width, m.height)
}
func runProjectEditor(settings config.Settings, path string) error {
	m := newProjectEditor(settings)
	if path != "" {
		p, err := project.Open(path)
		if err != nil {
			return err
		}
		if err = m.install(p); err != nil {
			return err
		}
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// selectID preserves identity when tree order changes after a document command.
func (m *projectEditorModel) selectID(id string) {
	for i, e := range m.document.Document().Hierarchy() {
		if e.ID == id {
			m.selected = i
			return
		}
	}
}
func (m *projectEditorModel) updateParentPicker(key string) {
	e, ok := m.selectedEntity()
	if !ok {
		m.modal = ""
		return
	}
	choices := m.document.Document().ParentChoices(e.ID)
	count := len(choices) + 1
	switch key {
	case "up":
		m.parentChoice = (m.parentChoice + count - 1) % count
	case "down", "tab":
		m.parentChoice = (m.parentChoice + 1) % count
	case "enter":
		parent := ""
		if m.parentChoice > 0 {
			parent = choices[m.parentChoice-1].ID
		}
		if err := m.document.Reparent(e.ID, parent); err != nil {
			m.message = err.Error()
			return
		}
		m.modal = ""
		m.selectID(e.ID)
		m.refresh()
		m.message = "Parent updated; local pose preserved"
	}
}
