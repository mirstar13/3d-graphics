package main

import (
	"fmt"
	"strings"
	"time"
)

// BenchmarkConfig holds benchmark configuration
type BenchmarkConfig struct {
	NumFrames       int
	SceneComplexity string
	RenderModes     []RenderMode
	NumWorkers      []int
}

// BenchmarkResult holds results for a single benchmark run
type BenchmarkResult struct {
	RenderMode      RenderMode
	NumWorkers      int
	AvgFrameTime    time.Duration
	MinFrameTime    time.Duration
	MaxFrameTime    time.Duration
	AvgFPS          float64
	TrianglesPerSec int
	TotalTriangles  int
}

// RunBenchmark runs performance benchmarks for different rendering modes
func RunBenchmark(config BenchmarkConfig) []BenchmarkResult {
	results := make([]BenchmarkResult, 0)

	for _, mode := range config.RenderModes {
		for _, workers := range config.NumWorkers {
			if mode == RenderModeSingle && workers > 1 {
				continue // Skip multi-worker for single-threaded
			}

			fmt.Printf("\nBenchmarking: %s with %d workers...\n",
				getRenderModeName(mode), workers)

			result := benchmarkRenderMode(mode, workers, config.NumFrames, config.SceneComplexity)
			results = append(results, result)
		}
	}

	return results
}

// benchmarkRenderMode benchmarks a specific rendering mode
func benchmarkRenderMode(mode RenderMode, workers int, numFrames int, complexity string) BenchmarkResult {
	// Create renderer (using in-memory buffers for benchmarking)
	baseRenderer := NewTerminalRenderer(nil, 51, 223)
	baseRenderer.SetUseColor(true)
	baseRenderer.SetShowDebugInfo(false)

	// Setup parallel renderer if needed
	var parallelRenderer *ParallelRenderer
	var jobRenderer *JobBasedRenderer

	switch mode {
	case RenderModeParallelTiles:
		parallelRenderer = NewParallelRenderer(baseRenderer, workers, 32)
	case RenderModeParallelJobs:
		jobRenderer = NewJobBasedRenderer(baseRenderer, workers)
	case RenderModeParallelBatched:
		parallelRenderer = NewParallelRenderer(baseRenderer, workers, 32)
	}

	// Create test scene
	scene := createBenchmarkScene(complexity)

	// Warm-up
	for i := 0; i < 5; i++ {
		renderFrame(baseRenderer, parallelRenderer, jobRenderer, scene, mode)
	}

	// Benchmark
	frameTimes := make([]time.Duration, numFrames)
	var totalTriangles int

	for i := 0; i < numFrames; i++ {
		start := time.Now()

		switch mode {
		case RenderModeParallelTiles:
			parallelRenderer.RenderSceneParallel(scene)
		case RenderModeParallelJobs:
			jobRenderer.RenderSceneJobs(scene)
		case RenderModeParallelBatched:
			parallelRenderer.RenderBatched(scene)
		default:
			baseRenderer.RenderScene(scene)
		}

		frameTimes[i] = time.Since(start)

		// Count triangles (approximate)
		if i == 0 {
			totalTriangles = countSceneTriangles(scene)
		}
	}

	// Calculate statistics
	var totalTime time.Duration
	minTime := frameTimes[0]
	maxTime := frameTimes[0]

	for _, t := range frameTimes {
		totalTime += t
		if t < minTime {
			minTime = t
		}
		if t > maxTime {
			maxTime = t
		}
	}

	avgTime := totalTime / time.Duration(numFrames)
	avgFPS := 1.0 / avgTime.Seconds()
	trianglesPerSec := int(float64(totalTriangles) * avgFPS)

	return BenchmarkResult{
		RenderMode:      mode,
		NumWorkers:      workers,
		AvgFrameTime:    avgTime,
		MinFrameTime:    minTime,
		MaxFrameTime:    maxTime,
		AvgFPS:          avgFPS,
		TrianglesPerSec: trianglesPerSec,
		TotalTriangles:  totalTriangles,
	}
}

// renderFrame renders a single frame based on mode
func renderFrame(base *TerminalRenderer, parallel *ParallelRenderer, job *JobBasedRenderer, scene *Scene, mode RenderMode) {
	switch mode {
	case RenderModeParallelTiles:
		if parallel != nil {
			parallel.RenderSceneParallel(scene)
		}
	case RenderModeParallelJobs:
		if job != nil {
			job.RenderSceneJobs(scene)
		}
	case RenderModeParallelBatched:
		if parallel != nil {
			parallel.RenderBatched(scene)
		}
	default:
		base.RenderScene(scene)
	}
}

// createBenchmarkScene creates a test scene for benchmarking
func createBenchmarkScene(complexity string) *Scene {
	scene := NewScene()
	material := NewMaterial()
	material.DiffuseColor = Color{180, 150, 200}

	switch complexity {
	case "low":
		// 25 cubes
		for i := 0; i < 5; i++ {
			for j := 0; j < 5; j++ {
				cube := scene.CreateCube(fmt.Sprintf("cube_%d_%d", i, j), 4, material)
				cube.Transform.SetPosition(float64(i*15-30), 0, float64(j*15-30))
			}
		}

	case "medium":
		// 100 spheres
		for i := 0; i < 10; i++ {
			for j := 0; j < 10; j++ {
				sphere := scene.CreateSphere(fmt.Sprintf("sphere_%d_%d", i, j),
					4, 8, 8, material)
				sphere.Transform.SetPosition(float64(i*12-54), 0, float64(j*12-54))
			}
		}

	case "high":
		// 200+ mixed objects
		for i := 0; i < 15; i++ {
			for j := 0; j < 15; j++ {
				if (i+j)%2 == 0 {
					sphere := scene.CreateSphere(fmt.Sprintf("obj_%d_%d", i, j),
						3, 12, 12, material)
					sphere.Transform.SetPosition(float64(i*10-70), 0, float64(j*10-70))
				} else {
					cube := scene.CreateCube(fmt.Sprintf("obj_%d_%d", i, j), 3, material)
					cube.Transform.SetPosition(float64(i*10-70), 0, float64(j*10-70))
				}
			}
		}

	default:
		// Default: medium
		for i := 0; i < 8; i++ {
			for j := 0; j < 8; j++ {
				cube := scene.CreateCube(fmt.Sprintf("cube_%d_%d", i, j), 4, material)
				cube.Transform.SetPosition(float64(i*12-42), 0, float64(j*12-42))
			}
		}
	}

	// Setup camera
	scene.Camera.Transform.SetPosition(0, 30, -80)
	scene.Camera.Transform.SetRotation(-0.3, 0, 0)

	return scene
}

// countSceneTriangles counts total triangles in scene
func countSceneTriangles(scene *Scene) int {
	total := 0
	nodes := scene.GetRenderableNodes()

	for _, node := range nodes {
		switch obj := node.Object.(type) {
		case *Mesh:
			total += len(obj.Triangles)
			total += len(obj.Quads) * 2 // Each quad = 2 triangles
		case *Triangle:
			total++
		case *Quad:
			total += 2
		}
	}

	return total
}

// PrintBenchmarkResults prints formatted benchmark results
func PrintBenchmarkResults(results []BenchmarkResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("BENCHMARK RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("\n%-25s %-10s %-15s %-10s %-12s %-15s\n",
		"Mode", "Workers", "Avg Frame (ms)", "FPS", "Triangles", "Tri/Sec")
	fmt.Println(strings.Repeat("=", 80))

	for _, result := range results {
		workerStr := fmt.Sprintf("%d", result.NumWorkers)
		if result.RenderMode == RenderModeSingle {
			workerStr = "N/A"
		}

		fmt.Printf("%-25s %-10s %-15.2f %-10.1f %-12d %-15d\n",
			getRenderModeName(result.RenderMode),
			workerStr,
			float64(result.AvgFrameTime.Microseconds())/1000.0,
			result.AvgFPS,
			result.TotalTriangles,
			result.TrianglesPerSec,
		)
	}

	fmt.Println(strings.Repeat("=", 80))

	// Print speedup analysis
	if len(results) > 1 {
		baselineResult := results[0]
		fmt.Println("\nSPEEDUP ANALYSIS (compared to first result):")
		fmt.Println(strings.Repeat("=", 80))

		for i, result := range results {
			if i == 0 {
				fmt.Printf("%-25s: Baseline (1.00x)\n", getRenderModeName(result.RenderMode))
			} else {
				speedup := float64(baselineResult.AvgFrameTime) / float64(result.AvgFrameTime)
				fmt.Printf("%-25s (%d workers): %.2fx speedup\n",
					getRenderModeName(result.RenderMode),
					result.NumWorkers,
					speedup)
			}
		}
		fmt.Println(strings.Repeat("=", 80))
	}
}

// RunFullBenchmarkSuite runs a comprehensive benchmark suite
func RunFullBenchmarkSuite() {
	fmt.Println("Starting Full Benchmark Suite...")
	fmt.Println("This will take a few minutes...")

	allResults := make([]BenchmarkResult, 0)

	// Test different complexities
	complexities := []string{"low", "medium", "high"}

	for _, complexity := range complexities {
		fmt.Printf("\n\n=== TESTING %s COMPLEXITY ===\n", complexity)

		config := BenchmarkConfig{
			NumFrames:       30, // 30 frames per test
			SceneComplexity: complexity,
			RenderModes: []RenderMode{
				RenderModeSingle,
				RenderModeParallelTiles,
				RenderModeParallelJobs,
				RenderModeParallelBatched,
			},
			NumWorkers: []int{1, 2, 4, 8},
		}

		results := RunBenchmark(config)
		allResults = append(allResults, results...)

		// Print results for this complexity
		PrintBenchmarkResults(results)
	}

	// Print overall summary
	fmt.Println("\n\n" + strings.Repeat("=", 80))
	fmt.Println("OVERALL SUMMARY")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\nRecommendations:")
	fmt.Println("  • For <50 objects: Single-threaded is often fastest")
	fmt.Println("  • For 50-200 objects: Parallel Jobs or Tiles (2-4 workers)")
	fmt.Println("  • For 200+ objects: Parallel Tiles (4-8 workers)")
	fmt.Println("  • For many similar materials: Parallel Batched")
	fmt.Println(strings.Repeat("=", 80))
}

// OptimalWorkerCount determines optimal worker count based on scene
func OptimalWorkerCount(scene *Scene) int {
	triangles := countSceneTriangles(scene)
	nodes := len(scene.GetRenderableNodes())
	_ = nodes

	// Heuristic based on complexity
	if triangles < 1000 {
		return 1 // Single-threaded is fine
	} else if triangles < 5000 {
		return 2 // 2 workers
	} else if triangles < 15000 {
		return 4 // 4 workers
	} else {
		return 8 // 8 workers for very complex scenes
	}
}

// RecommendRenderMode recommends a rendering mode based on scene characteristics
func RecommendRenderMode(scene *Scene) RenderMode {
	triangles := countSceneTriangles(scene)
	nodes := len(scene.GetRenderableNodes())

	// Check if nodes share materials (for batching)
	materialMap := make(map[Material]int)
	for _, node := range scene.GetRenderableNodes() {
		var mat Material
		switch obj := node.Object.(type) {
		case *Mesh:
			if len(obj.Triangles) > 0 {
				mat = obj.Triangles[0].Material
			}
		case *Triangle:
			mat = obj.Material
		}
		materialMap[mat]++
	}

	// Calculate material sharing ratio
	materialSharing := float64(len(materialMap)) / float64(nodes)

	// Recommendation logic
	if triangles < 500 {
		return RenderModeSingle
	}

	if materialSharing < 0.3 { // High material sharing
		return RenderModeParallelBatched
	}

	if nodes > 100 {
		return RenderModeParallelJobs
	}

	return RenderModeParallelTiles
}
