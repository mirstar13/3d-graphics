package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

var toolLabels = []string{"Impulse X", "Impulse Y", "Impulse Z", "Apply at (center/hit)", "Spawn shape (box/sphere)", "Size / radius", "Mass (kg)", "Ray distance (m)", "Entity budget"}
var defaultToolValues = [9]string{"0", "5", "0", "center", "box", "0.5", "1", "5", "128"}

type manipulationOptions struct {
	impulse              engine.Point
	atHit                bool
	shape                string
	size, mass, distance float64
	budget               int
}

func parseToolValues(values [9]string) (manipulationOptions, error) {
	var n [9]float64
	for _, i := range []int{0, 1, 2, 5, 6, 7, 8} {
		v, err := strconv.ParseFloat(values[i], 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return manipulationOptions{}, fmt.Errorf("%s must be finite", toolLabels[i])
		}
		n[i] = v
	}
	for _, i := range []int{0, 1, 2} {
		if math.Abs(n[i]) > 1000 {
			return manipulationOptions{}, fmt.Errorf("impulse components must be -1000..1000 N s")
		}
	}
	if values[3] != "center" && values[3] != "hit" {
		return manipulationOptions{}, fmt.Errorf("apply at must be center or hit")
	}
	if values[4] != "box" && values[4] != "sphere" {
		return manipulationOptions{}, fmt.Errorf("spawn shape must be box or sphere")
	}
	if n[5] < .05 || n[5] > 10 || n[6] < .01 || n[6] > 10000 || n[7] < .5 || n[7] > 100 || n[8] < 1 || n[8] > 128 || math.Trunc(n[8]) != n[8] {
		return manipulationOptions{}, fmt.Errorf("size .05..10, mass .01..10000, distance .5..100, budget integer 1..128")
	}
	return manipulationOptions{impulse: engine.Point{X: n[0], Y: n[1], Z: n[2]}, atHit: values[3] == "hit", shape: values[4], size: n[5], mass: n[6], distance: n[7], budget: int(n[8])}, nil
}
func (m *projectEditorModel) openTools() {
	if m.play == nil {
		m.message = "Start Play before opening manipulation tools"
		return
	}
	m.play.World.SetPaused(true)
	m.lastTick = time.Now()
	m.modal = "tools"
	m.toolField = 0
	m.message = "Play paused; commands apply on next step"
}
func (m *projectEditorModel) updateTools(key string) {
	switch key {
	case "up":
		m.toolField = (m.toolField + len(toolLabels) - 1) % len(toolLabels)
	case "down", "tab":
		m.toolField = (m.toolField + 1) % len(toolLabels)
	case "enter":
		m.modal = "tool_value"
		m.draft = m.toolValues[m.toolField]
	case "i":
		m.applyToolImpulse()
	case "g":
		m.previewSpawn()
	case "d":
		if e, ok := m.selectedEntity(); ok {
			if id, ok := m.play.Find(e.ID); ok {
				if err := m.play.QueueDeleteSpawned(id); err != nil {
					m.message = err.Error()
				} else {
					m.message = "Runtime deletion queued"
				}
			}
		} else {
			m.message = "Select a runtime spawn to delete"
		}
	case "r":
		m.resetPlay()
	}
}
func (m *projectEditorModel) saveToolValue() error {
	next := m.toolValues
	next[m.toolField] = strings.TrimSpace(strings.ToLower(m.draft))
	if _, err := parseToolValues(next); err != nil {
		return err
	}
	m.toolValues = next
	m.modal = "tools"
	m.message = "Tool value updated"
	return nil
}
func (m *projectEditorModel) applyToolImpulse() {
	if m.play == nil {
		return
	}
	options, err := parseToolValues(m.toolValues)
	if err != nil {
		m.message = err.Error()
		return
	}
	selected, ok := m.selectedEntity()
	if !ok {
		m.message = "Select a dynamic body"
		return
	}
	id, ok := m.play.Find(selected.ID)
	if !ok {
		m.message = "Selected entity no longer exists"
		return
	}
	if options.atHit {
		if m.hitAnchor == nil || m.hitAnchor.Entity != id {
			m.message = "Pick this dynamic collider before using hit mode"
			return
		}
		err = m.play.QueueImpulseAtAnchor(*m.hitAnchor, options.impulse)
	} else {
		err = m.play.QueueImpulse(id, options.impulse)
	}
	if err != nil {
		m.message = err.Error()
		return
	}
	m.message = "Impulse queued for next tick"
	if options.impulse == (engine.Point{Y: 5}) && !options.atHit {
		m.message = "Upward impulse queued for next tick"
	}
}
func (m *projectEditorModel) previewSpawn() {
	options, err := parseToolValues(m.toolValues)
	if err != nil {
		m.message = err.Error()
		return
	}
	h, w := len(m.frame), len(m.frame[0])
	x, y := m.reticleCell(w, h)
	origin, direction, _, err := m.play.Scene.Camera.CellRay(x, y, w, h)
	if err != nil {
		m.message = err.Error()
		return
	}
	var id string
	for {
		m.spawnSequence++
		id = fmt.Sprintf("runtime-%d", m.spawnSequence)
		if _, exists := m.play.Find(id); !exists {
			break
		}
	}
	candidate := project.NewEntity(id, fmt.Sprintf("Runtime %s %d", options.shape, m.spawnSequence))
	candidate.Shape = options.shape
	candidate.Size = engine.Point{X: options.size, Y: options.size, Z: options.size}
	candidate.Mass = options.mass
	candidate.Position = engine.Point{X: origin.X + direction.X*options.distance, Y: origin.Y + direction.Y*options.distance, Z: origin.Z + direction.Z*options.distance}
	if err = m.play.CheckSpawn(candidate, options.budget); err != nil {
		m.message = err.Error()
		return
	}
	m.spawnPreview = candidate
	m.modal = "spawn_confirm"
	m.message = "Placement clear; checked again at the tick boundary"
}
func (m *projectEditorModel) confirmSpawn() error {
	options, err := parseToolValues(m.toolValues)
	if err != nil {
		return err
	}
	if err = m.play.QueueCheckedSpawn(m.spawnPreview, options.budget); err != nil {
		return err
	}
	m.pendingSpawn = m.spawnPreview.ID
	m.modal = "tools"
	m.message = "Spawn queued; close tools and step"
	return nil
}
func (m *projectEditorModel) resetPlay() {
	paused := m.play.World.Stats().Paused
	session, err := game.NewSession(m.document.Document(), m.settings.Physics.WorldOptions())
	if err != nil {
		m.message = err.Error()
		return
	}
	session.World.SetPaused(paused)
	m.view, m.zoom = 0, 1
	m.play = session
	m.runtimeSelection = ""
	m.pendingSpawn = ""
	m.hitAnchor = nil
	m.lastTick = time.Now()
	m.renderPreview()
	m.message = "Play reset; pending commands and runtime selection cleared"
}
func (m projectEditorModel) toolsView() string {
	var out strings.Builder
	out.WriteString("\nPlay tools (paused)\n")
	first := max(0, m.toolField-1)
	for i := first; i < min(len(toolLabels), first+3); i++ {
		prefix := "  "
		if i == m.toolField {
			prefix = "> "
		}
		fmt.Fprintf(&out, "%s%s: %s\n", prefix, toolLabels[i], m.toolValues[i])
	}
	options, _ := parseToolValues(m.toolValues)
	magnitude := math.Sqrt(options.impulse.X*options.impulse.X + options.impulse.Y*options.impulse.Y + options.impulse.Z*options.impulse.Z)
	fmt.Fprintf(&out, "Impulse (%g,%g,%g), |J|=%g at %s\n", options.impulse.X, options.impulse.Y, options.impulse.Z, magnitude, m.toolValues[3])
	out.WriteString("Arrows choose | Enter edit | I impulse | G spawn preview\nD delete runtime | R reset | Esc close (stays paused)\n")
	return out.String()
}
func (m *projectEditorModel) reconcileRuntime() {
	if m.play == nil {
		return
	}
	if m.pendingSpawn != "" {
		if _, ok := m.play.Find(m.pendingSpawn); ok {
			m.runtimeSelection = m.pendingSpawn
			m.pendingSpawn = ""
			m.hitAnchor = nil
		} else if m.play.PendingCommands() == 0 {
			m.pendingSpawn = ""
		}
	}
	if _, ok := m.play.Find(m.runtimeSelection); !ok {
		m.runtimeSelection = ""
		m.hitAnchor = nil
	}
}
func (m projectEditorModel) hierarchy() []project.TreeEntry {
	if m.play != nil {
		return m.play.Hierarchy()
	}
	return m.document.Document().Hierarchy()
}
func (m projectEditorModel) selectionIndex() int {
	if m.play == nil {
		return m.selected
	}
	for i, e := range m.play.Hierarchy() {
		if e.ID == m.runtimeSelection {
			return i
		}
	}
	return -1
}
func (m *projectEditorModel) clearSelection() {
	if m.play != nil {
		m.runtimeSelection = ""
		m.hitAnchor = nil
	} else {
		m.selected = -1
	}
}
func (m *projectEditorModel) inspectorRows() []string {
	e, ok := m.selectedEntity()
	if !ok {
		return nil
	}
	if m.play == nil {
		var rows []string
		for i := range project.Properties {
			p := m.inspectorProperty(i, e)
			rows = append(rows, p.Label+": "+p.Get(e)+" "+p.Units)
		}
		return rows
	}
	id, ok := m.play.Find(e.ID)
	if !ok {
		return nil
	}
	inspection, err := m.play.Inspect(id)
	if err != nil {
		return []string{err.Error()}
	}
	p := inspection.Entity.Position
	r := inspection.Entity.Rotation
	rows := []string{"Name: " + e.Name, "Body: " + e.Body, fmt.Sprintf("Runtime spawn: %t", inspection.Spawned), fmt.Sprintf("Position: %.2f %.2f %.2f", p.X, p.Y, p.Z), fmt.Sprintf("Rotation: %.2f %.2f %.2f", r.X, r.Y, r.Z)}
	if b := inspection.Body; b != nil {
		rows = append(rows, fmt.Sprintf("Mass: %.2f kg", b.Mass), fmt.Sprintf("Velocity: %.2f %.2f %.2f", b.Velocity.X, b.Velocity.Y, b.Velocity.Z), fmt.Sprintf("Angular: %.2f %.2f %.2f", b.AngularVelocity.X, b.AngularVelocity.Y, b.AngularVelocity.Z), fmt.Sprintf("Sleeping: %t", b.Sleeping), fmt.Sprintf("Sensor: %t", b.Collider.Sensor), fmt.Sprintf("Friction: %g", b.Collider.Friction), fmt.Sprintf("Restitution: %g", b.Collider.Restitution))
	}
	return rows
}
func (m *projectEditorModel) inspectorSlice(visible int) []string {
	rows := m.inspectorRows()
	if len(rows) == 0 {
		return []string{"(no selection)"}
	}
	selected := min(m.property, len(rows)-1)
	first := max(0, selected-visible+1)
	var out []string
	for i := first; i < min(len(rows), first+visible); i++ {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		out = append(out, prefix+rows[i])
	}
	return out
}
