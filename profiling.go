package main

import (
	"fmt"
	"time"
)

// PerformanceStats tracks rendering performance metrics
type PerformanceStats struct {
	// Frame timing
	FrameTime   time.Duration
	UpdateTime  time.Duration
	RenderTime  time.Duration
	PresentTime time.Duration
	FPS         float64

	// Rendering stats
	TrianglesTotal    int
	TrianglesRendered int
	TrianglesCulled   int
	DrawCalls         int

	// Culling stats
	NodesTested  int
	NodesCulled  int
	NodesVisible int

	// LOD stats
	LODGroups  int
	LOD0Active int
	LOD1Active int
	LOD2Active int
	LODOther   int

	// Memory stats
	PoolTriangles int
	PoolQuads     int
	PoolPoints    int
	PoolMatrices  int

	// BVH/Octree stats
	BVHNodes       int
	OctreeNodes    int
	SpatialQueries int

	// Detailed timing
	CullingTime       time.Duration
	TransformTime     time.Duration
	LightingTime      time.Duration
	RasterizationTime time.Duration
	ClippingTime      time.Duration
}

// Profiler manages performance profiling
type Profiler struct {
	enabled bool
	stats   PerformanceStats

	// Frame history
	frameHistory   []PerformanceStats
	maxHistorySize int
	historyIndex   int

	// Timing markers
	frameStart     time.Time
	updateStart    time.Time
	renderStart    time.Time
	presentStart   time.Time
	cullingStart   time.Time
	transformStart time.Time
	lightingStart  time.Time
	rasterizeStart time.Time
	clippingStart  time.Time
}

// NewProfiler creates a new profiler
func NewProfiler(historySize int) *Profiler {
	return &Profiler{
		enabled:        true,
		maxHistorySize: historySize,
		frameHistory:   make([]PerformanceStats, historySize),
		historyIndex:   0,
	}
}

// BeginFrame marks the start of a frame
func (p *Profiler) BeginFrame() {
	if !p.enabled {
		return
	}
	p.frameStart = time.Now()
	p.stats = PerformanceStats{} // Reset stats
}

// EndFrame marks the end of a frame and calculates FPS
func (p *Profiler) EndFrame() {
	if !p.enabled {
		return
	}

	p.stats.FrameTime = time.Since(p.frameStart)
	if p.stats.FrameTime > 0 {
		p.stats.FPS = 1.0 / p.stats.FrameTime.Seconds()
	}

	// Add to history
	p.frameHistory[p.historyIndex] = p.stats
	p.historyIndex = (p.historyIndex + 1) % p.maxHistorySize
}

// BeginUpdate marks start of update phase
func (p *Profiler) BeginUpdate() {
	if !p.enabled {
		return
	}
	p.updateStart = time.Now()
}

// EndUpdate marks end of update phase
func (p *Profiler) EndUpdate() {
	if !p.enabled {
		return
	}
	p.stats.UpdateTime = time.Since(p.updateStart)
}

// BeginRender marks start of render phase
func (p *Profiler) BeginRender() {
	if !p.enabled {
		return
	}
	p.renderStart = time.Now()
}

// EndRender marks end of render phase
func (p *Profiler) EndRender() {
	if !p.enabled {
		return
	}
	p.stats.RenderTime = time.Since(p.renderStart)
}

// BeginPresent marks start of present phase
func (p *Profiler) BeginPresent() {
	if !p.enabled {
		return
	}
	p.presentStart = time.Now()
}

// EndPresent marks end of present phase
func (p *Profiler) EndPresent() {
	if !p.enabled {
		return
	}
	p.stats.PresentTime = time.Since(p.presentStart)
}

// Detailed timing markers
func (p *Profiler) BeginCulling() {
	if !p.enabled {
		return
	}
	p.cullingStart = time.Now()
}

func (p *Profiler) EndCulling() {
	if !p.enabled {
		return
	}
	p.stats.CullingTime += time.Since(p.cullingStart)
}

func (p *Profiler) BeginTransform() {
	if !p.enabled {
		return
	}
	p.transformStart = time.Now()
}

func (p *Profiler) EndTransform() {
	if !p.enabled {
		return
	}
	p.stats.TransformTime += time.Since(p.transformStart)
}

func (p *Profiler) BeginLighting() {
	if !p.enabled {
		return
	}
	p.lightingStart = time.Now()
}

func (p *Profiler) EndLighting() {
	if !p.enabled {
		return
	}
	p.stats.LightingTime += time.Since(p.lightingStart)
}

func (p *Profiler) BeginRasterization() {
	if !p.enabled {
		return
	}
	p.rasterizeStart = time.Now()
}

func (p *Profiler) EndRasterization() {
	if !p.enabled {
		return
	}
	p.stats.RasterizationTime += time.Since(p.rasterizeStart)
}

func (p *Profiler) BeginClipping() {
	if !p.enabled {
		return
	}
	p.clippingStart = time.Now()
}

func (p *Profiler) EndClipping() {
	if !p.enabled {
		return
	}
	p.stats.ClippingTime += time.Since(p.clippingStart)
}

// Stat recording methods
func (p *Profiler) RecordTriangle(rendered bool) {
	if !p.enabled {
		return
	}
	p.stats.TrianglesTotal++
	if rendered {
		p.stats.TrianglesRendered++
	} else {
		p.stats.TrianglesCulled++
	}
}

func (p *Profiler) RecordDrawCall() {
	if !p.enabled {
		return
	}
	p.stats.DrawCalls++
}

func (p *Profiler) RecordNode(visible bool) {
	if !p.enabled {
		return
	}
	p.stats.NodesTested++
	if visible {
		p.stats.NodesVisible++
	} else {
		p.stats.NodesCulled++
	}
}

func (p *Profiler) RecordLOD(level int) {
	if !p.enabled {
		return
	}
	p.stats.LODGroups++
	switch level {
	case 0:
		p.stats.LOD0Active++
	case 1:
		p.stats.LOD1Active++
	case 2:
		p.stats.LOD2Active++
	default:
		p.stats.LODOther++
	}
}

func (p *Profiler) RecordPoolUsage(triangles, quads, points, matrices int) {
	if !p.enabled {
		return
	}
	p.stats.PoolTriangles = triangles
	p.stats.PoolQuads = quads
	p.stats.PoolPoints = points
	p.stats.PoolMatrices = matrices
}

func (p *Profiler) RecordSpatialStructure(bvhNodes, octreeNodes, queries int) {
	if !p.enabled {
		return
	}
	p.stats.BVHNodes = bvhNodes
	p.stats.OctreeNodes = octreeNodes
	p.stats.SpatialQueries = queries
}

// GetStats returns current frame stats
func (p *Profiler) GetStats() PerformanceStats {
	return p.stats
}

// GetAverageStats returns average over frame history
func (p *Profiler) GetAverageStats() PerformanceStats {
	if !p.enabled {
		return PerformanceStats{}
	}

	avg := PerformanceStats{}
	count := 0

	for _, stats := range p.frameHistory {
		if stats.FrameTime > 0 {
			avg.FrameTime += stats.FrameTime
			avg.UpdateTime += stats.UpdateTime
			avg.RenderTime += stats.RenderTime
			avg.PresentTime += stats.PresentTime
			avg.FPS += stats.FPS
			avg.TrianglesTotal += stats.TrianglesTotal
			avg.TrianglesRendered += stats.TrianglesRendered
			avg.TrianglesCulled += stats.TrianglesCulled
			avg.DrawCalls += stats.DrawCalls
			avg.NodesTested += stats.NodesTested
			avg.NodesCulled += stats.NodesCulled
			avg.NodesVisible += stats.NodesVisible
			count++
		}
	}

	if count > 0 {
		div := time.Duration(count)
		avg.FrameTime /= div
		avg.UpdateTime /= div
		avg.RenderTime /= div
		avg.PresentTime /= div
		avg.FPS /= float64(count)
		avg.TrianglesTotal /= count
		avg.TrianglesRendered /= count
		avg.TrianglesCulled /= count
		avg.DrawCalls /= count
		avg.NodesTested /= count
		avg.NodesCulled /= count
		avg.NodesVisible /= count
	}

	return avg
}

// Enable/Disable profiling
func (p *Profiler) SetEnabled(enabled bool) {
	p.enabled = enabled
}

func (p *Profiler) IsEnabled() bool {
	return p.enabled
}

// Format stats as string
func (ps *PerformanceStats) String() string {
	return fmt.Sprintf(
		"FPS: %.1f | Frame: %.2fms | Render: %.2fms | Tris: %d/%d (%.1f%% culled) | Draws: %d | Nodes: %d/%d visible",
		ps.FPS,
		ps.FrameTime.Seconds()*1000,
		ps.RenderTime.Seconds()*1000,
		ps.TrianglesRendered,
		ps.TrianglesTotal,
		float64(ps.TrianglesCulled)/float64(ps.TrianglesTotal)*100.0,
		ps.DrawCalls,
		ps.NodesVisible,
		ps.NodesTested,
	)
}

// DetailedString returns detailed stats
func (ps *PerformanceStats) DetailedString() string {
	return fmt.Sprintf(`
=== Performance Stats ===
Frame Time:    %.2fms (%.1f FPS)
  Update:      %.2fms
  Render:      %.2fms
    Culling:   %.2fms
    Transform: %.2fms
    Lighting:  %.2fms
    Rasterize: %.2fms
    Clipping:  %.2fms
  Present:     %.2fms

Rendering:
  Triangles:   %d rendered / %d total (%d culled, %.1f%%)
  Draw Calls:  %d
  Nodes:       %d visible / %d tested (%d culled, %.1f%%)

LOD:
  Groups:      %d
  LOD0:        %d (%.1f%%)
  LOD1:        %d (%.1f%%)
  LOD2:        %d (%.1f%%)

Memory Pools:
  Triangles:   %d
  Quads:       %d
  Points:      %d
  Matrices:    %d

Spatial:
  BVH Nodes:   %d
  Octree:      %d
  Queries:     %d
`,
		ps.FrameTime.Seconds()*1000,
		ps.FPS,
		ps.UpdateTime.Seconds()*1000,
		ps.RenderTime.Seconds()*1000,
		ps.CullingTime.Seconds()*1000,
		ps.TransformTime.Seconds()*1000,
		ps.LightingTime.Seconds()*1000,
		ps.RasterizationTime.Seconds()*1000,
		ps.ClippingTime.Seconds()*1000,
		ps.PresentTime.Seconds()*1000,
		ps.TrianglesRendered,
		ps.TrianglesTotal,
		ps.TrianglesCulled,
		safePercent(ps.TrianglesCulled, ps.TrianglesTotal),
		ps.DrawCalls,
		ps.NodesVisible,
		ps.NodesTested,
		ps.NodesCulled,
		safePercent(ps.NodesCulled, ps.NodesTested),
		ps.LODGroups,
		ps.LOD0Active,
		safePercent(ps.LOD0Active, ps.LODGroups),
		ps.LOD1Active,
		safePercent(ps.LOD1Active, ps.LODGroups),
		ps.LOD2Active,
		safePercent(ps.LOD2Active, ps.LODGroups),
		ps.PoolTriangles,
		ps.PoolQuads,
		ps.PoolPoints,
		ps.PoolMatrices,
		ps.BVHNodes,
		ps.OctreeNodes,
		ps.SpatialQueries,
	)
}

func safePercent(num, denom int) float64 {
	if denom == 0 {
		return 0.0
	}
	return float64(num) / float64(denom) * 100.0
}
