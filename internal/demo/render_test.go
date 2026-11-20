package demo_test

import (
	"bufio"
	"io"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

func TestScenePresetsRender(t *testing.T) {
	for index, name := range demo.Names {
		t.Run(name, func(t *testing.T) {
			scene := engine.NewScene()
			demo.ConfigureCamera(scene.Camera, index, engine.OrientationTerminal)
			demo.BuildScene(scene, index)
			controller := engine.NewCameraController(scene.Camera)
			demo.ConfigureCameraController(controller, index)
			controller.Update(engine.InputState{}, engine.OrientationTerminal)
			demo.AnimateSceneDemo(scene, index, 0.1, 1.0/60)
			scene.Update(1.0 / 30)
			for _, mode := range []engine.RenderMode{engine.RenderModeSingle, engine.RenderModeParallelTiles} {
				base := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), 32, 96)
				base.ShadowRenderer = nil
				renderer := terminal.ComposeRenderer(base, engine.EngineConfig{RenderMode: mode, NumWorkers: 2, TileSize: 16})
				renderer.SetCamera(scene.Camera)
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
					t.Errorf("mode %d produced an empty frame", mode)
				}
			}
		})
	}
}
