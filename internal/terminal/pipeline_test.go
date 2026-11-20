package terminal

import (
	"bufio"
	"fmt"
	"io"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func TestRenderingModeAndAACombinations(t *testing.T) {
	for mode := engine.RenderModeSingle; mode <= engine.RenderModeParallelBatched; mode++ {
		for aa := engine.AANone; aa <= engine.AASSAA; aa++ {
			t.Run(fmt.Sprintf("mode%d-aa%d", mode, aa), func(t *testing.T) {
				base := NewTerminalRenderer(bufio.NewWriter(io.Discard), 24, 60)
				base.ShadowRenderer = nil
				scene := engine.NewScene()
				scene.Camera.SetPosition(0, 0, 0)
				triangle := engine.NewTriangle(engine.Point{X: -3, Y: -3, Z: 15}, engine.Point{X: 0, Y: 3, Z: 15}, engine.Point{X: 3, Y: -3, Z: 15}, 'X')
				scene.AddNode(engine.NewSceneNodeWithObject("triangle", triangle))
				renderer := ComposeRenderer(base, engine.EngineConfig{RenderMode: mode, AAMode: aa, NumWorkers: 2, TileSize: 8})
				renderer.SetCamera(scene.Camera)
				for frame := 0; frame < 2; frame++ {
					renderer.RenderScene(scene)
					pixels := 0
					for _, row := range base.Surface {
						for _, cell := range row {
							if cell != ' ' {
								pixels++
							}
						}
					}
					if pixels == 0 {
						t.Fatal("no geometry rendered")
					}
				}
				if aa != engine.AANone {
					aaRenderer, ok := renderer.(*AARenderer)
					if !ok {
						t.Fatal("AA pass missing")
					}
					resolved, ok := terminalBuffer(aaRenderer.Renderer)
					if !ok || resolved != base {
						t.Fatal("AA cannot access framebuffer")
					}
				}
			})
		}
	}
}
