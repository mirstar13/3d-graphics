package terminal

import (
	"math"
	"sync"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// ParallelRenderer extends Renderer with multithreading support
type ParallelRenderer struct {
	engine.Renderer
	NumWorkers int
	TileSize   int
	Batched    bool
	Work       TileWork
	Pools      *engine.RenderPools

	// Worker synchronization
	tileQueue chan RenderTile
	wg        sync.WaitGroup
}

// TileWork describes the last tiled frame's instance preparation and binning.
type TileWork struct {
	Instances, Vertices, InstanceTileReferences int
}

type tileDraw struct {
	node     *engine.SceneNode
	mesh     *engine.Mesh
	vertices []engine.Point
}

// RenderTile represents a tile to be rendered
type RenderTile struct {
	X, Y          int
	Width, Height int
	draws         []tileDraw
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
	bins := make([][]tileDraw, tilesX*tilesY)
	pr.Work = TileWork{}

	add := func(draw tileDraw, minX, minY, maxX, maxY int) int {
		references := 0
		if maxX >= 0 && maxY >= 0 && minX < width && minY < height {
			startTx := engine.ClampInt(minX/pr.TileSize, 0, tilesX-1)
			endTx := engine.ClampInt(maxX/pr.TileSize, 0, tilesX-1)
			startTy := engine.ClampInt(minY/pr.TileSize, 0, tilesY-1)
			endTy := engine.ClampInt(maxY/pr.TileSize, 0, tilesY-1)

			for ty := startTy; ty <= endTy; ty++ {
				for tx := startTx; tx <= endTx; tx++ {
					idx := ty*tilesX + tx
					bins[idx] = append(bins[idx], draw)
					references++
				}
			}
		}
		return references
	}
	for _, node := range nodes {
		if instances, ok := node.Object.(*engine.InstancedMesh); ok {
			if !instances.Enabled || instances.BaseMesh == nil || len(instances.BaseMesh.Vertices) == 0 {
				continue
			}
			world := node.Transform.GetWorldMatrix()
			for _, instance := range instances.Instances {
				mesh := instanceMesh(instances.BaseMesh, instance.Color)
				vertices := transformMeshVertices(&mesh, world.Multiply(instance.Transform))
				x0, y0, x1, y1 := instanceScreenBounds(vertices, camera, width, height)
				pr.Work.Instances++
				pr.Work.Vertices += len(vertices)
				pr.Work.InstanceTileReferences += add(tileDraw{mesh: &mesh, vertices: vertices}, x0, y0, x1, y1)
			}
			continue
		}
		x0, y0, x1, y1 := 0, 0, width-1, height-1
		if bounds := engine.ComputeNodeBounds(node); bounds != nil {
			x0, y0, x1, y1 = projectAABBToScreen(bounds, camera, width, height)
		}
		add(tileDraw{node: node}, x0, y0, x1, y1)
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
				draws: bins[idx], Camera: camera,
			}
			pr.tileQueue <- tile
		}
	}
}

// renderTile shares frame buffers while keeping clip bounds local to each worker.
func (pr *ParallelRenderer) renderTile(tile RenderTile) {
	renderer := *(pr.Renderer.(*TerminalRenderer))
	renderer.SetClipBounds(tile.X, tile.Y, tile.X+tile.Width, tile.Y+tile.Height)
	for _, draw := range tile.draws {
		if draw.mesh != nil {
			renderer.renderPreparedMesh(draw.mesh, draw.vertices, tile.Camera)
		} else {
			renderer.renderNode(draw.node, draw.node.Transform.GetWorldMatrix(), tile.Camera)
		}
	}
}

// RenderBatched bins enabled objects into tiles without the frustum pre-pass.
func (pr *ParallelRenderer) RenderBatched(scene *engine.Scene) {
	pr.renderScene(scene, false)
}

// In front of the near plane, projected triangle extrema occur at vertices.
func instanceScreenBounds(vertices []engine.Point, camera *engine.Camera, width, height int) (int, int, int, int) {
	x0, y0, x1, y1 := float64(width), float64(height), -1., -1.
	for _, v := range vertices {
		x, y, z := camera.ProjectPointFloat(v, height, width)
		if z <= camera.Near {
			return 0, 0, width - 1, height - 1
		}
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	return int(math.Floor(x0)) - 1, int(math.Floor(y0)) - 1, int(math.Ceil(x1)) + 1, int(math.Ceil(y1)) + 1
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
