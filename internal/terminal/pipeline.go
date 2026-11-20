package terminal

import "github.com/mirstar13/go-3d-graphics/internal/engine"

// ComposeRenderer applies CPU scheduling before the anti-aliasing pass.
func ComposeRenderer(base engine.Renderer, config engine.EngineConfig) engine.Renderer {
	renderer := base
	switch config.RenderMode {
	case engine.RenderModeParallelTiles, engine.RenderModeParallelBatched:
		parallel := NewParallelRenderer(base, config.NumWorkers, config.TileSize)
		parallel.Batched = config.RenderMode == engine.RenderModeParallelBatched
		renderer = parallel
	case engine.RenderModeParallelJobs:
		renderer = NewJobBasedRenderer(base, config.NumWorkers)
	}
	if config.AAMode != engine.AANone {
		renderer = NewAARenderer(renderer, config.AAMode)
	}
	return renderer
}

func terminalBuffer(renderer engine.Renderer) (*TerminalRenderer, bool) {
	switch r := renderer.(type) {
	case *TerminalRenderer:
		return r, true
	case *ParallelRenderer:
		return terminalBuffer(r.Renderer)
	case *JobBasedRenderer:
		return terminalBuffer(r.Renderer)
	case *AARenderer:
		return terminalBuffer(r.Renderer)
	default:
		return nil, false
	}
}
