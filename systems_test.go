package main

import (
	"testing"
)

func TestAllSystems(t *testing.T) {
	t.Log("=== Testing Fixed Systems ===")

	// Test 1: Bounding Volumes - Basic Overlap
	t.Run("BoundingVolumes_Overlap", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		// Create two cubes that DEFINITELY overlap
		cube1 := scene.CreateCube("Cube1", 10, mat)
		cube1.Transform.SetPosition(0, 0, 0)

		cube2 := scene.CreateCube("Cube2", 10, mat)
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

	// Test 2: Bounding Volumes - No Overlap
	t.Run("BoundingVolumes_Separation", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube1 := scene.CreateCube("Cube1", 10, mat)
		cube1.Transform.SetPosition(0, 0, 0)

		cube3 := scene.CreateCube("Cube3", 10, mat)
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

	// Test 3: Bounding Volumes with Rotation
	t.Run("BoundingVolumes_Rotation", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube4 := scene.CreateCube("Cube4", 10, mat)
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

	// Test 4: Raycasting - Direct Hit
	t.Run("Raycasting_DirectHit", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube := scene.CreateCube("Target", 10, mat)
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

	// Test 5: Raycasting - Miss
	t.Run("Raycasting_Miss", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube := scene.CreateCube("Target", 10, mat)
		cube.Transform.SetPosition(0, 0, 0)

		ray := NewRay(Point{X: 30, Y: 0, Z: -50}, Point{X: 0, Y: 0, Z: 1})
		hit := scene.Raycast(ray, 100.0)

		if hit.Hit {
			t.Error("Ray should miss the cube")
		}
	})

	// Test 6: Raycasting - Transformed Object
	t.Run("Raycasting_TransformedObject", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		cube := scene.CreateCube("TransformedTarget", 10, mat)
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

	// Test 7: Sphere Bounds
	t.Run("SphereBounds", func(t *testing.T) {
		scene := NewScene()
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue

		sphere := scene.CreateSphere("Sphere", 5, 16, 16, mat)
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
}
