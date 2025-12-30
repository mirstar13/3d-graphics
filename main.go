package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

const (
	DemoSolarSystem = iota
	DemoRobotArm
	DemoSpinningCubes
	DemoOrbitingObjects
	DemoWaveGrid
	DemoHelix
	DemoWireframe
	DemoAdvancedSystems
	DemoBoundingVolume
	DemoPerformanceTest
	DemoLineOfSight
	DemoOctree
	DemoBVH
	DemoOBB
	DemoMeshSimplification
	DemoSmoothLOD
	DemoCombinedAdvanced
	DemoStressTest
)

func main() {
	fmt.Println("=== 3D Engine Demo ===")
	fmt.Println()
	fmt.Println("Select a demo:")
	fmt.Println("  1 - Solar System (planets orbiting)")
	fmt.Println("  2 - Robot Arm (articulated joints)")
	fmt.Println("  3 - Spinning Cubes (3D grid)")
	fmt.Println("  4 - Orbiting Objects (circular motion)")
	fmt.Println("  5 - Wave Grid (sine wave animation)")
	fmt.Println("  6 - Helix (spiral structure)")
	fmt.Println("  7 - Wireframe Demo (mixed rendering)")
	fmt.Println("  8 - Advanced Systems Demo")
	fmt.Println("  9 - Bounding Volume Demo")
	fmt.Println(" 10 - Performance Test Demo")
	fmt.Println(" 11 - Line of Sight Demo")
	fmt.Println(" 12 - Octree Demo")
	fmt.Println(" 13 - BVH Demo")
	fmt.Println(" 14 - OBB Demo")
	fmt.Println(" 15 - Mesh Simplification Demo")
	fmt.Println(" 16 - Smooth LOD Demo")
	fmt.Println(" 17 - Combined Advanced Demo")
	fmt.Println(" 18 - Stress Test Demo")
	fmt.Println()
	fmt.Print("Enter choice (1-11): ")

	var choice int
	fmt.Scanln(&choice)

	if choice < 1 || choice > 18 {
		fmt.Println("Invalid choice, using Solar System demo")
		choice = 1
	}

	demoType := choice - 1

	fmt.Println()
	fmt.Println("Controls:")
	fmt.Println("  WASD     - Move camera")
	fmt.Println("  Q/E      - Move up/down")
	fmt.Println("  IJKL     - Rotate camera")
	fmt.Println("  R        - Reset camera")
	fmt.Println("  +/-      - Speed control")
	fmt.Println("  X or ESC - Quit")
	fmt.Println()
	fmt.Println("Starting in 3 seconds...")
	time.Sleep(3 * time.Second)

	// Create renderer
	writer := bufio.NewWriter(os.Stdout)
	renderer := NewRenderer(writer, 51, 223)
	renderer.SetUseColor(true)

	// Create scene
	scene := NewScene()

	// Configure camera based on demo
	configureCamera(scene.Camera, demoType)

	// Setup lighting
	lightingSystem := setupLighting(scene.Camera, demoType)
	renderer.SetLightingSystem(lightingSystem)

	// Create material
	material := NewMaterial()
	material.DiffuseColor = Color{200, 180, 150}
	material.SpecularColor = ColorWhite
	material.Shininess = 64.0
	material.SpecularStrength = 0.8

	// Build scene based on choice
	buildScene(scene, demoType, material)

	// Create camera controller
	inputManager := NewSilentInputManager()
	inputManager.Start()
	defer inputManager.Stop()

	cameraController := NewCameraController(scene.Camera)

	// Configure camera controller based on demo
	switch demoType {
	case DemoSolarSystem:
		cameraController.SetOrbitRadius(100.0)
	case DemoRobotArm:
		cameraController.SetOrbitRadius(60.0)
	case DemoSpinningCubes:
		cameraController.SetOrbitRadius(80.0)
	case DemoOrbitingObjects:
		cameraController.SetOrbitRadius(70.0)
	case DemoWaveGrid:
		cameraController.SetOrbitRadius(60.0)
	case DemoHelix:
		cameraController.SetOrbitRadius(70.0)
	}

	// Clear screen
	fmt.Print("\033[2J\033[H")

	fps := 60.0
	startTime := time.Now()

	// Update function
	updateFunc := func(scene *Scene, dt float64) {
		input := inputManager.GetInputState()

		// Check for quit
		if input.Quit {
			fmt.Print("\033[2J\033[H")
			fmt.Println("Exiting...")
			inputManager.Stop()
			os.Exit(0)
		}

		// Update camera
		cameraController.Update(input)

		// Animate based on demo type
		elapsedTime := time.Since(startTime).Seconds()
		animateSceneDemo(scene, demoType, elapsedTime)

		// Animate lights
		if renderer.LightingSystem != nil {
			for _, light := range renderer.LightingSystem.Lights {
				light.Rotate('y', 0.01)
			}
		}

		inputManager.ClearKeys()
	}

	// Start render loop
	renderer.RenderLoop(scene, fps, updateFunc)
}

// configureCamera sets up camera based on demo
func configureCamera(camera *Camera, demoType int) {
	camera.DZ = 0.0
	camera.Near = 0.5

	switch demoType {
	case DemoSolarSystem:
		camera.Far = 200.0
	case DemoRobotArm:
		camera.Far = 150.0
	case DemoSpinningCubes:
		camera.Far = 200.0
	default:
		camera.Far = 150.0
	}
}

// setupLighting creates lighting based on demo
func setupLighting(camera *Camera, demoType int) *LightingSystem {
	switch demoType {
	case DemoSolarSystem:
		return setupScenario3(camera) // Warm sun-like lighting
	case DemoRobotArm:
		return setupScenario5(camera) // Studio lighting
	default:
		return setupScenario5(camera) // Default studio lighting
	}
}

// buildScene creates the scene based on demo type
func buildScene(scene *Scene, demoType int, material Material) {
	switch demoType {
	case DemoSolarSystem:
		SolarSystemDemo(scene)
	case DemoRobotArm:
		RobotArmDemo(scene, material)
	case DemoSpinningCubes:
		SpinningCubesDemo(scene, material)
	case DemoOrbitingObjects:
		OrbitingObjectsDemo(scene, material)
	case DemoWaveGrid:
		WaveGridDemo(scene, material)
	case DemoHelix:
		HelixDemo(scene, material)
	case DemoWireframe:
		WireframeDemo(scene, material)
	case DemoAdvancedSystems:
		AdvancedSystemsDemo(scene)
	case DemoBoundingVolume:
		BoundingVolumeDemo(scene)
	case DemoPerformanceTest:
		PerformanceTestDemo(scene)
	case DemoLineOfSight:
		LineOfSightDemo(scene)
	case DemoOctree:
		OctreeDemo(scene)
	case DemoBVH:
		BVHDemo(scene)
	case DemoOBB:
		OBBDemo(scene)
	case DemoMeshSimplification:
		MeshSimplificationDemo(scene)
	case DemoSmoothLOD:
		SmoothLODDemo(scene)
	case DemoCombinedAdvanced:
		CombinedAdvancedDemo(scene)
	case DemoStressTest:
		StressTestDemo(scene)
	}
}

// animateScene animates the scene based on demo type
func animateSceneDemo(scene *Scene, demoType int, time float64) {
	switch demoType {
	case DemoSolarSystem:
		AnimateSolarSystem(scene)
	case DemoRobotArm:
		AnimateRobotArm(scene, time)
	case DemoSpinningCubes:
		AnimateSpinningCubes(scene)
	case DemoOrbitingObjects:
		AnimateOrbitingObjects(scene)
	case DemoWaveGrid:
		AnimateWaveGrid(scene, time)
	case DemoHelix:
		AnimateHelix(scene, time)
	case DemoWireframe: // NEW
		AnimateWireframe(scene, time)
	case DemoAdvancedSystems:
		AnimateAdvancedSystems(scene, time)
	case DemoBoundingVolume:
		AnimateBoundingVolume(scene, time)
	case DemoPerformanceTest:
		AnimatePerformanceTest(scene, time)
	case DemoLineOfSight:
		AnimateLineOfSight(scene, time)
	case DemoOctree:
		AnimateOctree(scene, time)
	case DemoBVH:
		AnimateBVH(scene, time)
	case DemoOBB:
		AnimateOBB(scene, time)
	case DemoMeshSimplification:
		AnimateMeshSimplification(scene, time)
	case DemoSmoothLOD:
		AnimateSmoothLOD(scene, time)
	case DemoCombinedAdvanced:
		AnimateCombinedAdvanced(scene, time)
	case DemoStressTest:
		AnimateStressTest(scene, time)
	}
}
