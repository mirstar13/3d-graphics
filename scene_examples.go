package main

import (
	"fmt"
	"math"
)

// ============================================================================
// DEMO CATEGORIES
// ============================================================================
// 1. Basic Geometry       - Fundamental shapes and primitives
// 2. Mesh & Generators    - Complex meshes using indexed geometry
// 3. Lighting Showcase    - All lighting scenarios and effects
// 4. Material Showcase    - Different material types and properties
// 5. Transform Hierarchy  - Scene graph, articulated objects
// 6. LOD System          - Level of detail demonstrations
// 7. Spatial Partitioning - Octree and BVH visualization
// 8. Collision & Physics  - Bounding volumes, raycasting
// 9. Advanced Rendering   - AA, clipping, special effects
// 10. Performance Test    - Stress testing with many objects
// ============================================================================

// ============================================================================
// DEMO 1: BASIC GEOMETRY - Fundamental Shapes
// ============================================================================

func BasicGeometryDemo(scene *Scene) {
	fmt.Println("=== Basic Geometry Demo ===")
	fmt.Println("Showcasing: Points, Lines, Triangles, Quads, Circles")

	// Single Point (rendered as small octahedron)
	point := NewPoint(0, 10, 0)
	pointNode := NewSceneNodeWithObject("Point", point)
	scene.AddNode(pointNode)

	// Line
	line := NewLine(Point{X: -15, Y: 0, Z: 0}, Point{X: -15, Y: 15, Z: 0})
	lineNode := NewSceneNodeWithObject("Line", line)
	scene.AddNode(lineNode)

	// Triangle
	triMat := NewMaterial()
	triMat.DiffuseColor = ColorRed
	tri := NewTriangle(
		Point{X: -5, Y: 0, Z: 0},
		Point{X: 0, Y: 10, Z: 0},
		Point{X: 5, Y: 0, Z: 0},
		'x',
	)
	tri.SetMaterial(&triMat)
	triNode := NewSceneNodeWithObject("Triangle", tri)
	scene.AddNode(triNode)

	// Quad
	quadMat := NewMaterial()
	quadMat.DiffuseColor = ColorGreen
	quad := NewQuad(
		Point{X: 10, Y: 0, Z: 0},
		Point{X: 15, Y: 0, Z: 0},
		Point{X: 15, Y: 8, Z: 0},
		Point{X: 10, Y: 8, Z: 0},
	)
	quad.SetMaterial(&quadMat)
	quadNode := NewSceneNodeWithObject("Quad", quad)
	scene.AddNode(quadNode)

	// Circle
	circle := NewCircle(0, -10, 0, 5, 24)
	circleNode := NewSceneNodeWithObject("Circle", circle)
	scene.AddNode(circleNode)

	fmt.Println("Created: 1 point, 1 line, 1 triangle, 1 quad, 1 circle")
}

func AnimateBasicGeometry(scene *Scene) {
	if tri := scene.FindNode("Triangle"); tri != nil {
		tri.RotateLocal(0, 0.02, 0)
	}
	if quad := scene.FindNode("Quad"); quad != nil {
		quad.RotateLocal(0, -0.02, 0)
	}
	if circle := scene.FindNode("Circle"); circle != nil {
		circle.RotateLocal(0.02, 0, 0)
	}
}

// ============================================================================
// DEMO 2: MESH & GENERATORS - Indexed Geometry
// ============================================================================

func MeshGeneratorsDemo(scene *Scene) {
	fmt.Println("=== Mesh Generators Demo ===")
	fmt.Println("Showcasing: Cube, Sphere, Torus with indexed geometry")

	// Cube (using indexed geometry)
	cubeMat := NewMaterial()
	cubeMat.DiffuseColor = Color{200, 100, 100}
	cube := scene.CreateCube("Cube", 8, &cubeMat)
	cube.Transform.SetPosition(-20, 0, 0)
	cube.AddTag("rotating")

	// Sphere (procedurally generated)
	sphereMat := NewMaterial()
	sphereMat.DiffuseColor = Color{100, 200, 100}
	sphere := scene.CreateSphere("Sphere", 6, 24, 24, &sphereMat)
	sphere.Transform.SetPosition(0, 0, 0)
	sphere.AddTag("rotating")

	// Torus (procedurally generated)
	torusMesh := GenerateTorus(8.0, 2.5, 32, 16)
	torusMat := NewMaterial()
	torusMat.DiffuseColor = Color{100, 100, 200}
	torusMesh.Material = &torusMat
	torus := NewSceneNodeWithObject("Torus", torusMesh)
	torus.Transform.SetPosition(20, 0, 0)
	scene.AddNode(torus)
	torus.AddTag("rotating")

	fmt.Printf("Cube: %d vertices, %d triangles\n", 8, 12)
	fmt.Printf("Sphere: %d vertices, %d triangles\n", len(sphere.Object.(*Mesh).Vertices), len(sphere.Object.(*Mesh).Indices)/3)
	fmt.Printf("Torus: %d vertices, %d triangles\n", len(torusMesh.Vertices), len(torusMesh.Indices)/3)
}

func AnimateMeshGenerators(scene *Scene) {
	rotating := scene.FindNodesByTag("rotating")
	for i, obj := range rotating {
		// Different rotation speeds
		switch i % 3 {
		case 0:
			obj.RotateLocal(0.02, 0, 0)
		case 1:
			obj.RotateLocal(0, 0.02, 0)
		case 2:
			obj.RotateLocal(0.01, 0.01, 0)
		}
	}
}

// ============================================================================
// DEMO 3: LIGHTING SHOWCASE - All Lighting Scenarios
// ============================================================================

func LightingShowcaseDemo(scene *Scene) {
	fmt.Println("=== Lighting Showcase Demo ===")
	fmt.Println("Showcasing: Multiple lighting scenarios with IMaterial")

	positions := []float64{-40, -20, 0, 20, 40}
	names := []string{"Ambient", "Directional", "3-Point", "Colored", "Dynamic"}

	for i, pos := range positions {
		// Create material using IMaterial interface
		mat := NewMaterial()
		mat.DiffuseColor = Color{180, 180, 180}
		mat.SpecularColor = ColorWhite
		mat.Shininess = 64
		mat.SpecularStrength = 0.8
		sphere := scene.CreateSphere(names[i], 6, 24, 24, &mat)
		sphere.Transform.SetPosition(pos, 0, 0)
		sphere.AddTag(fmt.Sprintf("lighting-%d", i))
	}

	fmt.Println("Each sphere demonstrates different lighting with proper material system")
}

func AnimateLightingShowcase(scene *Scene, time float64) {
	// Gentle rotation to show lighting from all angles
	for i := 0; i < 5; i++ {
		if sphere := scene.FindNode([]string{"Ambient", "Directional", "3-Point", "Colored", "Dynamic"}[i]); sphere != nil {
			sphere.RotateLocal(0, 0.015, 0)
		}
	}
}

// ============================================================================
// DEMO 4: MATERIAL SHOWCASE - Different Material Types
// ============================================================================

func MaterialShowcaseDemo(scene *Scene) {
	fmt.Println("=== Material Showcase Demo ===")
	fmt.Println("Showcasing: IMaterial system with various material types")

	positions := []float64{-30, -15, 0, 15, 30}

	// 1. Matte Material (Low Specular)
	mat1 := NewMaterial()
	mat1.DiffuseColor = ColorRed
	mat1.Shininess = 4
	mat1.SpecularStrength = 0.1
	sphere1 := scene.CreateSphere("Matte", 6, 24, 24, &mat1)
	sphere1.Transform.SetPosition(positions[0], 0, 0)

	// 2. Medium Specular
	mat2 := NewMaterial()
	mat2.DiffuseColor = ColorGreen
	mat2.Shininess = 32
	mat2.SpecularStrength = 0.5
	sphere2 := scene.CreateSphere("Medium", 6, 24, 24, &mat2)
	sphere2.Transform.SetPosition(positions[1], 0, 0)

	// 3. High Specular (Shiny)
	mat3 := NewMaterial()
	mat3.DiffuseColor = ColorBlue
	mat3.Shininess = 128
	mat3.SpecularStrength = 1.0
	sphere3 := scene.CreateSphere("Shiny", 6, 24, 24, &mat3)
	sphere3.Transform.SetPosition(positions[2], 0, 0)

	// 4. Wireframe Material
	mat4 := NewMaterial()
	mat4.Wireframe = true
	mat4.WireframeColor = ColorYellow
	sphere4 := scene.CreateSphere("Wireframe", 6, 16, 16, &mat4)
	sphere4.Transform.SetPosition(positions[3], 0, 0)

	// 5. PBR Material
	pbrMat := NewPBRMaterial()
	pbrMat.Albedo = ColorMagenta
	pbrMat.Metallic = 0.8
	pbrMat.Roughness = 0.2

	// Create sphere with PBR - note we need to convert PBR params to basic material for now
	mat5 := pbrMat
	sphere5 := scene.CreateSphere("PBR", 6, 24, 24, mat5)
	sphere5.Transform.SetPosition(positions[4], 0, 0)

	fmt.Println("Created 5 spheres demonstrating IMaterial system")
}

func AnimateMaterialShowcase(scene *Scene) {
	materials := []string{"Matte", "Medium", "Shiny", "Wireframe", "Solid", "Overlay"}
	for _, name := range materials {
		if sphere := scene.FindNode(name); sphere != nil {
			sphere.RotateLocal(0.01, 0.02, 0)
		}
	}
}

// ============================================================================
// DEMO 5: TRANSFORM HIERARCHY - Scene Graph & Articulated Objects
// ============================================================================

func TransformHierarchyDemo(scene *Scene) {
	fmt.Println("=== Transform Hierarchy Demo ===")
	fmt.Println("Showcasing: Parent-child transforms, articulated robot arm")

	// Create a simple solar system to demonstrate hierarchy

	// Sun (root)
	sunMat := NewMaterial()
	sunMat.DiffuseColor = ColorYellow
	sun := scene.CreateSphere("Sun", 5, 20, 20, &sunMat)
	sun.Transform.SetPosition(0, 0, 0)

	// Earth (child of sun)
	earthMat := NewMaterial()
	earthMat.DiffuseColor = ColorBlue
	earth := scene.CreateSphere("Earth", 3, 16, 16, &earthMat)
	earth.Transform.SetPosition(20, 0, 0)
	scene.AddNodeTo(earth, sun) // Earth is child of sun

	// Moon (child of earth)
	moonMat := NewMaterial()
	moonMat.DiffuseColor = Color{200, 200, 200}
	moon := scene.CreateSphere("Moon", 1, 12, 12, &moonMat)
	moon.Transform.SetPosition(5, 0, 0)
	scene.AddNodeTo(moon, earth) // Moon is child of earth

	// Robot arm on the side
	baseMat := NewMaterial()
	baseMat.DiffuseColor = Color{100, 100, 100}
	base := scene.CreateCube("RobotBase", 4, &baseMat)
	base.Transform.SetPosition(40, -8, 0)

	arm1Mat := NewMaterial()
	arm1Mat.DiffuseColor = ColorRed
	arm1 := scene.CreateCube("RobotArm1", 2, &arm1Mat)
	arm1.Transform.SetPosition(0, 6, 0)
	arm1.Transform.SetScale(0.5, 3, 0.5)
	scene.AddNodeTo(arm1, base)

	arm2Mat := NewMaterial()
	arm2Mat.DiffuseColor = ColorGreen
	arm2 := scene.CreateCube("RobotArm2", 2, &arm2Mat)
	arm2.Transform.SetPosition(0, 6, 0)
	arm2.Transform.SetScale(0.4, 2.5, 0.4)
	scene.AddNodeTo(arm2, arm1)

	fmt.Println("Created hierarchical solar system and robot arm")
}

func AnimateTransformHierarchy(scene *Scene, time float64) {
	// Rotate sun (affects entire solar system)
	if sun := scene.FindNode("Sun"); sun != nil {
		sun.RotateLocal(0, 0.01, 0)
	}

	// Earth orbits and rotates
	if earth := scene.FindNode("Earth"); earth != nil {
		earth.RotateLocal(0, 0.03, 0)
	}

	// Moon orbits earth
	if moon := scene.FindNode("Moon"); moon != nil {
		moon.RotateLocal(0, 0.05, 0)
	}

	// Animate robot arm
	if base := scene.FindNode("RobotBase"); base != nil {
		base.RotateLocal(0, 0.02, 0)
	}
	if arm1 := scene.FindNode("RobotArm1"); arm1 != nil {
		angle := math.Sin(time*0.5) * 0.5
		arm1.Transform.Rotation = QuaternionFromEuler(0, 0, angle)
		arm1.Transform.MarkDirty()
	}
	if arm2 := scene.FindNode("RobotArm2"); arm2 != nil {
		angle := math.Sin(time*0.7) * 0.6
		arm2.Transform.Rotation = QuaternionFromEuler(0, 0, angle)
		arm2.Transform.MarkDirty()
	}
}

// ============================================================================
// DEMO 6: LOD SYSTEM - Level of Detail
// ============================================================================

func LODSystemDemo(scene *Scene) {
	fmt.Println("=== LOD System Demo ===")
	fmt.Println("Showcasing: Automatic LOD switching based on camera distance")

	// Create a line of spheres at different distances with LOD
	numObjects := 8
	startZ := -120.0
	spacing := 35.0

	for i := 0; i < numObjects; i++ {
		// Create high-detail mesh
		highDetail := GenerateSphere(6.0, 24, 24)
		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(100 + i*15),
			G: uint8(150),
			B: uint8(200 - i*10),
		}
		highDetail.Material = &mat

		// Create LOD group
		lodGroup := NewLODGroup()
		lodGroup.AddLOD(highDetail, 40.0)

		medDetail := SimplifyMesh(highDetail, 0.6)
		medDetail.Material = &mat
		lodGroup.AddLOD(medDetail, 80.0)

		lowDetail := SimplifyMesh(highDetail, 0.3)
		lowDetail.Material = &mat
		lodGroup.AddLOD(lowDetail, 150.0)

		name := fmt.Sprintf("LOD_%d", i)
		node := scene.CreateEmpty(name)
		node.SetLODGroup(lodGroup)
		node.Transform.SetPosition(0, 0, startZ+float64(i)*spacing)
		node.AddTag("lod-object")
	}

	fmt.Printf("Created %d objects with 3 LOD levels each\n", numObjects)
	fmt.Println("Move camera to see LOD transitions")
}

func AnimateLODSystem(scene *Scene) {
	// Gentle rotation to see LOD quality
	lods := scene.FindNodesByTag("lod-object")
	for _, obj := range lods {
		obj.RotateLocal(0.01, 0.015, 0)
	}
	scene.UpdateLODs()
}

// ============================================================================
// DEMO 7: SPATIAL PARTITIONING - Octree & BVH
// ============================================================================

func SpatialPartitioningDemo(scene *Scene) {
	fmt.Println("=== Spatial Partitioning Demo ===")
	fmt.Println("Showcasing: Octree and BVH for efficient spatial queries")

	// Create randomly distributed objects
	numObjects := 50
	for i := 0; i < numObjects; i++ {
		// Pseudo-random position
		angle := float64(i) * 2.4
		radius := 20.0 + float64(i%5)*8.0
		height := math.Sin(float64(i)*0.7) * 20.0

		x := radius * math.Cos(angle)
		z := radius * math.Sin(angle)

		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(100 + i*3),
			G: uint8(150 + (i%10)*10),
			B: 200,
		}

		name := fmt.Sprintf("Spatial_%d", i)
		obj := scene.CreateCube(name, 4, &mat)
		obj.Transform.SetPosition(x, height, z)
		obj.AddTag("spatial")
	}

	// Create query sphere (visualizes spatial query)
	queryMat := NewMaterial()
	queryMat.DiffuseColor = ColorYellow
	queryMat.Wireframe = true
	query := scene.CreateSphere("QuerySphere", 15, 16, 16, &queryMat)
	query.Transform.SetPosition(0, 0, 0)

	fmt.Printf("Created %d objects for spatial queries\n", numObjects)
}

func AnimateSpatialPartitioning(scene *Scene, time float64) {
	// Move query sphere in a pattern
	if query := scene.FindNode("QuerySphere"); query != nil {
		query.Transform.Position.X = math.Cos(time*0.5) * 30.0
		query.Transform.Position.Z = math.Sin(time*0.5) * 30.0
		query.Transform.Position.Y = math.Sin(time*0.8) * 15.0
	}

	// Gentle rotation of objects
	spatials := scene.FindNodesByTag("spatial")
	for i, obj := range spatials {
		if i%3 == 0 {
			obj.RotateLocal(0.01, 0.01, 0)
		}
	}
}

// ============================================================================
// DEMO 8: COLLISION & PHYSICS - Bounding Volumes & Raycasting
// ============================================================================

func CollisionPhysicsDemo(scene *Scene) {
	fmt.Println("=== Collision & Physics Demo ===")
	fmt.Println("Showcasing: AABB, OBB, raycasting, line-of-sight")

	// Create moving probe with proper bounds
	probeMat := NewMaterial()
	probeMat.DiffuseColor = ColorYellow
	probeMat.Shininess = 64
	probe := scene.CreateCube("Probe", 5, &probeMat)
	probe.Transform.SetPosition(0, 0, 0)
	probe.AddTag("probe")

	// Create static obstacles with proper materials
	positions := []struct{ x, y, z float64 }{
		{-20, 0, -15}, {20, 0, -15},
		{-20, 0, 15}, {20, 0, 15},
		{0, 0, -25}, {0, 0, 25},
	}

	for i, pos := range positions {
		mat := NewMaterial()
		mat.DiffuseColor = Color{100, 150, 200}
		mat.Shininess = 32
		name := fmt.Sprintf("Obstacle_%d", i)
		obstacle := scene.CreateCube(name, 7, &mat)
		obstacle.Transform.SetPosition(pos.x, pos.y, pos.z)
		obstacle.Transform.SetRotation(float64(i)*0.2, float64(i)*0.3, 0)
		obstacle.AddTag("obstacle")
	}

	// Raycast target
	targetMat := NewMaterial()
	targetMat.DiffuseColor = ColorRed
	targetMat.Shininess = 64
	target := scene.CreateSphere("Target", 3, 16, 16, &targetMat)
	target.Transform.SetPosition(15, 5, 0)
	target.AddTag("target")

	fmt.Println("Probe detects collisions and casts rays to target")
	fmt.Println("Bounding volumes are properly calculated in world space")
}

func AnimateCollisionPhysics(scene *Scene, time float64) {
	// Move probe in circular pattern
	if probe := scene.FindNode("Probe"); probe != nil {
		probe.Transform.Position.X = math.Sin(time*0.6) * 20.0
		probe.Transform.Position.Z = math.Cos(time*0.6) * 20.0
		probe.RotateLocal(0.03, 0.02, 0.01)

		// Check collisions with obstacles
		probeBounds := scene.computeNodeBounds(probe)
		if probeBounds != nil {
			obstacles := scene.FindNodesByTag("obstacle")
			colliding := false

			for _, obs := range obstacles {
				obsBounds := scene.computeNodeBounds(obs)
				if obsBounds != nil && probeBounds.IntersectsAABB(obsBounds) {
					colliding = true
					break
				}
			}

			// Change probe color based on collision
			if mesh, ok := probe.Object.(*Mesh); ok {
				if mat, ok := mesh.Material.(*Material); ok {
					if colliding {
						mat.DiffuseColor = ColorRed
					} else {
						mat.DiffuseColor = ColorYellow
					}
				}
			}
		}

		// Raycast to target
		if target := scene.FindNode("Target"); target != nil {
			probePos := probe.Transform.GetWorldPosition()
			targetPos := target.Transform.GetWorldPosition()

			direction := Point{
				X: targetPos.X - probePos.X,
				Y: targetPos.Y - probePos.Y,
				Z: targetPos.Z - probePos.Z,
			}

			ray := NewRay(probePos, direction)
			hit := scene.Raycast(ray, 100.0)

			if hit.Hit {
				// Visual feedback - change target color if ray hits it
				if hit.Node == target {
					if mesh, ok := target.Object.(*Mesh); ok {
						if mat, ok := mesh.Material.(*Material); ok {
							mat.DiffuseColor = ColorGreen
						}
					}
				}
			}
		}
	}

	// Gentle rotation of obstacles
	obstacles := scene.FindNodesByTag("obstacle")
	for _, obs := range obstacles {
		obs.RotateLocal(0.005, 0.01, 0)
	}

	// Move target
	if target := scene.FindNode("Target"); target != nil {
		target.Transform.Position.X = math.Cos(time*0.4) * 18.0
		target.Transform.Position.Z = math.Sin(time*0.4) * 18.0
	}
}

// ============================================================================
// DEMO 9: ADVANCED RENDERING - AA, Clipping, Effects
// ============================================================================

func AdvancedRenderingDemo(scene *Scene) {
	fmt.Println("=== Advanced Rendering Demo ===")
	fmt.Println("Showcasing: Anti-aliasing (set in config), frustum culling, clipping")

	// Create objects at various distances to test clipping
	distances := []float64{10, 30, 60, 100, 150}

	for i, dist := range distances {
		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(255 - i*40),
			G: uint8(100 + i*30),
			B: uint8(100 + i*20),
		}

		name := fmt.Sprintf("Clip_%d", i)
		cube := scene.CreateCube(name, 8, &mat)
		cube.Transform.SetPosition(0, 0, -dist)
		cube.AddTag("clipping")
	}

	// Add some objects at extreme angles for frustum culling
	for i := 0; i < 8; i++ {
		angle := float64(i) * math.Pi / 4
		mat := NewMaterial()
		mat.DiffuseColor = ColorCyan

		name := fmt.Sprintf("Frustum_%d", i)
		sphere := scene.CreateSphere(name, 5, 16, 16, &mat)
		sphere.Transform.SetPosition(
			math.Cos(angle)*80,
			0,
			math.Sin(angle)*80,
		)
		sphere.AddTag("frustum")
	}

	fmt.Println("Objects at various distances demonstrate clipping")
	fmt.Println("Rotate camera to see frustum culling in action")
}

func AnimateAdvancedRendering(scene *Scene) {
	// Rotate clipping test objects
	clipping := scene.FindNodesByTag("clipping")
	for _, obj := range clipping {
		obj.RotateLocal(0.02, 0.015, 0.01)
	}

	// Rotate frustum test objects in place
	frustum := scene.FindNodesByTag("frustum")
	for _, obj := range frustum {
		obj.RotateLocal(0.01, 0.02, 0)
	}
}

// ============================================================================
// DEMO 10: PERFORMANCE TEST - Stress Testing
// ============================================================================

func PerformanceTestDemo(scene *Scene) {
	fmt.Println("=== Performance Test Demo ===")
	fmt.Println("Showcasing: Rendering many objects with LOD optimization")

	// Create a large grid of LOD objects
	gridSize := 8
	spacing := 18.0
	offset := -float64(gridSize-1) * spacing / 2

	objectCount := 0
	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			y := (x + z) % 3

			baseMesh := GenerateSphere(4.0, 20, 20)
			mat := NewMaterial()
			mat.DiffuseColor = Color{
				R: uint8(80 + (x * 20)),
				G: uint8(80 + (z * 20)),
				B: uint8(150 + (y * 30)),
			}
			baseMesh.Material = &mat

			lodGroup := NewLODGroup()
			lodGroup.AddLOD(baseMesh, 35.0)

			medMesh := SimplifyMesh(baseMesh, 0.6)
			medMesh.Material = &mat
			lodGroup.AddLOD(medMesh, 70.0)

			lowMesh := SimplifyMesh(baseMesh, 0.3)
			lowMesh.Material = &mat
			lodGroup.AddLOD(lowMesh, 140.0)

			name := fmt.Sprintf("Perf_%d_%d", x, z)
			node := scene.CreateEmpty(name)
			node.SetLODGroup(lodGroup)
			node.Transform.SetPosition(
				offset+float64(x)*spacing,
				float64(y)*8.0,
				offset+float64(z)*spacing,
			)
			node.AddTag("performance")
			objectCount++
		}
	}

	fmt.Printf("Created %d objects for performance testing\n", objectCount)
	fmt.Println("LOD system automatically optimizes rendering")
}

func AnimatePerformanceTest(scene *Scene) {
	// Minimal animation to maintain performance
	perf := scene.FindNodesByTag("performance")
	for i, obj := range perf {
		if i%10 == 0 {
			obj.RotateLocal(0.005, 0.008, 0)
		}
	}
	scene.UpdateLODs()
}
