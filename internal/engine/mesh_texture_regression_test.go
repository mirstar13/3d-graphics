package engine

import (
	"math"
	"testing"
)

func TestSphereTrianglesFaceOutward(t *testing.T) {
	mesh := GenerateSphere(3, 8, 12)
	checked := 0
	for i := 0; i < len(mesh.Indices); i += 3 {
		a, b, c := mesh.Vertices[mesh.Indices[i]], mesh.Vertices[mesh.Indices[i+1]], mesh.Vertices[mesh.Indices[i+2]]
		ux, uy, uz := b.X-a.X, b.Y-a.Y, b.Z-a.Z
		vx, vy, vz := c.X-a.X, c.Y-a.Y, c.Z-a.Z
		cx, cy, cz := uy*vz-uz*vy, uz*vx-ux*vz, ux*vy-uy*vx
		if cx*cx+cy*cy+cz*cz < 1e-16 {
			continue
		}
		n := CalculateSurfaceNormal(&a, &b, &c, nil, false)
		dot := n.X*(a.X+b.X+c.X) + n.Y*(a.Y+b.Y+c.Y) + n.Z*(a.Z+b.Z+c.Z)
		// Pole triangles can be degenerate.
		if math.Abs(dot) < 1e-8 {
			continue
		}
		if dot < 0 {
			t.Fatalf("triangle %d faces inward", i/3)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no nondegenerate faces")
	}
}
func TestTexturedMaterialSamplesMap(t *testing.T) {
	material := NewTexturedMaterial()
	material.DiffuseTexture = GenerateCheckerboard(8, 8, 4, ColorRed, ColorBlue)
	material.TextureFilter = FilterNearest
	material.UseTextures = true
	var surface IMaterial = &material
	if surface.SampleDiffuse(.1, .1) == surface.SampleDiffuse(.6, .1) {
		t.Fatal("texture replaced by solid color")
	}
	material.UseTextures = false
	if surface.SampleDiffuse(.1, .1) != material.DiffuseColor {
		t.Fatal("disabled texture does not use base color")
	}
}
