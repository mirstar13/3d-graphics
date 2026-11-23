package physics

import (
	"fmt"
	"math"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// queryTolerance admits nanometre-scale rounding at touching boundaries.
const queryTolerance = 1e-9

// QueryFilter selects collider layers and optionally sensors. Zero mask matches nothing.
// Collision response masks do not affect queries. Disabled bodies are always omitted.
type QueryFilter struct {
	LayerMask      uint32
	IncludeSensors bool
}

// QueryHit identifies geometry in an authoritative world pose. Child is -1 for
// ordinary shapes. Distance is metres along a ray, zero for overlap results.
type QueryHit struct {
	Body          BodyID
	Child         int
	Point, Normal engine.Point
	Distance      float64
}

// Ray is a finite world-space segment. Direction need not be normalized.
type Ray struct {
	Origin, Direction engine.Point
	MaxDistance       float64
}

func (r Ray) normalized() (Ray, error) {
	size := math.Hypot(math.Hypot(r.Direction.X, r.Direction.Y), r.Direction.Z)
	if !finitePoint(r.Origin) || !finitePoint(r.Direction) || !finite(size) || size < 1e-12 || !finite(r.MaxDistance) || r.MaxDistance < 0 {
		return Ray{}, fmt.Errorf("ray requires finite origin, nonzero direction and nonnegative distance")
	}
	r.Direction = mul(r.Direction, 1/size)
	if !finitePoint(add(r.Origin, mul(r.Direction, r.MaxDistance))) {
		return Ray{}, fmt.Errorf("ray endpoint overflow")
	}
	return r, nil
}
func queryIncludes(b *body, f QueryFilter) bool {
	return b != nil && b.active && !b.state.Disabled && b.state.Collider.Layer&f.LayerMask != 0 && (!b.state.Collider.Sensor || f.IncludeSensors)
}

// Raycast returns the closest hit, using stable slot/child order for exact ties.
// It does not mutate bodies, contacts, clocks, or sleeping state.
func (w *World) Raycast(ray Ray, filter QueryFilter) (QueryHit, bool, error) {
	r, err := ray.normalized()
	if err != nil {
		return QueryHit{}, false, err
	}
	var best QueryHit
	found := false
	for _, slot := range w.bodies {
		b := slot.body
		if !queryIncludes(b, filter) {
			continue
		}
		hit, ok := rayBody(r, b)
		if ok && (!found || hit.Distance < best.Distance) {
			best = hit
			found = true
		}
	}
	return best, found, nil
}

// RaycastShape tests an immutable shape in a supplied pose, for visual-only proxies.
// No world body is created. Hit.Body is zero; normals and points are world-space.
func RaycastShape(ray Ray, shape Shape, position engine.Point, rotation engine.Quaternion) (QueryHit, bool, error) {
	r, err := ray.normalized()
	if err != nil {
		return QueryHit{}, false, err
	}
	if !finitePoint(position) || !validRotation(rotation) {
		return QueryHit{}, false, fmt.Errorf("invalid query pose")
	}
	desc := NewBodyDesc(Static, shape)
	desc.Position = position
	desc.Rotation = rotation.Normalize()
	if err = desc.Collider.validate(Static); err != nil {
		return QueryHit{}, false, err
	}
	hit, ok := rayBody(r, &body{state: BodyState{BodyDesc: desc}})
	return hit, ok, nil
}
func rayBody(r Ray, b *body) (QueryHit, bool) {
	if b.state.Collider.Shape.kind == CompoundShape {
		var best QueryHit
		found := false
		for i, child := range b.state.Collider.Shape.compound.children {
			hit, ok := rayBody(r, compoundBody(b, child))
			if ok && (!found || hit.Distance < best.Distance) {
				hit.Child = i
				best = hit
				found = true
			}
		}
		return best, found
	}
	local := Ray{Origin: inverseRotate(b.state.Rotation, sub(r.Origin, b.state.Position)), Direction: inverseRotate(b.state.Rotation, r.Direction), MaxDistance: r.MaxDistance + queryTolerance}
	if !finitePoint(local.Origin) || !finitePoint(local.Direction) {
		return QueryHit{}, false
	}
	distance, normal, ok := rayLocal(local, b.state.Collider.Shape)
	if !ok || !finite(distance) || !finitePoint(normal) {
		return QueryHit{}, false
	}
	distance = math.Min(distance, r.MaxDistance)
	return QueryHit{Body: b.state.ID, Child: -1, Point: add(r.Origin, mul(r.Direction, distance)), Normal: b.state.Rotation.RotateVector(normal), Distance: distance}, true
}

// OverlapSphere returns one hit per overlapping body/compound child in stable
// slot/child order. Touching counts; normals point outward from target geometry.
func (w *World) OverlapSphere(center engine.Point, radius float64, filter QueryFilter) ([]QueryHit, error) {
	if !finitePoint(center) || !finite(radius) || radius <= 0 {
		return nil, fmt.Errorf("overlap requires finite centre and positive radius")
	}
	extent := vec{X: radius, Y: radius, Z: radius}
	if !finitePoint(add(center, extent)) || !finitePoint(sub(center, extent)) {
		return nil, fmt.Errorf("overlap bounds overflow")
	}
	probe := &body{state: BodyState{BodyDesc: NewBodyDesc(Static, Sphere(radius))}}
	probe.state.Position = center
	var hits []QueryHit
	var test func(*body, int)
	test = func(target *body, child int) {
		if target.state.Collider.Shape.kind == CompoundShape {
			for i, c := range target.state.Collider.Shape.compound.children {
				test(compoundBody(target, c), i)
			}
			return
		}
		contacts := collide(probe, target)
		var chosen *Contact
		for i := range contacts {
			if contacts[i].Separation <= queryTolerance && (chosen == nil || contacts[i].Separation < chosen.Separation) {
				chosen = &contacts[i]
			}
		}
		if chosen != nil {
			hits = append(hits, QueryHit{Body: target.state.ID, Child: child, Point: chosen.PointB, Normal: mul(chosen.Normal, -1)})
		}
	}
	for _, slot := range w.bodies {
		if queryIncludes(slot.body, filter) {
			test(slot.body, -1)
		}
	}
	return hits, nil
}
