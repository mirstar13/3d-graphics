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

	// Update lighting system
	if ctx.LightingSystem != nil {
		ctx.LightingSystem.SetCamera(scene.Camera)
	}

	// Get visible nodes using frustum culling
	frustum := BuildFrustumSimple(scene.Camera)
	visibleNodes := make([]*SceneNode, 0)
	FrustumCullNode(scene.Root, &frustum, &visibleNodes)

	// Start workers
	pr.startWorkers()

	// Generate tiles and queue them
	pr.generateTiles(visibleNodes, scene.Camera)

	// Wait for all tiles to complete
	pr.wg.Wait()
	close(pr.tileQueue)
}

// startWorkers starts the worker goroutines
func (pr *ParallelRenderer) startWorkers() {
	for i := 0; i < pr.NumWorkers; i++ {
		pr.wg.Add(1)
		go pr.worker()
	}
}

// worker processes tiles from the queue
func (pr *ParallelRenderer) worker() {
	defer pr.wg.Done()

	for tile := range pr.tileQueue {
		pr.renderTile(tile)
	}
}

// generateTiles splits rendering into tiles and queues them
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

			// Clamp to screen bounds
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
				Nodes:  nodes,
				Camera: camera,
			}

			pr.tileQueue <- tile
		}
	}
}

func (pr *ParallelRenderer) renderNodeWithMatrix(node *SceneNode, worldMatrix Matrix4x4, camera *Camera) {
	switch obj := node.Object.(type) {
	case *Mesh:
		pr.Renderer.RenderMesh(obj, worldMatrix, camera)
	case *Triangle:
		pr.Renderer.RenderTriangle(obj, worldMatrix, camera)
	case *Line:
		pr.Renderer.RenderLine(obj, worldMatrix, camera)
	case *Point:
		pr.Renderer.RenderPoint(obj, worldMatrix, camera)
	}
}

// renderTile renders a single tile
func (pr *ParallelRenderer) renderTile(tile RenderTile) {
	// Render each node within this tile
	for _, node := range tile.Nodes {
		pr.renderNodeInTile(node, tile)
	}
}

// renderNodeInTile renders a node within tile bounds
func (pr *ParallelRenderer) renderNodeInTile(node *SceneNode, tile RenderTile) {
	worldMatrix := node.Transform.GetWorldMatrix()

	switch obj := node.Object.(type) {
	case *Mesh:
		pr.renderMeshInTile(obj, worldMatrix, tile)
	case *Triangle:
		pr.renderTriangleInTile(obj, worldMatrix, tile)
	}
}

// renderMeshInTile renders a mesh within tile bounds
func (pr *ParallelRenderer) renderMeshInTile(mesh *Mesh, worldMatrix Matrix4x4, tile RenderTile) {
	// Transform and render triangles
	for _, tri := range mesh.Triangles {
		// Get pooled triangle
		transformed := pr.Pools.Triangles.Get()
		CopyTriangle(transformed, tri)

		// Transform vertices
		transformed.P0 = worldMatrix.TransformPoint(tri.P0)
		transformed.P1 = worldMatrix.TransformPoint(tri.P1)
		transformed.P2 = worldMatrix.TransformPoint(tri.P2)

		// Transform normal if set
		if tri.UseSetNormal && tri.Normal != nil {
			transformedNormal := worldMatrix.TransformDirection(*tri.Normal)
			transformed.Normal = &transformedNormal
		}

		// Check if triangle overlaps tile
		if pr.triangleOverlapsTile(transformed, tile) {
			// Render within tile bounds
			pr.renderTriangleClipped(transformed, worldMatrix, tile)
		}
	}
}

// renderTriangleInTile renders a single triangle within tile
func (pr *ParallelRenderer) renderTriangleInTile(tri *Triangle, worldMatrix Matrix4x4, tile RenderTile) {
	transformed := pr.Pools.Triangles.Get()
	CopyTriangle(transformed, tri)

	transformed.P0 = worldMatrix.TransformPoint(tri.P0)
	transformed.P1 = worldMatrix.TransformPoint(tri.P1)
	transformed.P2 = worldMatrix.TransformPoint(tri.P2)

	if tri.UseSetNormal && tri.Normal != nil {
		transformedNormal := worldMatrix.TransformDirection(*tri.Normal)
		transformed.Normal = &transformedNormal
	}

	if pr.triangleOverlapsTile(transformed, tile) {
		pr.renderTriangleClipped(transformed, worldMatrix, tile)
	}
}

// triangleOverlapsTile checks if triangle overlaps with tile
func (pr *ParallelRenderer) triangleOverlapsTile(tri *Triangle, tile RenderTile) bool {
	width, height := pr.Renderer.GetDimensions()
	ctx := pr.Renderer.GetRenderContext()

	// Project vertices
	x0, y0, _ := ctx.Camera.ProjectPoint(tri.P0, height, width)
	x1, y1, _ := ctx.Camera.ProjectPoint(tri.P1, height, width)
	x2, y2, _ := ctx.Camera.ProjectPoint(tri.P2, height, width)

	// Check if any vertex is behind camera
	if x0 == -1 && x1 == -1 && x2 == -1 {
		return false
	}

	// Calculate triangle bounds
	minX := min3(x0, x1, x2)
	maxX := max3(x0, x1, x2)
	minY := min3(y0, y1, y2)
	maxY := max3(y0, y1, y2)

	// Check overlap with tile
	if maxX < tile.X || minX >= tile.X+tile.Width {
		return false
	}
	if maxY < tile.Y || minY >= tile.Y+tile.Height {
		return false
	}

	return true
}

// renderTriangleClipped renders triangle clipped to tile bounds
func (pr *ParallelRenderer) renderTriangleClipped(tri *Triangle, worldMatrix Matrix4x4, tile RenderTile) {
	// Use standard rendering but restrict to tile bounds
	// This is a simplified version - full implementation would clip geometry
	pr.Renderer.RenderTriangle(tri, worldMatrix, pr.Renderer.GetRenderContext().Camera)
}

// RenderBatched renders scene with draw call batching
func (pr *ParallelRenderer) RenderBatched(scene *Scene) {
	pr.Renderer.BeginFrame()
	pr.Pools.ResetAll()

	ctx := pr.Renderer.GetRenderContext()

	if ctx.LightingSystem != nil {
		ctx.LightingSystem.SetCamera(scene.Camera)
	}

	// Get visible nodes
	frustum := BuildFrustumSimple(scene.Camera)
	visibleNodes := make([]*SceneNode, 0)
	FrustumCullNode(scene.Root, &frustum, &visibleNodes)

	// Batch by material
	batches := pr.batchByMaterial(visibleNodes)

	// Render each batch
	for _, batch := range batches {
		pr.renderBatch(batch)
	}
}

// RenderBatch represents a batch of nodes with same material
type RenderBatch struct {
	Material Material
	Nodes    []*SceneNode
}

// batchByMaterial groups nodes by material
func (pr *ParallelRenderer) batchByMaterial(nodes []*SceneNode) []RenderBatch {
	batchMap := make(map[Material]*RenderBatch)

	for _, node := range nodes {
		mat := pr.getMaterialFromNode(node)

		if batchMap[mat] == nil {
			batchMap[mat] = &RenderBatch{
				Material: mat,
				Nodes:    make([]*SceneNode, 0),
			}
		}
		batchMap[mat].Nodes = append(batchMap[mat].Nodes, node)
	}

	// Convert to slice
	batches := make([]RenderBatch, 0, len(batchMap))
	for _, batch := range batchMap {
		batches = append(batches, *batch)
	}

	return batches
}

// getMaterialFromNode extracts material from node's object
func (pr *ParallelRenderer) getMaterialFromNode(node *SceneNode) Material {
	switch obj := node.Object.(type) {
	case *Mesh:
		if len(obj.Triangles) > 0 {
			return obj.Triangles[0].Material
		}
		if len(obj.Quads) > 0 {
			return obj.Quads[0].Material
		}
	case *Triangle:
		return obj.Material
	case *Quad:
		return obj.Material
	}
	return NewMaterial()
}

// renderBatch renders all nodes in a batch
func (pr *ParallelRenderer) renderBatch(batch RenderBatch) {
	for _, node := range batch.Nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		pr.renderNodeWithMatrix(node, worldMatrix, pr.Renderer.GetRenderContext().Camera)
	}
}

// JobBasedRenderer implements job-based parallelism
type JobBasedRenderer struct {
	Renderer
	NumWorkers int
	jobQueue   chan RenderJob
	wg         sync.WaitGroup
	Pools      *RenderPools
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
				// Force matrix update
				node.Transform.GetWorldMatrix()
			}
		}(nodes[start:end])
	}

	wg.Wait()
}

// ParallelCulling performs frustum culling in parallel
func ParallelCulling(nodes []*SceneNode, frustum *ViewFrustum, numWorkers int) []*SceneNode {
	if len(nodes) == 0 {
		return nil
	}

	chunkSize := (len(nodes) + numWorkers - 1) / numWorkers
	resultChannels := make([]chan []*SceneNode, numWorkers)

	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(nodes) {
			end = len(nodes)
		}
		if start >= len(nodes) {
			break
		}

		resultChannels[i] = make(chan []*SceneNode, 1)

		go func(nodeChunk []*SceneNode, resultChan chan []*SceneNode) {
			visible := make([]*SceneNode, 0)
			for _, node := range nodeChunk {
				bounds := ComputeNodeBounds(node)
				if bounds != nil && frustum.TestAABB(bounds) {
					visible = append(visible, node)
				}
			}
			resultChan <- visible
			close(resultChan)
		}(nodes[start:end], resultChannels[i])
	}

	// Collect results
	allVisible := make([]*SceneNode, 0)
	for _, ch := range resultChannels {
		if ch != nil {
			visible := <-ch
			allVisible = append(allVisible, visible...)
		}
	}

	return allVisible
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

	width, _ := sr.Renderer.GetDimensions()

	// Queue scanlines
	for y := 0; y < width; y++ {
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
		// Render all triangles for this scanline
		for _, tri := range job.Triangles {
			sr.renderTriangleScanline(tri, job.Y, job.Camera)
		}
	}
}

// renderTriangleScanline renders a single scanline of a triangle
func (sr *ScanlineRenderer) renderTriangleScanline(tri *Triangle, y int, camera *Camera) {
	width, height := sr.Renderer.GetDimensions()

	// Project vertices
	x0, y0, _ := camera.ProjectPoint(tri.P0, height, width)
	x1, y1, _ := camera.ProjectPoint(tri.P1, height, width)
	x2, y2, _ := camera.ProjectPoint(tri.P2, height, width)

	if x0 == -1 || x1 == -1 || x2 == -1 {
		return
	}

	// Check if scanline intersects triangle
	minY := min3(y0, y1, y2)
	maxY := max3(y0, y1, y2)

	if y < minY || y > maxY {
		return
	}

	// Simplified - would need proper edge intersection
	// This is a placeholder for the full scanline rasterization
}
