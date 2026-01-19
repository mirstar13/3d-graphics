package main

import (
	"math"
	"testing"
)

// ============================================================================
// BOUNDING VOLUME TESTS
// ============================================================================

func TestBoundingVolumes(t *testing.T) {
	t.Run("BasicOverlap", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		// Create two cubes that DEFINITELY overlap
		cube1 := scene.CreateCube("Cube1", 10, &mat)
		cube1.Transform.SetPosition(0, 0, 0)

		cube2 := scene.CreateCube("Cube2", 10, &mat)
		cube2.Transform.SetPosition(15, 0, 0) // Overlapping by 5 units

		bounds1 := scene.computeNodeBounds(cube1)
		bounds2 := scene.computeNodeBounds(cube2)

		if bounds1 == nil || bounds2 == nil {
			t.Fatal("Could not compute bounds")
		}

		t.Logf("Cube1 bounds: Min(%.1f,%.1f,%.1f) Max(%.1f,%.1f,%.1f)",
			bounds1.Min.X, bounds1.Min.Y, bounds1.Min.Z,
			bounds1.Max.X, bounds1.Max.Y, bounds1.Max.Z)
		t.Logf("Cube2 bounds: Min(%.1f,%.1f,%.1f) Max(%.1f,%.1f,%.1f)",
			bounds2.Min.X, bounds2.Min.Y, bounds2.Min.Z,
			bounds2.Max.X, bounds2.Max.Y, bounds2.Max.Z)

		if !bounds1.IntersectsAABB(bounds2) {
			t.Error("Should detect overlap between overlapping cubes")
		}
	})

	t.Run("Separation", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube1 := scene.CreateCube("Cube1", 10, &mat)
		cube1.Transform.SetPosition(0, 0, 0)

		cube3 := scene.CreateCube("Cube3", 10, &mat)
		cube3.Transform.SetPosition(50, 0, 0) // Far from cube1

		bounds1 := scene.computeNodeBounds(cube1)
		bounds3 := scene.computeNodeBounds(cube3)

		if bounds1 == nil || bounds3 == nil {
			t.Fatal("Could not compute bounds")
		}

		if bounds1.IntersectsAABB(bounds3) {
			t.Error("Should NOT detect overlap between separated cubes")
		}
	})

	t.Run("WithRotation", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube4 := scene.CreateCube("Cube4", 10, &mat)
		cube4.Transform.SetPosition(0, 0, 0)
		cube4.Transform.SetRotation(0, 0.785, 0) // 45 degrees

		bounds4 := scene.computeNodeBounds(cube4)
		if bounds4 == nil {
			t.Fatal("Could not compute bounds for rotated cube")
		}

		// Rotated cube should have larger bounds due to corners
		size := bounds4.Max.X - bounds4.Min.X
		if size <= 20 { // Should be larger than 20 when rotated
			t.Errorf("Rotated bounds not expanded correctly: got %.1f, expected > 20", size)
		}
		t.Logf("Rotated cube bounds expanded correctly (size: %.1f)", size)
	})

	t.Run("SphereBounds", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		sphere := scene.CreateSphere("Sphere", 5, 16, 16, &mat)
		sphere.Transform.SetPosition(0, 0, 0)

		sphereBounds := scene.computeNodeBounds(sphere)
		if sphereBounds == nil {
			t.Fatal("Could not compute sphere bounds")
		}

		// Sphere of radius 5 should have bounds from roughly -5 to +5
		size := sphereBounds.Max.X - sphereBounds.Min.X
		if size < 9 || size > 11 { // Allow some tolerance for approximation
			t.Errorf("Sphere bounds incorrect: got %.1f, expected ~10", size)
		}
		t.Logf("Sphere bounds correct (size: %.1f, expected ~10)", size)
	})

	t.Run("WithScale", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorRed

		cube := scene.CreateCube("ScaledCube", 10, &mat)
		cube.Transform.SetPosition(0, 0, 0)
		cube.Transform.SetScale(2, 1, 0.5)

		bounds := scene.computeNodeBounds(cube)
		if bounds == nil {
			t.Fatal("Could not compute bounds for scaled cube")
		}

		// Check each dimension is scaled correctly
		sizeX := bounds.Max.X - bounds.Min.X
		sizeY := bounds.Max.Y - bounds.Min.Y
		sizeZ := bounds.Max.Z - bounds.Min.Z

		// Original size is 20, scaled by (2, 1, 0.5)
		if math.Abs(sizeX-40) > 1 {
			t.Errorf("X dimension incorrect: got %.1f, expected ~40", sizeX)
		}
		if math.Abs(sizeY-20) > 1 {
			t.Errorf("Y dimension incorrect: got %.1f, expected ~20", sizeY)
		}
		if math.Abs(sizeZ-10) > 1 {
			t.Errorf("Z dimension incorrect: got %.1f, expected ~10", sizeZ)
		}
		t.Logf("Scaled bounds correct: (%.1f, %.1f, %.1f)", sizeX, sizeY, sizeZ)
	})

	t.Run("ContainsPoint", func(t *testing.T) {
		bounds := &AABB{
			Min: Point{X: -10, Y: -10, Z: -10},
			Max: Point{X: 10, Y: 10, Z: 10},
		}

		testCases := []struct {
			point  Point
			inside bool
		}{
			{Point{X: 0, Y: 0, Z: 0}, true},    // Center
			{Point{X: 9, Y: 9, Z: 9}, true},    // Inside
			{Point{X: 15, Y: 0, Z: 0}, false},  // Outside X
			{Point{X: 0, Y: 15, Z: 0}, false},  // Outside Y
			{Point{X: 0, Y: 0, Z: 15}, false},  // Outside Z
			{Point{X: -10, Y: 0, Z: 0}, true},  // On boundary
			{Point{X: 10, Y: 10, Z: 10}, true}, // On corner
		}

		for _, tc := range testCases {
			result := bounds.Contains(tc.point)
			if result != tc.inside {
				t.Errorf("Point (%.1f,%.1f,%.1f) containment wrong: got %v, expected %v",
					tc.point.X, tc.point.Y, tc.point.Z, result, tc.inside)
			}
		}
	})
}

// ============================================================================
// RAYCASTING TESTS
// ============================================================================

func TestRaycasting(t *testing.T) {
	t.Run("DirectHit", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube := scene.CreateCube("Target", 10, &mat)
		cube.Transform.SetPosition(0, 0, 0)

		// Cast ray from -50 toward +Z
		ray := NewRay(Point{X: 0, Y: 0, Z: -50}, Point{X: 0, Y: 0, Z: 1})
		hit := scene.Raycast(ray, 100.0)

		if !hit.Hit {
			t.Error("Ray should hit cube")
			return
		}

		expectedDist := 40.0 // 50 - 10 (cube extends from -10 to +10)
		if hit.Distance < 39 || hit.Distance > 41 {
			t.Errorf("Hit at wrong distance: got %.2f, expected ~%.1f", hit.Distance, expectedDist)
		}
		t.Logf("Ray hit at distance %.2f (expected ~%.1f)", hit.Distance, expectedDist)
	})

	t.Run("Miss", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube := scene.CreateCube("Target", 10, &mat)
		cube.Transform.SetPosition(0, 0, 0)

		ray := NewRay(Point{X: 30, Y: 0, Z: -50}, Point{X: 0, Y: 0, Z: 1})
		hit := scene.Raycast(ray, 100.0)

		if hit.Hit {
			t.Error("Ray should miss the cube")
		}
	})

	t.Run("TransformedObject", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube := scene.CreateCube("TransformedTarget", 10, &mat)
		cube.Transform.SetPosition(20, 0, 0) // Offset to the right

		// Cast ray at the transformed position
		ray := NewRay(Point{X: 20, Y: 0, Z: -50}, Point{X: 0, Y: 0, Z: 1})
		hit := scene.Raycast(ray, 100.0)

		if !hit.Hit {
			t.Error("Ray should hit transformed cube")
			return
		}
		t.Logf("Ray hit transformed cube at distance %.2f", hit.Distance)
	})

	t.Run("MaxDistance", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorGreen

		cube := scene.CreateCube("FarTarget", 10, &mat)
		cube.Transform.SetPosition(0, 0, 0)

		// Ray with limited max distance
		ray := NewRay(Point{X: 0, Y: 0, Z: -100}, Point{X: 0, Y: 0, Z: 1})
		hit := scene.Raycast(ray, 50.0) // Only search first 50 units

		if hit.Hit {
			t.Error("Ray should not hit cube beyond max distance")
		}
	})

	t.Run("MultipleObjects", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorYellow

		// Create three cubes in a line
		cube1 := scene.CreateCube("Near", 10, &mat)
		cube1.Transform.SetPosition(0, 0, -20)

		cube2 := scene.CreateCube("Middle", 10, &mat)
		cube2.Transform.SetPosition(0, 0, 0)

		cube3 := scene.CreateCube("Far", 10, &mat)
		cube3.Transform.SetPosition(0, 0, 20)

		// Ray should hit the nearest one
		ray := NewRay(Point{X: 0, Y: 0, Z: -50}, Point{X: 0, Y: 0, Z: 1})
		hit := scene.Raycast(ray, 100.0)

		if !hit.Hit {
			t.Error("Ray should hit at least one cube")
			return
		}

		// Should hit cube1 (nearest)
		if hit.Node != cube1 {
			t.Error("Ray should hit nearest cube first")
		}
		t.Logf("Correctly hit nearest cube at distance %.2f", hit.Distance)
	})

	t.Run("FromInside", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorMagenta

		cube := scene.CreateCube("Container", 20, &mat)
		cube.Transform.SetPosition(0, 0, 0)

		// Ray starting from inside the cube
		ray := NewRay(Point{X: 0, Y: 0, Z: 0}, Point{X: 1, Y: 0, Z: 0})
		hit := scene.Raycast(ray, 100.0)

		// Behavior depends on implementation - document what happens
		t.Logf("Ray from inside: hit=%v", hit.Hit)
	})
}

// ============================================================================
// MATERIAL SYSTEM TESTS
// ============================================================================

func TestMaterialSystem(t *testing.T) {
	t.Run("BasicMaterial", func(t *testing.T) {
		mat := NewMaterial()
		mat.DiffuseColor = ColorRed
		mat.SpecularColor = ColorWhite
		mat.Shininess = 64.0
		mat.SpecularStrength = 0.8

		if mat.GetType() != MaterialTypeBasic {
			t.Error("Material type should be Basic")
		}

		color := mat.GetDiffuseColor(0, 0)
		if color != ColorRed {
			t.Error("Diffuse color should be red")
		}

		if mat.GetShininess() != 64.0 {
			t.Errorf("Shininess should be 64.0, got %.1f", mat.GetShininess())
		}
	})

	t.Run("PBRMaterial", func(t *testing.T) {
		pbr := NewPBRMaterial()
		pbr.Albedo = ColorBlue
		pbr.Metallic = 0.8
		pbr.Roughness = 0.2

		if pbr.GetType() != MaterialTypePBR {
			t.Error("Material type should be PBR")
		}

		if pbr.GetMetallic() != 0.8 {
			t.Errorf("Metallic should be 0.8, got %.2f", pbr.GetMetallic())
		}

		if pbr.GetRoughness() != 0.2 {
			t.Errorf("Roughness should be 0.2, got %.2f", pbr.GetRoughness())
		}

		// PBR converts roughness to shininess
		shininess := pbr.GetShininess()
		expectedShininess := (1.0 - 0.2) * 128.0
		if math.Abs(shininess-expectedShininess) > 0.1 {
			t.Errorf("Shininess conversion incorrect: got %.1f, expected %.1f", shininess, expectedShininess)
		}
	})

	t.Run("WireframeMaterial", func(t *testing.T) {
		mat := NewMaterial()
		mat.Wireframe = true
		mat.WireframeColor = ColorGreen

		if !mat.IsWireframe() {
			t.Error("Should be wireframe material")
		}

		if mat.GetWireframeColor() != ColorGreen {
			t.Error("Wireframe color should be green")
		}
	})

	t.Run("MaterialPolymorphism", func(t *testing.T) {
		// Test that different material types work through IMaterial interface
		materials := []IMaterial{
			&Material{DiffuseColor: ColorRed},
			NewPBRMaterial(),
		}

		for i, mat := range materials {
			// Should be able to call interface methods on all types
			_ = mat.GetDiffuseColor(0, 0)
			_ = mat.GetSpecularColor()
			_ = mat.GetShininess()
			_ = mat.IsWireframe()
			t.Logf("Material %d implements IMaterial correctly", i)
		}
	})

	t.Run("GeometryWithMaterial", func(t *testing.T) {
		mat := NewMaterial()
		mat.DiffuseColor = ColorCyan

		// Test Triangle
		tri := NewTriangle(
			Point{X: 0, Y: 0, Z: 0},
			Point{X: 1, Y: 0, Z: 0},
			Point{X: 0, Y: 1, Z: 0},
			'x',
		)
		tri.SetMaterial(&mat)

		if tri.Material.GetDiffuseColor(0, 0) != ColorCyan {
			t.Error("Triangle material not set correctly")
		}

		// Test Mesh
		mesh := NewMesh()
		mesh.Material = &mat

		if mesh.Material.GetDiffuseColor(0, 0) != ColorCyan {
			t.Error("Mesh material not set correctly")
		}
	})
}

// ============================================================================
// GEOMETRY TESTS
// ============================================================================

func TestGeometry(t *testing.T) {
	t.Run("TriangleCreation", func(t *testing.T) {
		p0 := Point{X: 0, Y: 0, Z: 0}
		p1 := Point{X: 1, Y: 0, Z: 0}
		p2 := Point{X: 0, Y: 1, Z: 0}

		tri := NewTriangle(p0, p1, p2, 'x')

		if tri.P0 != p0 || tri.P1 != p1 || tri.P2 != p2 {
			t.Error("Triangle vertices not set correctly")
		}
	})

	t.Run("QuadCreation", func(t *testing.T) {
		p0 := Point{X: 0, Y: 0, Z: 0}
		p1 := Point{X: 1, Y: 0, Z: 0}
		p2 := Point{X: 1, Y: 1, Z: 0}
		p3 := Point{X: 0, Y: 1, Z: 0}

		quad := NewQuad(p0, p1, p2, p3)

		if quad.P0 != p0 || quad.P1 != p1 || quad.P2 != p2 || quad.P3 != p3 {
			t.Error("Quad vertices not set correctly")
		}
	})

	t.Run("MeshVerticesAndIndices", func(t *testing.T) {
		mesh := NewMesh()

		// Add vertices
		i0 := mesh.AddVertex(0, 0, 0)
		i1 := mesh.AddVertex(1, 0, 0)
		i2 := mesh.AddVertex(0, 1, 0)

		if i0 != 0 || i1 != 1 || i2 != 2 {
			t.Error("Vertex indices not sequential")
		}

		// Add triangle
		mesh.AddTriangleIndices(i0, i1, i2)

		if len(mesh.Indices) != 3 {
			t.Errorf("Expected 3 indices, got %d", len(mesh.Indices))
		}
	})

	t.Run("QuadToTriangles", func(t *testing.T) {
		quad := NewQuad(
			Point{X: 0, Y: 0, Z: 0},
			Point{X: 1, Y: 0, Z: 0},
			Point{X: 1, Y: 1, Z: 0},
			Point{X: 0, Y: 1, Z: 0},
		)

		triangles := ConvertQuadToTriangles(quad)

		if len(triangles) != 2 {
			t.Errorf("Expected 2 triangles, got %d", len(triangles))
		}
	})

	t.Run("ProceduralSphere", func(t *testing.T) {
		mesh := GenerateSphere(5.0, 16, 16)

		if len(mesh.Vertices) == 0 {
			t.Error("Sphere should have vertices")
		}

		if len(mesh.Indices)%3 != 0 {
			t.Error("Sphere indices should be multiple of 3 (triangles)")
		}

		t.Logf("Sphere generated with %d vertices and %d triangles",
			len(mesh.Vertices), len(mesh.Indices)/3)
	})

	t.Run("SphereNormals", func(t *testing.T) {
		radius := 5.0
		mesh := GenerateSphere(radius, 16, 16)

		// Check that normals array has the correct length
		if len(mesh.Normals) != len(mesh.Vertices) {
			t.Errorf("Expected %d normals, got %d", len(mesh.Vertices), len(mesh.Normals))
		}

		// Verify that normals are normalized unit vectors
		for i, normal := range mesh.Normals {
			length := math.Sqrt(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)
			if math.Abs(length-1.0) > 0.01 {
				t.Errorf("Normal %d not normalized: length %.4f", i, length)
			}
		}

		// For a sphere, normals should point radially outward from center
		// Check a few sample vertices
		if len(mesh.Vertices) > 0 {
			vertex := mesh.Vertices[0]
			normal := mesh.Normals[0]

			// Vertex position normalized should match normal direction
			vLen := math.Sqrt(vertex.X*vertex.X + vertex.Y*vertex.Y + vertex.Z*vertex.Z)
			expectedNormal := Point{X: vertex.X / vLen, Y: vertex.Y / vLen, Z: vertex.Z / vLen}

			// Allow some tolerance due to floating point precision
			if math.Abs(normal.X-expectedNormal.X) > 0.01 ||
				math.Abs(normal.Y-expectedNormal.Y) > 0.01 ||
				math.Abs(normal.Z-expectedNormal.Z) > 0.01 {
				t.Errorf("Normal direction incorrect for sphere vertex 0")
			}
		}

		t.Logf("Sphere normals verified: %d unit vectors", len(mesh.Normals))
	})

	t.Run("ProceduralTorus", func(t *testing.T) {
		mesh := GenerateTorus(8.0, 2.5, 16, 16)

		if len(mesh.Vertices) == 0 {
			t.Error("Torus should have vertices")
		}

		if len(mesh.Indices)%3 != 0 {
			t.Error("Torus indices should be multiple of 3 (triangles)")
		}

		t.Logf("Torus generated with %d vertices and %d triangles",
			len(mesh.Vertices), len(mesh.Indices)/3)
	})

	t.Run("TorusNormals", func(t *testing.T) {
		mesh := GenerateTorus(8.0, 2.5, 16, 16)

		// Check that normals array has the correct length
		if len(mesh.Normals) != len(mesh.Vertices) {
			t.Errorf("Expected %d normals, got %d", len(mesh.Vertices), len(mesh.Normals))
		}

		// Verify that normals are normalized unit vectors
		for i, normal := range mesh.Normals {
			length := math.Sqrt(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)
			if math.Abs(length-1.0) > 0.01 {
				t.Errorf("Normal %d not normalized: length %.4f", i, length)
			}
		}

		t.Logf("Torus normals verified: %d unit vectors", len(mesh.Normals))
	})

	t.Run("CalculateNormals", func(t *testing.T) {
		// Create a simple mesh with a single triangle
		mesh := NewMesh()
		mesh.AddVertex(0, 0, 0)
		mesh.AddVertex(1, 0, 0)
		mesh.AddVertex(0, 1, 0)
		mesh.AddTriangleIndices(0, 1, 2)

		// Calculate normals
		mesh.CalculateNormals()

		// Check that normals were created
		if len(mesh.Normals) != 3 {
			t.Errorf("Expected 3 normals, got %d", len(mesh.Normals))
		}

		// All three vertices of this triangle should have the same normal (facing +Z)
		for i, normal := range mesh.Normals {
			// Check normalization
			length := math.Sqrt(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)
			if math.Abs(length-1.0) > 0.0001 {
				t.Errorf("Normal %d not normalized: length %.6f", i, length)
			}

			// The normal should point in the +Z direction (0, 0, 1)
			// since the triangle is in the XY plane
			if math.Abs(normal.X) > 0.0001 || math.Abs(normal.Y) > 0.0001 || math.Abs(normal.Z-1.0) > 0.0001 {
				t.Errorf("Normal %d direction incorrect: got (%.4f, %.4f, %.4f), expected (0, 0, 1)",
					i, normal.X, normal.Y, normal.Z)
			}
		}

		t.Logf("CalculateNormals verified for single triangle")
	})

	t.Run("CalculateNormalsSmoothing", func(t *testing.T) {
		// Create a mesh with two triangles sharing an edge to test smoothing
		mesh := NewMesh()
		mesh.AddVertex(0, 0, 0) // 0: shared vertex
		mesh.AddVertex(1, 0, 0) // 1: shared vertex
		mesh.AddVertex(0, 1, 0) // 2: triangle 1
		mesh.AddVertex(0, 0, 1) // 3: triangle 2

		// Triangle 1: (0, 1, 2) in XY plane
		mesh.AddTriangleIndices(0, 1, 2)
		// Triangle 2: (0, 3, 1) in XZ plane
		mesh.AddTriangleIndices(0, 3, 1)

		mesh.CalculateNormals()

		// Normals at vertices 0 and 1 should be averaged between the two triangles
		// They should not be pointing purely in +Z or +Y direction
		for i := 0; i < 2; i++ {
			normal := mesh.Normals[i]
			length := math.Sqrt(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)

			if math.Abs(length-1.0) > 0.0001 {
				t.Errorf("Shared vertex %d normal not normalized: length %.6f", i, length)
			}

			// Should have both Y and Z components (averaged from both triangles)
			if normal.Y < 0.1 || normal.Z < 0.1 {
				t.Errorf("Shared vertex %d normal not properly averaged: (%.4f, %.4f, %.4f)",
					i, normal.X, normal.Y, normal.Z)
			}
		}

		t.Logf("CalculateNormals smoothing verified")
	})

	t.Run("CalculateNormalsEmptyMesh", func(t *testing.T) {
		// Test edge case: empty mesh
		mesh := NewMesh()
		mesh.CalculateNormals()

		if len(mesh.Normals) != 0 {
			t.Errorf("Empty mesh should have 0 normals, got %d", len(mesh.Normals))
		}
	})

	t.Run("CalculateNormalsInvalidIndices", func(t *testing.T) {
		// Test edge case: mesh with invalid indices (out of bounds)
		mesh := NewMesh()
		mesh.AddVertex(0, 0, 0)
		mesh.AddVertex(1, 0, 0)
		mesh.AddVertex(0, 1, 0)

		// Add invalid indices
		mesh.Indices = []int{0, 1, 10} // Index 10 is out of bounds

		// Should not panic, but skip invalid triangles
		mesh.CalculateNormals()

		// Normals should still be created for all vertices (but zero for unused ones)
		if len(mesh.Normals) != 3 {
			t.Errorf("Expected 3 normals, got %d", len(mesh.Normals))
		}

		t.Logf("CalculateNormals handles invalid indices gracefully")
	})
}

// ============================================================================
// TRANSFORM TESTS
// ============================================================================

func TestTransforms(t *testing.T) {
	t.Run("Position", func(t *testing.T) {
		transform := NewTransform()
		transform.SetPosition(10, 20, 30)

		pos := transform.Position
		if pos.X != 10 || pos.Y != 20 || pos.Z != 30 {
			t.Errorf("Position incorrect: got (%.1f,%.1f,%.1f), expected (10,20,30)",
				pos.X, pos.Y, pos.Z)
		}
	})

	t.Run("Rotation", func(t *testing.T) {
		transform := NewTransform()
		transform.SetRotation(1.0, 2.0, 3.0)

		pitch, yaw, roll := transform.GetRotation()
		if math.Abs(pitch-1.0) > 0.1 || math.Abs(yaw-2.0) > 0.1 || math.Abs(roll-3.0) > 0.1 {
			t.Errorf("Rotation incorrect: got (%.1f,%.1f,%.1f), expected (1,2,3)",
				pitch, yaw, roll)
		}
	})

	t.Run("Scale", func(t *testing.T) {
		transform := NewTransform()
		transform.SetScale(2.0, 3.0, 4.0)

		scale := transform.Scale
		if scale.X != 2.0 || scale.Y != 3.0 || scale.Z != 4.0 {
			t.Errorf("Scale incorrect: got (%.1f,%.1f,%.1f), expected (2,3,4)",
				scale.X, scale.Y, scale.Z)
		}
	})

	t.Run("WorldPosition", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()

		parent := scene.CreateCube("Parent", 10, &mat)
		parent.Transform.SetPosition(10, 0, 0)

		child := scene.CreateCube("Child", 5, &mat)
		child.Transform.SetPosition(5, 0, 0)
		scene.AddNodeTo(child, parent)

		worldPos := child.Transform.GetWorldPosition()

		// Child's world position should be parent + child = 15
		expectedX := 15.0
		if math.Abs(worldPos.X-expectedX) > 0.1 {
			t.Errorf("World position X incorrect: got %.1f, expected %.1f", worldPos.X, expectedX)
		}
		t.Logf("Child world position: (%.1f, %.1f, %.1f)", worldPos.X, worldPos.Y, worldPos.Z)
	})
}

// ============================================================================
// SCENE HIERARCHY TESTS
// ============================================================================

func TestSceneHierarchy(t *testing.T) {
	t.Run("AddNode", func(t *testing.T) {
		scene := NewScene()
		node := NewSceneNode("TestNode")
		scene.AddNode(node)

		found := scene.FindNode("TestNode")
		if found != node {
			t.Error("Node not found in scene")
		}
	})

	t.Run("ParentChild", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()

		parent := scene.CreateCube("Parent", 10, &mat)
		child := scene.CreateCube("Child", 5, &mat)

		scene.AddNodeTo(child, parent)

		if child.Parent != parent {
			t.Error("Child parent not set correctly")
		}

		if len(parent.Children) != 1 || parent.Children[0] != child {
			t.Error("Parent children not set correctly")
		}
	})

	t.Run("FindByTag", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()

		cube1 := scene.CreateCube("Cube1", 10, &mat)
		cube1.AddTag("enemy")

		cube2 := scene.CreateCube("Cube2", 10, &mat)
		cube2.AddTag("enemy")

		cube3 := scene.CreateCube("Cube3", 10, &mat)
		cube3.AddTag("player")

		enemies := scene.FindNodesByTag("enemy")
		if len(enemies) != 2 {
			t.Errorf("Expected 2 enemies, found %d", len(enemies))
		}

		players := scene.FindNodesByTag("player")
		if len(players) != 1 {
			t.Errorf("Expected 1 player, found %d", len(players))
		}
	})

	t.Run("RemoveNode", func(t *testing.T) {
		scene := NewScene()
		node := NewSceneNode("ToRemove")
		scene.AddNode(node)

		scene.RemoveNode(node)

		found := scene.FindNode("ToRemove")
		if found != nil {
			t.Error("Node should be removed from scene")
		}
	})
}

// ============================================================================
// LIGHTING TESTS
// ============================================================================

func TestLighting(t *testing.T) {
	t.Run("LightCreation", func(t *testing.T) {
		light := NewLight(10, 10, 10, ColorWhite, 1.0)

		if light.Position.X != 10 || light.Position.Y != 10 || light.Position.Z != 10 {
			t.Error("Light position not set correctly")
		}

		if light.Intensity != 1.0 {
			t.Errorf("Light intensity should be 1.0, got %.1f", light.Intensity)
		}
	})

	t.Run("LightingSystem", func(t *testing.T) {
		camera := NewCamera()
		ls := NewLightingSystem(camera)

		light := NewLight(0, 10, 0, ColorWhite, 1.0)
		ls.AddLight(light)

		if len(ls.Lights) != 1 {
			t.Errorf("Expected 1 light, got %d", len(ls.Lights))
		}
	})

	t.Run("LightCalculation", func(t *testing.T) {
		camera := NewCamera()
		ls := NewLightingSystem(camera)

		light := NewLight(0, 10, 0, ColorWhite, 1.0)
		ls.AddLight(light)

		mat := NewMaterial()
		mat.DiffuseColor = ColorRed

		// Calculate lighting at origin with normal pointing up
		surfacePoint := Point{X: 0, Y: 0, Z: 0}
		normal := Point{X: 0, Y: 1, Z: 0}

		color := ls.CalculateLighting(surfacePoint, normal, &mat, 1.0)

		// Should have some color (ambient + diffuse)
		if color.R == 0 && color.G == 0 && color.B == 0 {
			t.Error("Lighting calculation should produce non-black color")
		}
		t.Logf("Lit color: R=%d G=%d B=%d", color.R, color.G, color.B)
	})
}

// ============================================================================
// MATH UTILITY TESTS
// ============================================================================

func TestMathUtils(t *testing.T) {
	t.Run("VectorNormalization", func(t *testing.T) {
		x, y, z := 3.0, 4.0, 0.0
		nx, ny, nz := normalizeVector(x, y, z)

		// Length should be 1
		length := math.Sqrt(nx*nx + ny*ny + nz*nz)
		if math.Abs(length-1.0) > 0.0001 {
			t.Errorf("Normalized vector length should be 1.0, got %.4f", length)
		}
	})

	t.Run("DotProduct", func(t *testing.T) {
		dot := dotProduct(1, 0, 0, 1, 0, 0)
		if dot != 1.0 {
			t.Errorf("Dot product of parallel vectors should be 1.0, got %.2f", dot)
		}

		dot = dotProduct(1, 0, 0, 0, 1, 0)
		if dot != 0.0 {
			t.Errorf("Dot product of perpendicular vectors should be 0.0, got %.2f", dot)
		}
	})

	t.Run("CrossProduct", func(t *testing.T) {
		// X cross Y = Z
		x, y, z := crossProduct(1, 0, 0, 0, 1, 0)
		if math.Abs(x) > 0.0001 || math.Abs(y) > 0.0001 || math.Abs(z-1) > 0.0001 {
			t.Errorf("X × Y should be Z, got (%.2f, %.2f, %.2f)", x, y, z)
		}
	})

	t.Run("Clamp", func(t *testing.T) {
		result := clamp(5.0, 0.0, 10.0)
		if result != 5.0 {
			t.Errorf("Clamp should return 5.0, got %.1f", result)
		}

		result = clamp(-5.0, 0.0, 10.0)
		if result != 0.0 {
			t.Errorf("Clamp should return 0.0, got %.1f", result)
		}

		result = clamp(15.0, 0.0, 10.0)
		if result != 10.0 {
			t.Errorf("Clamp should return 10.0, got %.1f", result)
		}
	})
}
