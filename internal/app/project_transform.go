package app

import (
	"math"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
)

type viewportDrag struct {
	id                   string
	handle               game.EntityID
	x, y                 int
	position, axis       engine.Point
	rotation             engine.Quaternion
	horizontal, vertical float64
	amount               float64
	rotate, snap         bool
}

func (m *projectEditorModel) transformSession() *game.Session {
	if m.play != nil {
		if !m.play.World.Stats().Paused {
			return nil
		}
		return m.play
	}
	return m.preview
}

func (m *projectEditorModel) beginTransform(x, y int) *viewportDrag {
	s := m.transformSession()
	entity, selected := m.selectedEntity()
	if s == nil || !selected || len(m.frame) == 0 {
		return nil
	}
	id, ok := s.Find(entity.ID)
	if !ok {
		return nil
	}
	pose, err := s.LocalPose(id)
	if err != nil {
		return nil
	}
	axis := [...]engine.Point{{X: 1}, {Y: 1}, {Z: 1}}[m.transformAxis]
	a, b := pose.Position, pose.Position
	b.X += axis.X
	b.Y += axis.Y
	b.Z += axis.Z
	if pose.Parent != nil {
		a, b = pose.Parent.TransformPoint(a), pose.Parent.TransformPoint(b)
	}
	h, w := len(m.frame), len(m.frame[0])
	ax, ay, az := s.Scene.Camera.ProjectPointFloat(a, h, w)
	bx, by, bz := s.Scene.Camera.ProjectPointFloat(b, h, w)
	dx, dy := bx-ax, by-ay
	length := dx*dx + dy*dy
	if az == 0 || bz == 0 || length < 1 {
		// End-on axes still respond to horizontal motion.
		dx, dy, length = 10, 0, 100
	}
	return &viewportDrag{id: entity.ID, handle: id, x: x, y: y, position: pose.Position, axis: axis,
		rotation: pose.Rotation, horizontal: dx / length, vertical: dy / length, rotate: m.transformTool == 2, snap: m.transformSnap}
}

func (d viewportDrag) pose() (engine.Point, engine.Quaternion) {
	p, q := d.position, d.rotation
	amount := d.amount
	if d.snap {
		step := .25
		if d.rotate {
			step = 15
		}
		amount = math.Round(amount/step) * step
	}
	if d.rotate {
		q = engine.QuaternionFromAxisAngle(d.axis, amount*math.Pi/180).Multiply(q)
	} else {
		p.X += d.axis.X * amount
		p.Y += d.axis.Y * amount
		p.Z += d.axis.Z * amount
	}
	return p, q
}

func (m *projectEditorModel) commitTransform(d viewportDrag) {
	m.transformDrag = nil
	p, q := d.pose()
	if p == d.position && q == d.rotation {
		m.renderPreview()
		return
	}
	if m.play != nil {
		if err := m.play.SetPausedPose(d.handle, p, q); err != nil {
			m.message = err.Error()
		}
		m.hitAnchor = nil
		m.renderPreview()
		return
	}
	x, y, z := q.ToEuler()
	rotation := engine.Point{X: x * 180 / math.Pi, Y: y * 180 / math.Pi, Z: z * 180 / math.Pi}
	if !d.rotate {
		entity, _ := m.selectedEntity()
		rotation = entity.Rotation
	}
	if err := m.document.SetPose(d.id, p, rotation); err != nil {
		m.message = err.Error()
	}
	m.refresh()
}

func (m *projectEditorModel) dragMouse(msg tea.MouseMsg) bool {
	if m.transformDrag == nil {
		return false
	}
	d := *m.transformDrag
	if msg.Action == tea.MouseActionMotion {
		d.amount = float64(msg.X-d.x)*d.horizontal + float64(msg.Y-d.y)*d.vertical
		if d.rotate {
			d.amount = 5 * float64(msg.X-d.x)
		}
		m.transformDrag = &d
		m.renderPreview()
	}
	if msg.Action == tea.MouseActionRelease {
		m.commitTransform(d)
	}
	return true
}

func (m *projectEditorModel) transformKey(key string) bool {
	if m.modal != "" || m.transformSession() == nil {
		return false
	}
	switch key {
	case "ui.transform", "alt+g":
		m.transformTool = (m.transformTool + 1) % 3
	case "ui.axis", "alt+j":
		m.transformAxis = (m.transformAxis + 1) % 3
	case "ui.snap", "alt+l":
		m.transformSnap = !m.transformSnap
	case "[", "]":
		if m.transformTool == 0 || m.focus != 2 {
			return false
		}
		if d := m.beginTransform(0, 0); d != nil {
			d.amount = .25
			if d.rotate {
				d.amount = 15
			}
			if key == "[" {
				d.amount = -d.amount
			}
			m.commitTransform(*d)
		}
	default:
		return false
	}
	m.transformDrag = nil
	m.renderPreview()
	return true
}

func (m *projectEditorModel) cancelTransform() {
	if m.transformDrag != nil {
		m.transformDrag = nil
		m.renderPreview()
	}
}
