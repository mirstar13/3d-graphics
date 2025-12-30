package main

import "math"

func GenerateSphere(radius float64, rings, sectors int, material Material) *Mesh {
	mesh := NewMesh()

	R := 1.0 / float64(rings-1)
	S := 1.0 / float64(sectors-1)

	for r := 0; r < rings-1; r++ {
		for s := 0; s < sectors-1; s++ {
			// Y is up/down (latitude), X/Z are the circle (longitude)

			// Point 1: Current
			y1 := math.Sin(-math.Pi/2 + math.Pi*float64(r)*R)
			r1 := math.Cos(-math.Pi/2 + math.Pi*float64(r)*R) // Radius at this height
			x1 := math.Cos(2*math.Pi*float64(s)*S) * r1
			z1 := math.Sin(2*math.Pi*float64(s)*S) * r1

			// Point 2: Next Longitude (Same Ring)
			y2 := math.Sin(-math.Pi/2 + math.Pi*float64(r)*R)
			r2 := r1
			x2 := math.Cos(2*math.Pi*float64(s+1)*S) * r2
			z2 := math.Sin(2*math.Pi*float64(s+1)*S) * r2

			// Point 3: Next Ring (Same Longitude)
			y3 := math.Sin(-math.Pi/2 + math.Pi*float64(r+1)*R)
			r3 := math.Cos(-math.Pi/2 + math.Pi*float64(r+1)*R)
			x3 := math.Cos(2*math.Pi*float64(s)*S) * r3
			z3 := math.Sin(2*math.Pi*float64(s)*S) * r3

			// Point 4: Next Ring, Next Longitude
			y4 := math.Sin(-math.Pi/2 + math.Pi*float64(r+1)*R)
			r4 := r3
			x4 := math.Cos(2*math.Pi*float64(s+1)*S) * r4
			z4 := math.Sin(2*math.Pi*float64(s+1)*S) * r4

			p1 := *NewPoint(x1*radius, y1*radius, z1*radius)
			p2 := *NewPoint(x2*radius, y2*radius, z2*radius)
			p3 := *NewPoint(x3*radius, y3*radius, z3*radius)
			p4 := *NewPoint(x4*radius, y4*radius, z4*radius)

			quad := NewQuad(p1, p2, p4, p3)
			quad.SetMaterial(material)

			mesh.AddQuad(quad)
		}
	}

	return mesh
}
