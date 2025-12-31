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
	DemoStressTest // New stress test demo
)

// RenderMode specifies the rendering approach
type RenderMode int

const (
	RenderModeSingle RenderMode = iota
	RenderModeParallelTiles
	RenderModeParallelJobs
	RenderModeParallelBatched
)

// EngineConfig holds engine configuration
type EngineConfig struct {
	Width           int
	Height          int
	FPS             float64
	UseColor        bool
	ShowDebugInfo   bool
	RenderMode      RenderMode
	NumWorkers      int
	TileSize        int
	EnableProfiling bool
}

func main() {
	fmt.Println("=== 3D Engine Demo (Parallel Rendering Integrated) ===")
	fmt.Println()
	fmt.Println("Select a demo:")
	fmt.Println("  1 - Solar System (planets orbiting)")
	fmt.Println("  2 - Robot Arm (articulated joints)")
	fmt.Println("  3 - Spinning Cubes (3D grid)")
	fmt.Println("  4 - Orbiting Objects (circular motion)")
	fmt.Println("  5 - Wave Grid (sine wave animation)")
	fmt.Println("  6 - Helix (spiral structure)")
	fmt.Println("  7 - Wireframe Demo (mixed rendering)")
	fmt.Println("  8 - Stress Test (200+ objects with LOD)")
	fmt.Println()
	fmt.Print("Enter choice (1-8): ")

	var choice int
	fmt.Scanln(&choice)

	if choice < 1 || choice > 8 {
		fmt.Println("Invalid choice, using Solar System demo")
		choice = 1
	}

	demoType := choice - 1

	// Configure rendering mode
	fmt.Println()
	fmt.Println("Select rendering mode:")
	fmt.Println("  1 - Single-threaded (default)")
	fmt.Println("  2 - Parallel Tiles (best for large scenes)")
	fmt.Println("  3 - Parallel Jobs (best for many objects)")
	fmt.Println("  4 - Parallel Batched (best for similar materials)")
	fmt.Println()
	fmt.Print("Enter rendering mode (1-4, default=1): ")

	var renderChoice int
	fmt.Scanln(&renderChoice)

	renderMode := RenderModeSingle
	switch renderChoice {
	case 2:
		renderMode = RenderModeParallelTiles
	case 3:
		renderMode = RenderModeParallelJobs
	case 4:
		renderMode = RenderModeParallelBatched
	default:
		renderMode = RenderModeSingle
	}

	// Create engine config
	config := EngineConfig{
		Width:           223,
		Height:          51,
		FPS:             60.0,
		UseColor:        true,
		ShowDebugInfo:   true,
		RenderMode:      renderMode,
		NumWorkers:      4, // Default to 4 workers
		TileSize:        32,
		EnableProfiling: true,
	}

	fmt.Println()
	fmt.Println("Controls:")
	fmt.Println("  WASD     - Move camera")
	fmt.Println("  Q/E      - Move up/down")
	fmt.Println("  IJKL     - Rotate camera")
	fmt.Println("  R        - Reset camera")
	fmt.Println("  +/-      - Speed control")
	fmt.Println("  X or ESC - Quit")
	fmt.Println()
	fmt.Printf("Mode: %s with %d workers\n", getRenderModeName(renderMode), config.NumWorkers)
	fmt.Println("Starting in 3 seconds...")
	time.Sleep(3 * time.Second)

	// Run the engine
	runEngine(demoType, config)
}

func getRenderModeName(mode RenderMode) string {
	switch mode {
	case RenderModeSingle:
		return "Single-threaded"
	case RenderModeParallelTiles:
		return "Parallel Tiles"
	case RenderModeParallelJobs:
		return "Parallel Jobs"
	case RenderModeParallelBatched:
		return "Parallel Batched"
	default:
		return "Unknown"
	}
}

func runEngine(demoType int, config EngineConfig) {
	// Create terminal renderer
	writer := bufio.NewWriter(os.Stdout)
	baseRenderer := NewTerminalRenderer(writer, config.Height, config.Width)
	baseRenderer.SetUseColor(config.UseColor)
	baseRenderer.SetShowDebugInfo(config.ShowDebugInfo)

	// Wrap with profiler if enabled
	var profiler *Profiler
	if config.EnableProfiling {
		profiler = NewProfiler(60) // Keep 60 frames of history
		fmt.Println("Profiling enabled")
	}

	// Wrap with parallel renderer if needed
	var renderer Renderer
	var parallelRenderer *ParallelRenderer
	var jobRenderer *JobBasedRenderer

	switch config.RenderMode {
	case RenderModeParallelTiles:
		parallelRenderer = NewParallelRenderer(baseRenderer, config.NumWorkers, config.TileSize)
		renderer = baseRenderer // Still use base for interface
	case RenderModeParallelJobs:
		jobRenderer = NewJobBasedRenderer(baseRenderer, config.NumWorkers)
		renderer = baseRenderer
	case RenderModeParallelBatched:
		parallelRenderer = NewParallelRenderer(baseRenderer, config.NumWorkers, config.TileSize)
		renderer = baseRenderer
	default:
		renderer = baseRenderer
	}

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

	fps := config.FPS
	startTime := time.Now()
	frameCount := 0
	lastStatsTime := time.Now()

	// Main render loop
	dt := 1.0 / fps
	ticker := time.NewTicker(time.Duration(dt*1000) * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C

		// Profiling: Begin frame
		if profiler != nil {
			profiler.BeginFrame()
		}

		// Update phase
		if profiler != nil {
			profiler.BeginUpdate()
		}

		input := inputManager.GetInputState()
		if input.Quit {
			fmt.Print("\033[2J\033[H")
			fmt.Println("Exiting...")
			return
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

		// Update scene
		scene.Update(dt)

		if profiler != nil {
			profiler.EndUpdate()
		}

		inputManager.ClearKeys()

		// Render phase
		if profiler != nil {
			profiler.BeginRender()
		}

		// Choose rendering path based on mode
		switch config.RenderMode {
		case RenderModeParallelTiles:
			parallelRenderer.RenderSceneParallel(scene)
		case RenderModeParallelJobs:
			jobRenderer.RenderSceneJobs(scene)
		case RenderModeParallelBatched:
			parallelRenderer.RenderBatched(scene)
		default:
			renderer.RenderScene(scene)
		}

		if profiler != nil {
			profiler.EndRender()
		}

		// Present phase
		if profiler != nil {
			profiler.BeginPresent()
		}

		renderer.Present()

		if profiler != nil {
			profiler.EndPresent()
			profiler.EndFrame()
		}

		frameCount++

		// Print stats every 2 seconds
		if profiler != nil && time.Since(lastStatsTime).Seconds() >= 2.0 {
			stats := profiler.GetAverageStats()
			fmt.Printf("\n%s\n", stats.String())
			lastStatsTime = time.Now()
		}
	}
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

	case DemoStressTest:
		camera.Transform.SetPosition(0, 50, -150)
		camera.Transform.SetRotation(-0.3, 0, 0)
		camera.Far = 500.0

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

	case DemoStressTest:
		controller.SetOrbitRadius(200.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(50.0)
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
	case DemoStressTest:
		StressTestDemo(scene)
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
	case DemoStressTest:
		AnimateStressTest(scene, time)
	}
}
