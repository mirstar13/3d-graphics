package main

import (
	"bufio"
	"fmt"
	"math"
)

// SolarSystemDemo creates a solar system with orbiting planets
func SolarSystemDemo(scene *Scene) {
	// Sun at center
	sun := scene.CreateEmpty("Sun")
	sun.Transform.SetPosition(0, 0, 0)

	sunSphere := scene.CreateSphere("SunSphere", 8, 8, 8, NewMaterial())
	sunSphere.Object.(*Mesh).Quads[0].Material.DiffuseColor = ColorYellow
	scene.AddNodeTo(sunSphere, sun)

	// Earth
	earth := scene.CreateEmpty("Earth")
	earth.Transform.SetPosition(40, 0, 0)

	earthSphere := scene.CreateSphere("EarthSphere", 4, 8, 8, NewMaterial())
	earthSphere.Object.(*Mesh).Quads[0].Material.DiffuseColor = ColorBlue
	scene.AddNodeTo(earthSphere, earth)
	scene.AddNodeTo(earth, sun)

	// Moon
	moon := scene.CreateEmpty("Moon")
	moon.Transform.SetPosition(12, 0, 0)

	moonSphere := scene.CreateSphere("MoonSphere", 2, 6, 6, NewMaterial())
	moonSphere.Object.(*Mesh).Quads[0].Material.DiffuseColor = Color{200, 200, 200}
	scene.AddNodeTo(moonSphere, moon)
	scene.AddNodeTo(moon, earth)

	// Mars
	mars := scene.CreateEmpty("Mars")
	mars.Transform.SetPosition(60, 0, 0)

	marsSphere := scene.CreateSphere("MarsSphere", 3, 8, 8, NewMaterial())
	marsSphere.Object.(*Mesh).Quads[0].Material.DiffuseColor = ColorRed
	scene.AddNodeTo(marsSphere, mars)
	scene.AddNodeTo(mars, sun)

	// Tag for animation
	sun.AddTag("sun")
	earth.AddTag("planet")
	mars.AddTag("planet")
	moon.AddTag("moon")
}

// AnimateSolarSystem animates the solar system
func AnimateSolarSystem(scene *Scene) {
	// Rotate sun
	sun := scene.FindNode("Sun")
	if sun != nil {
		sun.RotateLocal(0, 0.005, 0)
	}

	// Rotate planets around sun (by rotating their parent)
	// Earth orbits sun
	earth := scene.FindNode("Earth")
	if earth != nil {
		earth.RotateLocal(0, 0.02, 0)
	}

	// Mars orbits sun slower
	mars := scene.FindNode("Mars")
	if mars != nil {
		mars.RotateLocal(0, 0.015, 0)
	}

	// Moon orbits earth
	moon := scene.FindNode("Moon")
	if moon != nil {
		moon.RotateLocal(0, 0.04, 0)
	}
}

// RobotArmDemo creates a robotic arm with multiple joints
func RobotArmDemo(scene *Scene, material Material) {
	// Base
	base := scene.CreateCube("Base", 5, material)
	base.Transform.SetPosition(0, -10, 0)
	base.Transform.SetScale(3, 1, 3)

	// Shoulder joint
	shoulder := scene.CreateCube("Shoulder", 3, material)
	shoulder.Transform.SetPosition(0, 8, 0)
	scene.AddNodeTo(shoulder, base)

	// Upper arm
	upperArm := scene.CreateCube("UpperArm", 2, material)
	upperArm.Transform.SetPosition(0, 12, 0)
	upperArm.Transform.SetScale(0.8, 3, 0.8)
	scene.AddNodeTo(upperArm, shoulder)

	// Elbow
	elbow := scene.CreateCube("Elbow", 2, material)
	elbow.Transform.SetPosition(0, 8, 0)
	scene.AddNodeTo(elbow, upperArm)

	// Lower arm
	lowerArm := scene.CreateCube("LowerArm", 2, material)
	lowerArm.Transform.SetPosition(0, 10, 0)
	lowerArm.Transform.SetScale(0.7, 3, 0.7)
	scene.AddNodeTo(lowerArm, elbow)

	// Wrist
	wrist := scene.CreateCube("Wrist", 1.5, material)
	wrist.Transform.SetPosition(0, 6, 0)
	scene.AddNodeTo(wrist, lowerArm)

	// Hand
	hand := scene.CreateCube("Hand", 2, material)
	hand.Transform.SetPosition(0, 4, 0)
	hand.Transform.SetScale(1.5, 0.5, 0.8)
	scene.AddNodeTo(hand, wrist)

	// Tag for animation
	shoulder.AddTag("robot-joint")
	elbow.AddTag("robot-joint")
	wrist.AddTag("robot-joint")
}

// AnimateRobotArm animates the robotic arm
func AnimateRobotArm(scene *Scene, time float64) {
	shoulder := scene.FindNode("Shoulder")
	if shoulder != nil {
		shoulder.Transform.SetRotation(0, time*0.5, 0)
	}

	elbow := scene.FindNode("Elbow")
	if elbow != nil {
		elbow.Transform.SetRotation(0, 0, math.Sin(time)*0.5)
	}

	wrist := scene.FindNode("Wrist")
	if wrist != nil {
		wrist.Transform.SetRotation(math.Sin(time*2)*0.3, 0, 0)
	}
}

// SpinningCubesDemo creates a grid of spinning cubes
func SpinningCubesDemo(scene *Scene, material Material) {
	container := scene.CreateEmpty("CubeGrid")

	gridSize := 3
	spacing := 20.0

	for x := 0; x < gridSize; x++ {
		for y := 0; y < gridSize; y++ {
			for z := 0; z < gridSize; z++ {
				name := fmt.Sprintf("Cube_%d_%d_%d", x, y, z)
				cube := scene.CreateCube(name, 4, material)

				posX := (float64(x) - float64(gridSize)/2) * spacing
				posY := (float64(y) - float64(gridSize)/2) * spacing
				posZ := (float64(z) - float64(gridSize)/2) * spacing

				cube.Transform.SetPosition(posX, posY, posZ)
				cube.AddTag("spinning")

				scene.AddNodeTo(cube, container)
			}
		}
	}
}

// AnimateSpinningCubes animates the cube grid
func AnimateSpinningCubes(scene *Scene) {
	cubes := scene.FindNodesByTag("spinning")
	for i, cube := range cubes {
		// Each cube rotates at different speed
		speed := 0.01 + float64(i)*0.001
		cube.RotateLocal(speed, speed*0.7, speed*0.5)
	}
}

// OrbitingObjectsDemo creates objects orbiting around a center
func OrbitingObjectsDemo(scene *Scene, material Material) {
	center := scene.CreateEmpty("OrbitCenter")
	center.Transform.SetPosition(0, 0, 0)

	numObjects := 8
	for i := 0; i < numObjects; i++ {
		// Create orbit container at different radius
		orbitContainer := scene.CreateEmpty(fmt.Sprintf("Orbit_%d", i))
		radius := 20.0 + float64(i)*5.0
		angle := (float64(i) / float64(numObjects)) * 2 * math.Pi

		orbitContainer.Transform.SetPosition(
			radius*math.Cos(angle),
			0,
			radius*math.Sin(angle),
		)
		scene.AddNodeTo(orbitContainer, center)

		// Create object
		obj := scene.CreateCube(fmt.Sprintf("Object_%d", i), 3, material)
		obj.AddTag("orbiting")
		scene.AddNodeTo(obj, orbitContainer)
	}

	center.AddTag("orbit-center")
}

// AnimateOrbitingObjects animates orbiting objects
func AnimateOrbitingObjects(scene *Scene) {
	center := scene.FindNode("OrbitCenter")
	if center != nil {
		// Rotate center -> all objects orbit
		center.RotateLocal(0, 0.02, 0)
	}

	// Each object also spins
	objects := scene.FindNodesByTag("orbiting")
	for _, obj := range objects {
		obj.RotateLocal(0.03, 0.02, 0.01)
	}
}

// WaveGridDemo creates a grid that waves up and down
func WaveGridDemo(scene *Scene, material Material) {
	gridSize := 10
	spacing := 4.0

	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			name := fmt.Sprintf("WaveCube_%d_%d", x, z)
			cube := scene.CreateCube(name, 1.5, material)

			posX := (float64(x) - float64(gridSize)/2) * spacing
			posZ := (float64(z) - float64(gridSize)/2) * spacing

			cube.Transform.SetPosition(posX, 0, posZ)
			cube.AddTag("wave")

			// Store grid position in metadata (hacky but works for demo)
			cube.Tags = append(cube.Tags, fmt.Sprintf("gridX:%d", x))
			cube.Tags = append(cube.Tags, fmt.Sprintf("gridZ:%d", z))
		}
	}
}

// AnimateWaveGrid animates the wave grid
func AnimateWaveGrid(scene *Scene, time float64) {
	waveCubes := scene.FindNodesByTag("wave")

	for _, cube := range waveCubes {
		// Extract grid position from tags (hacky but works)
		var gridX, gridZ int
		fmt.Sscanf(cube.Name, "WaveCube_%d_%d", &gridX, &gridZ)

		// Calculate wave height
		wave := math.Sin(time+float64(gridX)*0.5) * math.Cos(time*0.7+float64(gridZ)*0.5)
		height := wave * 10.0

		pos := cube.Transform.Position
		cube.Transform.SetPosition(pos.X, height, pos.Z)
	}
}

// HelixDemo creates a helix structure
func HelixDemo(scene *Scene, material Material) {
	numCubes := 20
	radius := 15.0
	height := 40.0

	for i := 0; i < numCubes; i++ {
		t := float64(i) / float64(numCubes)
		angle := t * 4 * math.Pi

		x := radius * math.Cos(angle)
		y := (t - 0.5) * height
		z := radius * math.Sin(angle)

		cube := scene.CreateCube(fmt.Sprintf("HelixCube_%d", i), 2, material)
		cube.Transform.SetPosition(x, y, z)
		cube.AddTag("helix")
	}
}

// AnimateHelix animates the helix
func AnimateHelix(scene *Scene, time float64) {
	cubes := scene.FindNodesByTag("helix")

	for i, cube := range cubes {
		// Rotate helix
		baseAngle := (float64(i) / float64(len(cubes))) * 4 * math.Pi
		angle := baseAngle + time

		radius := 15.0
		height := 40.0
		t := float64(i) / float64(len(cubes))

		x := radius * math.Cos(angle)
		y := (t - 0.5) * height
		z := radius * math.Sin(angle)

		cube.Transform.SetPosition(x, y, z)
		cube.RotateLocal(0.05, 0.03, 0.02)
	}
}

func WireframeDemo(scene *Scene, material Material) {
	// Solid sphere in center
	solidMat := material
	solidMat.DiffuseColor = ColorBlue
	sphere := scene.CreateSphere("SolidSphere", 8, 12, 12, solidMat)
	sphere.Transform.SetPosition(0, 0, 20)

	// Wireframe cube around it
	wireMat := NewWireframeMaterial(ColorYellow)
	cube1 := scene.CreateCube("WireCube1", 12, wireMat)
	cube1.Transform.SetPosition(0, 0, 20)
	cube1.AddTag("rotating")

	// Wireframe sphere offset
	wireMat2 := NewWireframeMaterial(ColorCyan)
	wireframeSphere := scene.CreateSphere("WireSphere", 6, 8, 10, wireMat2)
	wireframeSphere.Transform.SetPosition(-25, 0, 20)
	wireframeSphere.AddTag("rotating")

	// Solid cube offset
	solidMat2 := material
	solidMat2.DiffuseColor = ColorRed
	solidCube := scene.CreateCube("SolidCube", 6, solidMat2)
	solidCube.Transform.SetPosition(25, 0, 20)
	solidCube.AddTag("rotating")

	// Mixed mode - solid with wireframe overlay
	solidMat3 := material
	solidMat3.DiffuseColor = ColorGreen
	mixedSolid := scene.CreateCube("MixedSolid", 5, solidMat3)
	mixedSolid.Transform.SetPosition(0, 20, 20)
	mixedSolid.AddTag("rotating")

	wireMat3 := NewWireframeMaterial(ColorWhite)
	mixedWire := scene.CreateCube("MixedWire", 5.2, wireMat3)
	scene.AddNodeTo(mixedWire, mixedSolid)
}

func AnimateWireframe(scene *Scene, time float64) {
	rotating := scene.FindNodesByTag("rotating")
	for _, obj := range rotating {
		obj.RotateLocal(0.02, 0.03, 0.01)
	}
}

// AdvancedSystemsDemo showcases bounding volumes, raycasting, and LOD
func AdvancedSystemsDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = Color{100, 150, 200}

	// Create a grid of objects with LOD
	gridSize := 5
	spacing := 40.0

	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			name := fmt.Sprintf("LODSphere_%d_%d", x, z)

			// Create different LOD levels
			highDetail := GenerateSphere(6.0, 16, 16, material)
			medDetail := GenerateSphere(6.0, 10, 10, material)
			lowDetail := GenerateSphere(6.0, 6, 6, material)

			// Create LOD group
			lodGroup := NewLODGroup()
			lodGroup.AddLOD(highDetail, 60.0) // High detail within 60 units
			lodGroup.AddLOD(medDetail, 120.0) // Medium detail within 120 units
			lodGroup.AddLOD(lowDetail, 250.0) // Low detail beyond 120 units

			// Create node
			node := scene.CreateEmpty(name)
			node.SetLODGroup(lodGroup)

			// Position in grid
			posX := (float64(x) - float64(gridSize)/2) * spacing
			posZ := (float64(z) - float64(gridSize)/2) * spacing
			node.Transform.SetPosition(posX, 0, posZ)

			// Add tags for functionality
			node.AddTag("pickable")
			node.AddTag("rotating")

			// Compute and store bounding volume (for demonstration)
			bounds := ComputeMeshBounds(highDetail)
			_ = bounds // In real usage, you'd store this in the node
		}
	}

	// Create a "picked" indicator (cursor)
	cursorMat := NewMaterial()
	cursorMat.DiffuseColor = ColorYellow
	cursor := scene.CreateSphere("Cursor", 2.0, 8, 8, cursorMat)
	cursor.Transform.SetPosition(0, -20, 0) // Start off-screen
	cursor.AddTag("cursor")

	// Create ground plane for reference
	groundMat := NewMaterial()
	groundMat.DiffuseColor = Color{80, 100, 80}
	ground := scene.CreateCube("Ground", 200, groundMat)
	ground.Transform.SetPosition(0, -15, 0)
	ground.Transform.SetScale(5, 0.1, 5)

	// Create raycast visualization line
	rayLine := scene.CreateEmpty("RayLine")
	rayLine.AddTag("ray-viz")
}

// AnimateAdvancedSystems handles the advanced demo
func AnimateAdvancedSystems(scene *Scene, time float64) {
	// Update LOD for all objects
	scene.UpdateLODs()

	// Rotate objects slowly
	rotating := scene.FindNodesByTag("rotating")
	for _, obj := range rotating {
		obj.RotateLocal(0.01, 0.02, 0.0)
	}

	// Simulate continuous raycasting from camera center
	screenWidth := 223
	screenHeight := 51

	// Cast ray from center of screen
	hit := scene.RaycastFromScreen(
		screenWidth/2,
		screenHeight/2,
		screenWidth,
		screenHeight,
		500.0,
	)

	// Update cursor position based on raycast
	cursor := scene.FindNode("Cursor")
	if cursor != nil {
		if hit.Hit {
			// Move cursor to hit point
			cursor.Transform.SetPosition(hit.Point.X, hit.Point.Y, hit.Point.Z)
			cursor.SetEnabled(true)

			// Highlight hit object
			if hit.Node != nil && hit.Node.HasTag("pickable") {
				// Get LOD group and change color
				if lodGroup := hit.Node.GetLODGroup(); lodGroup != nil {
					currentMesh := lodGroup.GetCurrentMesh()
					if currentMesh != nil {
						// Pulse color based on time
						intensity := 0.5 + 0.5*math.Sin(time*5.0)
						for _, tri := range currentMesh.Triangles {
							tri.Material.DiffuseColor = Color{
								R: uint8(200 + 55*intensity),
								G: uint8(150 + 105*intensity),
								B: 100,
							}
						}
						for _, quad := range currentMesh.Quads {
							quad.Material.DiffuseColor = Color{
								R: uint8(200 + 55*intensity),
								G: uint8(150 + 105*intensity),
								B: 100,
							}
						}
					}
				}
			}
		} else {
			// No hit - hide cursor
			cursor.SetEnabled(false)
		}
	}
}

// BoundingVolumeDemo demonstrates bounding volume usage
func BoundingVolumeDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = ColorBlue

	// Create objects with explicit bounding volumes
	for i := 0; i < 8; i++ {
		angle := (float64(i) / 8.0) * 2 * math.Pi
		radius := 60.0

		x := radius * math.Cos(angle)
		z := radius * math.Sin(angle)

		name := fmt.Sprintf("BoundedObj_%d", i)
		sphere := scene.CreateSphere(name, 8.0, 12, 12, material)
		sphere.Transform.SetPosition(x, 0, z)
		sphere.AddTag("bounded")

		// Compute bounds
		if mesh, ok := sphere.Object.(*Mesh); ok {
			bounds := ComputeMeshBounds(mesh)
			_ = bounds // Store in custom field if needed

			// Demonstrate bounds usage
			center := bounds.GetCenter()
			radius := bounds.GetRadius()
			_ = center
			_ = radius
		}
	}

	// Create moving "probe" that checks intersections
	probeMat := NewMaterial()
	probeMat.DiffuseColor = ColorRed
	probe := scene.CreateSphere("Probe", 5.0, 8, 8, probeMat)
	probe.Transform.SetPosition(0, 0, 0)
	probe.AddTag("probe")
}

// AnimateBoundingVolume demonstrates collision detection
func AnimateBoundingVolume(scene *Scene, time float64) {
	probe := scene.FindNode("Probe")
	if probe == nil {
		return
	}

	// Move probe in a circle
	radius := 60.0
	x := radius * math.Cos(time*0.5)
	z := radius * math.Sin(time*0.5)
	probe.Transform.SetPosition(x, 0, z)

	// Get probe bounds
	var probeBounds *AABB
	if mesh, ok := probe.Object.(*Mesh); ok {
		localBounds := ComputeMeshBounds(mesh)
		probeBounds = TransformAABB(localBounds, probe.Transform)
	}

	// Check intersections with other objects
	bounded := scene.FindNodesByTag("bounded")
	for _, obj := range bounded {
		if mesh, ok := obj.Object.(*Mesh); ok {
			localBounds := ComputeMeshBounds(mesh)
			objBounds := TransformAABB(localBounds, obj.Transform)

			// Check intersection
			if probeBounds != nil && probeBounds.IntersectsAABB(objBounds) {
				// Collision! Change color
				for _, tri := range mesh.Triangles {
					tri.Material.DiffuseColor = ColorYellow
				}
				for _, quad := range mesh.Quads {
					quad.Material.DiffuseColor = ColorYellow
				}
			} else {
				// No collision - reset color
				for _, tri := range mesh.Triangles {
					tri.Material.DiffuseColor = ColorBlue
				}
				for _, quad := range mesh.Quads {
					quad.Material.DiffuseColor = ColorBlue
				}
			}
		}
	}
}

// PerformanceTestDemo creates many objects to test LOD performance
func PerformanceTestDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = Color{150, 150, 200}

	// Create a large grid to test LOD performance
	gridSize := 10
	spacing := 25.0

	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			name := fmt.Sprintf("PerfTest_%d_%d", x, z)

			// Create LOD chain automatically
			baseMesh := GenerateSphere(4.0, 20, 20, material)
			lodGroup := GenerateLODChain(baseMesh, 3)

			node := scene.CreateEmpty(name)
			node.SetLODGroup(lodGroup)

			posX := (float64(x) - float64(gridSize)/2) * spacing
			posZ := (float64(z) - float64(gridSize)/2) * spacing
			node.Transform.SetPosition(posX, 0, posZ)

			node.AddTag("perf-test")
		}
	}
}

// AnimatePerformanceTest shows LOD statistics
func AnimatePerformanceTest(scene *Scene, time float64) {
	scene.UpdateLODs()

	// Print stats periodically
	if int(time*10)%100 == 0 { // Every 10 seconds
		stats := scene.GetLODStats()
		fmt.Printf("\n=== PERFORMANCE STATS ===\n")
		fmt.Printf("Total LOD Groups: %d\n", stats.TotalLODGroups)
		fmt.Printf("High Detail (LOD0): %d (%.1f%%)\n",
			stats.ActiveLOD0,
			float64(stats.ActiveLOD0)/float64(stats.TotalLODGroups)*100)
		fmt.Printf("Med Detail (LOD1):  %d (%.1f%%)\n",
			stats.ActiveLOD1,
			float64(stats.ActiveLOD1)/float64(stats.TotalLODGroups)*100)
		fmt.Printf("Low Detail (LOD2):  %d (%.1f%%)\n",
			stats.ActiveLOD2,
			float64(stats.ActiveLOD2)/float64(stats.TotalLODGroups)*100)
		fmt.Printf("Total Triangles:    %d\n", stats.TotalTriangles)
	}
}

// LineOfSightDemo demonstrates line-of-sight raycasting
func LineOfSightDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = ColorBlue

	// Create "guard" object
	guardMat := NewMaterial()
	guardMat.DiffuseColor = ColorRed
	guard := scene.CreateSphere("Guard", 5.0, 10, 10, guardMat)
	guard.Transform.SetPosition(-40, 0, 0)
	guard.AddTag("guard")

	// Create "target" object
	targetMat := NewMaterial()
	targetMat.DiffuseColor = ColorGreen
	target := scene.CreateSphere("Target", 5.0, 10, 10, targetMat)
	target.Transform.SetPosition(40, 0, 0)
	target.AddTag("target")

	// Create obstacles
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("Obstacle_%d", i)
		obstacle := scene.CreateCube(name, 8, material)
		obstacle.Transform.SetPosition(
			float64(i-2)*15.0,
			0,
			float64((i%2)*20-10),
		)
		obstacle.AddTag("obstacle")
	}
}

// AnimateLineOfSight checks and visualizes line of sight
func AnimateLineOfSight(scene *Scene, time float64) {
	guard := scene.FindNode("Guard")
	target := scene.FindNode("Target")

	if guard == nil || target == nil {
		return
	}

	// Move target in a pattern
	target.Transform.SetPosition(
		40*math.Cos(time*0.3),
		5*math.Sin(time*0.7),
		40*math.Sin(time*0.3),
	)

	// Check line of sight
	guardPos := guard.Transform.GetWorldPosition()
	targetPos := target.Transform.GetWorldPosition()

	hasLOS := scene.LineOfSight(guardPos, targetPos, 1000.0)

	// Change guard color based on LOS
	if mesh, ok := guard.Object.(*Mesh); ok {
		color := ColorRed
		if hasLOS {
			color = ColorYellow // Can see target!
		}

		for _, tri := range mesh.Triangles {
			tri.Material.DiffuseColor = color
		}
		for _, quad := range mesh.Quads {
			quad.Material.DiffuseColor = color
		}
	}

	// Rotate obstacles
	obstacles := scene.FindNodesByTag("obstacle")
	for _, obs := range obstacles {
		obs.RotateLocal(0, 0.02, 0)
	}
}

// Helper function to create a visual ray (for debugging)
func CreateRayVisualization(scene *Scene, ray Ray, length float64) *Line {
	start := ray.Origin
	end := ray.GetPoint(length)
	return NewLine(start, end)
}

func OctreeDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = Color{100, 150, 200}

	// Create many objects scattered in space
	numObjects := 100
	spread := 200.0

	for i := 0; i < numObjects; i++ {
		angle := (float64(i) / float64(numObjects)) * 2 * math.Pi
		radius := spread * (0.3 + 0.7*float64(i)/float64(numObjects))
		height := (math.Sin(float64(i)*0.5) * 50.0)

		x := radius * math.Cos(angle)
		y := height
		z := radius * math.Sin(angle)

		name := fmt.Sprintf("OctreeObj_%d", i)
		sphere := scene.CreateSphere(name, 4.0, 8, 8, material)
		sphere.Transform.SetPosition(x, y, z)
		sphere.AddTag("octree-object")
	}

	// Build octree (will be used for culling/queries)
	// Store reference in scene metadata
	fmt.Println("\n=== Building Octree ===")
	octree := scene.BuildOctree(5, 10) // Max depth 5, max 10 objects per leaf
	if octree != nil {
		fmt.Printf("Octree built: %d nodes, %d objects\n", octree.TotalNodes, octree.TotalObjects)
	}
}

func AnimateOctree(scene *Scene, time float64) {
	// Rotate objects slowly
	objects := scene.FindNodesByTag("octree-object")
	for _, obj := range objects {
		obj.RotateLocal(0.01, 0.02, 0.0)
	}

	// Demonstrate octree query every 5 seconds
	if int(time*10)%50 == 0 {
		octree := scene.BuildOctree(5, 10)
		if octree != nil {
			// Query objects near camera
			camPos := scene.Camera.GetPosition()
			queryBounds := NewAABB(
				Point{X: camPos.X - 50, Y: camPos.Y - 50, Z: camPos.Z - 50},
				Point{X: camPos.X + 50, Y: camPos.Y + 50, Z: camPos.Z + 50},
			)

			nearbyObjects := octree.Query(queryBounds)
			fmt.Printf("\n=== Octree Query ===\n")
			fmt.Printf("Found %d objects near camera\n", len(nearbyObjects))
		}
	}
}

func BVHDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = Color{200, 100, 150}

	// Create dynamic objects
	for i := 0; i < 50; i++ {
		angle := (float64(i) / 50.0) * 2 * math.Pi
		radius := 80.0

		x := radius * math.Cos(angle)
		z := radius * math.Sin(angle)

		name := fmt.Sprintf("BVHObj_%d", i)
		cube := scene.CreateCube(name, 6, material)
		cube.Transform.SetPosition(x, 0, z)
		cube.AddTag("bvh-object")
	}

	// Build BVH
	fmt.Println("\n=== Building BVH ===")
	bvh := scene.BuildBVH()
	if bvh != nil {
		fmt.Printf("BVH built: %d nodes\n", bvh.TotalNodes)
	}
}

func AnimateBVH(scene *Scene, time float64) {
	// Move objects dynamically
	objects := scene.FindNodesByTag("bvh-object")
	for i, obj := range objects {
		angle := (float64(i)/float64(len(objects)))*2*math.Pi + time*0.5
		radius := 80.0 + 20.0*math.Sin(time+float64(i)*0.3)

		x := radius * math.Cos(angle)
		z := radius * math.Sin(angle)
		y := 10.0 * math.Sin(time*2.0+float64(i)*0.5)

		obj.Transform.SetPosition(x, y, z)
		obj.RotateLocal(0.02, 0.03, 0.01)
	}

	// Rebuild BVH periodically (every 2 seconds)
	if int(time*10)%20 == 0 {
		bvh := scene.BuildBVH()
		if bvh != nil {
			fmt.Printf("\n=== BVH Rebuilt ===\n")
			fmt.Printf("Nodes: %d\n", bvh.TotalNodes)
		}
	}
}

// ============================================================================
// DEMO: OBB Collision Detection
// ============================================================================

func OBBDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = ColorBlue

	// Create rotating boxes
	for i := 0; i < 8; i++ {
		angle := (float64(i) / 8.0) * 2 * math.Pi
		radius := 60.0

		x := radius * math.Cos(angle)
		z := radius * math.Sin(angle)

		name := fmt.Sprintf("OBBBox_%d", i)
		cube := scene.CreateCube(name, 8, material)
		cube.Transform.SetPosition(x, 0, z)
		cube.AddTag("obb-box")
	}

	// Create moving probe
	probeMat := NewMaterial()
	probeMat.DiffuseColor = ColorRed
	probe := scene.CreateCube("OBBProbe", 6, probeMat)
	probe.Transform.SetPosition(0, 0, 0)
	probe.AddTag("obb-probe")
}

func AnimateOBB(scene *Scene, time float64) {
	// Rotate all boxes
	boxes := scene.FindNodesByTag("obb-box")
	for i, box := range boxes {
		box.RotateLocal(0.02+float64(i)*0.005, 0.03, 0.01)
	}

	// Move probe
	probe := scene.FindNode("OBBProbe")
	if probe != nil {
		radius := 60.0
		x := radius * math.Cos(time*0.5)
		z := radius * math.Sin(time*0.5)
		probe.Transform.SetPosition(x, 0, z)
		probe.RotateLocal(0.05, 0.04, 0.02)

		// Get probe OBB
		if probeMesh, ok := probe.Object.(*Mesh); ok {
			probeAABB := ComputeMeshBounds(probeMesh)
			probeOBB := NewOBBFromTransformedAABB(probeAABB, probe.Transform)

			// Check collisions with OBB
			for _, box := range boxes {
				if boxMesh, ok := box.Object.(*Mesh); ok {
					boxAABB := ComputeMeshBounds(boxMesh)
					boxOBB := NewOBBFromTransformedAABB(boxAABB, box.Transform)

					// Test OBB-OBB intersection
					if probeOBB.IntersectsOBB(boxOBB) {
						// Collision! Change color
						for _, tri := range boxMesh.Triangles {
							tri.Material.DiffuseColor = ColorYellow
						}
					} else {
						// No collision - reset color
						for _, tri := range boxMesh.Triangles {
							tri.Material.DiffuseColor = ColorBlue
						}
					}
				}
			}
		}
	}
}

func MeshSimplificationDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = Color{150, 200, 150}

	// Create high-detail sphere
	highDetail := GenerateSphere(12.0, 24, 24, material)

	// Generate simplified versions using QEM
	fmt.Println("\n=== Mesh Simplification (QEM) ===")
	fmt.Printf("Original: %d triangles\n", len(highDetail.Triangles))

	med1 := SimplifyMeshQEM(highDetail, 200)
	fmt.Printf("75%% simplified: %d triangles\n", len(med1.Triangles))

	med2 := SimplifyMeshQEM(highDetail, 100)
	fmt.Printf("50%% simplified: %d triangles\n", len(med2.Triangles))

	low := SimplifyMeshQEM(highDetail, 50)
	fmt.Printf("25%% simplified: %d triangles\n", len(low.Triangles))

	// Display side by side
	highNode := scene.CreateEmpty("HighDetail")
	highNode.Object = highDetail
	highNode.Transform.SetPosition(-40, 0, 0)

	med1Node := scene.CreateEmpty("Med1Detail")
	med1Node.Object = med1
	med1Node.Transform.SetPosition(-13, 0, 0)

	med2Node := scene.CreateEmpty("Med2Detail")
	med2Node.Object = med2
	med2Node.Transform.SetPosition(13, 0, 0)

	lowNode := scene.CreateEmpty("LowDetail")
	lowNode.Object = low
	lowNode.Transform.SetPosition(40, 0, 0)

	// Tag for rotation
	highNode.AddTag("qem-rotating")
	med1Node.AddTag("qem-rotating")
	med2Node.AddTag("qem-rotating")
	lowNode.AddTag("qem-rotating")
}

func AnimateMeshSimplification(scene *Scene, time float64) {
	rotating := scene.FindNodesByTag("qem-rotating")
	for _, obj := range rotating {
		obj.RotateLocal(0, 0.02, 0)
	}
}

func SmoothLODDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = Color{200, 150, 100}

	// Create objects with smooth LOD transitions
	numObjects := 12
	for i := 0; i < numObjects; i++ {
		angle := (float64(i) / float64(numObjects)) * 2 * math.Pi
		radius := 80.0

		x := radius * math.Cos(angle)
		z := radius * math.Sin(angle)

		name := fmt.Sprintf("SmoothLOD_%d", i)

		// Generate LOD chain with simplification
		baseMesh := GenerateSphere(6.0, 20, 20, material)
		lodGroup := GenerateAdvancedLODChain(baseMesh, 4, true) // Use QEM

		// Wrap in transition-enabled LOD group
		transitionGroup := NewLODGroupWithTransitions(LODTransitionFade, 0.5)
		transitionGroup.LODGroup = lodGroup

		node := scene.CreateEmpty(name)
		node.SetLODGroupWithTransition(transitionGroup)
		node.Transform.SetPosition(x, 0, z)

		node.AddTag("smooth-lod")
	}

	fmt.Println("\n=== Smooth LOD Transitions ===")
	fmt.Println("LOD will smoothly fade as camera moves")
	fmt.Println("Transition duration: 0.5 seconds")
}

func AnimateSmoothLOD(scene *Scene, time float64) {
	// Update LODs with transitions
	scene.UpdateLODsWithTransitions(time)

	// Rotate objects
	objects := scene.FindNodesByTag("smooth-lod")
	for _, obj := range objects {
		obj.RotateLocal(0.01, 0.02, 0.0)
	}

	// Print transition info periodically
	if int(time*10)%50 == 0 {
		transitioning := 0
		for _, obj := range objects {
			if lodGroup, ok := obj.Object.(*LODGroupWithTransitions); ok {
				if lodGroup.TransitionState.IsTransitioning {
					transitioning++
				}
			}
		}

		if transitioning > 0 {
			fmt.Printf("\n=== LOD Status ===\n")
			fmt.Printf("%d objects transitioning\n", transitioning)
		}
	}
}

func CombinedAdvancedDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = Color{180, 140, 200}

	// Create grid with all advanced features
	gridSize := 8
	spacing := 30.0

	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			name := fmt.Sprintf("Advanced_%d_%d", x, z)

			// Generate LOD with QEM simplification
			baseMesh := GenerateSphere(5.0, 16, 16, material)
			lodGroup := GenerateAdvancedLODChain(baseMesh, 3, true)

			// Enable smooth transitions
			transitionGroup := NewLODGroupWithTransitions(LODTransitionFade, 0.3)
			transitionGroup.LODGroup = lodGroup

			node := scene.CreateEmpty(name)
			node.SetLODGroupWithTransition(transitionGroup)

			posX := (float64(x) - float64(gridSize)/2) * spacing
			posZ := (float64(z) - float64(gridSize)/2) * spacing
			node.Transform.SetPosition(posX, 0, posZ)

			node.AddTag("advanced-object")
		}
	}

	// Build spatial structures
	fmt.Println("\n=== Combined Advanced Features ===")
	octree := scene.BuildOctree(5, 8)
	if octree != nil {
		fmt.Printf("Octree: %d nodes, %d objects\n", octree.TotalNodes, octree.TotalObjects)
	}

	bvh := scene.BuildBVH()
	if bvh != nil {
		fmt.Printf("BVH: %d nodes\n", bvh.TotalNodes)
	}

	fmt.Println("Features enabled:")
	fmt.Println("  - Octree spatial partitioning")
	fmt.Println("  - BVH for dynamic culling")
	fmt.Println("  - QEM mesh simplification")
	fmt.Println("  - Smooth LOD transitions (fade)")
	fmt.Println("  - OBB collision detection")
}

func AnimateCombinedAdvanced(scene *Scene, time float64) {
	// Update LODs with smooth transitions
	scene.UpdateLODsWithTransitions(time)

	// Rotate objects
	objects := scene.FindNodesByTag("advanced-object")
	for _, obj := range objects {
		obj.RotateLocal(0.005, 0.01, 0.0)
	}

	// Rebuild BVH periodically
	if int(time*10)%30 == 0 {
		scene.BuildBVH()
	}

	// Print comprehensive stats every 5 seconds
	if int(time*10)%50 == 0 {
		stats := scene.GetLODStats()
		fmt.Printf("\n=== Performance Stats ===\n")
		fmt.Printf("Total LOD Groups: %d\n", stats.TotalLODGroups)
		fmt.Printf("High Detail: %d (%.1f%%)\n",
			stats.ActiveLOD0,
			float64(stats.ActiveLOD0)/float64(stats.TotalLODGroups)*100)
		fmt.Printf("Med Detail: %d (%.1f%%)\n",
			stats.ActiveLOD1,
			float64(stats.ActiveLOD1)/float64(stats.TotalLODGroups)*100)
		fmt.Printf("Low Detail: %d (%.1f%%)\n",
			stats.ActiveLOD2,
			float64(stats.ActiveLOD2)/float64(stats.TotalLODGroups)*100)
		fmt.Printf("Total Triangles: %d\n", stats.TotalTriangles)

		// Count transitioning objects
		transitioning := 0
		for _, obj := range objects {
			if lodGroup, ok := obj.Object.(*LODGroupWithTransitions); ok {
				if lodGroup.TransitionState.IsTransitioning {
					transitioning++
				}
			}
		}
		fmt.Printf("Transitioning: %d\n", transitioning)
	}
}

func StressTestDemo(scene *Scene) {
	material := NewMaterial()
	material.DiffuseColor = Color{120, 160, 200}

	// Create LOTS of objects
	numObjects := 200
	spread := 300.0

	fmt.Println("\n=== Stress Test ===")
	fmt.Printf("Creating %d objects...\n", numObjects)

	for i := 0; i < numObjects; i++ {
		// Random-ish distribution
		angle := (float64(i) / float64(numObjects)) * 6 * math.Pi
		radius := spread * math.Sqrt(float64(i)/float64(numObjects))
		height := 50.0 * math.Sin(float64(i)*0.2)

		x := radius * math.Cos(angle)
		y := height
		z := radius * math.Sin(angle)

		name := fmt.Sprintf("Stress_%d", i)

		// Use aggressive LOD for performance
		baseMesh := GenerateSphere(4.0, 12, 12, material)
		lodGroup := GenerateAdvancedLODChain(baseMesh, 4, false) // Use clustering (faster)

		transitionGroup := NewLODGroupWithTransitions(LODTransitionFade, 0.2)
		transitionGroup.LODGroup = lodGroup

		node := scene.CreateEmpty(name)
		node.SetLODGroupWithTransition(transitionGroup)
		node.Transform.SetPosition(x, y, z)
		node.AddTag("stress-object")
	}

	// Build optimized structures
	octree := scene.BuildOctree(6, 20)
	bvh := scene.BuildBVH()

	fmt.Printf("Octree: %d nodes\n", octree.TotalNodes)
	fmt.Printf("BVH: %d nodes\n", bvh.TotalNodes)
	fmt.Println("Ready!")
}

func AnimateStressTest(scene *Scene, time float64) {
	scene.UpdateLODsWithTransitions(time)

	// Rotate some objects
	objects := scene.FindNodesByTag("stress-object")
	for i, obj := range objects {
		if i%3 == 0 { // Only rotate 1/3 of objects for performance
			obj.RotateLocal(0.005, 0.01, 0.0)
		}
	}

	// Stats every 3 seconds
	if int(time*10)%30 == 0 {
		stats := scene.GetLODStats()
		fmt.Printf("\n=== Stress Test Stats ===\n")
		fmt.Printf("Objects: %d\n", stats.TotalLODGroups)
		fmt.Printf("Triangles: %d\n", stats.TotalTriangles)
		fmt.Printf("LOD Distribution: L0:%d L1:%d L2:%d\n",
			stats.ActiveLOD0, stats.ActiveLOD1, stats.ActiveLOD2)
	}
}

// Example usage showing the difference:
func LightingComparisonDemo(scene *Scene, writer *bufio.Writer) {
	renderer := NewTerminalRenderer(writer, 51, 223)

	// Set up lighting system with multiple lights
	ls := NewLightingSystem(scene.Camera)

	// Add key light
	keyLight := NewLight(30, 30, -20, ColorWhite, 1.0)
	ls.AddLight(keyLight)

	// Add fill light
	fillLight := NewLight(-20, 10, -10, Color{150, 150, 200}, 0.4)
	ls.AddLight(fillLight)

	// Add rim light
	rimLight := NewLight(0, 20, 30, Color{255, 200, 150}, 0.6)
	ls.AddLight(rimLight)

	scene.AddNode(scene.CreateSphere("CenterSphere", 10.0, 16, 16, NewMaterial()))

	renderer.SetLightingSystem(ls)

	renderer.RenderScene(scene)
}
