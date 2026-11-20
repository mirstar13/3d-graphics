package app

import (
	"bufio"
	"fmt"
	"os"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

// BackendAdapter joins rendering, input, and camera conventions.
// NewInput runs after the renderer has initialized, so window adapters can
// attach their input handlers to a live window.
type BackendAdapter struct {
	Renderer    engine.Renderer
	NewInput    func() engine.InputManager
	Orientation engine.OrientationType
	Terminal    bool
}

type BackendFactory func(engine.EngineConfig) (BackendAdapter, error)

var backendFactories = map[string]BackendFactory{
	"terminal": func(config engine.EngineConfig) (BackendAdapter, error) {
		renderer := terminal.NewTerminalRenderer(bufio.NewWriter(os.Stdout), config.Height, config.Width)
		renderer.ColorMode = config.ColorMode
		renderer.RequestedWidth, renderer.RequestedHeight = config.RequestedWidth, config.RequestedHeight
		return BackendAdapter{
			Renderer:    renderer,
			NewInput:    func() engine.InputManager { return terminal.NewTerminalInputManager() },
			Orientation: engine.OrientationTerminal,
			Terminal:    true,
		}, nil
	},
}

// RegisterBackend lets a compiled-in adapter register from its init function.
func RegisterBackend(name string, factory BackendFactory) {
	if name == "" || factory == nil {
		panic("backend registration requires a name and factory")
	}
	if _, exists := backendFactories[name]; exists {
		panic("backend already registered: " + name)
	}
	backendFactories[name] = factory
}

func createBackend(config engine.EngineConfig) (BackendAdapter, error) {
	factory, ok := backendFactories[config.Backend]
	if !ok {
		return BackendAdapter{}, fmt.Errorf("unknown renderer %q", config.Backend)
	}
	adapter, err := factory(config)
	if err != nil {
		return BackendAdapter{}, err
	}
	if adapter.Renderer == nil || adapter.NewInput == nil {
		return BackendAdapter{}, fmt.Errorf("renderer %q has an incomplete adapter", config.Backend)
	}
	return adapter, nil
}
