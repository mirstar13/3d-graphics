package physics

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
)

// DebugLines returns collider outlines and contact normals from authoritative poses.
// Consumers draw them after rendering; these shapes never participate in simulation.
func (w *World) DebugLines() []engine.DebugLine {
	var lines []engine.DebugLine
	line := func(a, b vec, c engine.Color) { lines = append(lines, engine.DebugLine{Start: a, End: b, Color: c}) }
	var bodies []*body
	for _, slot := range w.bodies {
		b := slot.body
		if b == nil || !b.active || b.state.Disabled {
			continue
		}
		if b.state.Collider.Shape.kind == CompoundShape {
			for _, child := range b.state.Collider.Shape.compound.children {
				bodies = append(bodies, compoundBody(b, child))
			}
		} else {
			bodies = append(bodies, b)
		}
	}
	for _, b := range bodies {
		if b == nil || !b.active || b.state.Disabled {
			continue
		}
		s := b.state.Collider.Shape
		color := engine.Color{R: 60, G: 220, B: 220}
		if b.state.Type == Dynamic {
			color = engine.Color{R: 100, G: 255, B: 100}
		}
		switch s.kind {
		case SphereShape:
			for plane := 0; plane < 3; plane++ {
				u, v := axis((plane+1)%3), axis((plane+2)%3)
				for i := 0; i < 24; i++ {
					a, c := float64(i)*math.Pi/12, float64(i+1)*math.Pi/12
					point := func(t float64) vec {
						return add(b.state.Position, mul(add(mul(u, math.Cos(t)), mul(v, math.Sin(t))), s.radius))
					}
					line(point(a), point(c), color)
				}
			}
		case BoxShape:
			vertices := boxVertices(b)
			for i := 0; i < 8; i++ {
				for j := 0; j < 3; j++ {
					k := i ^ (1 << j)
					if i < k {
						line(vertices[i], vertices[k], color)
					}
				}
			}
		case PlaneShape:
			n, offset := planeWorld(b)
			u := unit(cross(n, vec{X: 1}))
			if math.Abs(n.X) > .9 {
				u = unit(cross(n, vec{Y: 1}))
			}
			v := cross(n, u)
			center := add(b.state.Position, mul(n, offset-dot(n, b.state.Position)))
			for i := 0; i < 4; i++ {
				a, c := float64(i)*math.Pi/2+math.Pi/4, float64(i+1)*math.Pi/2+math.Pi/4
				line(add(center, mul(add(mul(u, math.Cos(a)), mul(v, math.Sin(a))), 16)), add(center, mul(add(mul(u, math.Cos(c)), mul(v, math.Sin(c))), 16)), color)
			}
		case MeshShape:
			for i := range s.mesh.triangles {
				tri := s.mesh.triangle(i)
				for j := 0; j < 3; j++ {
					line(add(b.state.Position, b.state.Rotation.RotateVector(tri[j])), add(b.state.Position, b.state.Rotation.RotateVector(tri[(j+1)%3])), color)
				}
			}
		}
	}
	for _, c := range w.contacts {
		p := contactPoint(c)
		line(p, add(p, c.Normal), engine.ColorRed)
		for i := 0; i < 3; i++ {
			d := mul(axis(i), .12)
			line(sub(p, d), add(p, d), engine.ColorWhite)
		}
	}
	for _, slot := range w.joints {
		j := slot.joint
		if j == nil || !j.active {
			continue
		}
		a, b, e := w.jointBodies(j.state.JointDesc)
		if e != nil || a.state.Disabled || b.state.Disabled {
			continue
		}
		pa, pb := jointPoint(a, j.state.AnchorA), jointPoint(b, j.state.AnchorB)
		line(pa, pb, engine.ColorWhite)
		if j.state.Kind == HingeJoint {
			line(pa, add(pa, a.state.Rotation.RotateVector(j.state.AxisA)), engine.ColorWhite)
		}
	}
	return lines
}
