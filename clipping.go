package main

import "math"

// ClipTriangleToNearPlane clips a triangle against the camera's near plane
// Returns 0, 1, or 2 triangles depending on how the original triangle intersects the plane
func ClipTriangleToNearPlane(t *Triangle, camera *Camera) []*Triangle {
	nearPlane := camera.Near

	// Transform vertices to view space
	v0 := camera.TransformToViewSpace(t.P0)
	v1 := camera.TransformToViewSpace(t.P1)
	v2 := camera.TransformToViewSpace(t.P2)

	// Calculate z-depth for each vertex in view space
	z0 := v0.Z
	z1 := v1.Z
	z2 := v2.Z

	behind := [3]bool{z0 <= nearPlane, z1 <= nearPlane, z2 <= nearPlane}
	behindCount := 0
	if behind[0] {
		behindCount++
	}
	if behind[1] {
		behindCount++
	}
	if behind[2] {
		behindCount++
	}

	if behindCount == 0 {
		return []*Triangle{t}
	}
	if behindCount == 3 {
		return []*Triangle{}
	}

	vertices := [3]Point{t.P0, t.P1, t.P2}
	zDepths := [3]float64{z0, z1, z2}

	if behindCount == 1 {
		return clipOneVertexBehind(vertices, zDepths, behind, nearPlane, 0.0, t, camera)
	}
	return clipTwoVerticesBehind(vertices, zDepths, behind, nearPlane, 0.0, t, camera)
}

// clipOneVertexBehind handles case where one vertex is behind near plane
// Returns 2 triangles forming a quad
func clipOneVertexBehind(vertices [3]Point, zDepths [3]float64, behind [3]bool, nearPlane, dz float64, original *Triangle, camera *Camera) []*Triangle {
	// Find the vertex that's behind
	behindIdx := 0
	for i := 0; i < 3; i++ {
		if behind[i] {
			behindIdx = i
			break
		}
	}

	// Get indices in order: behind, front1, front2
	idx0 := behindIdx
	idx1 := (behindIdx + 1) % 3
	idx2 := (behindIdx + 2) % 3

	vBehind := vertices[idx0]
	vFront1 := vertices[idx1]
	vFront2 := vertices[idx2]

	zBehind := zDepths[idx0]
	zFront1 := zDepths[idx1]
	zFront2 := zDepths[idx2]

	// Intersect edge (behind -> front1) with near plane
	intersection1 := intersectEdgeWithPlane(vBehind, vFront1, zBehind, zFront1, nearPlane, dz, camera)

	// Intersect edge (behind -> front2) with near plane
	intersection2 := intersectEdgeWithPlane(vBehind, vFront2, zBehind, zFront2, nearPlane, dz, camera)

	// Create two triangles from the quad: [intersection1, front1, front2] and [intersection1, front2, intersection2]
	t1 := NewTriangle(intersection1, vFront1, vFront2, original.char)
	t1.Material = original.Material
	if original.UseSetNormal {
		t1.SetNormal(*original.Normal)
	}

	t2 := NewTriangle(intersection1, vFront2, intersection2, original.char)
	t2.Material = original.Material
	if original.UseSetNormal {
		t2.SetNormal(*original.Normal)
	}

	return []*Triangle{t1, t2}
}

// clipTwoVerticesBehind handles case where two vertices are behind near plane
// Returns 1 triangle
func clipTwoVerticesBehind(vertices [3]Point, zDepths [3]float64, behind [3]bool, nearPlane, dz float64, original *Triangle, camera *Camera) []*Triangle {
	// Find the vertex that's in front
	frontIdx := 0
	for i := 0; i < 3; i++ {
		if !behind[i] {
			frontIdx = i
			break
		}
	}

	// Get indices: front, behind1, behind2
	idx0 := frontIdx
	idx1 := (frontIdx + 1) % 3
	idx2 := (frontIdx + 2) % 3

	vFront := vertices[idx0]
	vBehind1 := vertices[idx1]
	vBehind2 := vertices[idx2]

	zFront := zDepths[idx0]
	zBehind1 := zDepths[idx1]
	zBehind2 := zDepths[idx2]

	// Intersect edge (front -> behind1) with near plane
	intersection1 := intersectEdgeWithPlane(vFront, vBehind1, zFront, zBehind1, nearPlane, dz, camera)

	// Intersect edge (front -> behind2) with near plane
	intersection2 := intersectEdgeWithPlane(vFront, vBehind2, zFront, zBehind2, nearPlane, dz, camera)

	// Create one triangle from the remaining visible portion
	t := NewTriangle(vFront, intersection1, intersection2, original.char)
	t.Material = original.Material
	if original.UseSetNormal {
		t.SetNormal(*original.Normal)
	}

	return []*Triangle{t}
}

// intersectEdgeWithPlane finds the intersection point of an edge with the near plane
func intersectEdgeWithPlane(v0, v1 Point, z0, z1, nearPlane, dz float64, camera *Camera) Point {
	// Calculate interpolation parameter t where the edge crosses the near plane
	// z0 + t*(z1 - z0) = nearPlane
	t := (nearPlane - z0) / (z1 - z0)

	// Clamp t to [0, 1] for safety
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	// Linear interpolation of vertex position in world space
	return Point{
		X: v0.X + t*(v1.X-v0.X),
		Y: v0.Y + t*(v1.Y-v0.Y),
		Z: v0.Z + t*(v1.Z-v0.Z),
	}
}

// Helper function to check if a point is in front of the near plane
func isInFrontOfNearPlane(p Point, camera *Camera) bool {
	viewPoint := camera.TransformToViewSpace(p)
	z := viewPoint.Z + camera.DZ
	return z > camera.Near
}

// ClipLineToNearPlane clips a line against the near plane
// Returns true if the line is visible after clipping, false if completely behind
func ClipLineToNearPlane(line *Line, camera *Camera) (*Line, bool) {
	nearPlane := camera.Near
	dz := camera.DZ

	// Transform to view space
	v0 := camera.TransformToViewSpace(line.Start)
	v1 := camera.TransformToViewSpace(line.End)

	z0 := v0.Z + dz
	z1 := v1.Z + dz

	// Both in front - no clipping needed
	if z0 > nearPlane && z1 > nearPlane {
		return line, true
	}

	// Both behind - discard
	if z0 <= nearPlane && z1 <= nearPlane {
		return nil, false
	}

	// One behind, one in front - clip
	var clippedStart, clippedEnd Point

	if z0 <= nearPlane {
		// Start is behind, clip it
		clippedStart = intersectEdgeWithPlane(line.Start, line.End, z0, z1, nearPlane, dz, camera)
		clippedEnd = line.End
	} else {
		// End is behind, clip it
		clippedStart = line.Start
		clippedEnd = intersectEdgeWithPlane(line.Start, line.End, z0, z1, nearPlane, dz, camera)
	}

	return NewLine(clippedStart, clippedEnd), true
}

// SmoothNearPlaneTransition provides a smooth fade effect near the clipping plane
// Returns alpha value [0, 1] for transparency/fade based on distance to near plane
func SmoothNearPlaneTransition(z, nearPlane, fadeDistance float64) float64 {
	if z >= nearPlane+fadeDistance {
		return 1.0 // Fully visible
	}
	if z <= nearPlane {
		return 0.0 // Clipped
	}

	// Linear fade in the transition region
	return (z - nearPlane) / fadeDistance
}

// GetClippedTriangleArea calculates the area of a triangle (useful for LOD)
func GetClippedTriangleArea(t *Triangle) float64 {
	// Vector from P0 to P1
	ux := t.P1.X - t.P0.X
	uy := t.P1.Y - t.P0.Y
	uz := t.P1.Z - t.P0.Z

	// Vector from P0 to P2
	vx := t.P2.X - t.P0.X
	vy := t.P2.Y - t.P0.Y
	vz := t.P2.Z - t.P0.Z

	// Cross product
	cx, cy, cz := crossProduct(ux, uy, uz, vx, vy, vz)

	// Magnitude of cross product / 2 = area
	return math.Sqrt(cx*cx+cy*cy+cz*cz) / 2.0
}
