package engine

import (
	"math"
)

// OBB represents an Oriented Bounding Box
type OBB struct {
	Center      Point    // Center of the box
	Axes        [3]Point // Local axes (normalized)
	HalfExtents Point    // Half-widths along each axis
}

// NewOBB creates a new OBB
func NewOBB(center Point, axes [3]Point, halfExtents Point) *OBB {
	// Normalize axes
	for i := 0; i < 3; i++ {
		x, y, z := NormalizeVector(axes[i].X, axes[i].Y, axes[i].Z)
		axes[i] = Point{X: x, Y: y, Z: z}
	}

	return &OBB{
		Center:      center,
		Axes:        axes,
		HalfExtents: halfExtents,
	}
}

// NewOBBFromAABB creates an OBB from an AABB (axis-aligned)
func NewOBBFromAABB(aabb *AABB) *OBB {
	center := aabb.GetCenter()
	size := aabb.GetSize()

	return &OBB{
		Center: center,
		Axes: [3]Point{
			{X: 1, Y: 0, Z: 0}, // X axis
			{X: 0, Y: 1, Z: 0}, // Y axis
			{X: 0, Y: 0, Z: 1}, // Z axis
		},
		HalfExtents: Point{
			X: size.X / 2,
			Y: size.Y / 2,
			Z: size.Z / 2,
		},
	}
}

// NewOBBFromTransformedAABB creates an OBB from an AABB with a transform
func NewOBBFromTransformedAABB(aabb *AABB, transform *Transform) *OBB {
	// Get AABB center and size
	localCenter := aabb.GetCenter()
	size := aabb.GetSize()

	// Transform center to world space
	worldCenter := transform.TransformPoint(localCenter)

	// Transform axes (just the rotation, no translation)
	axes := [3]Point{
		transform.TransformDirection(Point{X: 1, Y: 0, Z: 0}),
		transform.TransformDirection(Point{X: 0, Y: 1, Z: 0}),
		transform.TransformDirection(Point{X: 0, Y: 0, Z: 1}),
	}

	// Apply scale to half extents
	halfExtents := Point{
		X: (size.X / 2) * transform.Scale.X,
		Y: (size.Y / 2) * transform.Scale.Y,
		Z: (size.Z / 2) * transform.Scale.Z,
	}

	return &OBB{
		Center:      worldCenter,
		Axes:        axes,
		HalfExtents: halfExtents,
	}
}

func (obb *OBB) Contains(p Point) bool {
	// Transform point to OBB's local space
	d := Point{
		X: p.X - obb.Center.X,
		Y: p.Y - obb.Center.Y,
		Z: p.Z - obb.Center.Z,
	}

	// Check if point is within bounds on each axis
	for i := 0; i < 3; i++ {
		// Project distance onto axis
		dist := DotProduct(d.X, d.Y, d.Z, obb.Axes[i].X, obb.Axes[i].Y, obb.Axes[i].Z)

		halfExtent := obb.getHalfExtent(i)
		if math.Abs(dist) > halfExtent {
			return false
		}
	}

	return true
}

func (obb *OBB) Intersects(other BoundingVolume) bool {
	switch v := other.(type) {
	case *OBB:
		return obb.IntersectsOBB(v)
	case *AABB:
		// Convert AABB to OBB for intersection test
		otherOBB := NewOBBFromAABB(v)
		return obb.IntersectsOBB(otherOBB)
	case *BoundingSphere:
		return obb.IntersectsSphere(v)
	}
	return false
}

func (obb *OBB) GetCenter() Point {
	return obb.Center
}

func (obb *OBB) GetRadius() float64 {
	// Return radius of bounding sphere
	return math.Sqrt(
		obb.HalfExtents.X*obb.HalfExtents.X +
			obb.HalfExtents.Y*obb.HalfExtents.Y +
			obb.HalfExtents.Z*obb.HalfExtents.Z,
	)
}

func (obb *OBB) IntersectsRay(ray Ray) (bool, float64) {
	// Transform ray to OBB's local space
	// This is equivalent to testing against an AABB in local space

	// Vector from ray origin to OBB center
	p := Point{
		X: obb.Center.X - ray.Origin.X,
		Y: obb.Center.Y - ray.Origin.Y,
		Z: obb.Center.Z - ray.Origin.Z,
	}

	// For each axis of the OBB
	tMin := -math.MaxFloat64
	tMax := math.MaxFloat64

	for i := 0; i < 3; i++ {
		axis := obb.Axes[i]
		halfExtent := obb.getHalfExtent(i)

		// Project onto axis
		e := DotProduct(axis.X, axis.Y, axis.Z, p.X, p.Y, p.Z)
		f := DotProduct(axis.X, axis.Y, axis.Z, ray.Direction.X, ray.Direction.Y, ray.Direction.Z)

		if math.Abs(f) > 1e-6 {
			t1 := (e + halfExtent) / f
			t2 := (e - halfExtent) / f

			if t1 > t2 {
				t1, t2 = t2, t1
			}

			if t1 > tMin {
				tMin = t1
			}
			if t2 < tMax {
				tMax = t2
			}

			if tMin > tMax || tMax < 0 {
				return false, 0
			}
		} else {
			// Ray is parallel to slab
			if -e-halfExtent > 0 || -e+halfExtent < 0 {
				return false, 0
			}
		}
	}

	if tMin > 0 {
		return true, tMin
	}
	return true, tMax
}

// IntersectsOBB tests if two OBBs intersect using SAT
func (obb *OBB) IntersectsOBB(other *OBB) bool {
	// Separating Axis Theorem (SAT):
	// Two OBBs don't intersect if there exists a separating axis

	// Test 15 potential separating axes:
	// - 3 axes of first OBB
	// - 3 axes of second OBB
	// - 9 cross products of axes

	// Vector between centers
	t := Point{
		X: other.Center.X - obb.Center.X,
		Y: other.Center.Y - obb.Center.Y,
		Z: other.Center.Z - obb.Center.Z,
	}

	// Test axes of first OBB
	for i := 0; i < 3; i++ {
		if !obb.testAxis(other, t, obb.Axes[i]) {
			return false
		}
	}

	// Test axes of second OBB
	for i := 0; i < 3; i++ {
		if !obb.testAxis(other, t, other.Axes[i]) {
			return false
		}
	}

	// Test cross product axes
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			axisX, axisY, axisZ := crossProduct(
				obb.Axes[i].X, obb.Axes[i].Y, obb.Axes[i].Z,
				other.Axes[j].X, other.Axes[j].Y, other.Axes[j].Z,
			)

			// Skip if cross product is near zero (parallel axes)
			length := math.Sqrt(axisX*axisX + axisY*axisY + axisZ*axisZ)
			if length < 1e-6 {
				continue
			}

			// Normalize
			axisX /= length
			axisY /= length
			axisZ /= length

			axis := Point{X: axisX, Y: axisY, Z: axisZ}
			if !obb.testAxis(other, t, axis) {
				return false
			}
		}
	}

	// No separating axis found - OBBs intersect
	return true
}

// testAxis tests if axis is a separating axis
func (obb *OBB) testAxis(other *OBB, t Point, axis Point) bool {
	// Project both OBBs onto the axis
	ra := obb.projectOntoAxis(axis)
	rb := other.projectOntoAxis(axis)

	// Project center distance onto axis
	distance := math.Abs(DotProduct(t.X, t.Y, t.Z, axis.X, axis.Y, axis.Z))

	// Check if projections overlap
	return distance <= ra+rb
}

// projectOntoAxis projects the OBB onto an axis and returns the radius
func (obb *OBB) projectOntoAxis(axis Point) float64 {
	return math.Abs(DotProduct(obb.Axes[0].X, obb.Axes[0].Y, obb.Axes[0].Z, axis.X, axis.Y, axis.Z))*obb.HalfExtents.X +
		math.Abs(DotProduct(obb.Axes[1].X, obb.Axes[1].Y, obb.Axes[1].Z, axis.X, axis.Y, axis.Z))*obb.HalfExtents.Y +
		math.Abs(DotProduct(obb.Axes[2].X, obb.Axes[2].Y, obb.Axes[2].Z, axis.X, axis.Y, axis.Z))*obb.HalfExtents.Z
}

// IntersectsSphere tests if OBB intersects with a sphere
func (obb *OBB) IntersectsSphere(sphere *BoundingSphere) bool {
	// Find closest point on OBB to sphere center
	closest := obb.ClosestPoint(sphere.Center)

	// Check distance
	dx := closest.X - sphere.Center.X
	dy := closest.Y - sphere.Center.Y
	dz := closest.Z - sphere.Center.Z
	distSq := dx*dx + dy*dy + dz*dz

	return distSq <= sphere.Radius*sphere.Radius
}

// ClosestPoint finds the closest point on the OBB to a given point
func (obb *OBB) ClosestPoint(p Point) Point {
	// Vector from center to point
	d := Point{
		X: p.X - obb.Center.X,
		Y: p.Y - obb.Center.Y,
		Z: p.Z - obb.Center.Z,
	}

	// Start at center
	result := obb.Center

	// For each axis, clamp the distance
	for i := 0; i < 3; i++ {
		// Project d onto axis
		dist := DotProduct(d.X, d.Y, d.Z, obb.Axes[i].X, obb.Axes[i].Y, obb.Axes[i].Z)

		// Clamp to box extents
		halfExtent := obb.getHalfExtent(i)
		if dist > halfExtent {
			dist = halfExtent
		}
		if dist < -halfExtent {
			dist = -halfExtent
		}

		// Add clamped distance along axis
		result.X += dist * obb.Axes[i].X
		result.Y += dist * obb.Axes[i].Y
		result.Z += dist * obb.Axes[i].Z
	}

	return result
}

// GetCorners returns the 8 corners of the OBB
func (obb *OBB) GetCorners() [8]Point {
	var corners [8]Point

	// Generate all 8 combinations of +/- half extents
	for i := 0; i < 8; i++ {
		point := obb.Center

		// X component
		if i&1 != 0 {
			point.X += obb.Axes[0].X * obb.HalfExtents.X
			point.Y += obb.Axes[0].Y * obb.HalfExtents.X
			point.Z += obb.Axes[0].Z * obb.HalfExtents.X
		} else {
			point.X -= obb.Axes[0].X * obb.HalfExtents.X
			point.Y -= obb.Axes[0].Y * obb.HalfExtents.X
			point.Z -= obb.Axes[0].Z * obb.HalfExtents.X
		}

		// Y component
		if i&2 != 0 {
			point.X += obb.Axes[1].X * obb.HalfExtents.Y
			point.Y += obb.Axes[1].Y * obb.HalfExtents.Y
			point.Z += obb.Axes[1].Z * obb.HalfExtents.Y
		} else {
			point.X -= obb.Axes[1].X * obb.HalfExtents.Y
			point.Y -= obb.Axes[1].Y * obb.HalfExtents.Y
			point.Z -= obb.Axes[1].Z * obb.HalfExtents.Y
		}

		// Z component
		if i&4 != 0 {
			point.X += obb.Axes[2].X * obb.HalfExtents.Z
			point.Y += obb.Axes[2].Y * obb.HalfExtents.Z
			point.Z += obb.Axes[2].Z * obb.HalfExtents.Z
		} else {
			point.X -= obb.Axes[2].X * obb.HalfExtents.Z
			point.Y -= obb.Axes[2].Y * obb.HalfExtents.Z
			point.Z -= obb.Axes[2].Z * obb.HalfExtents.Z
		}

		corners[i] = point
	}

	return corners
}

// ToAABB converts the OBB to an axis-aligned bounding box
func (obb *OBB) ToAABB() *AABB {
	corners := obb.GetCorners()
	return NewAABBFromPoints(corners[:])
}

// getHalfExtent returns the half extent along axis i
func (obb *OBB) getHalfExtent(i int) float64 {
	switch i {
	case 0:
		return obb.HalfExtents.X
	case 1:
		return obb.HalfExtents.Y
	case 2:
		return obb.HalfExtents.Z
	}
	return 0
}

// ComputeMeshOBB returns an axis-aligned OBB enclosing the mesh and its local offset.
func ComputeMeshOBB(mesh *Mesh) *OBB {
	aabbVol := ComputeMeshBounds(mesh)
	if aabb, ok := aabbVol.(*AABB); ok {
		return NewOBBFromAABB(aabb)
	}
	return NewOBBFromAABB(NewAABB(Point{}, Point{}))
}

// ComputeOptimalOBB returns an enclosing axis-aligned OBB for compatibility.
// Despite the historical name, it does not perform principal-component fitting.
func ComputeOptimalOBB(points []Point) *OBB {
	return NewOBBFromAABB(NewAABBFromPoints(points))
}
