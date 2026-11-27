package app

import (
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

func TestBackendSelection(t *testing.T) {
	adapter, err := createBackend(engine.EngineConfig{Backend: "terminal", Width: 80, Height: 24})
	if err != nil {
		t.Fatal(err)
	}
	width, height := adapter.Renderer.GetDimensions()
	if width != 80 || height != 24 || !adapter.Terminal {
		t.Fatalf("unexpected terminal adapter dimensions %dx%d", width, height)
	}
	if adapter.NewInput() == nil {
		t.Fatal("missing terminal input")
	}
	if _, err := createBackend(engine.EngineConfig{Backend: "missing"}); err == nil {
		t.Fatal("unknown backend accepted")
	}
}

func TestBackendRegistration(t *testing.T) {
	RegisterBackend("test-adapter", func(config engine.EngineConfig) (BackendAdapter, error) {
		config.Backend = "terminal"
		return createBackend(config)
	})
	defer delete(backendFactories, "test-adapter")
	if _, err := createBackend(engine.EngineConfig{Backend: "test-adapter", Width: 80, Height: 24}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenGLBackendsUseTerminalPipeline(t *testing.T) {
	for _, backend := range []string{"terminal-gl", "terminal-gl-raster"} {
		for _, mode := range []engine.AAMode{engine.AAFXAA, engine.AAMSAA4x} {
			config := engine.EngineConfig{Backend: backend, Width: 80, Height: 24, AAMode: mode}
			adapter, err := createBackend(config)
			if err != nil {
				t.Fatal(err)
			}
			renderer, ok := adapter.Renderer.(*terminal.TerminalRenderer)
			if !ok || !adapter.Terminal || !isTerminalBackend(config.Backend) {
				t.Fatal("OpenGL backend lost terminal input or presentation")
			}
			if renderer.UseOpenGLFXAA != (backend == "terminal-gl" && mode == engine.AAFXAA) ||
				renderer.UseOpenGLRaster != (backend == "terminal-gl-raster") {
				t.Fatal("wrong OpenGL pass selected")
			}
		}
	}
}
