package app

import (
	"fmt"
	"strings"

	"github.com/mirstar13/go-3d-graphics/internal/input"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

func (m *projectEditorModel) updateBindings(key string) {
	actions := input.Definitions()
	switch key {
	case "up":
		m.bindingSelected = (m.bindingSelected + len(actions) - 1) % len(actions)
	case "down", "tab":
		m.bindingSelected = (m.bindingSelected + 1) % len(actions)
	case "enter":
		m.modal = "binding"
		m.draft = strings.Join(input.Keys(m.project.Manifest.Bindings, actions[m.bindingSelected].ID), ",")
	case "d":
		if err := m.project.SaveBindings(nil); err != nil {
			m.message = err.Error()
		} else {
			m.message = "Default bindings saved"
		}
	}
}
func (m *projectEditorModel) saveBinding() error {
	bindings := input.Clone(m.project.Manifest.Bindings)
	if bindings == nil {
		bindings = input.Bindings{}
	}
	keys := strings.Split(strings.ToLower(m.draft), ",")
	for i := range keys {
		keys[i] = strings.TrimSpace(keys[i])
	}
	bindings[input.Definitions()[m.bindingSelected].ID] = keys
	if err := m.project.SaveBindings(bindings); err != nil {
		return err
	}
	m.modal = "bindings"
	m.message = "Project bindings saved"
	return nil
}
func (m projectEditorModel) bindingsView() string {
	actions := input.Definitions()
	var out strings.Builder
	out.WriteString("\nProject key bindings\n")
	first := max(0, m.bindingSelected-1)
	for i := first; i < min(len(actions), first+3); i++ {
		prefix := "  "
		if i == m.bindingSelected {
			prefix = "> "
		}
		d := actions[i]
		fmt.Fprintf(&out, "%s%s: %s\n", prefix, d.Label, strings.Join(input.Keys(m.project.Manifest.Bindings, d.ID), ","))
	}
	out.WriteString("Arrows choose | Enter edit | D defaults | Esc close\n")
	return out.String()
}
func (m *projectEditorModel) cycleSelection(delta int) {
	count := len(m.document.Document().Entities)
	if count == 0 {
		m.selected = -1
		return
	}
	if m.selected < 0 && delta < 0 {
		m.selected = 0
	}
	m.selected = (m.selected + count + delta) % count
	m.renderPreview()
}
func (m *projectEditorModel) moveReticle(dx, dy int) {
	if len(m.frame) == 0 {
		return
	}
	h, w := len(m.frame), len(m.frame[0])
	x, y := m.reticleCell(w, h)
	x = max(0, min(w-1, x+dx))
	y = max(0, min(h-1, y+dy))
	m.reticleU = (float64(x) + .5) / float64(w)
	m.reticleV = (float64(y) + .5) / float64(h)
	m.renderPreview()
}
func (m projectEditorModel) reticleCell(w, h int) (int, int) {
	return max(0, min(w-1, int(m.reticleU*float64(w)))), max(0, min(h-1, int(m.reticleV*float64(h))))
}
func (m *projectEditorModel) pickReticle() {
	if len(m.frame) == 0 {
		return
	}
	session := m.preview
	if m.play != nil {
		session = m.play
	}
	h, w := len(m.frame), len(m.frame[0])
	x, y := m.reticleCell(w, h)
	origin, direction, distance, err := session.Scene.Camera.CellRay(x, y, w, h)
	if err != nil {
		m.message = err.Error()
		return
	}
	hit, ok, err := session.Pick(physics.Ray{Origin: origin, Direction: direction, MaxDistance: distance})
	if err != nil {
		m.message = err.Error()
		return
	}
	if !ok {
		m.selected = -1
		m.message = "No hit; selection cleared"
	} else {
		m.selectID(hit.FileID)
		kind := "collider"
		if hit.VisualProxy {
			kind = "visual proxy"
		}
		m.message = fmt.Sprintf("Picked %s (%s) at %.2fm normal %.2f,%.2f,%.2f", hit.FileID, kind, hit.Distance, hit.Normal.X, hit.Normal.Y, hit.Normal.Z)
	}
	m.renderPreview()
}
