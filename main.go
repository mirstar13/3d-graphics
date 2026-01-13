package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"time"
)

const (
	DemoBasicGeometry = iota
	DemoMeshGenerators
	DemoLightingShowcase
	DemoMaterialShowcase
	DemoTransformHierarchy
	DemoLODSystem
	DemoSpatialPartitioning
	DemoCollisionPhysics
	DemoAdvancedRendering
	DemoPerformanceTest
	DemoAdvancedFeatures
	DemoTextureShowcase
	DemoShadowMapping
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
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to file")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Printf("could not create CPU profile: %v\n", err)
			return
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Printf("could not start CPU profile: %v\n", err)
			return
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("CPU profiling enabled, writing to %s\n", *cpuprofile)
	}

	if *memprofile != "" {
		defer func() {
			f, err := os.Create(*memprofile)
			if err != nil {
				fmt.Printf("could not create memory profile: %v\n", err)
				return
			}
			defer f.Close()
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Printf("could not write memory profile: %v\n", err)
			}
			fmt.Printf("Memory profile written to %s\n", *memprofile)
		}()
	}

	fmt.Println("=== 3D Engine - Complete Feature Showcase ===")
	fmt.Println()
	fmt.Println("Select a demo:")
	fmt.Println("  === ORGANIZED FEATURE DEMONSTRATIONS ===")
	fmt.Println("  1  - Basic Geometry (Points, Lines, Triangles, Quads, Circles)")
	fmt.Println("  2  - Mesh Generators (Cube, Sphere, Torus with indexed geometry)")
	fmt.Println("  3  - Lighting Showcase (10 different lighting scenarios)")
	fmt.Println("  4  - Material Showcase (Matte, Glossy, Wireframe, Combined)")
	fmt.Println("  5  - Transform Hierarchy (Scene graph, solar system, robot arm)")
	fmt.Println("  6  - LOD System (Automatic level-of-detail switching)")
	fmt.Println("  7  - Spatial Partitioning (Octree & BVH demonstrations)")
	fmt.Println("  8  - Collision & Physics (AABB, OBB, Raycasting)")
	fmt.Println("  9  - Advanced Rendering (Anti-aliasing, Clipping, Frustum)")
	fmt.Println("  10 - Performance Test (Stress test with many objects)")
	fmt.Println("  11 - Advanced Features (PBR, Textures, Shadows, Instancing)")
	fmt.Println("  12 - Texture Showcase (UV mapping, procedural textures)")
	fmt.Println("  13 - Shadow Mapping (Real-time shadows with PCF)")
	fmt.Println()
	fmt.Print("Enter choice (1-13): ")

	var choice int
	fmt.Scanln(&choice)

	if choice < 1 || choice > 13 {
		fmt.Println("Invalid choice, using Basic Geometry demo")
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
	var inputManager InputManager

	switch config.Backend {
	case BackendTerminal:
		silentInput := NewTerminalInputManager()
		silentInput.Start()
		defer silentInput.Stop()
		inputManager = silentInput

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
		baseRenderer.Initialize()
		baseRenderer.SetUseColor(config.UseColor)
		baseRenderer.SetShowDebugInfo(config.ShowDebugInfo)

		if glRenderer, ok := baseRenderer.(*OpenGLRenderer); ok {
			inputManager = NewGLFWInputManager(glRenderer.GetWindow())
		}

		orientation = OrientationOpenGL
	case BackendVulkan:
		/*
					if vulkanRenderer, ok := baseRenderer.(*VulkanRenderer); ok {
			            // Assuming VulkanRenderer also has GetWindow()
			            inputManager = NewGLFWInputManager(vulkanRenderer.GetWindow())
			        }
		*/

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

		if inputManager.ShouldClose() {
			break
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

func configureCamera(camera *Camera, demoType int, orientation OrientationType) {
	camera.DZ = 0.0
	camera.Near = 0.5
	camera.FOV = Point{X: 60.0, Y: 30.0, Z: 0}

	switch demoType {
	case DemoBasicGeometry:
		camera.Transform.SetPosition(0, 5, -40*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	case DemoMeshGenerators:
		camera.Transform.SetPosition(0, 10, -60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	case DemoLightingShowcase:
		camera.Transform.SetPosition(0, 15, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0

	case DemoMaterialShowcase:
		camera.Transform.SetPosition(0, 12, -60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 250.0

	case DemoTransformHierarchy:
		camera.Transform.SetPosition(0, 25, -70*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0

	case DemoLODSystem:
		camera.Transform.SetPosition(0, 50, -150*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 500.0

	case DemoSpatialPartitioning:
		camera.Transform.SetPosition(0, 40, -100*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0

	case DemoCollisionPhysics:
		camera.Transform.SetPosition(0, 30, -70*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0

	case DemoAdvancedRendering:
		camera.Transform.SetPosition(0, 20, -50*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0

	case DemoPerformanceTest:
		camera.Transform.SetPosition(0, 50, -130*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 500.0

	case DemoAdvancedFeatures:
		camera.Transform.SetPosition(0, 20, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0

	case DemoTextureShowcase:
		camera.Transform.SetPosition(0, 0, 60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	case DemoShadowMapping:
		camera.Transform.SetPosition(0, 15, 40*float64(orientation))
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
	case DemoBasicGeometry:
		controller.SetOrbitRadius(50.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(10.0)

	case DemoMeshGenerators:
		controller.SetOrbitRadius(70.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(15.0)

	case DemoLightingShowcase:
		controller.SetOrbitRadius(90.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(20.0)

	case DemoMaterialShowcase:
		controller.SetOrbitRadius(70.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(15.0)

	case DemoTransformHierarchy:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(30.0)

	case DemoLODSystem:
		controller.SetOrbitRadius(180.0)
		controller.SetOrbitCenter(0, 0, -40)
		controller.SetOrbitHeight(50.0)

	case DemoSpatialPartitioning:
		controller.SetOrbitRadius(120.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(40.0)

	case DemoCollisionPhysics:
		controller.SetOrbitRadius(90.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(30.0)

	case DemoAdvancedRendering:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, -50)
		controller.SetOrbitHeight(25.0)

	case DemoPerformanceTest:
		controller.SetOrbitRadius(150.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(50.0)

	case DemoAdvancedFeatures:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(25.0)

	case DemoTextureShowcase:
		controller.SetOrbitRadius(70.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(15.0)

	case DemoShadowMapping:
		controller.SetOrbitRadius(60.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(20.0)

	default:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(20.0)
	}

	controller.EnableAutoOrbit(true)
}

func setupLighting(camera *Camera, demoType int) *LightingSystem {
	lighting := GetLightingScenario(demoType, camera)
	scenarioName := GetLightingScenarioName(demoType)
	fmt.Printf("Lighting: %s\n", scenarioName)
	return lighting
}

func buildScene(scene *Scene, demoType int, material Material) {
	//assetManager := GetGlobalAssetManager()
	//
	//assetManager.RegisterMaterial("default", material)

	switch demoType {
	case DemoBasicGeometry:
		BasicGeometryDemo(scene)
	case DemoMeshGenerators:
		MeshGeneratorsDemo(scene)
	case DemoLightingShowcase:
		LightingShowcaseDemo(scene)
	case DemoMaterialShowcase:
		MaterialShowcaseDemo(scene)
	case DemoTransformHierarchy:
		TransformHierarchyDemo(scene)
	case DemoLODSystem:
		LODSystemDemo(scene)
	case DemoSpatialPartitioning:
		SpatialPartitioningDemo(scene)
	case DemoCollisionPhysics:
		CollisionPhysicsDemo(scene)
	case DemoAdvancedRendering:
		AdvancedRenderingDemo(scene)
	case DemoPerformanceTest:
		PerformanceTestDemo(scene)
	case DemoAdvancedFeatures:
		AdvancedFeaturesDemo(scene)
	case DemoTextureShowcase:
		TextureShowcaseDemo(scene)
	case DemoShadowMapping:
		ShadowMappingDemo(scene)
	default:
		BasicGeometryDemo(scene)
	}
}

func animateSceneDemo(scene *Scene, demoType int, time float64) {
	switch demoType {
	case DemoBasicGeometry:
		AnimateBasicGeometry(scene)
	case DemoMeshGenerators:
		AnimateMeshGenerators(scene)
	case DemoLightingShowcase:
		AnimateLightingShowcase(scene, time)
	case DemoMaterialShowcase:
		AnimateMaterialShowcase(scene)
	case DemoTransformHierarchy:
		AnimateTransformHierarchy(scene, time)
	case DemoLODSystem:
		AnimateLODSystem(scene)
	case DemoSpatialPartitioning:
		AnimateSpatialPartitioning(scene, time)
	case DemoCollisionPhysics:
		AnimateCollisionPhysics(scene, time)
	case DemoAdvancedRendering:
		AnimateAdvancedRendering(scene)
	case DemoPerformanceTest:
		AnimatePerformanceTest(scene)
	case DemoAdvancedFeatures:
		AnimateAdvancedFeatures(scene, time)
	case DemoTextureShowcase:
		AnimateAdvancedFeatures(scene, time) // Reuse advanced features animation
	case DemoShadowMapping:
		AnimateAdvancedFeatures(scene, time) // Reuse animation for shadow casters
	}

	/*
		// Animate dynamic lights if applicable
		if scene.LightingSystem != nil && demoType == DemoCollisionPhysics {
			// Get the lighting system from the renderer
			// This would need to be passed through or accessed differently
			// For now, this is a placeholder for the concept
		}
	*/
}
