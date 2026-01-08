package main

import (
	"fmt"
	"math"
)

// ============================================================================
// BASIC DEMOS - Simple, Working Examples
// ============================================================================

// SolarSystemDemo creates a solar system with orbiting planets
func SolarSystemDemo(scene *Scene) {
	fmt.Println("=== Solar System Demo ===")

	// Sun (Center)
	sunMat := NewMaterial()
	sunMat.DiffuseColor = ColorYellow
	sun := scene.CreateSphere("Sun", 8, 24, 24, sunMat)
	sun.Transform.SetPosition(0, 0, 0)
	sun.AddTag("sun")

	// Earth
	earthMat := NewMaterial()
	earthMat.DiffuseColor = ColorBlue
	earth := scene.CreateSphere("Earth", 4, 16, 16, earthMat)
	earth.Transform.SetPosition(40, 0, 0)
	earth.AddTag("planet")

	// Moon
	moonMat := NewMaterial()
	moonMat.DiffuseColor = Color{200, 200, 200}
	moon := scene.CreateSphere("Moon", 2, 12, 12, moonMat)
	moon.Transform.SetPosition(50, 0, 0)
	moon.AddTag("moon")

	// Mars
	marsMat := NewMaterial()
	marsMat.DiffuseColor = ColorRed
	mars := scene.CreateSphere("Mars", 3, 16, 16, marsMat)
	mars.Transform.SetPosition(65, 0, 0)
	mars.AddTag("planet")

	// Venus
	venusMat := NewMaterial()
	venusMat.DiffuseColor = Color{255, 200, 100}
	venus := scene.CreateSphere("Venus", 3.5, 16, 16, venusMat)
	venus.Transform.SetPosition(-30, 0, 0)
	venus.AddTag("planet")
}

func AnimateSolarSystem(scene *Scene) {
	// Rotate sun
	sun := scene.FindNode("Sun")
	if sun != nil {
		sun.RotateLocal(0, 0.01, 0)
	}

	// Orbit planets
	planets := scene.FindNodesByTag("planet")
	for _, planet := range planets {
		// Get distance from sun
		x := planet.Transform.Position.X
		z := planet.Transform.Position.Z
		distance := math.Sqrt(x*x + z*z)

		// Orbital speed inversely proportional to distance
		speed := 0.5 / distance
		angle := speed

		// Rotate around sun
		newX := x*math.Cos(angle) - z*math.Sin(angle)
		newZ := x*math.Sin(angle) + z*math.Cos(angle)
		planet.Transform.Position.X = newX
		planet.Transform.Position.Z = newZ

		// Rotate planet itself
		planet.RotateLocal(0, 0.02, 0)
	}

	// Moon orbits faster
	moon := scene.FindNode("Moon")
	if moon != nil {
		x := moon.Transform.Position.X
		z := moon.Transform.Position.Z
		angle := 0.03
		newX := x*math.Cos(angle) - z*math.Sin(angle)
		newZ := x*math.Sin(angle) + z*math.Cos(angle)
		moon.Transform.Position.X = newX
		moon.Transform.Position.Z = newZ
		moon.RotateLocal(0, 0.04, 0)
	}
}

// RobotArmDemo creates a robotic arm with multiple joints
func RobotArmDemo(scene *Scene, material Material) {
	fmt.Println("=== Robot Arm Demo ===")

	baseMat := NewMaterial()
	baseMat.DiffuseColor = Color{150, 150, 150}

	// Base platform
	base := scene.CreateCube("Base", 6, baseMat)
	base.Transform.SetPosition(0, -5, 0)
	base.Transform.SetScale(2, 0.5, 2)

	// Lower arm
	lowerMat := NewMaterial()
	lowerMat.DiffuseColor = ColorRed
	lower := scene.CreateCube("LowerArm", 3, lowerMat)
	lower.Transform.SetPosition(0, 5, 0)
	lower.Transform.SetScale(1, 3, 1)
	lower.AddTag("arm")

	// Upper arm
	upperMat := NewMaterial()
	upperMat.DiffuseColor = ColorBlue
	upper := scene.CreateCube("UpperArm", 3, upperMat)
	upper.Transform.SetPosition(0, 15, 0)
	upper.Transform.SetScale(0.8, 2.5, 0.8)
	upper.AddTag("arm")

	// Gripper
	gripperMat := NewMaterial()
	gripperMat.DiffuseColor = ColorYellow
	gripper := scene.CreateCube("Gripper", 2, gripperMat)
	gripper.Transform.SetPosition(0, 22, 0)
	gripper.Transform.SetScale(0.6, 1, 0.6)
	gripper.AddTag("arm")
}

func AnimateRobotArm(scene *Scene, time float64) {
	// Rotate base
	base := scene.FindNode("Base")
	if base != nil {
		base.RotateLocal(0, 0.01, 0)
	}

	// Animate arms
	lower := scene.FindNode("LowerArm")
	if lower != nil {
		angle := math.Sin(time*0.5) * 0.3
		lower.Transform.Rotation.Z = angle
	}

	upper := scene.FindNode("UpperArm")
	if upper != nil {
		angle := math.Sin(time*0.7) * 0.4
		upper.Transform.Rotation.Z = angle
	}

	gripper := scene.FindNode("Gripper")
	if gripper != nil {
		gripper.RotateLocal(0.02, 0.03, 0.01)
	}
}

// SpinningCubesDemo creates a 3D grid of spinning cubes
func SpinningCubesDemo(scene *Scene, material Material) {
	fmt.Println("=== Spinning Cubes Demo ===")

	gridSize := 5
	spacing := 12.0
	offset := -float64(gridSize-1) * spacing / 2

	colors := []Color{
		ColorRed, ColorGreen, ColorBlue,
		ColorYellow, ColorMagenta, ColorCyan,
	}

	for x := 0; x < gridSize; x++ {
		for y := 0; y < gridSize; y++ {
			for z := 0; z < gridSize; z++ {
				mat := NewMaterial()
				mat.DiffuseColor = colors[(x+y+z)%len(colors)]

				name := fmt.Sprintf("Cube_%d_%d_%d", x, y, z)
				cube := scene.CreateCube(name, 4, mat)
				cube.Transform.SetPosition(
					offset+float64(x)*spacing,
					offset+float64(y)*spacing,
					offset+float64(z)*spacing,
				)
				cube.AddTag("spinning")
			}
		}
	}

	fmt.Printf("Created %d cubes in a %dx%dx%d grid\n", gridSize*gridSize*gridSize, gridSize, gridSize, gridSize)
}

func AnimateSpinningCubes(scene *Scene) {
	cubes := scene.FindNodesByTag("spinning")
	for _, cube := range cubes {
		cube.RotateLocal(0.02, 0.03, 0.01)
	}
}

// OrbitingObjectsDemo creates objects orbiting a center point
func OrbitingObjectsDemo(scene *Scene, material Material) {
	fmt.Println("=== Orbiting Objects Demo ===")

	// Center sphere
	centerMat := NewMaterial()
	centerMat.DiffuseColor = ColorYellow
	center := scene.CreateSphere("Center", 8, 24, 24, centerMat)
	center.Transform.SetPosition(0, 0, 0)

	// Create orbiting objects
	numObjects := 8
	radius := 30.0

	for i := 0; i < numObjects; i++ {
		angle := (float64(i) / float64(numObjects)) * 2 * math.Pi
		x := radius * math.Cos(angle)
		z := radius * math.Sin(angle)

		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(128 + 127*math.Sin(angle)),
			G: uint8(128 + 127*math.Cos(angle)),
			B: uint8(200),
		}

		name := fmt.Sprintf("Orbiter_%d", i)
		obj := scene.CreateCube(name, 4, mat)
		obj.Transform.SetPosition(x, 0, z)
		obj.AddTag("orbiter")
	}
}

func AnimateOrbitingObjects(scene *Scene) {
	orbiters := scene.FindNodesByTag("orbiter")
	for _, obj := range orbiters {
		// Rotate around center
		x := obj.Transform.Position.X
		z := obj.Transform.Position.Z
		angle := 0.02

		newX := x*math.Cos(angle) - z*math.Sin(angle)
		newZ := x*math.Sin(angle) + z*math.Cos(angle)

		obj.Transform.Position.X = newX
		obj.Transform.Position.Z = newZ

		// Spin the object itself
		obj.RotateLocal(0.03, 0.02, 0.01)
	}
}

// WaveGridDemo creates a wave animation on a grid
func WaveGridDemo(scene *Scene, material Material) {
	fmt.Println("=== Wave Grid Demo ===")

	gridSize := 10
	spacing := 6.0
	offset := -float64(gridSize-1) * spacing / 2

	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			mat := NewMaterial()
			// Color gradient across grid
			mat.DiffuseColor = Color{
				R: uint8(100 + (x * 155 / gridSize)),
				G: uint8(100 + (z * 155 / gridSize)),
				B: 200,
			}

			name := fmt.Sprintf("Wave_%d_%d", x, z)
			cube := scene.CreateCube(name, 3, mat)
			cube.Transform.SetPosition(
				offset+float64(x)*spacing,
				0,
				offset+float64(z)*spacing,
			)
			cube.AddTag("wave")
		}
	}
}

func AnimateWaveGrid(scene *Scene, time float64) {
	waves := scene.FindNodesByTag("wave")
	for _, wave := range waves {
		// Extract grid position from name
		x := wave.Transform.Position.X
		z := wave.Transform.Position.Z

		// Create wave pattern
		height := 15.0 * math.Sin(x*0.2+time*2) * math.Cos(z*0.2+time*2)
		wave.Transform.Position.Y = height

		// Rotate slightly
		wave.RotateLocal(0.01, 0.01, 0)
	}
}

// HelixDemo creates a helix/spiral structure
func HelixDemo(scene *Scene, material Material) {
	fmt.Println("=== Helix Demo ===")

	numPoints := 40
	radius := 20.0
	height := 60.0

	for i := 0; i < numPoints; i++ {
		t := float64(i) / float64(numPoints)
		angle := t * 4 * math.Pi // 2 complete rotations

		x := radius * math.Cos(angle)
		y := -height/2 + t*height
		z := radius * math.Sin(angle)

		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(100 + 155*t),
			G: uint8(200 - 100*t),
			B: 200,
		}

		name := fmt.Sprintf("Helix_%d", i)
		sphere := scene.CreateSphere(name, 2.5, 12, 12, mat)
		sphere.Transform.SetPosition(x, y, z)
		sphere.AddTag("helix")
	}
}

func AnimateHelix(scene *Scene, time float64) {
	helixPoints := scene.FindNodesByTag("helix")
	for i, point := range helixPoints {
		// Rotate the entire helix
		x := point.Transform.Position.X
		z := point.Transform.Position.Z
		angle := 0.01

		newX := x*math.Cos(angle) - z*math.Sin(angle)
		newZ := x*math.Sin(angle) + z*math.Cos(angle)

		point.Transform.Position.X = newX
		point.Transform.Position.Z = newZ

		// Pulse individual spheres
		scale := 1.0 + 0.3*math.Sin(time*2+float64(i)*0.5)
		point.Transform.SetScale(scale, scale, scale)
	}
}

// WireframeDemo shows wireframe and solid rendering
func WireframeDemo(scene *Scene, material Material) {
	fmt.Println("=== Wireframe Demo ===")

	// Solid cube
	solidMat := NewMaterial()
	solidMat.DiffuseColor = ColorGreen
	solidMat.Wireframe = false
	solid := scene.CreateCube("SolidCube", 8, solidMat)
	solid.Transform.SetPosition(-15, 0, 0)
	solid.AddTag("rotating")

	// Wireframe cube
	wireMat := NewMaterial()
	wireMat.DiffuseColor = ColorRed
	wireMat.Wireframe = true
	wire := scene.CreateCube("WireCube", 8, wireMat)
	wire.Transform.SetPosition(0, 0, 0)
	wire.AddTag("rotating")

	// Mixed sphere - solid with wireframe overlay
	sphereMat := NewMaterial()
	sphereMat.DiffuseColor = ColorBlue
	sphere := scene.CreateSphere("SolidSphere", 6, 24, 24, sphereMat)
	sphere.Transform.SetPosition(15, 0, 0)
	sphere.AddTag("rotating")

	// Wireframe sphere overlay
	wireSphereMat := NewMaterial()
	wireSphereMat.DiffuseColor = ColorYellow
	wireSphereMat.Wireframe = true
	wireSphere := scene.CreateSphere("WireSphere", 6.2, 16, 16, wireSphereMat)
	wireSphere.Transform.SetPosition(15, 0, 0)
	wireSphere.AddTag("rotating")
}

func AnimateWireframe(scene *Scene, time float64) {
	rotating := scene.FindNodesByTag("rotating")
	for _, obj := range rotating {
		obj.RotateLocal(0.02, 0.03, 0.01)
	}
}

// ============================================================================
// ADVANCED DEMOS - LOD, Spatial Partitioning, etc.
// ============================================================================

// StressTestDemo creates many objects with LOD
func StressTestDemo(scene *Scene) {
	fmt.Println("=== Stress Test Demo ===")
	fmt.Println("Creating 200+ objects with LOD system...")

	gridSize := 7
	spacing := 20.0
	offset := -float64(gridSize-1) * spacing / 2

	objectCount := 0
	for x := 0; x < gridSize; x++ {
		for y := 0; y < gridSize; y++ {
			for z := 0; z < gridSize; z++ {
				// Create LOD group
				baseMesh := GenerateSphere(4.0, 16, 16)
				baseMat := NewMaterial()
				baseMat.DiffuseColor = Color{
					R: uint8(100 + (x * 20)),
					G: uint8(100 + (y * 20)),
					B: uint8(100 + (z * 20)),
				}
				baseMesh.Material = baseMat

				lodGroup := NewLODGroup()
				lodGroup.AddLOD(baseMesh, 50.0)

				// Medium detail
				medMesh := SimplifyMesh(baseMesh, 0.6)
				medMesh.Material = baseMat
				lodGroup.AddLOD(medMesh, 100.0)

				// Low detail
				lowMesh := SimplifyMesh(baseMesh, 0.3)
				lowMesh.Material = baseMat
				lodGroup.AddLOD(lowMesh, 200.0)

				name := fmt.Sprintf("LOD_%d_%d_%d", x, y, z)
				node := scene.CreateEmpty(name)
				node.SetLODGroup(lodGroup)
				node.Transform.SetPosition(
					offset+float64(x)*spacing,
					offset+float64(y)*spacing,
					offset+float64(z)*spacing,
				)
				node.AddTag("lod-object")
				objectCount++
			}
		}
	}

	fmt.Printf("Created %d LOD objects\n", objectCount)
}

func AnimateStressTest(scene *Scene, time float64) {
	// Only rotate a subset to save performance
	objects := scene.FindNodesByTag("lod-object")
	for i, obj := range objects {
		if i%5 == 0 {
			obj.RotateLocal(0.005, 0.01, 0)
		}
	}

	// Update LOD system
	scene.UpdateLODs()
}

// AnimateAdvancedSystems - Animate LOD and raycasting demo
func AnimateAdvancedSystems(scene *Scene, time float64) {
	// Rotate LOD objects
	objects := scene.FindNodesByTag("lod-object")
	for i, obj := range objects {
		// Rotate every other object
		if i%2 == 0 {
			obj.RotateLocal(0.01, 0.015, 0)
		}
	}

	// Move cursor in a circular pattern
	cursor := scene.FindNode("Cursor")
	if cursor != nil {
		radius := 20.0
		cursor.Transform.Position.X = radius * math.Cos(time*0.5)
		cursor.Transform.Position.Z = radius * math.Sin(time*0.5)
		cursor.Transform.Position.Y = 5.0 + 3.0*math.Sin(time)
		cursor.RotateLocal(0.03, 0.02, 0.01)
	}

	// Update LOD system
	scene.UpdateLODs()
}

// AnimateBoundingVolume - Animate collision detection demo
func AnimateBoundingVolume(scene *Scene, time float64) {
	// Move probe in a circular pattern
	probe := scene.FindNode("Probe")
	if probe != nil {
		probe.Transform.Position.X = math.Sin(time*0.8) * 25.0
		probe.Transform.Position.Z = math.Cos(time*0.8) * 25.0
		probe.RotateLocal(0.02, 0.03, 0.01)
	}

	// Gently rotate obstacles
	obstacles := scene.FindNodesByTag("obstacle")
	for i, obstacle := range obstacles {
		obstacle.RotateLocal(0.01, 0.01*float64(i%3), 0)
	}
}

// AnimatePerformanceTest - Add subtle animation to performance test
func AnimatePerformanceTest(scene *Scene, time float64) {
	// Rotate only a few objects to maintain performance
	objects := scene.FindNodesByTag("lod-object")
	for i, obj := range objects {
		if i%10 == 0 {
			obj.RotateLocal(0.003, 0.005, 0)
		}
	}

	// Update LOD system
	scene.UpdateLODs()
}

// AnimateLineOfSight - Animate visibility demo
func AnimateLineOfSight(scene *Scene, time float64) {
	// Rotate guard
	guard := scene.FindNode("Guard")
	if guard != nil {
		guard.RotateLocal(0, 0.02, 0)
	}

	// Move target in circular pattern
	target := scene.FindNode("Target")
	if target != nil {
		target.Transform.Position.X = math.Sin(time*0.5) * 30.0
		target.Transform.Position.Z = math.Cos(time*0.5) * 30.0
		target.RotateLocal(0.01, 0.02, 0)
	}

	// Subtle animation on walls
	walls := scene.FindNodesByTag("obstacle")
	for _, wall := range walls {
		wall.RotateLocal(0, 0.005, 0)
	}
}

// AnimateOctree - Animate spatial partitioning demo
func AnimateOctree(scene *Scene, time float64) {
	// Rotate all spatial objects with varied speeds
	objects := scene.FindNodesByTag("spatial-object")
	for i, obj := range objects {
		speed := 0.01 + float64(i%5)*0.002
		obj.RotateLocal(speed, speed*0.8, 0)

		// Add subtle floating motion
		baseY := obj.Transform.Position.Y
		obj.Transform.Position.Y = baseY + math.Sin(time+float64(i)*0.3)*2.0
	}

	// Move query sphere
	query := scene.FindNode("QuerySphere")
	if query != nil {
		query.Transform.Position.X = math.Cos(time*0.6) * 35.0
		query.Transform.Position.Z = math.Sin(time*0.6) * 35.0
		query.Transform.Position.Y = math.Sin(time*0.8) * 20.0
	}
}

// AnimateBVH - Animate BVH demo
func AnimateBVH(scene *Scene, time float64) {
	// Rotate objects within their clusters
	objects := scene.FindNodesByTag("bvh-object")
	for _, obj := range objects {
		obj.RotateLocal(0.01, 0.015, 0.005)

		// Add orbital motion within cluster
		x := obj.Transform.Position.X
		z := obj.Transform.Position.Z

		// Get cluster center by finding nearest cluster
		angle := math.Atan2(z, x)
		distance := math.Sqrt(x*x + z*z)

		// Rotate slightly around cluster center
		angle += 0.005
		obj.Transform.Position.X = distance * math.Cos(angle)
		obj.Transform.Position.Z = distance * math.Sin(angle)
	}
}

// AnimateOBB - Animate OBB collision demo
func AnimateOBB(scene *Scene, time float64) {
	// Rotate probe continuously
	probe := scene.FindNode("Probe")
	if probe != nil {
		probe.RotateLocal(0.05, 0.04, 0.02)

		// Move probe in a pattern
		probe.Transform.Position.X = math.Sin(time*0.7) * 15.0
		probe.Transform.Position.Z = math.Cos(time*0.7) * 15.0
	}

	// Rotate OBB boxes slowly to show orientation changes
	boxes := scene.FindNodesByTag("obb-box")
	for i, box := range boxes {
		speed := 0.005 + float64(i)*0.002
		box.RotateLocal(speed, speed*1.2, speed*0.8)
	}
}

// AnimateMeshSimplification - Animate mesh simplification demo
func AnimateMeshSimplification(scene *Scene, time float64) {
	// Rotate all examples so we can see the detail levels
	examples := scene.FindNodesByTag("mesh-example")
	for _, obj := range examples {
		obj.RotateLocal(0.015, 0.02, 0)
	}
}

// AnimateSmoothLOD - Animate smooth LOD transition demo
func AnimateSmoothLOD(scene *Scene, time float64) {
	// Rotate objects to show LOD transitions
	objects := scene.FindNodesByTag("lod-object")
	for i, obj := range objects {
		if i%2 == 0 {
			obj.RotateLocal(0.008, 0.012, 0)
		}
	}

	// Update LOD system
	scene.UpdateLODs()
}

// AnimateCombinedAdvanced - Animate combined advanced demo
func AnimateCombinedAdvanced(scene *Scene, time float64) {
	// Animate LOD objects
	lodObjects := scene.FindNodesByTag("lod-object")
	for i, obj := range lodObjects {
		if i%3 == 0 {
			obj.RotateLocal(0.005, 0.01, 0)
		}
	}

	// Rotate objects with "rotating" tag
	rotating := scene.FindNodesByTag("rotating")
	for _, obj := range rotating {
		// Orbital motion
		x := obj.Transform.Position.X
		z := obj.Transform.Position.Z
		angle := 0.015

		newX := x*math.Cos(angle) - z*math.Sin(angle)
		newZ := x*math.Sin(angle) + z*math.Cos(angle)

		obj.Transform.Position.X = newX
		obj.Transform.Position.Z = newZ

		// Self rotation
		obj.RotateLocal(0.02, 0.03, 0.01)
	}

	// Animate torus meshes
	toruses := scene.FindNodesByTag("torus")
	for i, torus := range toruses {
		switch i {
		case 0:
			torus.RotateLocal(0.01, 0, 0)
		case 1:
			torus.RotateLocal(0, 0.01, 0)
		case 2:
			torus.RotateLocal(0.007, 0.007, 0)
		}
	}

	// Update LOD system
	scene.UpdateLODs()
}

// ============================================================================
// TORUS DEMO
// ============================================================================

// TorusDemo creates a scene with various torus shapes
func TorusDemo(scene *Scene) {
	fmt.Println("=== Torus Demo ===")
	fmt.Println("Showcasing various torus configurations...")

	// Camera setup
	scene.Camera.Transform.SetPosition(0, 15, 40)
	scene.Camera.LookAt(Point{X: 0, Y: 0, Z: 0})

	// Create different torus shapes
	torusConfigs := []struct {
		name        string
		majorRadius float64
		minorRadius float64
		majorSegs   int
		minorSegs   int
		x, y, z     float64
		color       Color
		wireframe   bool
	}{
		{"Torus_Thick", 8.0, 3.0, 32, 16, -15, 0, 0, ColorGreen, false},
		{"Torus_Thin", 8.0, 1.5, 32, 12, 0, 0, 0, ColorBlue, false},
		{"Torus_Wire", 8.0, 2.0, 24, 12, 15, 0, 0, ColorRed, true},
		{"Torus_LowPoly", 6.0, 2.0, 12, 8, -10, -12, 0, ColorYellow, false},
		{"Torus_Ring", 10.0, 0.8, 48, 12, 10, -12, 0, ColorMagenta, false},
	}

	for _, config := range torusConfigs {
		mesh := GenerateTorus(config.majorRadius, config.minorRadius, config.majorSegs, config.minorSegs)
		material := NewMaterial()
		material.DiffuseColor = config.color
		material.Wireframe = config.wireframe
		mesh.Material = material

		node := scene.CreateEmpty(config.name)
		node.Object = mesh
		node.Transform.SetPosition(config.x, config.y, config.z)
		node.AddTag("torus")
	}

	fmt.Println("Top row: thick, thin, wireframe")
	fmt.Println("Bottom row: low-poly, ring")
}

func UpdateTorusDemo(scene *Scene, deltaTime float64) {
	// Rotate all toruses with smooth, varied animations
	toruses := scene.FindNodesByTag("torus")
	for i, node := range toruses {
		// Each torus rotates on different axes with small, smooth increments
		switch i {
		case 0: // Thick - slow X axis rotation
			node.RotateLocal(0.008, 0, 0)
		case 1: // Thin - slow Y axis rotation
			node.RotateLocal(0, 0.008, 0)
		case 2: // Wire - diagonal rotation (X and Y)
			node.RotateLocal(0.006, 0.006, 0)
		case 3: // LowPoly - Z axis rotation
			node.RotateLocal(0, 0, 0.010)
		case 4: // Ring - tumbling (all axes)
			node.RotateLocal(0.005, 0.007, 0.004)
		}
	}
}

// ============================================================================
// ADVANCED DEMOS - Proper Implementations
// ============================================================================

// AdvancedSystemsDemo - LOD and Raycasting demonstration
func AdvancedSystemsDemo(scene *Scene) {
	fmt.Println("=== Advanced Systems Demo ===")
	fmt.Println("Demonstrating LOD system with raycasting...")

	// Create a grid of LOD objects
	gridSize := 5
	spacing := 15.0
	offset := -float64(gridSize-1) * spacing / 2

	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			// Create LOD group for each object
			baseMesh := GenerateSphere(4.0, 20, 20)
			mat := NewMaterial()
			mat.DiffuseColor = Color{
				R: uint8(100 + (x * 30)),
				G: uint8(100 + (z * 30)),
				B: 200,
			}
			baseMesh.Material = mat

			lodGroup := NewLODGroup()
			lodGroup.AddLOD(baseMesh, 40.0)

			medMesh := SimplifyMesh(baseMesh, 0.6)
			medMesh.Material = mat
			lodGroup.AddLOD(medMesh, 80.0)

			lowMesh := SimplifyMesh(baseMesh, 0.3)
			lowMesh.Material = mat
			lodGroup.AddLOD(lowMesh, 150.0)

			name := fmt.Sprintf("LOD_%d_%d", x, z)
			node := scene.CreateEmpty(name)
			node.SetLODGroup(lodGroup)
			node.Transform.SetPosition(
				offset+float64(x)*spacing,
				0,
				offset+float64(z)*spacing,
			)
			node.AddTag("lod-object")
			node.AddTag("pickable")
		}
	}

	// Create a cursor for raycasting
	cursorMat := NewMaterial()
	cursorMat.DiffuseColor = ColorYellow
	cursorMat.Wireframe = true
	cursor := scene.CreateSphere("Cursor", 2, 12, 12, cursorMat)
	cursor.Transform.SetPosition(0, 5, 0)
	cursor.AddTag("cursor")

	fmt.Printf("Created %d LOD objects with 3 detail levels each\n", gridSize*gridSize)
}

// BoundingVolumeDemo - AABB collision detection
func BoundingVolumeDemo(scene *Scene) {
	fmt.Println("=== Bounding Volume Demo ===")
	fmt.Println("Demonstrating AABB collision detection...")

	// Create moving probe
	probeMat := NewMaterial()
	probeMat.DiffuseColor = ColorYellow
	probe := scene.CreateCube("Probe", 5, probeMat)
	probe.Transform.SetPosition(0, 0, 0)
	probe.AddTag("probe")

	// Create static obstacles in a pattern
	positions := []struct{ x, y, z float64 }{
		{-20, 0, -20},
		{20, 0, -20},
		{-20, 0, 20},
		{20, 0, 20},
		{0, 0, 25},
		{0, 0, -25},
		{25, 0, 0},
		{-25, 0, 0},
	}

	for i, pos := range positions {
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue
		name := fmt.Sprintf("Obstacle_%d", i)
		obstacle := scene.CreateCube(name, 6, mat)
		obstacle.Transform.SetPosition(pos.x, pos.y, pos.z)
		obstacle.AddTag("obstacle")
	}

	fmt.Println("Probe will change color when colliding with obstacles")
}

// PerformanceTestDemo - Static LOD performance testing
func PerformanceTestDemo(scene *Scene) {
	fmt.Println("=== Performance Test Demo ===")
	fmt.Println("Testing LOD system with many static objects...")

	// Create a larger grid for performance testing
	gridSize := 10
	spacing := 15.0
	offset := -float64(gridSize-1) * spacing / 2

	objectCount := 0
	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			y := int((x + z) % 3) // Stagger heights

			baseMesh := GenerateSphere(3.0, 16, 16)
			mat := NewMaterial()
			mat.DiffuseColor = Color{
				R: uint8(80 + (x * 17)),
				G: uint8(80 + (z * 17)),
				B: uint8(150 + (y * 30)),
			}
			baseMesh.Material = mat

			lodGroup := NewLODGroup()
			lodGroup.AddLOD(baseMesh, 35.0)

			medMesh := SimplifyMesh(baseMesh, 0.6)
			medMesh.Material = mat
			lodGroup.AddLOD(medMesh, 70.0)

			lowMesh := SimplifyMesh(baseMesh, 0.3)
			lowMesh.Material = mat
			lodGroup.AddLOD(lowMesh, 140.0)

			name := fmt.Sprintf("Perf_%d_%d", x, z)
			node := scene.CreateEmpty(name)
			node.SetLODGroup(lodGroup)
			node.Transform.SetPosition(
				offset+float64(x)*spacing,
				float64(y)*10.0,
				offset+float64(z)*spacing,
			)
			node.AddTag("lod-object")
			objectCount++
		}
	}

	fmt.Printf("Created %d objects for performance testing\n", objectCount)
}

// LineOfSightDemo - Visibility checking between objects
func LineOfSightDemo(scene *Scene) {
	fmt.Println("=== Line of Sight Demo ===")
	fmt.Println("Demonstrating line-of-sight visibility checks...")

	// Create observer (guard)
	guardMat := NewMaterial()
	guardMat.DiffuseColor = ColorRed
	guard := scene.CreateCube("Guard", 5, guardMat)
	guard.Transform.SetPosition(0, 0, 0)
	guard.AddTag("guard")

	// Create target that moves
	targetMat := NewMaterial()
	targetMat.DiffuseColor = ColorGreen
	target := scene.CreateSphere("Target", 3, 16, 16, targetMat)
	target.Transform.SetPosition(30, 0, 0)
	target.AddTag("target")

	// Create obstacles that block line of sight
	obstaclePositions := []struct{ x, y, z float64 }{
		{15, 0, 5},
		{15, 0, -5},
		{-15, 0, 10},
		{-15, 0, -10},
		{0, 0, 20},
	}

	for i, pos := range obstaclePositions {
		mat := NewMaterial()
		mat.DiffuseColor = Color{150, 150, 150}
		name := fmt.Sprintf("Wall_%d", i)
		wall := scene.CreateCube(name, 8, mat)
		wall.Transform.SetPosition(pos.x, pos.y, pos.z)
		wall.Transform.SetScale(1, 2, 1)
		wall.AddTag("obstacle")
	}

	fmt.Println("Guard turns yellow when it can see the target")
}

// OctreeDemo - Spatial partitioning demonstration
func OctreeDemo(scene *Scene) {
	fmt.Println("=== Octree Demo ===")
	fmt.Println("Demonstrating spatial organization (conceptual)...")

	// Create scattered objects in a spatial pattern
	numObjects := 60
	for i := 0; i < numObjects; i++ {
		angle := (float64(i) / float64(numObjects)) * 2 * math.Pi
		radius := 30.0 + math.Sin(float64(i)*0.5)*20.0
		height := math.Cos(float64(i)*0.3) * 30.0

		x := radius * math.Cos(angle)
		z := radius * math.Sin(angle)

		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(100 + i*2),
			G: uint8(200 - i*2),
			B: 200,
		}

		name := fmt.Sprintf("Object_%d", i)
		obj := scene.CreateCube(name, 4, mat)
		obj.Transform.SetPosition(x, height, z)
		obj.AddTag("spatial-object")
	}

	// Create a query sphere
	queryMat := NewMaterial()
	queryMat.DiffuseColor = ColorYellow
	queryMat.Wireframe = true
	query := scene.CreateSphere("QuerySphere", 15, 16, 16, queryMat)
	query.Transform.SetPosition(0, 0, 0)
	query.AddTag("query")

	fmt.Printf("Created %d objects in spatial patterns\n", numObjects)
}

// BVHDemo - Bounding Volume Hierarchy demonstration
func BVHDemo(scene *Scene) {
	fmt.Println("=== BVH Demo ===")
	fmt.Println("Demonstrating hierarchical spatial organization...")

	// Create clusters of objects
	clusterCount := 5
	objectsPerCluster := 12

	for c := 0; c < clusterCount; c++ {
		angle := (float64(c) / float64(clusterCount)) * 2 * math.Pi
		centerX := 40.0 * math.Cos(angle)
		centerZ := 40.0 * math.Sin(angle)

		for i := 0; i < objectsPerCluster; i++ {
			offsetAngle := (float64(i) / float64(objectsPerCluster)) * 2 * math.Pi
			offsetRadius := 8.0

			x := centerX + offsetRadius*math.Cos(offsetAngle)
			z := centerZ + offsetRadius*math.Sin(offsetAngle)
			y := math.Sin(float64(i)*0.5) * 5.0

			mat := NewMaterial()
			mat.DiffuseColor = Color{
				R: uint8(100 + c*30),
				G: uint8(150),
				B: uint8(100 + i*12),
			}

			name := fmt.Sprintf("BVH_%d_%d", c, i)
			obj := scene.CreateCube(name, 3, mat)
			obj.Transform.SetPosition(x, y, z)
			obj.AddTag("bvh-object")
		}
	}

	fmt.Printf("Created %d objects in %d clusters\n",
		clusterCount*objectsPerCluster, clusterCount)
}

// OBBDemo - Oriented Bounding Box collision
func OBBDemo(scene *Scene) {
	fmt.Println("=== OBB Demo ===")
	fmt.Println("Demonstrating OBB (Oriented Bounding Box) collision...")

	// Create rotating probe
	probeMat := NewMaterial()
	probeMat.DiffuseColor = ColorYellow
	probeMat.Wireframe = true
	probe := scene.CreateCube("Probe", 6, probeMat)
	probe.Transform.SetPosition(0, 0, 0)
	probe.AddTag("probe")

	// Create rotated boxes for OBB testing
	positions := []struct{ x, y, z, rx, ry, rz float64 }{
		{-20, 0, 0, 0, 0.5, 0},
		{20, 0, 0, 0.5, 0, 0},
		{0, 0, -20, 0, 0, 0.5},
		{0, 0, 20, 0.3, 0.3, 0},
		{15, 0, 15, 0.2, 0.5, 0.3},
		{-15, 0, -15, 0.4, 0.2, 0.1},
	}

	for i, pos := range positions {
		mat := NewMaterial()
		mat.DiffuseColor = ColorBlue
		name := fmt.Sprintf("OBBBox_%d", i)
		box := scene.CreateCube(name, 7, mat)
		box.Transform.SetPosition(pos.x, pos.y, pos.z)
		box.Transform.SetRotation(pos.rx, pos.ry, pos.rz)
		box.AddTag("obb-box")
	}

	fmt.Println("Rotating probe tests OBB collisions with angled boxes")
}

// MeshSimplificationDemo - QEM mesh simplification
func MeshSimplificationDemo(scene *Scene) {
	fmt.Println("=== Mesh Simplification Demo ===")
	fmt.Println("Comparing different simplification levels...")

	// Show sphere at different LOD levels
	positions := []float64{-30, -15, 0, 15, 30}
	labels := []string{"100%", "75%", "50%", "25%", "10%"}
	ratios := []float64{1.0, 0.75, 0.5, 0.25, 0.1}

	for i, pos := range positions {
		highDetail := GenerateSphere(8.0, 24, 24)
		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(100 + i*35),
			G: 150,
			B: uint8(250 - i*45),
		}

		var mesh *Mesh
		if ratios[i] == 1.0 {
			mesh = highDetail
		} else {
			mesh = SimplifyMesh(highDetail, ratios[i])
		}
		mesh.Material = mat

		node := scene.CreateEmpty(fmt.Sprintf("Simplified_%s", labels[i]))
		node.Object = mesh
		node.Transform.SetPosition(pos, 0, 0)
		node.AddTag("mesh-example")

		fmt.Printf("%s detail: %d triangles\n", labels[i], len(mesh.Indices)/3)
	}
}

// SmoothLODDemo - LOD transitions with smooth blending
func SmoothLODDemo(scene *Scene) {
	fmt.Println("=== Smooth LOD Transitions Demo ===")
	fmt.Println("Demonstrating smooth LOD level transitions...")

	// Create a line of objects at different distances
	numObjects := 8
	spacing := 25.0
	startZ := -100.0

	for i := 0; i < numObjects; i++ {
		baseMesh := GenerateSphere(6.0, 20, 20)
		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(100 + i*15),
			G: uint8(150 + i*10),
			B: 200,
		}
		baseMesh.Material = mat

		// Create LOD group with smooth transitions
		lodGroup := NewLODGroup()
		lodGroup.FadeTransition = true
		lodGroup.TransitionRange = 0.2

		lodGroup.AddLOD(baseMesh, 40.0)

		medMesh := SimplifyMesh(baseMesh, 0.6)
		medMesh.Material = mat
		lodGroup.AddLOD(medMesh, 80.0)

		lowMesh := SimplifyMesh(baseMesh, 0.3)
		lowMesh.Material = mat
		lodGroup.AddLOD(lowMesh, 150.0)

		name := fmt.Sprintf("SmoothLOD_%d", i)
		node := scene.CreateEmpty(name)
		node.SetLODGroup(lodGroup)
		node.Transform.SetPosition(0, 0, startZ+float64(i)*spacing)
		node.AddTag("lod-object")
	}

	fmt.Println("Objects at different distances show smooth LOD transitions")
}

// CombinedAdvancedDemo - All advanced features together
func CombinedAdvancedDemo(scene *Scene) {
	fmt.Println("=== Combined Advanced Demo ===")
	fmt.Println("Showcasing all advanced features together...")

	// Region 1: LOD Grid (left)
	gridSize := 4
	spacing := 20.0
	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			baseMesh := GenerateSphere(4.0, 16, 16)
			mat := NewMaterial()
			mat.DiffuseColor = Color{
				R: uint8(200),
				G: uint8(100 + x*30),
				B: uint8(100 + z*30),
			}
			baseMesh.Material = mat

			lodGroup := NewLODGroup()
			lodGroup.AddLOD(baseMesh, 50.0)
			medMesh := SimplifyMesh(baseMesh, 0.6)
			medMesh.Material = mat
			lodGroup.AddLOD(medMesh, 100.0)

			name := fmt.Sprintf("LOD_%d_%d", x, z)
			node := scene.CreateEmpty(name)
			node.SetLODGroup(lodGroup)
			node.Transform.SetPosition(
				-80+float64(x)*spacing,
				0,
				-40+float64(z)*spacing,
			)
			node.AddTag("lod-object")
		}
	}

	// Region 2: Rotating objects (center)
	for i := 0; i < 12; i++ {
		angle := (float64(i) / 12.0) * 2 * math.Pi
		radius := 30.0

		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(100 + i*10),
			G: 200,
			B: uint8(200 - i*10),
		}

		name := fmt.Sprintf("Rotating_%d", i)
		obj := scene.CreateCube(name, 5, mat)
		obj.Transform.SetPosition(
			radius*math.Cos(angle),
			0,
			radius*math.Sin(angle),
		)
		obj.AddTag("rotating")
	}

	// Region 3: Torus formations (right)
	for i := 0; i < 3; i++ {
		mesh := GenerateTorus(8.0, 2.0, 24, 12)
		mat := NewMaterial()
		mat.DiffuseColor = Color{
			R: uint8(100),
			G: uint8(150 + i*30),
			B: uint8(200),
		}
		if i == 1 {
			mat.Wireframe = true
		}
		mesh.Material = mat

		node := scene.CreateEmpty(fmt.Sprintf("Torus_%d", i))
		node.Object = mesh
		node.Transform.SetPosition(80, float64(i-1)*15, 0)
		node.AddTag("torus")
	}

	fmt.Println("Combined demo with LOD, rotating objects, and torus meshes")
}
