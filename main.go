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
)

func main() {
	fmt.Println("=== 3D Engine Demo (Refactored Architecture) ===")
	fmt.Println()
	fmt.Println("Select a demo:")
	fmt.Println("  1 - Solar System (planets orbiting)")
	fmt.Println("  2 - Robot Arm (articulated joints)")
	fmt.Println("  3 - Spinning Cubes (3D grid)")
	fmt.Println("  4 - Orbiting Objects (circular motion)")
	fmt.Println("  5 - Wave Grid (sine wave animation)")
	fmt.Println("  6 - Helix (spiral structure)")
	fmt.Println("  7 - Wireframe Demo (mixed rendering)")
	fmt.Println()
	fmt.Print("Enter choice (1-7): ")

	var choice int
	fmt.Scanln(&choice)

	if choice < 1 || choice > 7 {
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

	// Create terminal renderer
	writer := bufio.NewWriter(os.Stdout)
	renderer := NewTerminalRenderer(writer, 51, 223)
	renderer.SetUseColor(true)

	// Initialize renderer
	if err := renderer.Initialize(); err != nil {
		panic(err)
	}
	defer renderer.Shutdown()

	// Create scene
	scene := NewScene()

	// Configure camera
	configureCamera(scene.Camera, demoType)
	renderer.SetCamera(scene.Camera)

	// Setup lighting
	lightingSystem := setupLighting(scene.Camera, demoType)
	renderer.SetLightingSystem(lightingSystem)

	// Create material
	material := NewMaterial()
	material.DiffuseColor = Color{200, 180, 150}
	material.SpecularColor = ColorWhite
	material.Shininess = 64.0
	material.SpecularStrength = 0.8

	// Build scene
	buildScene(scene, demoType, material)

	// Create input manager and camera controller
	inputManager := NewSilentInputManager()
	inputManager.Start()
	defer inputManager.Stop()

	cameraController := NewCameraController(scene.Camera)
	configureCameraController(cameraController, demoType)

	// Clear screen
	fmt.Print("\033[2J\033[H")

	fps := 60.0
	startTime := time.Now()

	// Update function
	updateFunc := func(scene *Scene, dt float64) {
		input := inputManager.GetInputState()

		if input.Quit {
			fmt.Print("\033[2J\033[H")
			fmt.Println("Exiting...")
			inputManager.Stop()
			renderer.Shutdown()
			os.Exit(0)
		}

		cameraController.Update(input)

		elapsedTime := time.Since(startTime).Seconds()
		animateSceneDemo(scene, demoType, elapsedTime)

		// Animate lights
		if lightingSystem != nil {
			for _, light := range lightingSystem.Lights {
				light.Rotate('y', 0.01)
			}
		}

		inputManager.ClearKeys()
	}

	// Start render loop using the renderer
	renderer.RenderLoop(scene, fps, updateFunc)
}

// configureCamera sets up camera based on demo
func configureCamera(camera *Camera, demoType int) {
	camera.DZ = 0.0
	camera.Near = 0.5
	camera.FOV = Point{X: 60.0, Y: 30.0, Z: 0}

	switch demoType {
	case DemoSolarSystem:
		camera.Transform.SetPosition(0, 30, -100)
		camera.Transform.SetRotation(-0.2, 0, 0)
		camera.Far = 300.0

	case DemoRobotArm:
		camera.Transform.SetPosition(0, 10, -60)
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	case DemoSpinningCubes:
		camera.Transform.SetPosition(0, 0, -80)
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0

	case DemoOrbitingObjects:
		camera.Transform.SetPosition(0, 20, -80)
		camera.Transform.SetRotation(-0.2, 0, 0)
		camera.Far = 200.0

	case DemoWaveGrid:
		camera.Transform.SetPosition(0, 30, -50)
		camera.Transform.SetRotation(-0.4, 0, 0)
		camera.Far = 200.0

	case DemoHelix:
		camera.Transform.SetPosition(0, 0, -60)
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	case DemoWireframe:
		camera.Transform.SetPosition(0, 0, -50)
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	default:
		camera.Transform.SetPosition(0, 10, -60)
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	}
}

// configureCameraController sets controller parameters
func configureCameraController(controller *CameraController, demoType int) {
	switch demoType {
	case DemoSolarSystem:
		controller.SetOrbitRadius(120.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.EnableAutoOrbit(true)

	case DemoRobotArm:
		controller.SetOrbitRadius(60.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.EnableAutoOrbit(true)

	case DemoSpinningCubes:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.EnableAutoOrbit(true)

	case DemoOrbitingObjects:
		controller.SetOrbitRadius(90.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.EnableAutoOrbit(true)

	case DemoWaveGrid:
		controller.SetOrbitRadius(70.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.EnableAutoOrbit(true)

	case DemoHelix:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.EnableAutoOrbit(true)

	default:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.EnableAutoOrbit(true)
	}
}

// setupLighting creates lighting based on demo
func setupLighting(camera *Camera, demoType int) *LightingSystem {
	switch demoType {
	case DemoSolarSystem:
		return setupScenario3(camera)
	case DemoRobotArm:
		return setupScenario5(camera)
	default:
		return setupScenario5(camera)
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
	}
}

// animateSceneDemo animates the scene
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
	case DemoWireframe:
		AnimateWireframe(scene, time)
	}
}
