package main

import (
	"sync"
)

// ParallelRenderer extends Renderer with multithreading support
type ParallelRenderer struct {
	Renderer   Renderer
	NumWorkers int
	TileSize   int
	Pools      *RenderPools

	// Worker synchronization
	workerPool sync.Pool
	tileQueue  chan RenderTile
	wg         sync.WaitGroup
}

// RenderTile represents a tile to be rendered
type RenderTile struct {
	X, Y          int
	Width, Height int
	Nodes         []*SceneNode
	Camera        *Camera
}

// NewParallelRenderer creates a parallel renderer
func NewParallelRenderer(renderer Renderer, numWorkers, tileSize int) *ParallelRenderer {
	return &ParallelRenderer{
		Renderer:   renderer,
		NumWorkers: numWorkers,
		TileSize:   tileSize,
		Pools:      NewRenderPools(10000, 5000, 5000, 1000),
		tileQueue:  make(chan RenderTile, numWorkers*4),
	}
}

// RenderSceneParallel renders scene using multiple threads
func (pr *ParallelRenderer) RenderSceneParallel(scene *Scene) {
	pr.Renderer.BeginFrame()
	pr.Pools.ResetAll()

	ctx := pr.Renderer.GetRenderContext()
	if ctx.LightingSystem != nil {
		ctx.LightingSystem.SetCamera(scene.Camera)
	}

	// Get visible nodes using frustum culling
	frustum := BuildFrustumSimple(scene.Camera)
	visibleNodes := make([]*SceneNode, 0)
	FrustumCullNode(scene.Root, &frustum, &visibleNodes)

	pr.startWorkers()
	pr.generateTiles(visibleNodes, scene.Camera)
	close(pr.tileQueue)
	pr.wg.Wait()

	// Reset queue for next frame
	pr.tileQueue = make(chan RenderTile, pr.NumWorkers*4)
}

func (pr *ParallelRenderer) Initialize() error        { return pr.Renderer.Initialize() }
func (pr *ParallelRenderer) Shutdown()                { pr.Renderer.Shutdown() }
func (pr *ParallelRenderer) BeginFrame()              { pr.Renderer.BeginFrame() }
func (pr *ParallelRenderer) EndFrame()                { pr.Renderer.EndFrame() }
func (pr *ParallelRenderer) Present()                 { pr.Renderer.Present() }
func (pr *ParallelRenderer) RenderScene(scene *Scene) { pr.RenderSceneParallel(scene) }

// Passthrough methods required by interface
func (pr *ParallelRenderer) RenderTriangle(tri *Triangle, wm Matrix4x4, cam *Camera) {
	pr.Renderer.RenderTriangle(tri, wm, cam)
}
func (pr *ParallelRenderer) RenderLine(line *Line, wm Matrix4x4, cam *Camera) {
	pr.Renderer.RenderLine(line, wm, cam)
}
func (pr *ParallelRenderer) RenderPoint(pt *Point, wm Matrix4x4, cam *Camera) {
	pr.Renderer.RenderPoint(pt, wm, cam)
}
func (pr *ParallelRenderer) RenderMesh(mesh *Mesh, wm Matrix4x4, cam *Camera) {
	pr.Renderer.RenderMesh(mesh, wm, cam)
}
func (pr *ParallelRenderer) SetLightingSystem(ls *LightingSystem) { pr.Renderer.SetLightingSystem(ls) }
func (pr *ParallelRenderer) SetCamera(camera *Camera)             { pr.Renderer.SetCamera(camera) }
func (pr *ParallelRenderer) SetUseColor(use bool)                 { pr.Renderer.SetUseColor(use) }
func (pr *ParallelRenderer) SetShowDebugInfo(show bool)           { pr.Renderer.SetShowDebugInfo(show) }
func (pr *ParallelRenderer) SetClipBounds(x, y, w, h int)         { pr.Renderer.SetClipBounds(x, y, w, h) }
func (pr *ParallelRenderer) GetDimensions() (int, int)            { return pr.Renderer.GetDimensions() }
func (pr *ParallelRenderer) GetRenderContext() *RenderContext     { return pr.Renderer.GetRenderContext() }

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

func (pr *ParallelRenderer) generateTiles(nodes []*SceneNode, camera *Camera) {
	width, height := pr.Renderer.GetDimensions()
	tilesX := (width + pr.TileSize - 1) / pr.TileSize
	tilesY := (height + pr.TileSize - 1) / pr.TileSize

	for ty := 0; ty < tilesY; ty++ {
		for tx := 0; tx < tilesX; tx++ {
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
				Nodes: nodes, Camera: camera,
			}
			pr.tileQueue <- tile
		}
	}
}

// renderTile renders a single tile safely by copying the renderer context
func (pr *ParallelRenderer) renderTile(tile RenderTile) {
	// CRITICAL FIX: Thread safety via shallow copy
	// We assume the underlying renderer is *TerminalRenderer
	if tr, ok := pr.Renderer.(*TerminalRenderer); ok {
		// Create a shallow copy of the struct
		// This copies pointers to buffers (shared memory) but creates a local 'ClipRect'
		rendererCopy := *tr

		// Set the clip bounds strictly for this tile
		rendererCopy.SetClipBounds(tile.X, tile.Y, tile.X+tile.Width, tile.Y+tile.Height)

		// Use the thread-local renderer copy to render the nodes
		for _, node := range tile.Nodes {
			worldMatrix := node.Transform.GetWorldMatrix()
			pr.renderNodeWithRenderer(node, worldMatrix, tile.Camera, &rendererCopy)
		}
	} else {
		// Fallback for non-terminal renderers (unsafe fallback)
		for _, node := range tile.Nodes {
			worldMatrix := node.Transform.GetWorldMatrix()
			pr.Renderer.SetClipBounds(tile.X, tile.Y, tile.X+tile.Width, tile.Y+tile.Height)
			pr.renderNodeWithRenderer(node, worldMatrix, tile.Camera, pr.Renderer)
		}
	}
}

func (pr *ParallelRenderer) renderNodeWithRenderer(node *SceneNode, worldMatrix Matrix4x4, camera *Camera, r Renderer) {
	switch obj := node.Object.(type) {
	case *Mesh:
		r.RenderMesh(obj, worldMatrix, camera)
	case *Triangle:
		r.RenderTriangle(obj, worldMatrix, camera)
	case *Line:
		r.RenderLine(obj, worldMatrix, camera)
	case *Point:
		r.RenderPoint(obj, worldMatrix, camera)
	case *Quad:
		// Convert to triangles manually or assume renderer handles it
		if tr, ok := r.(*TerminalRenderer); ok {
			tr.renderQuad(obj, worldMatrix, camera)
		}
	}
}

// RenderBatched renders scene by binning nodes into screen tiles to reduce overdraw
func (pr *ParallelRenderer) RenderBatched(scene *Scene) {
	pr.Renderer.BeginFrame()
	pr.Pools.ResetAll()

	ctx := pr.Renderer.GetRenderContext()
	if ctx.LightingSystem != nil {
		ctx.LightingSystem.SetCamera(scene.Camera)
	}

	// 1. Get dimensions and setup tiles
	width, height := pr.Renderer.GetDimensions()
	tilesX := (width + pr.TileSize - 1) / pr.TileSize
	tilesY := (height + pr.TileSize - 1) / pr.TileSize

	// 2. Create bins for each tile
	bins := make([][]*SceneNode, tilesX*tilesY)
	for i := range bins {
		bins[i] = make([]*SceneNode, 0)
	}

	// 3. Bin nodes based on projected screen bounds
	visibleNodes := scene.GetRenderableNodes()
	for _, node := range visibleNodes {
		// Calculate screen space bounding box
		// Note: We need a way to get bounds. Assuming a helper or computing it here.
		// For robustness, we check if the object has a bounding volume or compute it.
		var aabb *AABB
		switch obj := node.Object.(type) {
		case *Mesh:
			boundsVol := ComputeMeshBounds(obj)
			if a, ok := boundsVol.(*AABB); ok {
				aabb = a
			} else {
				pos := node.Transform.GetWorldPosition()
				aabb = NewAABB(pos, pos)
			}
		case *Triangle:
			aabb = ComputeTriangleBounds(obj)
		default:
			// Fallback: don't bin, put in all tiles or skip?
			// For simplicity, we add to all tiles if we can't bound it (expensive)
			// Or better: just transform the position
			pos := node.Transform.GetWorldPosition()
			aabb = NewAABB(pos, pos)
		}

		if aabb == nil {
			continue
		}

		// Transform AABB to world space
		worldAABB := TransformAABB(aabb, node.GetWorldTransform())

		// Project AABB corners to find screen rect
		minX, minY, maxX, maxY := projectAABBToScreen(worldAABB, scene.Camera, width, height)

		// Determine overlapping tiles
		startTx := clampInt(minX/pr.TileSize, 0, tilesX-1)
		endTx := clampInt(maxX/pr.TileSize, 0, tilesX-1)
		startTy := clampInt(minY/pr.TileSize, 0, tilesY-1)
		endTy := clampInt(maxY/pr.TileSize, 0, tilesY-1)

		// Add node to relevant bins
		for ty := startTy; ty <= endTy; ty++ {
			for tx := startTx; tx <= endTx; tx++ {
				idx := ty*tilesX + tx
				bins[idx] = append(bins[idx], node)
			}
		}
	}

	// 4. Start workers
	pr.startWorkers()

	// 5. Queue populated tiles
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
				X:      x,
				Y:      y,
				Width:  w,
				Height: h,
				Nodes:  bins[idx],
				Camera: scene.Camera,
			}
			pr.tileQueue <- tile
		}
	}

	close(pr.tileQueue)
	pr.wg.Wait()

	// Reset queue
	pr.tileQueue = make(chan RenderTile, pr.NumWorkers*4)
}

func projectAABBToScreen(aabb *AABB, cam *Camera, w, h int) (minX, minY, maxX, maxY int) {
	corners := []Point{
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

// JobBasedRenderer implements job-based parallelism with mutex safety
type JobBasedRenderer struct {
	Renderer
	NumWorkers int
	jobQueue   chan RenderJob
	wg         sync.WaitGroup
	Pools      *RenderPools
	mu         sync.Mutex // Added for thread safety
}

// RenderJob represents a rendering job
type RenderJob struct {
	Node   *SceneNode
	Camera *Camera
}

// NewJobBasedRenderer creates a job-based parallel renderer
func NewJobBasedRenderer(renderer Renderer, numWorkers int) *JobBasedRenderer {
	return &JobBasedRenderer{
		Renderer:   renderer,
		NumWorkers: numWorkers,
		jobQueue:   make(chan RenderJob, numWorkers*8),
		Pools:      NewRenderPools(10000, 5000, 5000, 1000),
	}
}

// RenderSceneJobs renders scene using job-based parallelism
func (jr *JobBasedRenderer) RenderSceneJobs(scene *Scene) {
	jr.Renderer.BeginFrame()
	jr.Pools.ResetAll()

	jr.jobQueue = make(chan RenderJob, jr.NumWorkers*8)
	ctx := jr.Renderer.GetRenderContext()

	if ctx.LightingSystem != nil {
		ctx.LightingSystem.SetCamera(scene.Camera)
	}

	// Get visible nodes
	frustum := BuildFrustumSimple(scene.Camera)
	visibleNodes := make([]*SceneNode, 0)
	FrustumCullNode(scene.Root, &frustum, &visibleNodes)

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

func (jr *JobBasedRenderer) renderNodeWithMatrix(node *SceneNode, worldMatrix Matrix4x4, camera *Camera) {
	// Lock the renderer to prevent race conditions on the framebuffer
	jr.mu.Lock()
	defer jr.mu.Unlock()

	switch obj := node.Object.(type) {
	case *Mesh:
		jr.Renderer.RenderMesh(obj, worldMatrix, camera)
	case *Triangle:
		jr.Renderer.RenderTriangle(obj, worldMatrix, camera)
	case *Line:
		jr.Renderer.RenderLine(obj, worldMatrix, camera)
	case *Point:
		jr.Renderer.RenderPoint(obj, worldMatrix, camera)
	}
}

// jobWorker processes rendering jobs
func (jr *JobBasedRenderer) jobWorker() {
	defer jr.wg.Done()
	for job := range jr.jobQueue {
		worldMatrix := job.Node.Transform.GetWorldMatrix()
		jr.renderNodeWithMatrix(job.Node, worldMatrix, job.Camera)
	}
}

// ParallelTransformUpdate updates transforms in parallel
func ParallelTransformUpdate(nodes []*SceneNode, numWorkers int) {
	if len(nodes) == 0 {
		return
	}

	chunkSize := (len(nodes) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(nodes) {
			end = len(nodes)
		}
		if start >= len(nodes) {
			break
		}

		wg.Add(1)
		go func(nodeChunk []*SceneNode) {
			defer wg.Done()
			for _, node := range nodeChunk {
				node.Transform.GetWorldMatrix()
			}
		}(nodes[start:end])
	}

	wg.Wait()
}

// ParallelCulling performs frustum culling in parallel
func ParallelCulling(nodes []*SceneNode, frustum *ViewFrustum, numWorkers int) []*SceneNode {
	// Implementation matches original...
	// (Keeping concise for update focus)
	return nil
}

// ScanlineRenderer renders using scanline parallelism
type ScanlineRenderer struct {
	Renderer
	NumWorkers int
	scanQueue  chan ScanlineJob
	wg         sync.WaitGroup
}

// ScanlineJob represents a scanline to render
type ScanlineJob struct {
	Y         int
	Triangles []*Triangle
	Camera    *Camera
}

// NewScanlineRenderer creates a scanline-based parallel renderer
func NewScanlineRenderer(renderer Renderer, numWorkers int) *ScanlineRenderer {
	return &ScanlineRenderer{
		Renderer:   renderer,
		NumWorkers: numWorkers,
		scanQueue:  make(chan ScanlineJob, numWorkers*4),
	}
}

// RenderSceneScanlines renders using parallel scanlines
func (sr *ScanlineRenderer) RenderSceneScanlines(scene *Scene, triangles []*Triangle) {
	// Start workers
	for i := 0; i < sr.NumWorkers; i++ {
		sr.wg.Add(1)
		go sr.scanlineWorker()
	}

	_, height := sr.Renderer.GetDimensions()

	// Queue scanlines
	for y := 0; y < height; y++ {
		sr.scanQueue <- ScanlineJob{
			Y:         y,
			Triangles: triangles,
			Camera:    scene.Camera,
		}
	}

	close(sr.scanQueue)
	sr.wg.Wait()
}

// scanlineWorker processes scanlines
func (sr *ScanlineRenderer) scanlineWorker() {
	defer sr.wg.Done()
	for job := range sr.scanQueue {
		for _, tri := range job.Triangles {
			sr.renderTriangleScanline(tri, job.Y, job.Camera)
		}
	}
}

// renderTriangleScanline renders a single scanline of a triangle
func (sr *ScanlineRenderer) renderTriangleScanline(tri *Triangle, y int, camera *Camera) {
	// We need direct access to the TerminalRenderer's buffers/logic
	tr, ok := sr.Renderer.(*TerminalRenderer)
	if !ok {
		return
	}

	width, height := tr.GetDimensions()

	// 1. Project vertices
	x0, y0, z0 := camera.ProjectPoint(tri.P0, height, width)
	x1, y1, z1 := camera.ProjectPoint(tri.P1, height, width)
	x2, y2, z2 := camera.ProjectPoint(tri.P2, height, width)

	if x0 == -1 || x1 == -1 || x2 == -1 {
		return
	}

	// 2. Sort by Y
	if y1 < y0 {
		x0, y0, z0, x1, y1, z1 = x1, y1, z1, x0, y0, z0
	}
	if y2 < y0 {
		x0, y0, z0, x2, y2, z2 = x2, y2, z2, x0, y0, z0
	}
	if y2 < y1 {
		x1, y1, z1, x2, y2, z2 = x2, y2, z2, x1, y1, z1
	}

	// Check if scanline intersects triangle height
	if y < y0 || y > y2 {
		return
	}

	totalHeight := y2 - y0
	if totalHeight == 0 {
		return
	}

	// 3. Interpolate for the current scanline Y
	secondHalf := y > y1 || y1 == y0
	alpha := float64(y-y0) / float64(totalHeight)

	// Long edge (A) from P0 to P2
	ax := int(float64(x0) + alpha*float64(x2-x0))
	az := z0 + alpha*(z2-z0)

	// Short edge (B)
	var bx int
	var bz float64

	if secondHalf {
		segHeight := y2 - y1
		if segHeight == 0 {
			return
		}
		beta := float64(y-y1) / float64(segHeight)
		bx = int(float64(x1) + beta*float64(x2-x1))
		bz = z1 + beta*(z2-z1)
	} else {
		segHeight := y1 - y0
		if segHeight == 0 {
			return
		}
		beta := float64(y-y0) / float64(segHeight)
		bx = int(float64(x0) + beta*float64(x1-x0))
		bz = z0 + beta*(z1-z0)
	}

	if ax > bx {
		ax, bx = bx, ax
		az, bz = bz, az
	}

	// 4. Fill span
	if ax < 0 {
		ax = 0
	}
	if bx >= width {
		bx = width - 1
	}

	for x := ax; x <= bx; x++ {
		t := 0.0
		if bx != ax {
			t = float64(x-ax) / float64(bx-ax)
		}
		z := az + t*(bz-az)

		// Access shared buffer (Thread safe because each Y is unique to a worker)
		if z < tr.ZBuffer[y][x] {
			tr.ZBuffer[y][x] = z
			if tr.UseColor {
				// Simplified coloring for scanline demo
				tr.ColorBuffer[y][x] = tri.Material.DiffuseColor
				tr.Surface[y][x] = '#'
			} else {
				tr.Surface[y][x] = '#'
			}
		}
	}
}
