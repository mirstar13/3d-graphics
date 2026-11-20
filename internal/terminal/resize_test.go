package terminal

import (
	"bufio"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"io"
	"testing"
)

func TestResizeAcrossPipelines(t *testing.T) {
	for mode := 0; mode < 4; mode++ {
		for aa := 0; aa < 5; aa++ {
			r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 20, 30)
			pipeline := ComposeRenderer(r, engine.EngineConfig{RenderMode: engine.RenderMode(mode), AAMode: engine.AAMode(aa), NumWorkers: 2, TileSize: 8})
			scene := qualityScene()
			for _, size := range [][2]int{{9, 7}, {80, 24}, {1, 1}} {
				if !r.Resize(size[0], size[1]) {
					t.Fatal("resize rejected")
				}
				pipeline.RenderScene(scene)
				pipeline.RenderScene(scene)
				if len(r.Surface) != size[1] || len(r.Surface[0]) != size[0] || r.ClipMaxX != size[0] {
					t.Fatal("stale buffers or clipping")
				}
			}
		}
	}
}
