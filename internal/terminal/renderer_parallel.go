package terminal

import (
	"sync"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// ParallelRenderer extends Renderer with multithreading support
type ParallelRenderer struct {
	engine.Renderer
	NumWorkers int
	TileSize   int
	Batched    bool
	Pools      *engine.RenderPools

	// Worker synchronization
	tileQueue chan RenderTile
	wg        sync.WaitGroup
}

// RenderTile represents a tile to be rendered
type RenderTile struct {
	X, Y          int
	Width, Height int
	Nodes         []*engine.SceneNode
	Camera        *engine.Camera
}

// NewParallelRenderer creates a parallel renderer
func NewParallelRenderer(renderer engine.Renderer, numWorkers, tileSize int) *ParallelRenderer {
	numWorkers, tileSize = max(1, numWorkers), max(1, tileSize)
	return &ParallelRenderer{
		Renderer:   renderer,
		NumWorkers: numWorkers,
		TileSize:   tileSize,
		Pools:      engine.NewRenderPools(10000, 5000, 5000, 1000),
		tileQueue:  make(chan RenderTile, numWorkers*4),
	}
}

// RenderSceneParallel culls objects and schedules disjoint screen tiles.
func (pr *ParallelRenderer) RenderSceneParallel(scene *engine.Scene) {
	pr.renderScene(scene, true)
}

func (pr *ParallelRenderer) RenderScene(scene *engine.Scene) {
	if pr.Batched {
		pr.RenderBatched(scene)
	} else {
		pr.RenderSceneParallel(scene)
	}
}

func (pr *ParallelRenderer) renderScene(scene *engine.Scene, cull bool) {
	renderer, ok := pr.Renderer.(*TerminalRenderer)
	if !ok || engine.RequiresMainThread(pr.Renderer) {
		pr.Renderer.RenderScene(scene)
		return
	}
	nodes := renderer.prepareScene(scene)
	if cull {
		frustum := engine.BuildFrustum(scene.Camera)
		nodes = nil
		engine.FrustumCullNode(scene.Root, &frustum, &nodes)
	}
	pr.Pools.ResetAll()
	pr.startWorkers()
	pr.generateTiles(nodes, scene.Camera)
	close(pr.tileQueue)
	pr.wg.Wait()
	pr.tileQueue = make(chan RenderTile, pr.NumWorkers*4)
}

func (pr *ParallelRenderer) startWorkers() {
	for i := 0; i < pr.NumWorkers; i++ {
		pr.wg.Add(1)
		go pr.worker()
	}
}

func (pr *ParallelRenderer) worker() {
	defer pr.wg.Done()
	for tile := range pr.tileQueue {
		pr.renderTile(tile)
	}
}

func (pr *ParallelRenderer) generateTiles(nodes []*engine.SceneNode, camera *engine.Camera) {
	width, height := pr.Renderer.GetDimensions()
	tilesX := (width + pr.TileSize - 1) / pr.TileSize
	tilesY := (height + pr.TileSize - 1) / pr.TileSize

	// Pre-allocate bins
	bins := make([][]*engine.SceneNode, tilesX*tilesY)
	for i := range bins {
		bins[i] = make([]*engine.SceneNode, 0)
	}

	// Bin nodes
	for _, node := range nodes {
		minX, minY, maxX, maxY := 0, 0, width-1, height-1
		if bounds := engine.ComputeNodeBounds(node); bounds != nil {
			minX, minY, maxX, maxY = projectAABBToScreen(bounds, camera, width, height)
		}
		if maxX >= 0 && maxY >= 0 && minX < width && minY < height {
			startTx := engine.ClampInt(minX/pr.TileSize, 0, tilesX-1)
			endTx := engine.ClampInt(maxX/pr.TileSize, 0, tilesX-1)
			startTy := engine.ClampInt(minY/pr.TileSize, 0, tilesY-1)
			endTy := engine.ClampInt(maxY/pr.TileSize, 0, tilesY-1)

			for ty := startTy; ty <= endTy; ty++ {
				for tx := startTx; tx <= endTx; tx++ {
					idx := ty*tilesX + tx
					bins[idx] = append(bins[idx], node)
				}
			}
		}
	}

	for ty := 0; ty < tilesY; ty++ {
		for tx := 0; tx < tilesX; tx++ {
			idx := ty*tilesX + tx
			if len(bins[idx]) == 0 {
				continue
			}

			x := tx * pr.TileSize
			y := ty * pr.TileSize
			w := pr.TileSize
			h := pr.TileSize
			if x+w > width {
				w = width - x
			}
			if y+h > height {
				h = height - y
			}

			tile := RenderTile{
				X: x, Y: y, Width: w, Height: h,
				Nodes: bins[idx], Camera: camera,
			}
			pr.tileQueue <- tile
		}
	}
}

// renderTile shares frame buffers while keeping clip bounds local to each worker.
func (pr *ParallelRenderer) renderTile(tile RenderTile) {
	renderer := *(pr.Renderer.(*TerminalRenderer))
	renderer.SetClipBounds(tile.X, tile.Y, tile.X+tile.Width, tile.Y+tile.Height)
	for _, node := range tile.Nodes {
		renderer.renderNode(node, node.Transform.GetWorldMatrix(), tile.Camera)
	}
}

// RenderBatched bins enabled objects into tiles without the frustum pre-pass.
func (pr *ParallelRenderer) RenderBatched(scene *engine.Scene) {
	pr.renderScene(scene, false)
}

func projectAABBToScreen(aabb *engine.AABB, cam *engine.Camera, w, h int) (minX, minY, maxX, maxY int) {
	corners := []engine.Point{
		{X: aabb.Min.X, Y: aabb.Min.Y, Z: aabb.Min.Z},
		{X: aabb.Max.X, Y: aabb.Min.Y, Z: aabb.Min.Z},
		{X: aabb.Min.X, Y: aabb.Max.Y, Z: aabb.Min.Z},
		{X: aabb.Max.X, Y: aabb.Max.Y, Z: aabb.Min.Z},
		{X: aabb.Min.X, Y: aabb.Min.Y, Z: aabb.Max.Z},
		{X: aabb.Max.X, Y: aabb.Min.Y, Z: aabb.Max.Z},
		{X: aabb.Min.X, Y: aabb.Max.Y, Z: aabb.Max.Z},
		{X: aabb.Max.X, Y: aabb.Max.Y, Z: aabb.Max.Z},
	}

	minX, minY = w, h
	maxX, maxY = 0, 0
	initialized := false

	for _, p := range corners {
		sx, sy, z := cam.ProjectPoint(p, h, w)
		// Handle behind camera (simple clip)
		if z <= cam.Near {
			// If bounding box is partially behind camera, this naive projection is wrong.
			// A robust implementation would clip the AABB against the near plane.
			// For this implementation, if any point is behind, we assume it covers full screen for safety
			return 0, 0, w, h
		}

		if sx < minX {
			minX = sx
		}
		if sx > maxX {
			maxX = sx
		}
		if sy < minY {
			minY = sy
		}
		if sy > maxY {
			maxY = sy
		}
		initialized = true
	}

	if !initialized {
		return 0, 0, 0, 0
	}
	return
}

// JobBasedRenderer schedules node jobs while serializing framebuffer writes.
// Prefer tiled rendering when rasterization itself should run concurrently.
type JobBasedRenderer struct {
	engine.Renderer
	NumWorkers int
	jobQueue   chan RenderJob
	wg         sync.WaitGroup
	Pools      *engine.RenderPools
	mu         sync.Mutex // Serializes writes to the shared renderer.
}

// RenderJob represents a rendering job
type RenderJob struct {
	Node   *engine.SceneNode
	Camera *engine.Camera
}

// NewJobBasedRenderer creates a job-based parallel renderer
func NewJobBasedRenderer(renderer engine.Renderer, numWorkers int) *JobBasedRenderer {
	numWorkers = max(1, numWorkers)
	return &JobBasedRenderer{
		Renderer:   renderer,
		NumWorkers: numWorkers,
		jobQueue:   make(chan RenderJob, numWorkers*8),
		Pools:      engine.NewRenderPools(10000, 5000, 5000, 1000),
	}
}

// RenderSceneJobs renders scene using job-based parallelism
func (jr *JobBasedRenderer) RenderSceneJobs(scene *engine.Scene) {
	renderer, ok := jr.Renderer.(*TerminalRenderer)
	if !ok || engine.RequiresMainThread(jr.Renderer) {
		jr.Renderer.RenderScene(scene)
		return
	}
	renderer.prepareScene(scene)
	jr.Pools.ResetAll()
	jr.jobQueue = make(chan RenderJob, jr.NumWorkers*8)
	frustum := engine.BuildFrustum(scene.Camera)
	var visibleNodes []*engine.SceneNode
	engine.FrustumCullNode(scene.Root, &frustum, &visibleNodes)

	// Start workers
	for i := 0; i < jr.NumWorkers; i++ {
		jr.wg.Add(1)
		go jr.jobWorker()
	}

	// Queue jobs
	for _, node := range visibleNodes {
		jr.jobQueue <- RenderJob{
			Node:   node,
			Camera: scene.Camera,
		}
	}

	close(jr.jobQueue)
	jr.wg.Wait()
}

func (jr *JobBasedRenderer) renderNodeWithMatrix(node *engine.SceneNode, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	// Lock the renderer to prevent race conditions on the framebuffer
	jr.mu.Lock()
	defer jr.mu.Unlock()

	jr.Renderer.(*TerminalRenderer).renderNode(node, worldMatrix, camera)
}

// jobWorker processes rendering jobs
func (jr *JobBasedRenderer) jobWorker() {
	defer jr.wg.Done()
	for job := range jr.jobQueue {
		worldMatrix := job.Node.Transform.GetWorldMatrix()
		jr.renderNodeWithMatrix(job.Node, worldMatrix, job.Camera)
	}
}

// RenderScene dispatches the job renderer through the shared renderer interface.
func (jr *JobBasedRenderer) RenderScene(scene *engine.Scene) {
	jr.RenderSceneJobs(scene)
}
