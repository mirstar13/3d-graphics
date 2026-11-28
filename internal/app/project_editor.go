package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/audio"
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
	files           editorFiles
	fileSelection   int
	actionSelection int
	keyboard        bool
	keyPage         int
	audio           *audio.Manager
	audition        *audio.Scope
	soundSelection  int

	scriptedPlay                                   bool
	lookDrag                                       input.Pointer
	lifetime                                       context.Context
	restartChild                                   bool
	build                                          buildPanel
	sceneField                                     int
	settings                                       config.Settings
	project                                        *project.Project
	document                                       *project.Editor
	preview, play                                  *game.Session
	previewRenderer                                *terminal.TerminalRenderer
	width, height, selected, property, focus, view int
	zoom                                           float64
	modal, draft, pendingAction, message           string
	frame, coloredFrame                            []string
	colorMode                                      string
	lastTick                                       time.Time
	timerScheduled                                 bool
	parentChoice                                   int
	activeResource                                 *project.Resource
	resourcePath                                   string
	assets                                         []project.ResourceInfo
	assetSelected                                  int
	meshPaths                                      []string
	meshSelection                                  int
	assetTarget                                    string
	bindingSelected                                int
	reticleU, reticleV                             float64
	toolValues                                     [9]string
	toolField                                      int
	playActor                                      string
	editFocus, editView, editProperty              int
	editZoom                                       float64
	transformTool, transformAxis                   int
	transformSnap                                  bool
	transformDrag                                  *viewportDrag
	runtimeSelection, pendingSpawn                 string
	hitAnchor                                      *game.Anchor
	spawnPreview                                   project.Entity
	spawnSequence                                  uint64
}

func newProjectEditor(settings config.Settings) projectEditorModel {
	mode, _ := terminal.ResolveColorMode(settings.Color)
	return projectEditorModel{audio: audio.NewNative(), colorMode: mode, lifetime: context.Background(), settings: settings, width: 80, height: 30, zoom: 1, reticleU: .5, reticleV: .5, toolValues: defaultToolValues}
}
func (m projectEditorModel) Init() tea.Cmd { return waitAudio(m.audio) }
func (m *projectEditorModel) install(p *project.Project) error {
	ed, err := project.NewEditor(p.Scene)
	if err != nil {
		return err
	}
	preview, err := game.NewProjectSession(p.Root, p.Scene, m.settings.Physics.WorldOptions())
	if err != nil {
		return err
	}
	m.audition.Close()
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
	preview, err := game.NewProjectSession(m.project.Root, m.document.Document(), m.settings.Physics.WorldOptions())
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
	w, h := max(12, m.width-4-min(42, m.width/2)), m.contentHeight()
	if m.width < 65 {
		w = max(1, m.width-1)
		h = max(1, m.contentHeight()-1)
	}
	if m.gameViewActive() {
		w = max(1, m.width-1)
		h = max(1, m.height-1-strings.Count(objectiveHUD(m.play.Objective()), "\n"))
		w, h = terminal.FitViewport(m.settings.Width, m.settings.Height, w, h)
	}
	if m.previewRenderer == nil {
		m.previewRenderer = terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), h, w)
		m.previewRenderer.UseColor = false
		m.previewRenderer.ShadowRenderer = nil
	} else {
		m.previewRenderer.Resize(w, h)
	}
	renderer := m.previewRenderer
	renderer.SetLightingSystem(session.Lighting)
	camera := session.InitialCamera
	eye := camera.Position
	target := camera.Target
	if m.view == 1 {
		eye = engine.Point{Y: 8, Z: -24}
	}
	if m.view == 2 {
		eye = engine.Point{X: 24, Y: 8}
	}
	if m.view != 0 {
		target = engine.Point{Y: 1}
	}
	if m.play == nil {
		session.Scene.Camera.SetPosition(target.X+(eye.X-target.X)*m.zoom, target.Y+(eye.Y-target.Y)*m.zoom, target.Z+(eye.Z-target.Z)*m.zoom)
		session.Scene.Camera.LookAt(target)
	}
	draw := func() { renderer.RenderScene(session.Scene) }
	if m.transformDrag != nil {
		p, q := m.transformDrag.pose()
		session.PreviewPose(m.transformDrag.handle, p, q, draw)
	} else {
		draw()
	}
	if e, ok := m.selectedEntity(); ok && m.transformSession() != nil && m.transformDrag == nil {
		if id, ok := session.Find(e.ID); ok {
			renderer.RenderPhysicsDebug(session.SelectionLines(id), session.Scene.Camera)
		}
	}
	m.frame = terminal.GlyphRows(m.frame, renderer.Surface)
	x, y := m.reticleCell(w, h)
	if m.gameViewActive() {
		m.colorPreview(renderer, -1, -1)
		return
	}
	row := []rune(m.frame[y])
	row[x] = '+'
	m.frame[y] = string(row)
	m.colorPreview(renderer, x, y)
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
func (m *projectEditorModel) newPlaySession() (*game.Session, error) {
	s, err := game.NewProjectSession(m.project.Root, m.document.Document(), m.settings.Physics.WorldOptions())
	if err == nil {
		var clips map[string]*audio.Clip
		clips, err = m.project.LoadSounds()
		if err == nil {
			err = s.ConfigureAudio(clips)
			s.SetResultSounds(m.project.Manifest.WinSound, m.project.Manifest.LossSound)
		}
	}
	if err == nil && m.scriptedPlay {
		s.EnableBehaviors(m.lifetime, m.project)
	}
	if err != nil && s != nil {
		s.Close()
		s = nil
	}
	return s, err
}

func (m *projectEditorModel) stopPlay() {
	if m.play != nil {
		m.play.Close()
		m.focus, m.view, m.zoom, m.property = m.editFocus, m.editView, m.editZoom, m.editProperty
		m.runtimeSelection, m.pendingSpawn, m.hitAnchor = "", "", nil
		m.lookDrag.Reset(false)
		m.transformDrag = nil
		m.play = nil
	}
	m.scriptedPlay = false
	m.renderPreview()
	m.message = "Stopped; authored scene preserved"
}

func (m *projectEditorModel) startPlay() tea.Cmd {
	if m.document == nil {
		return nil
	}
	session, err := m.newPlaySession()
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
	if m.play == nil {
		m.editFocus, m.editView, m.editZoom, m.editProperty = m.focus, m.view, m.zoom, m.property
		if m.preview != nil {
			*session.Scene.Camera.Transform = *m.preview.Scene.Camera.Transform
		}
	}
	m.transformDrag = nil
	m.focus = 2
	m.playActor = ""
	for _, entry := range m.document.Document().Hierarchy() {
		if entry.Body == "dynamic" && (m.playActor == "" || entry.ID == selected.ID) {
			m.playActor = entry.ID
		}
	}
	m.lookDrag.Reset(false)
	if m.play != nil {
		m.play.Close()
	}
	m.audition.Close()
	session.AttachAudio(m.audio)
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
	case interactiveChildResult:
		m.message = childResult{err: msg.err}.status()
		return m, nil
	case childResult:
		m.build.cancel()
		m.build.cancel = nil
		if m.restartChild {
			return m.startChild()
		}
		m.build.output = msg.status()
		return m, nil
	case projectExportResult:
		m.build.cancel()
		m.build.cancel = nil
		m.build.output = "Exported " + msg.path
		if msg.err != nil {
			m.build.output = "Export failed\n" + msg.err.Error()
		}
		return m, nil
	case projectBuildResult:
		if m.build.cancel != nil {
			m.build.cancel()
			m.build.cancel = nil
		}
		m.build.output = "Built " + msg.path
		if msg.err != nil {
			m.build.output = "Build failed\n" + msg.err.Error()
		}
		return m, nil
	case audioFailure:
		if msg.err != nil {
			m.message = msg.err.Error()
		}
		return m, nil
	case tea.BlurMsg:
		m.cancelTransform()
		m.lookDrag.Reset(false)
		return m, nil
	case tea.MouseMsg:
		if m.dragMouse(msg) {
			return m, nil
		}
		if m.modal != "" {
			return m.dialogMouse(msg)
		}
		if m.gameViewActive() {
			switch {
			case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonRight:
				m.lookDrag.Captured = true
				m.lookDrag.Reanchor()
				m.lookDrag.Motion(msg.X, msg.Y)
			case msg.Action == tea.MouseActionRelease:
				m.lookDrag.EndDrag()
			case msg.Action == tea.MouseActionMotion:
				m.lookDrag.Motion(msg.X, msg.Y)
			}
		}
		if next, cmd, ok := m.clickButton(msg); ok {
			return next, cmd
		}
		if next, cmd, ok := m.clickSidebar(msg); ok {
			return next, cmd
		}
		m.viewportMouse(msg)
		return m, nil
	case tea.WindowSizeMsg:
		m.transformDrag = nil
		m.lookDrag.Reset(false)
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
		if m.gameViewActive() {
			dx, dy := m.lookDrag.Take()
			m.play.Scene.Camera.Turn(m.project.Manifest.MouseLook.Angles(dx, dy))
		} else {
			m.lookDrag.Reset(false)
		}
		if err := m.play.Advance(elapsed); err != nil {
			m.stopPlay()
			m.timerScheduled = false
			m.message = err.Error()
			return m, nil
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
		return m.handleEditorKey(msg, msg.String())
	}
	return m, nil
}

func (m projectEditorModel) handleEditorKey(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	if key == "esc" && m.transformDrag != nil {
		m.cancelTransform()
		return m, nil
	}
	if m.transformKey(key) {
		return m, nil
	}
	if m.transformDrag != nil {
		m.cancelTransform()
	}
	if key == "ui.keyboard" {
		m.keyboard = !m.keyboard
		m.keyPage = 0
		return m, nil
	}
	if key == "ui.next_keys" {
		m.keyPage++
		return m, nil
	}
	if key == "ui.clear_draft" {
		m.draft = ""
		return m, nil
	}
	if key == "ui.files" {
		m.browseFiles()
		return m, nil
	}
	if m.modal == "files" {
		return m.fileKey(key)
	}
	if m.modal == "commands" {
		return m.commandKey(key)
	}
	if key == "ui.commands" && m.modal == "" {
		m.actionSelection, m.modal = 0, "commands"
		return m, nil
	}
	if m.modal == "shortcuts" {
		if key == "esc" || key == "?" || key == "q" {
			m.modal = ""
		}
		return m, nil
	}
	if m.project != nil && m.modal == "" && key == "?" {
		m.modal = "shortcuts"
		return m, nil
	}
	if m.modal == "build" || m.modal == "child" || m.modal == "export" {
		return m.buildKey(key)
	}
	if m.modal == "scene" || m.modal == "scene_edit" {
		m.sceneKey(msg, key)
		return m, nil
	}
	if m.modal != "" {
		if key == "esc" || key == "ctrl+c" {
			if m.modal == "sound_assets" || m.modal == "sound_import" {
				m.audition.Close()
			}
			m.modal = ""
			return m, nil
		}
		if m.modal == "sound_assets" {
			m.updateSoundBrowser(key)
			return m, nil
		}
		if m.modal == "mesh_assets" {
			m.updateMeshBrowser(key)
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
			case "sound_import":
				var cue string
				cue, err = m.project.ImportSound(strings.TrimSpace(m.draft))
				if err == nil {
					m.modal = "sound_assets"
					m.soundSelection = slices.Index(m.soundCues(), cue)
					m.message = "Imported " + cue
				}
			case "mesh_import":
				var name string
				name, err = m.project.ImportGLB(strings.TrimSpace(m.draft))
				if err == nil {
					m.browseMeshes()
					m.meshSelection = max(0, slices.Index(m.meshPaths, name))
					m.message = "Imported " + name
				}
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
					prop := project.Properties[m.property]
					if prop.Key == "mesh_asset" || prop.Key == "material_asset" {
						err = prop.Set(&e, m.draft)
						if err == nil {
							_, _, err = e.LoadVisualAssets(m.project.Root)
						}
					}
					if err == nil {
						err = m.document.Set(e.ID, prop.Key, m.draft)
					}
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
	if m.gameViewActive() {
		turn := map[string][2]float64{"shift+left": {-.12, 0}, "shift+right": {.12, 0}, "shift+up": {0, .1}, "shift+down": {0, -.1}}
		if angles, ok := turn[key]; ok {
			m.play.Scene.Camera.Turn(angles[0], angles[1])
			m.renderPreview()
			return m, nil
		}
	}
	if m.play != nil && m.focus == 2 {
		m.play.BehaviorInput(key)
	}
	if m.play != nil && m.focus == 2 && m.playActor != "" {
		if impulse, ok := playerImpulse(key); ok {
			if id, exists := m.play.Find(m.playActor); exists && !m.play.Objective().Finished() {
				if err := m.play.QueueImpulse(id, impulse); err != nil {
					m.message = err.Error()
				}
			}
			return m, nil
		}
	}
	switch key {
	case "r":
		if m.play != nil && m.play.Objective().Finished() {
			cmd := m.startPlay()
			return m, cmd
		}
	case "ctrl+w":
		if m.project != nil && m.play == nil {
			if _, err := m.project.WorkerExecutable(); err != nil {
				m.message = err.Error()
				return m, nil
			}
			m.scriptedPlay = true
			cmd := m.startPlay()
			if m.play == nil {
				m.scriptedPlay = false
			}
			return m, cmd
		}
		return m, nil
	case "ctrl+g":
		if m.project != nil && m.play == nil && m.activeResource == nil {
			return m.startInteractiveChild()
		}
		return m, nil
	case "f6":
		if m.play != nil {
			break
		}
		if m.project != nil && m.activeResource == nil && m.play == nil {
			return m.startChild()
		}
		return m, nil
	case "f9":
		if m.project != nil && m.activeResource == nil && m.play == nil {
			return m.startExport()
		}
		return m, nil
	case "ctrl+b":
		if m.project != nil && m.activeResource == nil && m.play == nil {
			return m.startBuild()
		}
		return m, nil
	case "ctrl+e":
		if m.document != nil && m.play == nil && m.activeResource == nil {
			m.modal = "scene"
			m.sceneField = 0
			m.message = ""
		}
		return m, nil
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
	case "esc", "q", "ctrl+c":
		if m.play != nil {
			m.stopPlay()
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
	case "ui.hierarchy", "ui.inspector", "ui.viewport":
		m.focus = map[string]int{"ui.hierarchy": 0, "ui.inspector": 1, "ui.viewport": 2}[key]
		m.renderPreview()
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
		if m.play != nil {
			break
		}
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
		m.stopPlay()
	case "ctrl+y":
		if m.play != nil {
			m.toggleAudioMute()
			return m, nil
		}
		e, _ := m.selectedEntity()
		if m.document.Redo() {
			m.selectID(e.ID)
			m.refresh()
			m.message = "Redo"
		}
	case input.Pause:
		if m.play != nil {
			m.lookDrag.Reset(false)
			m.play.SetPaused(!m.play.World.Stats().Paused)
			m.renderPreview()
			m.lastTick = time.Now()
		}
	case input.Step:
		if m.play != nil && m.play.World.Stats().Paused {
			if err := m.play.Step(); err != nil {
				m.stopPlay()
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
		case "ctrl+k":
			m.soundSelection, m.modal = 0, "sound_assets"
		case "ctrl+x":
			m.browseMeshes()
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
		}
	}
	return m, nil
}
func (m projectEditorModel) gameViewActive() bool {
	return m.play != nil && m.focus == 2 && m.modal == "" && !m.play.World.Stats().Paused
}
func (m projectEditorModel) View() string {
	if m.gameViewActive() {
		hud := fitView(objectiveHUD(m.play.Objective())+m.buttonText()+" [PLAY]  Right drag look | Tab inspect | F8 stop", m.width, m.height)
		rows := m.frame
		if len(m.coloredFrame) > 0 {
			rows = m.coloredFrame
		}
		if len(m.frame) == 0 {
			return hud
		}
		return terminal.CenteredGameView(hud, rows, len(m.frame[0]), m.width, m.height)
	}

	if m.modal != "" {
		view, _ := m.dialogLayout()
		return view
	}
	var out strings.Builder
	if m.project == nil {
		out.WriteString("Project editor\n\n" + m.buttonText() + " N New project | O Open project\n\nCreate a new directory or open one containing project.yaml.\nEsc returns to launcher.\n")
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
		fmt.Fprintf(&out, "Project editor - %s%s [%s]%s\n%s Resource: %s\n", m.project.Manifest.Name, dirty, mode, runtimeStatus, m.buttonText(), path)
		if m.play != nil {
			out.WriteString(objectiveHUD(m.play.Objective()))
		}
		if m.width >= 65 {
			out.WriteString("── Viewport · click to pick · wheel to zoom ──\n")
		}
		rows := m.sidebarRows()
		side := make([]string, len(rows))
		for i, row := range rows {
			side[i] = row.text
		}
		contentHeight := m.contentHeight()

		if m.width >= 65 {
			w := max(12, m.width-4-min(42, m.width/2))
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
		playKey := input.Help(m.project.Manifest.Bindings, input.Play)
		if m.play != nil {
			if m.focus == 2 && m.playActor != "" {
				fmt.Fprintf(&out, "Arrows move %s · Space jump · Tab inspect · F6 pause · F8 stop\n", m.playActor)
			} else {
				fmt.Fprintf(&out, "? shortcuts | F8 stop | Ctrl+L tools | %s\n", input.Help(m.project.Manifest.Bindings, input.Pause, input.Step))
			}
		} else {
			fmt.Fprintf(&out, "Actions or ? shortcuts | %s | E edit | Tab focus\n", playKey)
		}

	}
	out.WriteString("\n" + printableEditorText(m.message))
	view := out.String()
	view = fitView(view, m.width, m.height)
	return m.stylePreview(view)
}
func runProjectEditor(settings config.Settings, path, action string) error {
	m := newProjectEditor(settings)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.lifetime = ctx
	m.modal = action
	if path != "" {
		p, err := project.Open(path)
		if err != nil {
			return err
		}
		if err = m.install(p); err != nil {
			return err
		}
	}
	defer m.audio.Close()
	final, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithReportFocus()).Run()
	if editor, ok := final.(projectEditorModel); ok && editor.play != nil {
		editor.play.Close()
	}
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

// sampleProjectPath finds the editable example in a checkout or alongside the executable.
func sampleProjectPath(name string) string {
	executable, _ := os.Executable()
	for _, base := range []string{".", filepath.Dir(executable)} {
		path := filepath.Join(base, "examples", name)
		if info, err := os.Stat(filepath.Join(path, "project.yaml")); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

func (m projectEditorModel) shortcutView() string {
	var out strings.Builder
	out.WriteString("Editor shortcuts · Esc close\n\n")
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
		out.WriteString("Ctrl+K audio | Ctrl+X mesh assets | Ctrl+W scripted Play | F9 export | F6 run build | Ctrl+G run in terminal | Ctrl+B build | Ctrl+E scene settings | H parent | 1/2/3 camera | +/- zoom | Q exit editor\n")
	}
	if m.play != nil {
		fmt.Fprintf(&out, "Ctrl+L tools | Impulse (%s,%s,%s) %s\n", m.toolValues[0], m.toolValues[1], m.toolValues[2], m.toolValues[3])
	} else {
		out.WriteString("A assets | R extract | Ctrl+R refresh | Ctrl+U unpack | Ctrl+T reset\n")
	}
	rows := strings.Split(strings.TrimSpace(out.String()), "\n")
	var wrapped []string
	for _, row := range rows {
		line := ""
		for _, item := range strings.Split(row, " | ") {
			if line != "" && len(line)+3+len(item) >= m.width {
				wrapped = append(wrapped, line)
				line = ""
			}
			if line != "" {
				line += " | "
			}
			line += item
		}
		wrapped = append(wrapped, line)
	}
	return fitView(strings.Join(wrapped, "\n"), m.width, m.height)
}

func (m projectEditorModel) contentHeight() int {
	extra := 0
	if m.play != nil {
		extra = strings.Count(objectiveHUD(m.play.Objective()), "\n")
	}
	return max(1, m.height-6-extra)
}
