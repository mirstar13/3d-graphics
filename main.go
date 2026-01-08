package main

import (
	"bufio"
	"fmt"
	"math"
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
	DemoStressTest
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
	DemoTorus
)

// RenderMode specifies the rendering approach
type RenderMode int

const (
	RenderModeSingle RenderMode = iota
	RenderModeParallelTiles
	RenderModeParallelJobs
	RenderModeParallelBatched
)

// BackendType specifies the rendering backend
type BackendType int

const (
	BackendTerminal BackendType = iota
	BackendOpenGL
	BackendVulkan
)

// OrientationType. For some reason in opengl yaw axis and y axis are inversed compared to terminal renderer.
// This enum helps to manage that difference.
type OrientationType int

const (
	OrientationTerminal OrientationType = 1
	OrientationOpenGL   OrientationType = -1
	OrientationVulkan   OrientationType = 1
)

// EngineConfig holds engine configuration
type EngineConfig struct {
	Width           int
	Height          int
	FPS             float64
	UseColor        bool
	ShowDebugInfo   bool
	RenderMode      RenderMode
	Backend         BackendType
	NumWorkers      int
	TileSize        int
	EnableProfiling bool
	AAMode          AAMode
}

func main() {
	fmt.Println("=== 3D Engine Demo (with Anti-Aliasing & Vulkan) ===")
	fmt.Println()
	fmt.Println("Select a demo:")
	fmt.Println("  --- Basic Demos ---")
	fmt.Println("  1  - Solar System (planets orbiting)")
	fmt.Println("  2  - Robot Arm (articulated joints)")
	fmt.Println("  3  - Spinning Cubes (3D grid)")
	fmt.Println("  4  - Orbiting Objects (circular motion)")
	fmt.Println("  5  - Wave Grid (sine wave animation)")
	fmt.Println("  6  - Helix (spiral structure)")
	fmt.Println("  7  - Wireframe Demo (mixed rendering)")
	fmt.Println("  8  - Stress Test (200+ objects with LOD)")
	fmt.Println()
	fmt.Println("  --- Advanced Feature Demos ---")
	fmt.Println("  9  - Advanced Systems (LOD, Raycasting)")
	fmt.Println("  10 - Bounding Volume (AABB collision)")
	fmt.Println("  11 - Performance Test (Static LOD)")
	fmt.Println("  12 - Line of Sight (Visibility)")
	fmt.Println("  13 - Octree (Spatial Partitioning)")
	fmt.Println("  14 - BVH (Bounding Volume Hierarchy)")
	fmt.Println("  15 - OBB (Oriented Bounding Boxes)")
	fmt.Println("  16 - Mesh Simplification (QEM)")
	fmt.Println("  17 - Smooth LOD Transitions")
	fmt.Println("  18 - Combined Advanced (All Features)")
	fmt.Println("  19 - Torus Showcase (Mesh Generator)")
	fmt.Println()
	fmt.Print("Enter choice (1-19): ")

	var choice int
	fmt.Scanln(&choice)

	if choice < 1 || choice > 19 {
		fmt.Println("Invalid choice, using Solar System demo")
		choice = 1
	}

	demoType := choice - 1

	// Backend Selection
	fmt.Println()
	fmt.Println("Select rendering backend:")
	fmt.Println("  1 - Terminal (ASCII/ANSI)")
	fmt.Println("  2 - OpenGL (Hardware Accelerated - Full 3D)")
	fmt.Println("  3 - Vulkan (Hardware Accelerated - Advanced)")
	fmt.Println()
	fmt.Print("Enter backend choice (1-3, default=1): ")

	fmt.Scanln(&choice)

	if choice < 1 || choice > 3 {
		fmt.Println("Invalid choice, using Terminal Backend")
		choice = 1
	}

	backendChoice := BackendType(choice - 1)

	// Configure rendering mode
	fmt.Println()
	fmt.Println("Select CPU processing mode:")
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

	// Anti-aliasing configuration (Terminal only)
	var aaMode AAMode = AANone
	if backendChoice == BackendTerminal {
		fmt.Println()
		fmt.Println("Select anti-aliasing mode (Terminal Only):")
		fmt.Println("  1 - None (fastest)")
		fmt.Println("  2 - FXAA (Fast Approximate)")
		fmt.Println("  3 - MSAA 2x (Multi-Sample 2x)")
		fmt.Println("  4 - MSAA 4x (Multi-Sample 4x)")
		fmt.Println("  5 - SSAA (Super-Sample, best quality)")
		fmt.Println()
		fmt.Print("Enter AA mode (1-5, default=1): ")

		var aaChoice int
		fmt.Scanln(&aaChoice)

		switch aaChoice {
		case 2:
			aaMode = AAFXAA
		case 3:
			aaMode = AAMSAA2x
		case 4:
			aaMode = AAMSAA4x
		case 5:
			aaMode = AASSAA
		default:
			aaMode = AANone
		}
	}

	config := EngineConfig{
		Width:           223,
		Height:          51,
		FPS:             60.0,
		UseColor:        true,
		ShowDebugInfo:   true,
		RenderMode:      renderMode,
		Backend:         backendChoice,
		NumWorkers:      4,
		TileSize:        32,
		EnableProfiling: true,
		AAMode:          aaMode,
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
	fmt.Printf("Backend: %s | Mode: %s | Workers: %d\n",
		getBackendName(config.Backend),
		getRenderModeName(renderMode),
		config.NumWorkers)
	fmt.Println("Starting in 3 seconds...")
	time.Sleep(3 * time.Second)

	// Run the engine
	runEngine(demoType, config)
}

func getBackendName(backend BackendType) string {
	switch backend {
	case BackendTerminal:
		return "Terminal"
	case BackendOpenGL:
		return "OpenGL"
	case BackendVulkan:
		return "Vulkan"
	default:
		return "Unknown"
	}
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

func getAAModeName(mode AAMode) string {
	switch mode {
	case AANone:
		return "None"
	case AAFXAA:
		return "FXAA"
	case AAMSAA2x:
		return "MSAA 2x"
	case AAMSAA4x:
		return "MSAA 4x"
	case AASSAA:
		return "SSAA"
	default:
		return "Unknown"
	}
}

func runEngine(demoType int, config EngineConfig) {
	// 1. Select Base Renderer
	var baseRenderer Renderer
	var orientation OrientationType

	switch config.Backend {
	case BackendTerminal:
		// Use Terminal Renderer
		writer := bufio.NewWriter(os.Stdout)
		termRenderer := NewTerminalRenderer(writer, config.Height, config.Width)
		termRenderer.SetUseColor(config.UseColor)
		termRenderer.SetShowDebugInfo(config.ShowDebugInfo)
		baseRenderer = termRenderer

		orientation = OrientationTerminal
	case BackendOpenGL:
		// Use the CGO-based OpenGL renderer
		baseRenderer = NewOpenGLRenderer(800, 600)
		baseRenderer.SetUseColor(config.UseColor)
		baseRenderer.SetShowDebugInfo(config.ShowDebugInfo)

		orientation = OrientationOpenGL
	case BackendVulkan:
		// Use the CGO-based Vulkan renderer
		baseRenderer = NewVulkanRenderer(800, 600)
		baseRenderer.SetUseColor(config.UseColor)
		baseRenderer.SetShowDebugInfo(config.ShowDebugInfo)

		orientation = OrientationVulkan
	default:
		fmt.Println("Unsupported backend, exiting.")
		return
	}

	// 2. Wrap with Profiler if enabled
	var profiler *Profiler
	if config.EnableProfiling {
		profiler = NewProfiler(60)
		fmt.Println("Profiling enabled")
	}

	// 3. Wrap with AA Renderer (Terminal Only)
	var effectiveBaseRenderer Renderer = baseRenderer
	if config.Backend == BackendTerminal && config.AAMode != AANone {
		aaRenderer := NewAARenderer(baseRenderer, config.AAMode)
		effectiveBaseRenderer = aaRenderer
		fmt.Printf("Anti-aliasing enabled: %s\n", getAAModeName(config.AAMode))
	}

	// 4. Wrap with Parallel Renderer (if selected)
	var finalRenderer Renderer

	switch config.RenderMode {
	case RenderModeParallelTiles:
		finalRenderer = NewParallelRenderer(effectiveBaseRenderer, config.NumWorkers, config.TileSize)
	case RenderModeParallelJobs:
		finalRenderer = NewJobBasedRenderer(effectiveBaseRenderer, config.NumWorkers)
	case RenderModeParallelBatched:
		finalRenderer = NewParallelRenderer(effectiveBaseRenderer, config.NumWorkers, config.TileSize)
	default:
		finalRenderer = effectiveBaseRenderer
	}

	// Initialize renderer
	if err := finalRenderer.Initialize(); err != nil {
		fmt.Printf("Failed to initialize renderer: %v\n", err)
		return
	}
	defer finalRenderer.Shutdown()

	// Create scene
	scene := NewScene()

	// Configure camera
	configureCamera(scene.Camera, demoType, orientation)
	finalRenderer.SetCamera(scene.Camera)

	// Setup lighting
	lightingSystem := setupLighting(scene.Camera, demoType)
	finalRenderer.SetLightingSystem(lightingSystem)

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

	// Clear screen (Terminal only)
	if config.Backend == BackendTerminal {
		fmt.Print("\033[2J\033[H")
	}

	fps := config.FPS
	startTime := time.Now()
	// lastStatsTime initialized to now
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

		finalRenderer.BeginFrame()

		// Update phase
		if profiler != nil {
			profiler.BeginUpdate()
		}

		input := inputManager.GetInputState()
		if input.Quit {
			if config.Backend == BackendTerminal {
				fmt.Print("\033[2J\033[H")
			}
			fmt.Println("Exiting...")
			return
		}

		cameraController.Update(input, orientation)
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

		// Use the configured renderer
		if config.RenderMode == RenderModeParallelBatched {
			if pr, ok := finalRenderer.(*ParallelRenderer); ok {
				pr.RenderBatched(scene)
			} else {
				finalRenderer.RenderScene(scene)
			}
		} else if config.RenderMode == RenderModeParallelJobs {
			if jr, ok := finalRenderer.(*JobBasedRenderer); ok {
				jr.RenderSceneJobs(scene)
			} else {
				finalRenderer.RenderScene(scene)
			}
		} else {
			finalRenderer.RenderScene(scene)
		}

		if profiler != nil {
			profiler.EndRender()
		}

		// Present phase
		if profiler != nil {
			profiler.BeginPresent()
		}

		finalRenderer.EndFrame()
		finalRenderer.Present()

		if profiler != nil {
			profiler.EndPresent()
			profiler.EndFrame()
		}

		// Print stats every 2 seconds
		if profiler != nil && time.Since(lastStatsTime).Seconds() >= 2.0 {
			stats := profiler.GetAverageStats()

			if config.Backend == BackendTerminal {
				fmt.Printf("\n%s\n", stats.String())
			} else {
				if stats.TotalTime > 0.000001 {
					currentFPS := 1.0 / stats.TotalTime
					fmt.Printf("FPS: %.2f | Frame Time: %.4f ms\n", currentFPS, stats.TotalTime*1000)
				} else {
					fmt.Printf("FPS: >9999 | Frame Time: <0.001 ms\n")
				}
			}
			lastStatsTime = time.Now()

			fmt.Printf("\033[1A\033")
		}
	}
}

// Helper functions (same as original)
func configureCamera(camera *Camera, demoType int, orientation OrientationType) {
	camera.DZ = 0.0
	camera.Near = 0.5
	camera.FOV = Point{X: 60.0, Y: 30.0, Z: 0}

	switch demoType {
	case DemoSolarSystem:
		camera.Transform.SetPosition(0, 30, -100*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	case DemoRobotArm:
		camera.Transform.SetPosition(0, 10, -60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0
	case DemoSpinningCubes:
		camera.Transform.SetPosition(0, 0, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	case DemoOrbitingObjects:
		camera.Transform.SetPosition(0, 20, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0
	case DemoWaveGrid:
		camera.Transform.SetPosition(0, 30, -50*float64(orientation))
		camera.Transform.SetRotation(0, 0, (float64(orientation)*math.Pi)/6)
		camera.Far = 200.0
	case DemoHelix:
		camera.Transform.SetPosition(0, 0, -60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0
	case DemoWireframe:
		camera.Transform.SetPosition(0, 0, -50*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0
	case DemoStressTest:
		camera.Transform.SetPosition(0, 50, -150)
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 500.0
	case DemoAdvancedSystems:
		camera.Transform.SetPosition(0, 30, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	case DemoBoundingVolume:
		camera.Transform.SetPosition(0, 20, -60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0
	case DemoPerformanceTest:
		camera.Transform.SetPosition(0, 40, -120*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0
	case DemoLineOfSight:
		camera.Transform.SetPosition(0, 30, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	case DemoOctree:
		camera.Transform.SetPosition(0, 50, -100*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0
	case DemoBVH:
		camera.Transform.SetPosition(0, 50, -100*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0
	case DemoOBB:
		camera.Transform.SetPosition(0, 25, -70*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	case DemoMeshSimplification:
		camera.Transform.SetPosition(0, 20, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	case DemoSmoothLOD:
		camera.Transform.SetPosition(0, 50, -150*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 500.0
	case DemoCombinedAdvanced:
		camera.Transform.SetPosition(0, 60, -180*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 600.0
	case DemoTorus:
		camera.Transform.SetPosition(0, 15, -40*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0
	default:
		camera.Transform.SetPosition(0, 10, -60)
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	}
}

func configureCameraController(controller *CameraController, demoType int) {
	switch demoType {
	case DemoSolarSystem:
		controller.SetOrbitRadius(120.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoRobotArm:
		controller.SetOrbitRadius(60.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoSpinningCubes:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoOrbitingObjects:
		controller.SetOrbitRadius(90.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoWaveGrid:
		controller.SetOrbitRadius(70.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoHelix:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoStressTest:
		controller.SetOrbitRadius(200.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(50.0)
	case DemoAdvancedSystems:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoBoundingVolume:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoPerformanceTest:
		controller.SetOrbitRadius(150.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(40.0)
	case DemoLineOfSight:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoOctree:
		controller.SetOrbitRadius(120.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(50.0)
	case DemoBVH:
		controller.SetOrbitRadius(120.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(50.0)
	case DemoOBB:
		controller.SetOrbitRadius(90.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoMeshSimplification:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, 0)
	case DemoSmoothLOD:
		controller.SetOrbitRadius(180.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(50.0)
	case DemoCombinedAdvanced:
		controller.SetOrbitRadius(220.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(60.0)
	case DemoTorus:
		controller.SetOrbitRadius(60.0)
		controller.SetOrbitCenter(0, 0, 0)
	default:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
	}

	controller.EnableAutoOrbit(false)
}

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
	case DemoTorus:
		TorusDemo(scene)
	}
}

func animateSceneDemo(scene *Scene, demoType int, time float64) {
	dt := 1.0 / 60.0 // Approximate delta time

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
	case DemoTorus:
		UpdateTorusDemo(scene, dt)
	}
}
