package physics

import (
	"fmt"
	"math"
)

// ShapeKind identifies a collision shape, independent of render geometry.
type ShapeKind uint8

const (
	SphereShape ShapeKind = iota
	BoxShape
	PlaneShape
	MeshShape
)

// Shape is immutable once constructed; body creation validates its dimensions.
// Mesh geometry is copied and validated by NewTriangleMesh.
type Shape struct {
	kind   ShapeKind
	radius float64
	half   vec
	normal vec
	offset float64
	mesh   *TriangleMesh
}

// Sphere constructs a sphere centred at the body origin.
func Sphere(radius float64) Shape { return Shape{kind: SphereShape, radius: radius} }

// Box constructs an oriented box from positive local half extents.
func Box(halfExtents vec) Shape { return Shape{kind: BoxShape, half: halfExtents} }

// Plane defines permitted space dot(normal,p)>=offset in the body's local frame.
func Plane(normal vec, offset float64) Shape {
	return Shape{kind: PlaneShape, normal: normal, offset: offset}
}

// Mesh references immutable triangle geometry for a static body.
func Mesh(mesh *TriangleMesh) Shape { return Shape{kind: MeshShape, mesh: mesh} }

// Kind identifies the collision geometry.
func (s Shape) Kind() ShapeKind { return s.kind }

// Radius returns the sphere radius, or zero for other shapes.
func (s Shape) Radius() float64 { return s.radius }

// HalfExtents returns box dimensions measured from its centre.
func (s Shape) HalfExtents() vec { return s.half }

// Collider describes contact material, filtering, and sensor behavior. Sensors
// report contacts but do not exert impulses. Both layer/mask directions must match.
type Collider struct {
	Shape                 Shape
	Friction, Restitution float64
	Layer, Mask           uint32
	Sensor                bool
}

func (c Collider) validate(kind BodyType) error {
	s := c.Shape
	if !finite(c.Friction) || c.Friction < 0 || !finite(c.Restitution) || c.Restitution < 0 || c.Restitution > 1 {
		return fmt.Errorf("invalid collider material")
	}
	switch s.kind {
	case SphereShape:
		if !finite(s.radius) || s.radius <= 0 {
			return fmt.Errorf("sphere radius must be positive and finite")
		}
	case BoxShape:
		if !finitePoint(s.half) || s.half.X <= 0 || s.half.Y <= 0 || s.half.Z <= 0 {
			return fmt.Errorf("box half extents must be positive and finite")
		}
	case PlaneShape:
		if kind != Static || !finitePoint(s.normal) || length(s.normal) < 1e-10 || !finite(s.offset) {
			return fmt.Errorf("plane requires a static body and a finite nonzero normal")
		}
	case MeshShape:
		if kind != Static || s.mesh == nil {
			return fmt.Errorf("triangle mesh requires a static body and validated geometry")
		}
	default:
		return fmt.Errorf("unsupported collider shape")
	}
	return nil
}
func inertiaFor(s Shape, mass float64) vec {
	if s.kind == SphereShape {
		v := .4 * mass * s.radius * s.radius
		return vec{X: v, Y: v, Z: v}
	}
	h := s.half
	return mul(vec{X: h.Y*h.Y + h.Z*h.Z, Y: h.X*h.X + h.Z*h.Z, Z: h.X*h.X + h.Y*h.Y}, mass/3)
}

type bounds struct{ min, max vec }

func emptyBounds() bounds {
	return bounds{min: vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}, max: vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}}
}
func (b *bounds) include(p vec) {
	b.min = vec{X: math.Min(b.min.X, p.X), Y: math.Min(b.min.Y, p.Y), Z: math.Min(b.min.Z, p.Z)}
	b.max = vec{X: math.Max(b.max.X, p.X), Y: math.Max(b.max.Y, p.Y), Z: math.Max(b.max.Z, p.Z)}
}
func (b bounds) overlaps(c bounds) bool {
	return b.min.X <= c.max.X && b.max.X >= c.min.X && b.min.Y <= c.max.Y && b.max.Y >= c.min.Y && b.min.Z <= c.max.Z && b.max.Z >= c.min.Z
}
func (b bounds) expanded(d float64) bounds {
	v := vec{X: d, Y: d, Z: d}
	return bounds{min: sub(b.min, v), max: add(b.max, v)}
}
func boxAxes(b *body) [3]vec {
	return [3]vec{b.state.Rotation.RotateVector(axis(0)), b.state.Rotation.RotateVector(axis(1)), b.state.Rotation.RotateVector(axis(2))}
}
func boxVertices(b *body) [8]vec {
	axes := boxAxes(b)
	h := b.state.Collider.Shape.half
	var result [8]vec
	for i := range result {
		p := b.state.Position
		for j := 0; j < 3; j++ {
			sign := -1.0
			if i&(1<<j) != 0 {
				sign = 1
			}
			p = add(p, mul(axes[j], sign*component(h, j)))
		}
		result[i] = p
	}
	return result
}
func bodyBounds(b *body) bounds {
	s := b.state.Collider.Shape
	p := b.state.Position
	switch s.kind {
	case SphereShape:
		r := vec{X: s.radius, Y: s.radius, Z: s.radius}
		return bounds{min: sub(p, r), max: add(p, r)}
	case BoxShape:
		result := emptyBounds()
		for _, v := range boxVertices(b) {
			result.include(v)
		}
		return result
	case MeshShape:
		result := emptyBounds()
		for _, v := range s.mesh.vertices {
			result.include(add(p, b.state.Rotation.RotateVector(v)))
		}
		return result
	default:
		return bounds{min: vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}, max: vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}}
	}
}
