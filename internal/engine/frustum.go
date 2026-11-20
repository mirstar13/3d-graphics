package engine

import (
	"math"
)

// FrustumPlane represents a plane in the view frustum
type FrustumPlane struct {
	Normal   Point
	Distance float64
}

// ViewFrustum represents the camera's view frustum with 6 planes
type ViewFrustum struct {
	Planes [6]FrustumPlane // Left, Right, Top, Bottom, Near, Far
}

const (
	FrustumLeft = iota
	FrustumRight
	FrustumTop
	FrustumBottom
	FrustumNear
	FrustumFar
)

// Degrees to Radians conversion factor
const DegToRad = math.Pi / 180.0

// BuildFrustum constructs inward-facing, normalized planes matching Camera.ProjectPoint.
// Plane distances are measured in world units, allowing sphere-radius tests.
func BuildFrustum(camera *Camera) ViewFrustum {
	position := camera.GetPosition()
	forward := camera.GetForwardVectorPoint()
	right := camera.GetRightVectorPoint()
	up := camera.GetUpVectorPoint()
	halfX, halfY := camera.projectionHalfExtents()
	plane := func(normal Point, offset float64) FrustumPlane {
		normal.X, normal.Y, normal.Z = NormalizeVector(normal.X, normal.Y, normal.Z)
		return FrustumPlane{Normal: normal,
			Distance: -DotProduct(normal.X, normal.Y, normal.Z, position.X, position.Y, position.Z) + offset}
	}
	side := func(axis Point, tangent, sign float64) FrustumPlane {
		return plane(Point{
			X: forward.X*tangent + axis.X*sign,
			Y: forward.Y*tangent + axis.Y*sign,
			Z: forward.Z*tangent + axis.Z*sign,
		}, 0)
	}
	var frustum ViewFrustum
	frustum.Planes[FrustumNear] = plane(forward, -camera.Near)
	frustum.Planes[FrustumFar] = plane(Point{X: -forward.X, Y: -forward.Y, Z: -forward.Z}, camera.Far)
	frustum.Planes[FrustumLeft] = side(right, halfX, 1)
	frustum.Planes[FrustumRight] = side(right, halfX, -1)
	frustum.Planes[FrustumTop] = side(up, halfY, -1)
	frustum.Planes[FrustumBottom] = side(up, halfY, 1)
	return frustum
}

// TestPoint tests if a point is inside the frustum
func (f *ViewFrustum) TestPoint(p Point) bool {
	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		distance := DotProduct(plane.Normal.X, plane.Normal.Y, plane.Normal.Z, p.X, p.Y, p.Z) + plane.Distance

		if distance < 0 {
			return false // Outside this plane
		}
	}
	return true
}

// TestSphere tests if a sphere intersects the frustum
func (f *ViewFrustum) TestSphere(center Point, radius float64) bool {
	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		distance := DotProduct(plane.Normal.X, plane.Normal.Y, plane.Normal.Z, center.X, center.Y, center.Z) + plane.Distance

		if distance < -radius {
			return false // Completely outside this plane
		}
	}
	return true
}

// TestAABB tests if an AABB intersects the frustum
func (f *ViewFrustum) TestAABB(aabb *AABB) bool {
	// Get AABB corners
	corners := [8]Point{
		{X: aabb.Min.X, Y: aabb.Min.Y, Z: aabb.Min.Z},
		{X: aabb.Max.X, Y: aabb.Min.Y, Z: aabb.Min.Z},
		{X: aabb.Min.X, Y: aabb.Max.Y, Z: aabb.Min.Z},
		{X: aabb.Max.X, Y: aabb.Max.Y, Z: aabb.Min.Z},
		{X: aabb.Min.X, Y: aabb.Min.Y, Z: aabb.Max.Z},
		{X: aabb.Max.X, Y: aabb.Min.Y, Z: aabb.Max.Z},
		{X: aabb.Min.X, Y: aabb.Max.Y, Z: aabb.Max.Z},
		{X: aabb.Max.X, Y: aabb.Max.Y, Z: aabb.Max.Z},
	}

	// Test each plane
	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		insideCount := 0

		// Check if any corner is inside this plane
		for _, corner := range corners {
			distance := DotProduct(plane.Normal.X, plane.Normal.Y, plane.Normal.Z, corner.X, corner.Y, corner.Z) + plane.Distance
			if distance >= 0 {
				insideCount++
			}
		}

		// If all corners are outside this plane, AABB is outside frustum
		if insideCount == 0 {
			return false
		}
	}

	return true
}

// TestOBB tests if an OBB intersects the frustum
func (f *ViewFrustum) TestOBB(obb *OBB) bool {
	corners := obb.GetCorners()

	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		insideCount := 0

		for _, corner := range corners {
			distance := DotProduct(plane.Normal.X, plane.Normal.Y, plane.Normal.Z, corner.X, corner.Y, corner.Z) + plane.Distance
			if distance >= 0 {
				insideCount++
			}
		}

		if insideCount == 0 {
			return false
		}
	}

	return true
}

// FrustumCullNode recursively culls scene nodes against frustum
func FrustumCullNode(node *SceneNode, frustum *ViewFrustum, visible *[]*SceneNode) {
	if !node.IsEnabled() {
		return
	}

	// Test node's bounding volume
	if node.Object != nil {
		bounds := ComputeNodeBounds(node)
		// Object bounds do not enclose descendants; test each child separately.
		if bounds == nil || frustum.TestAABB(bounds) {
			*visible = append(*visible, node)
		}
	}

	// Recurse to children
	for _, child := range node.Children {
		FrustumCullNode(child, frustum, visible)
	}
}

// BuildFrustumSimple is a compatibility alias for BuildFrustum.
func BuildFrustumSimple(camera *Camera) ViewFrustum {
	return BuildFrustum(camera)
}
