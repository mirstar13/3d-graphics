package main

import "math"

// BoundingVolume interface for all bounding volume types
type BoundingVolume interface {
	Contains(p Point) bool
	Intersects(other BoundingVolume) bool
	IntersectsRay(ray Ray) (bool, float64)
	GetCenter() Point
	GetRadius() float64 // For sphere culling
}

// AABB represents an Axis-Aligned Bounding Box
type AABB struct {
	Min Point
	Max Point
}

// BoundingSphere represents a bounding sphere
type BoundingSphere struct {
	Center Point
	Radius float64
}

// NewAABB creates a new AABB
func NewAABB(min, max Point) *AABB {
	return &AABB{Min: min, Max: max}
}

// NewAABBFromPoints creates an AABB that encompasses all points
func NewAABBFromPoints(points []Point) *AABB {
	if len(points) == 0 {
		return &AABB{
			Min: Point{X: 0, Y: 0, Z: 0},
			Max: Point{X: 0, Y: 0, Z: 0},
		}
	}

	aabb := &AABB{
		Min: points[0],
		Max: points[0],
	}

	for _, p := range points[1:] {
		if p.X < aabb.Min.X {
			aabb.Min.X = p.X
		}
		if p.Y < aabb.Min.Y {
			aabb.Min.Y = p.Y
		}
		if p.Z < aabb.Min.Z {
			aabb.Min.Z = p.Z
		}

		if p.X > aabb.Max.X {
			aabb.Max.X = p.X
		}
		if p.Y > aabb.Max.Y {
			aabb.Max.Y = p.Y
		}
		if p.Z > aabb.Max.Z {
			aabb.Max.Z = p.Z
		}
	}

	return aabb
}

// NewBoundingSphere creates a new bounding sphere
func NewBoundingSphere(center Point, radius float64) *BoundingSphere {
	return &BoundingSphere{Center: center, Radius: radius}
}

// NewBoundingSphereFromPoints creates a bounding sphere from points (simple center + max distance)
func NewBoundingSphereFromPoints(points []Point) *BoundingSphere {
	if len(points) == 0 {
		return &BoundingSphere{Center: Point{}, Radius: 0}
	}

	// Calculate center
	center := Point{}
	for _, p := range points {
		center.X += p.X
		center.Y += p.Y
		center.Z += p.Z
	}
	center.X /= float64(len(points))
	center.Y /= float64(len(points))
	center.Z /= float64(len(points))

	// Find max distance from center
	maxDist := 0.0
	for _, p := range points {
		dx := p.X - center.X
		dy := p.Y - center.Y
		dz := p.Z - center.Z
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if dist > maxDist {
			maxDist = dist
		}
	}

	return &BoundingSphere{Center: center, Radius: maxDist}
}

// === AABB Methods ===

func (aabb *AABB) Contains(p Point) bool {
	return p.X >= aabb.Min.X && p.X <= aabb.Max.X &&
		p.Y >= aabb.Min.Y && p.Y <= aabb.Max.Y &&
		p.Z >= aabb.Min.Z && p.Z <= aabb.Max.Z
}

func (aabb *AABB) Intersects(other BoundingVolume) bool {
	switch v := other.(type) {
	case *AABB:
		return aabb.IntersectsAABB(v)
	case *BoundingSphere:
		return aabb.IntersectsSphere(v)
	}
	return false
}

func (aabb *AABB) IntersectsAABB(other *AABB) bool {
	return aabb.Min.X <= other.Max.X && aabb.Max.X >= other.Min.X &&
		aabb.Min.Y <= other.Max.Y && aabb.Max.Y >= other.Min.Y &&
		aabb.Min.Z <= other.Max.Z && aabb.Max.Z >= other.Min.Z
}

func (aabb *AABB) IntersectsSphere(sphere *BoundingSphere) bool {
	// Find closest point on AABB to sphere center
	closestX := clamp(sphere.Center.X, aabb.Min.X, aabb.Max.X)
	closestY := clamp(sphere.Center.Y, aabb.Min.Y, aabb.Max.Y)
	closestZ := clamp(sphere.Center.Z, aabb.Min.Z, aabb.Max.Z)

	// Check distance from closest point to sphere center
	dx := closestX - sphere.Center.X
	dy := closestY - sphere.Center.Y
	dz := closestZ - sphere.Center.Z

	distSq := dx*dx + dy*dy + dz*dz
	return distSq <= sphere.Radius*sphere.Radius
}

func (aabb *AABB) IntersectsRay(ray Ray) (bool, float64) {
	// Slab method for ray-AABB intersection
	tMin := (aabb.Min.X - ray.Origin.X) / ray.Direction.X
	tMax := (aabb.Max.X - ray.Origin.X) / ray.Direction.X

	if tMin > tMax {
		tMin, tMax = tMax, tMin
	}

	tyMin := (aabb.Min.Y - ray.Origin.Y) / ray.Direction.Y
	tyMax := (aabb.Max.Y - ray.Origin.Y) / ray.Direction.Y

	if tyMin > tyMax {
		tyMin, tyMax = tyMax, tyMin
	}

	if tMin > tyMax || tyMin > tMax {
		return false, 0
	}

	if tyMin > tMin {
		tMin = tyMin
	}
	if tyMax < tMax {
		tMax = tyMax
	}

	tzMin := (aabb.Min.Z - ray.Origin.Z) / ray.Direction.Z
	tzMax := (aabb.Max.Z - ray.Origin.Z) / ray.Direction.Z

	if tzMin > tzMax {
		tzMin, tzMax = tzMax, tzMin
	}

	if tMin > tzMax || tzMin > tMax {
		return false, 0
	}

	if tzMin > tMin {
		tMin = tzMin
	}

	// tMin is the distance to intersection
	return tMin >= 0, tMin
}

func (aabb *AABB) GetCenter() Point {
	return Point{
		X: (aabb.Min.X + aabb.Max.X) / 2,
		Y: (aabb.Min.Y + aabb.Max.Y) / 2,
		Z: (aabb.Min.Z + aabb.Max.Z) / 2,
	}
}

func (aabb *AABB) GetRadius() float64 {
	// Return radius of bounding sphere
	center := aabb.GetCenter()
	dx := aabb.Max.X - center.X
	dy := aabb.Max.Y - center.Y
	dz := aabb.Max.Z - center.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (aabb *AABB) GetSize() Point {
	return Point{
		X: aabb.Max.X - aabb.Min.X,
		Y: aabb.Max.Y - aabb.Min.Y,
		Z: aabb.Max.Z - aabb.Min.Z,
	}
}

func (aabb *AABB) Expand(amount float64) *AABB {
	return &AABB{
		Min: Point{
			X: aabb.Min.X - amount,
			Y: aabb.Min.Y - amount,
			Z: aabb.Min.Z - amount,
		},
		Max: Point{
			X: aabb.Max.X + amount,
			Y: aabb.Max.Y + amount,
			Z: aabb.Max.Z + amount,
		},
	}
}

func (aabb *AABB) Merge(other *AABB) *AABB {
	return &AABB{
		Min: Point{
			X: math.Min(aabb.Min.X, other.Min.X),
			Y: math.Min(aabb.Min.Y, other.Min.Y),
			Z: math.Min(aabb.Min.Z, other.Min.Z),
		},
		Max: Point{
			X: math.Max(aabb.Max.X, other.Max.X),
			Y: math.Max(aabb.Max.Y, other.Max.Y),
			Z: math.Max(aabb.Max.Z, other.Max.Z),
		},
	}
}

func (bs *BoundingSphere) Contains(p Point) bool {
	dx := p.X - bs.Center.X
	dy := p.Y - bs.Center.Y
	dz := p.Z - bs.Center.Z
	distSq := dx*dx + dy*dy + dz*dz
	return distSq <= bs.Radius*bs.Radius
}

func (bs *BoundingSphere) Intersects(other BoundingVolume) bool {
	switch v := other.(type) {
	case *BoundingSphere:
		return bs.IntersectsSphere(v)
	case *AABB:
		return v.IntersectsSphere(bs)
	}
	return false
}

func (bs *BoundingSphere) IntersectsSphere(other *BoundingSphere) bool {
	dx := bs.Center.X - other.Center.X
	dy := bs.Center.Y - other.Center.Y
	dz := bs.Center.Z - other.Center.Z
	distSq := dx*dx + dy*dy + dz*dz
	radiusSum := bs.Radius + other.Radius
	return distSq <= radiusSum*radiusSum
}

func (bs *BoundingSphere) IntersectsRay(ray Ray) (bool, float64) {
	// Ray-sphere intersection
	// Vector from ray origin to sphere center
	ocX := ray.Origin.X - bs.Center.X
	ocY := ray.Origin.Y - bs.Center.Y
	ocZ := ray.Origin.Z - bs.Center.Z

	a := ray.Direction.X*ray.Direction.X + ray.Direction.Y*ray.Direction.Y + ray.Direction.Z*ray.Direction.Z
	b := 2.0 * (ocX*ray.Direction.X + ocY*ray.Direction.Y + ocZ*ray.Direction.Z)
	c := ocX*ocX + ocY*ocY + ocZ*ocZ - bs.Radius*bs.Radius

	discriminant := b*b - 4*a*c

	if discriminant < 0 {
		return false, 0
	}

	t := (-b - math.Sqrt(discriminant)) / (2.0 * a)
	if t < 0 {
		t = (-b + math.Sqrt(discriminant)) / (2.0 * a)
	}

	return t >= 0, t
}

func (bs *BoundingSphere) GetCenter() Point {
	return bs.Center
}

func (bs *BoundingSphere) GetRadius() float64 {
	return bs.Radius
}

// ComputeTriangleBounds computes AABB for a triangle
func ComputeTriangleBounds(t *Triangle) *AABB {
	return NewAABBFromPoints([]Point{t.P0, t.P1, t.P2})
}

// ComputeMeshBounds computes AABB for a mesh
func ComputeMeshBounds(mesh *Mesh) *AABB {
	points := make([]Point, 0)

	for _, tri := range mesh.Triangles {
		points = append(points, tri.P0, tri.P1, tri.P2)
	}

	for _, quad := range mesh.Quads {
		points = append(points, quad.P0, quad.P1, quad.P2, quad.P3)
	}

	return NewAABBFromPoints(points)
}

// TransformAABB transforms an AABB by a transform (creates new oriented bounding box as AABB)
func TransformAABB(aabb *AABB, transform *Transform) *AABB {
	// Get 8 corners of AABB
	corners := []Point{
		{X: aabb.Min.X, Y: aabb.Min.Y, Z: aabb.Min.Z},
		{X: aabb.Max.X, Y: aabb.Min.Y, Z: aabb.Min.Z},
		{X: aabb.Min.X, Y: aabb.Max.Y, Z: aabb.Min.Z},
		{X: aabb.Max.X, Y: aabb.Max.Y, Z: aabb.Min.Z},
		{X: aabb.Min.X, Y: aabb.Min.Y, Z: aabb.Max.Z},
		{X: aabb.Max.X, Y: aabb.Min.Y, Z: aabb.Max.Z},
		{X: aabb.Min.X, Y: aabb.Max.Y, Z: aabb.Max.Z},
		{X: aabb.Max.X, Y: aabb.Max.Y, Z: aabb.Max.Z},
	}

	// Transform all corners
	transformed := make([]Point, len(corners))
	for i, corner := range corners {
		transformed[i] = transform.TransformPoint(corner)
	}

	// Create new AABB from transformed corners
	return NewAABBFromPoints(transformed)
}
