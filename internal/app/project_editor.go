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
	"github.com/mirstar13/go-3d-graphics/internal/input"
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
	activeResource                                 *project.Resource
	resourcePath                                   string
	assets                                         []project.ResourceInfo
	assetSelected                                  int
	assetTarget                                    string
	bindingSelected                                int
	reticleU, reticleV                             float64
	toolValues                                     [9]string
	toolField                                      int
	runtimeSelection, pendingSpawn                 string
	hitAnchor                                      *game.Anchor
	spawnPreview                                   project.Entity
	spawnSequence                                  uint64
}

func newProjectEditor(settings config.Settings) projectEditorModel {
	return projectEditorModel{settings: settings, width: 80, height: 30, zoom: 1, reticleU: .5, reticleV: .5, toolValues: defaultToolValues}
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
	m.activeResource = nil
	m.resourcePath = ""
	m.project = p
	m.document = ed
	m.preview = preview
	m.play = nil
	m.selected = 0
	m.focus = 0
	m.property = 0
	m.modal = ""
	m.message = "Opened " + p.Manifest.Name
	m.renderPreview()
	return nil
}
func (m *projectEditorModel) selectedEntity() (project.Entity, bool) {
	if m.play != nil {
		if id, ok := m.play.Find(m.runtimeSelection); ok {
			e, err := m.play.State(id)
			return e, err == nil
		}
		return project.Entity{}, false
	}
	if m.document == nil || m.selected < 0 {
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
	m.reconcileRuntime()
	session := m.preview
	if m.play != nil {
		session = m.play
	}
	if session == nil {
		m.frame = nil
		return
	}
	w, h := max(12, min(70, (m.width-3)/2)), max(1, min(26, m.height-11))
	if m.width < 65 {
		w = max(1, m.width)
		h = max(1, min(26, m.height-12))
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
	x, y := m.reticleCell(w, h)
	row := []rune(m.frame[y])
	row[x] = '+'
	m.frame[y] = string(row)

}
func (m *projectEditorModel) save() {
	if m.document == nil {
		return
	}
	var err error
	path := m.project.Manifest.EntryScene
	if m.activeResource != nil {
		resource := *m.activeResource
		resource.Scene = m.document.Document()
		err = m.project.SaveResource(m.resourcePath, resource)
		path = m.resourcePath
		if err == nil {
			m.activeResource = &resource
		}
	} else {
		err = m.project.Save(m.document.Document())
	}
	if err != nil {
		m.message = err.Error()
	} else {
		m.document.MarkSaved()
		m.message = "Saved " + path
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
	return m.finishAction(action)
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
	selected, ok := m.selectedEntity()
	m.runtimeSelection = ""
	m.pendingSpawn = ""
	m.hitAnchor = nil
	if ok {
		m.runtimeSelection = selected.ID
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
		if m.modal == "" && msg.Type == tea.KeyRunes {
			if msg.Paste {
				return m, nil
			}
			if len(msg.Runes) > 1 {
				var commands []tea.Cmd
				for _, r := range msg.Runes {
					next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: msg.Alt})
					m = next.(projectEditorModel)
					if cmd != nil {
						commands = append(commands, cmd)
					}
				}
				return m, tea.Batch(commands...)
			}
		}
		key := msg.String()
		if m.modal != "" {
			if key == "esc" || key == "ctrl+c" {
				m.modal = ""
				return m, nil
			}
			if m.modal == "tools" {
				m.updateTools(key)
				return m, nil
			}
			if m.modal == "bindings" {
				m.updateBindings(key)
				return m, nil
			}
			if m.modal == "assets" {
				cmd := m.updateAssets(key)
				return m, cmd
			}
			if m.modal == "parent" {
				m.updateParentPicker(key)
				return m, nil
			}
			if m.modal == "discard" {
				if key == "d" {
					action := m.pendingAction
					m.modal = ""
					cmd := m.finishAction(action)
					return m, cmd
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
				case "tool_value":
					err = m.saveToolValue()
				case "spawn_confirm":
					err = m.confirmSpawn()
				case "binding":
					err = m.saveBinding()
				case "extract":
					if e, ok := m.selectedEntity(); ok {
						var path string
						path, err = m.project.Extract(m.document.Document(), e.ID, strings.TrimSpace(m.draft))
						if err == nil {
							m.modal = ""
							m.message = "Created resource " + path + "; A browses resources"
						}
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
		case "ctrl+l":
			m.openTools()
			return m, nil
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
		if action := input.Resolve(m.project.Manifest.Bindings, key, m.focus == 2); action != "" {
			key = action
		}
		switch key {
		case input.Pick:
			m.pickReticle()
		case input.Clear:
			m.clearSelection()
			m.message = "Selection cleared"
			m.renderPreview()
		case input.Next:
			m.cycleSelection(1)
		case input.Previous:
			m.cycleSelection(-1)
		case input.AimUp:
			m.moveReticle(0, -1)
		case input.AimDown:
			m.moveReticle(0, 1)
		case input.AimLeft:
			m.moveReticle(-1, 0)
		case input.AimRight:
			m.moveReticle(1, 0)
		case "tab":
			m.focus = (m.focus + 1) % 3
			m.renderPreview()
		case "up", "down":
			if m.focus == 2 {
				break
			}
			delta := 1
			if key == "up" {
				delta = -1
			}
			if m.focus == 0 {
				m.cycleSelection(delta)
			} else {
				count := max(1, len(m.inspectorRows()))
				m.property = (m.property + count + delta) % count
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
		case input.Play:
			if m.play == nil {
				cmd := m.startPlay()
				return m, cmd
			}
		case "f8":
			m.play = nil
			m.renderPreview()
			m.message = "Stopped; authored scene preserved"
		case input.Pause:
			if m.play != nil {
				m.play.World.SetPaused(!m.play.World.Stats().Paused)
				m.lastTick = time.Now()
			}
		case input.Step:
			if m.play != nil && m.play.World.Stats().Paused {
				if err := m.play.Step(); err != nil {
					m.message = err.Error()
				}
				m.renderPreview()
			}
		case input.Impulse:
			m.applyToolImpulse()
		default:
			if m.play != nil {
				m.message = "Stop Play to edit authored properties"
				return m, nil
			}
			switch key {
			case "k":
				m.modal = "bindings"
				m.bindingSelected = 0
			case "a", "r", "ctrl+r", "ctrl+u", "ctrl+t":
				m.resourceCommand(key)
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
		path := m.project.Manifest.EntryScene
		if m.activeResource != nil {
			path = m.resourcePath + " [SOURCE]"
		}
		fmt.Fprintf(&out, "Project editor - %s%s [%s]%s\nResource: %s\n", m.project.Manifest.Name, dirty, mode, runtimeStatus, path)
		side := []string{"Hierarchy (Tab switches focus)"}
		entities := m.hierarchy()
		selected := m.selectionIndex()
		for i := range entities {
			if _, _, ok := m.document.Document().Link(entities[i].ID); ok {
				entities[i].Name += " [linked]"
			}
		}
		first := max(0, selected-2)
		for i := first; i < min(len(entities), first+4); i++ {
			prefix := "  "
			if i == selected {
				prefix = "> "
			}
			side = append(side, prefix+strings.Repeat("  ", min(entities[i].Depth, 8))+entities[i].Name)
		}
		if len(entities) == 0 {
			side = append(side, "(empty: B box / G sphere)")
		}
		focus := "Hierarchy"
		if m.focus == 2 {
			focus = "Viewport"
		}
		if m.focus == 1 {
			focus = "Inspector"
		}
		modeLabel := "local"
		if m.play != nil {
			modeLabel = "live, read-only"
		}
		side = append(side, "", "Inspector ("+modeLabel+") - focus: "+focus)
		side = append(side, m.inspectorSlice(max(2, len(m.frame)-len(side)))...)
		contentHeight := max(1, m.height-11)
		if m.width < 65 || contentHeight < 10 {
			side = nil
			if m.focus == 0 {
				side = append(side, "Hierarchy (Tab: Inspector)")
				first := max(0, selected-contentHeight+2)
				for i := first; i < min(len(entities), first+contentHeight-1); i++ {
					prefix := "  "
					if i == selected {
						prefix = "> "
					}
					side = append(side, prefix+strings.Repeat("  ", min(entities[i].Depth, 8))+entities[i].Name)
				}
			} else if m.focus == 2 {
				side = []string{"Viewport (Tab switches focus)"}
				if m.width < 65 {
					side = append(side, m.frame...)
				}
			} else {
				side = append(side, "Inspector ("+modeLabel+"; Tab: Viewport)")
				side = append(side, m.inspectorSlice(max(1, contentHeight-1))...)
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
		bindings := m.project.Manifest.Bindings
		out.WriteString(input.Help(bindings, input.Play, input.Pause, input.Step, input.Impulse) + " | F8/Esc Stop\n")
		out.WriteString(input.Help(bindings, input.Pick, input.Clear, input.Previous, input.Next) + " | K bindings\n")
		if m.play != nil {
			out.WriteString("Live hierarchy/inspector | Tab focus | Ctrl+L tools\nCtrl+S saves authored scene | Stop discards runtime changes\n")
		} else {
			out.WriteString("B box / G sphere | E edit | Del delete | Ctrl+D duplicate\nCtrl+S save | Ctrl+Z/Y undo/redo | Ctrl+N/O new/open\n")
		}
		if m.focus == 2 {
			out.WriteString(input.Help(bindings, input.AimUp, input.AimDown, input.AimLeft, input.AimRight) + "\n")
		} else {
			out.WriteString("H parent | 1/2/3 camera | +/- zoom | Q exit editor\n")
		}
		if m.play != nil {
			fmt.Fprintf(&out, "Ctrl+L tools | Impulse (%s,%s,%s) %s\n", m.toolValues[0], m.toolValues[1], m.toolValues[2], m.toolValues[3])
		} else {
			out.WriteString("A assets | R extract | Ctrl+R refresh | Ctrl+U unpack | Ctrl+T reset\n")
		}
	}
	if m.modal != "" {
		if m.modal == "discard" {
			out.WriteString("\nUnsaved edits: D discard | Esc cancel\n")
		} else if m.modal == "tools" {
			out.WriteString(m.toolsView())
		} else if m.modal == "tool_value" {
			fmt.Fprintf(&out, "\nEdit %s: %s_\nEnter apply | Esc cancel\n", toolLabels[m.toolField], m.draft)
		} else if m.modal == "spawn_confirm" {
			e := m.spawnPreview
			fmt.Fprintf(&out, "\nSpawn preview: %s\nPosition %.2f %.2f %.2f\nSize %.2f m | Mass %.2f kg\nEnter queue | Esc cancel\n", e.Shape, e.Position.X, e.Position.Y, e.Position.Z, e.Size.X, e.Mass)
		} else if m.modal == "bindings" {
			out.WriteString(m.bindingsView())
		} else if m.modal == "binding" {
			fmt.Fprintf(&out, "\nBind %s: %s_\nComma-separated keys | Enter save | Esc cancel\n", input.Definitions()[m.bindingSelected].Label, m.draft)
		} else if m.modal == "assets" {
			out.WriteString(m.resourceBrowser())
		} else if m.modal == "extract" {
			fmt.Fprintf(&out, "\nResource name: %s_\nEnter create | Esc cancel\n", m.draft)
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
	if m.play != nil {
		if _, ok := m.play.Find(id); ok {
			m.runtimeSelection = id
			m.hitAnchor = nil
		}
		return
	}
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
