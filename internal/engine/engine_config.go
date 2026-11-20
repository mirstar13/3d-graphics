package engine

// RenderMode specifies the rendering approach
type RenderMode int

const (
	RenderModeSingle RenderMode = iota
	RenderModeParallelTiles
	RenderModeParallelJobs
	RenderModeParallelBatched
)

// OrientationType controls camera axis conventions for a backend.
type OrientationType int

const OrientationTerminal OrientationType = 1

// EngineConfig holds engine configuration
type EngineConfig struct {
	RequestedWidth, RequestedHeight int // User viewport limits before terminal fitting.
	FOVX                            float64
	FOVY                            float64
	AutoOrbit                       bool
	ColorMode                       string
	Width                           int
	Height                          int
	FPS                             float64
	UseColor                        bool
	ShowDebugInfo                   bool
	RenderMode                      RenderMode
	Backend                         string
	NumWorkers                      int
	TileSize                        int
	EnableProfiling                 bool
	AAMode                          AAMode
}
