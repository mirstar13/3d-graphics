package app

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

func runEngine(demoType int, config engine.EngineConfig) bool {
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

	demo.BuildScene(scene, demoType)

	cameraController := engine.NewCameraController(scene.Camera)
	demo.ConfigureCameraController(cameraController, demoType)
	cameraController.EnableAutoOrbit(config.AutoOrbit)

	// Clear screen (Terminal only)
	if adapter.Terminal {
		fmt.Print("\033[2J\033[H")
	}

	fps := config.FPS
	startTime := time.Now()
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
		dt := now.Sub(lastFrame).Seconds()
		lastFrame = now
		if tr, ok := baseRenderer.(*terminal.TerminalRenderer); ok {
			tr.SyncSize()
		}
		finalRenderer.BeginFrame()

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
		elapsedTime := time.Since(startTime).Seconds()
		demo.AnimateSceneDemo(scene, demoType, elapsedTime, dt)

		// Animate lights
		if lightingSystem != nil {
			for _, light := range lightingSystem.Lights {
				light.Rotate('y', 0.6*dt)
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

		finalRenderer.RenderScene(scene)

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
