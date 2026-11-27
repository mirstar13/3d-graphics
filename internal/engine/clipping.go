package engine

import (
	"math"
)

// ClipTriangleToNearPlane clips world-space geometry while preserving winding,
// material, normals, and interpolated texture coordinates.
func ClipTriangleToNearPlane(triangle *Triangle, camera *Camera) []*Triangle {
	depths := [3]float64{camera.TransformToViewSpace(triangle.P0).Z, camera.TransformToViewSpace(triangle.P1).Z, camera.TransformToViewSpace(triangle.P2).Z}
	var out [2]Triangle
	count := ClipTriangleAtDepths(triangle, depths, camera.Near, &out)
	switch count {
	case 0:
		return nil
	case 1:
		if depths[0] >= camera.Near && depths[1] >= camera.Near && depths[2] >= camera.Near {
			return []*Triangle{triangle}
		}
		return []*Triangle{&out[0]}
	default:
		return []*Triangle{&out[0], &out[1]}
	}
}

// ClipTriangleAtDepths writes up to two triangles using precomputed view depths.
func ClipTriangleAtDepths(triangle *Triangle, depths [3]float64, near float64, out *[2]Triangle) int {
	type vertex struct {
		point Point
		uv    TextureCoord
		depth float64
	}
	input := []vertex{
		{triangle.P0, triangle.UV0, depths[0]},
		{triangle.P1, triangle.UV1, depths[1]},
		{triangle.P2, triangle.UV2, depths[2]},
	}
	inside := 0
	for _, v := range input {
		if v.depth >= near {
			inside++
		}
	}
	if inside == 3 {
		out[0] = *triangle
		return 1
	}
	if inside == 0 {
		return 0
	}

	var vertices [4]vertex
	output := vertices[:0]
	previous := input[len(input)-1]
	for _, current := range input {
		previousInside, currentInside := previous.depth >= near, current.depth >= near
		if previousInside != currentInside {
			fraction := (near - previous.depth) / (current.depth - previous.depth)
			output = append(output, vertex{
				point: lerpPoint(previous.point, current.point, fraction),
				uv: TextureCoord{
					U: previous.uv.U + (current.uv.U-previous.uv.U)*fraction,
					V: previous.uv.V + (current.uv.V-previous.uv.V)*fraction,
				},
				depth: near,
			})
		}
		if currentInside {
			output = append(output, current)
		}
		previous = current
	}
	count := 0
	for i := 1; i+1 < len(output); i++ {
		clipped := *triangle
		clipped.P0, clipped.P1, clipped.P2 = output[0].point, output[i].point, output[i+1].point
		clipped.UV0, clipped.UV1, clipped.UV2 = output[0].uv, output[i].uv, output[i+1].uv
		out[count] = clipped
		count++
	}
	return count
}

// ClipLineToNearPlane clips a line against the near plane
// Returns true if the line is visible after clipping, false if completely behind
func ClipLineToNearPlane(line *Line, camera *Camera) (*Line, bool) {
	nearPlane := camera.Near

	// Transform to view space
	v0 := camera.TransformToViewSpace(line.Start)
	v1 := camera.TransformToViewSpace(line.End)

	z0 := v0.Z
	z1 := v1.Z

	// Both in front - no clipping needed
	if z0 >= nearPlane && z1 >= nearPlane {
		return line, true
	}

	// Both behind - discard
	if z0 < nearPlane && z1 < nearPlane {
		return nil, false
	}

	// One behind, one in front - clip
	var clippedStart, clippedEnd Point

	if z0 < nearPlane {
		// Start is behind, clip it
		clippedStart = lerpPoint(line.Start, line.End, (nearPlane-z0)/(z1-z0))
		clippedEnd = line.End
	} else {
		// End is behind, clip it
		clippedStart = line.Start
		clippedEnd = lerpPoint(line.Start, line.End, (nearPlane-z0)/(z1-z0))
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
