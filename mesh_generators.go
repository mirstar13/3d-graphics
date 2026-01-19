package main

import "math"

func GenerateSphere(radius float64, rings, sectors int) *Mesh {
	mesh := NewMesh()

	// --- Phase 1: Generate Vertices ---
	// We iterate 0 to rings (inclusive) to get the top and bottom rows.
	// We iterate 0 to sectors (inclusive) to close the loop (the last vertex overlaps the first).
	for r := 0; r <= rings; r++ {

		// Calculate the percentage of height (0.0 to 1.0)
		v := float64(r) / float64(rings)

		// Calculate Latitude angles (Y axis)
		// -Pi/2 is South Pole, +Pi/2 is North Pole
		latAngle := -math.Pi/2 + math.Pi*v

		y := math.Sin(latAngle) * radius
		ringRadius := math.Cos(latAngle) * radius // The radius of the slice at this height

		for s := 0; s <= sectors; s++ {

			// Calculate the percentage around the circle (0.0 to 1.0)
			u := float64(s) / float64(sectors)

			// Calculate Longitude angles (X/Z circle)
			lonAngle := 2 * math.Pi * u

			x := math.Cos(lonAngle) * ringRadius
			z := math.Sin(lonAngle) * ringRadius

			// Add the single unique vertex to the list
			// Note: We are no longer making Quads here. Just points.
			mesh.AddVertex(x, y, z)

			// The normal for a sphere is the normalized vector from the center to the vertex
			nx := x / radius
			ny := y / radius
			nz := z / radius
			mesh.AddNormal(nx, ny, nz)
			
			// Add UV coordinates
			mesh.AddUV(u, 1.0-v) // Flip V for OpenGL convention
		}
	}

	// --- Phase 2: Generate Indices (Triangles) ---
	// We loop through the "squares" of the grid and cut them into triangles.
	// Note: We stop *before* the last row/col because we are looking "forward" to i+1
	for r := 0; r < rings; r++ {
		for s := 0; s < sectors; s++ {

			// The number of vertices in a single row (used to jump to the next row)
			stride := sectors + 1

			// Calculate the unique Index ID for the 4 corners of the current square
			curr := r*stride + s                 // Top-Left
			next := r*stride + (s + 1)           // Top-Right
			bottom := (r+1)*stride + s           // Bottom-Left
			bottomNext := (r+1)*stride + (s + 1) // Bottom-Right

			// First Triangle (Top-Left, Top-Right, Bottom-Left)
			mesh.AddIndex(curr)
			mesh.AddIndex(next)
			mesh.AddIndex(bottom)

			// Second Triangle (Top-Right, Bottom-Right, Bottom-Left)
			mesh.AddIndex(next)
			mesh.AddIndex(bottomNext)
			mesh.AddIndex(bottom)
		}
	}

	return mesh
}

// GenerateTorus generates a torus mesh with indexed geometry
// majorRadius: distance from center of torus to center of tube
// minorRadius: radius of the tube itself
// majorSegments: number of segments around the major circle
// minorSegments: number of segments around the tube
func GenerateTorus(majorRadius, minorRadius float64, majorSegments, minorSegments int) *Mesh {
	mesh := NewMesh()

	// --- Phase 1: Generate Vertices ---
	// We iterate 0 to majorSegments (inclusive) to close the major loop
	// We iterate 0 to minorSegments (inclusive) to close the minor loop
	for i := 0; i <= majorSegments; i++ {
		// Angle around the major circle (torus ring)
		u := float64(i) / float64(majorSegments)
		theta := u * 2.0 * math.Pi

		cosTheta := math.Cos(theta)
		sinTheta := math.Sin(theta)

		for j := 0; j <= minorSegments; j++ {
			// Angle around the minor circle (tube)
			v := float64(j) / float64(minorSegments)
			phi := v * 2.0 * math.Pi

			cosPhi := math.Cos(phi)
			sinPhi := math.Sin(phi)

			// Torus parametric equations:
			// x = (R + r*cos(phi)) * cos(theta)
			// y = r * sin(phi)
			// z = (R + r*cos(phi)) * sin(theta)
			x := (majorRadius + minorRadius*cosPhi) * cosTheta
			y := minorRadius * sinPhi
			z := (majorRadius + minorRadius*cosPhi) * sinTheta

			mesh.AddVertex(x, y, z)

			// Calculate and add the normal for the torus
			// This is derived from the partial derivatives of the parametric equations
			nx := cosPhi * cosTheta
			ny := sinPhi
			nz := cosPhi * sinTheta
			mesh.AddNormal(nx, ny, nz)

			mesh.AddUV(u, v)
		}
	}

	// --- Phase 2: Generate Indices (Triangles) ---
	// We loop through the grid and create two triangles per quad
	for i := 0; i < majorSegments; i++ {
		for j := 0; j < minorSegments; j++ {
			// The number of vertices in a single minor ring
			stride := minorSegments + 1

			// Calculate the indices for the 4 corners of the current quad
			curr := i*stride + j                 // Current ring, current segment
			next := i*stride + (j + 1)           // Current ring, next segment
			bottom := (i+1)*stride + j           // Next ring, current segment
			bottomNext := (i+1)*stride + (j + 1) // Next ring, next segment

			// First Triangle (curr, next, bottom)
			mesh.AddIndex(curr)
			mesh.AddIndex(next)
			mesh.AddIndex(bottom)

			// Second Triangle (next, bottomNext, bottom)
			mesh.AddIndex(next)
			mesh.AddIndex(bottomNext)
			mesh.AddIndex(bottom)
		}
	}

	return mesh
}
