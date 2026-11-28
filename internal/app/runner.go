package app

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	settingsconfig "github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

func runEngine(demoType int, config engine.EngineConfig, physicsSettings settingsconfig.PhysicsSettings, tracePath string) bool {
	adapter, err := createBackend(config)
	if err != nil {
		fmt.Println(err)
		return false
	}
	baseRenderer := adapter.Renderer
	orientation := adapter.Orientation
	baseRenderer.SetUseColor(config.UseColor)
	baseRenderer.SetShowDebugInfo(config.ShowDebugInfo)

	// Frame timings are collected only when enabled.
	var profiler *engine.Profiler
	if config.EnableProfiling {
		profiler = engine.NewProfiler(60)
		fmt.Println("Profiling enabled")
	}

	var frames []frameTiming
	if tracePath != "" {
		frames = make([]frameTiming, 0, frameTimingWindow)
		defer func() {
			if len(frames) == 0 {
				return
			}
			if err := writeFrameTrace(tracePath, frames); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			fmt.Fprintln(os.Stderr, frameTimingSummary(frames))
		}()
	}
	var timingHUD bool
	var previous frameTiming

	finalRenderer := terminal.ComposeRenderer(baseRenderer, config)

	// Initialize renderer
	if err := finalRenderer.Initialize(); err != nil {
		fmt.Printf("Failed to initialize renderer: %v\n", err)
		return false
	}
	defer finalRenderer.Shutdown()
	inputManager := adapter.NewInput()
	if err := inputManager.Start(); err != nil {
		fmt.Printf("Failed to initialize input: %v\n", err)
		return false
	}
	defer inputManager.Stop()

	// Create scene
	scene := engine.NewScene()

	// Configure camera
	demo.ConfigureCamera(scene.Camera, demoType, orientation)
	scene.Camera.FOV.X = config.FOVX
	scene.Camera.FOV.Y = config.FOVY
	finalRenderer.SetCamera(scene.Camera)

	// Setup lighting
	lightingSystem := demo.SetupLighting(scene.Camera, demoType)
	finalRenderer.SetLightingSystem(lightingSystem)
	if tr, ok := baseRenderer.(*terminal.TerminalRenderer); ok && !demo.PresetAt(demoType).Shadows {
		tr.ShadowRenderer = nil
	}

	demo.BuildScene(scene, demoType)

	cameraController := engine.NewCameraController(scene.Camera)
	demo.ConfigureCameraController(cameraController, demoType)
	cameraController.EnableAutoOrbit(config.AutoOrbit)

	// Clear screen (Terminal only)
	if adapter.Terminal {
		fmt.Print("\033[2J\033[H")
	}

	fps := config.FPS
	var presentLatest func()
	var presentationBehind func(time.Duration) bool
	tr, isTerminal := baseRenderer.(*terminal.TerminalRenderer)
	adaptive := isTerminal && os.Getenv("GRAPHICS_ADAPTIVE_QUALITY") == "1"
	var quality terminal.AdaptiveQuality
	var work time.Duration
	var layout [4]int
	if adaptive {
		presentation := tr.StartLatestPresentation()
		defer presentation.Close()
		presentLatest = func() { tr.QueuePresentation(presentation) }
		presentationBehind = presentation.Behind
	}
	simulation, err := newConfiguredSimulation(scene, lightingSystem, demoType, physicsSettings)
	if err != nil {
		fmt.Printf("Failed to initialize simulation: %v\n", err)
		return false
	}
	lastStatsTime := time.Now()

	// Main render loop
	lastFrame := time.Now()
	ticker := time.NewTicker(time.Duration(float64(time.Second) / fps))
	defer ticker.Stop()

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	for {
		select {
		case <-interrupts:
			return false
		case <-ticker.C:
		}

		// Profiling: Begin frame
		if profiler != nil {
			profiler.BeginFrame()
		}

		now := time.Now()
		elapsed := now.Sub(lastFrame)
		dt := elapsed.Seconds()
		lastFrame = now
		if isTerminal {
			tr.SyncSize()
		} else {
			finalRenderer.BeginFrame()
		}

		// Update phase
		if profiler != nil {
			profiler.BeginUpdate()
		}

		input := inputManager.GetInputState()

		if input.Settings {
			return true
		}
		if input.Quit || inputManager.ShouldClose() {
			break
		}

		cameraController.Update(input, orientation, dt)
		var commands engine.SimulationCommands
		if source, ok := inputManager.(engine.SimulationCommandInput); ok {
			commands = source.DrainSimulationCommands()
		}
		for _, command := range commands.Commands {
			if command == engine.ToggleFrameTiming {
				timingHUD = !timingHUD
			}
		}
		measuring := timingHUD || (frames != nil && len(frames) < cap(frames))
		var startTick uint64
		if measuring {
			startTick = simulation.world.Stats().Tick
		}
		if err := simulation.advance(elapsed, commands); err != nil {
			fmt.Printf("Simulation error: %v\n", err)
			return false
		}
		if display, ok := baseRenderer.(engine.SimulationStatusRenderer); ok {
			status := simulation.status()
			if isTerminal {
				tr.SetFrameTiming(timingHUD, previous.update, previous.raster, previous.present)
				tr.MeasureFrameWrite(measuring)
			}
			display.SetSimulationStatus(status)
		}

		if profiler != nil {
			profiler.EndUpdate()
		}

		inputManager.ClearKeys()

		// Render phase
		if profiler != nil {
			profiler.BeginRender()
		}

		var rasterStart, presentStart time.Time
		if measuring {
			rasterStart = time.Now()
		}
		simulation.prepareRender()
		if adaptive {
			next := [4]int{tr.Width, tr.Height, tr.OffsetX, tr.OffsetY}
			if next != layout {
				quality = terminal.AdaptiveQuality{}
				layout = next
			}
			budget := time.Duration(float64(time.Second) / fps)
			tier := quality.Update(scene.Camera, work, presentationBehind(budget), budget)
			tr.RenderQuality(tier, func() { finalRenderer.RenderScene(scene) })
		} else {
			finalRenderer.RenderScene(scene)
		}
		if simulation.overlay {
			if display, ok := baseRenderer.(engine.PhysicsDebugRenderer); ok {
				display.RenderPhysicsDebug(simulation.world.DebugLines(), scene.Camera)
			}
		}

		if profiler != nil {
			profiler.EndRender()
		}

		// Present phase
		if profiler != nil {
			profiler.BeginPresent()
		}

		if measuring {
			presentStart = time.Now()
		}
		finalRenderer.EndFrame()
		if adaptive {
			presentLatest()
			work = time.Since(now)
		} else {
			finalRenderer.Present()
		}
		if measuring {
			tick := simulation.world.Stats().Tick
			w, h := finalRenderer.GetDimensions()
			previous = frameTiming{interval: elapsed, update: rasterStart.Sub(now),
				raster: presentStart.Sub(rasterStart), present: time.Since(presentStart),
				ticks: tick - startTick, columns: w, rows: h}
			if isTerminal {
				previous.bytes, previous.write = tr.FrameOutput()
			}
			if frames != nil && len(frames) < cap(frames) {
				frames = append(frames, previous)
			}
		}
		simulation.bindings.synchronize()

		if profiler != nil {
			profiler.EndPresent()
			profiler.EndFrame()
		}

		// Print stats every 2 seconds
		if !adapter.Terminal && profiler != nil && time.Since(lastStatsTime).Seconds() >= 2.0 {
			stats := profiler.GetAverageStats()

			if stats.TotalTime > 0.000001 {
				fmt.Printf("FPS: %.2f | Frame Time: %.4f ms\n", 1/stats.TotalTime, stats.TotalTime*1000)
			} else {
				fmt.Println("FPS: >9999 | Frame Time: <0.001 ms")
			}
			lastStatsTime = time.Now()

		}
	}
	return false
}
