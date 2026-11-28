package terminal

import (
	"reflect"
	"testing"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func TestAdaptiveQualityPreservesGridUntilOutputFallsBehind(t *testing.T) {
	var quality AdaptiveQuality
	camera := engine.NewCamera()
	budget := 16 * time.Millisecond
	advance := func(overload, behind bool) QualityTier {
		camera.MoveBy(.01, 0, 0)
		work := budget / 2
		if overload {
			work = budget * 2
		}
		return quality.Update(camera, work, behind, budget)
	}
	advance(false, false)
	advance(true, false)
	if got := advance(false, false); got != FullQuality {
		t.Fatal(got)
	}
	for i := 0; i < 4; i++ {
		advance(true, false)
	}
	if quality.Tier != ReducedShading {
		t.Fatal(quality.Tier)
	}
	for i := 0; i < 8; i++ {
		advance(true, false)
	}
	if quality.Tier != ReducedShading {
		t.Fatal(quality.Tier)
	}
	for i := 0; i < 4; i++ {
		advance(true, true)
	}
	if quality.Tier != ReducedGrid {
		t.Fatal(quality.Tier)
	}
	for i := 0; i < 8; i++ {
		quality.Update(camera, budget*2, true, budget)
	}
	if quality.Tier != FullQuality {
		t.Fatal(quality.Tier)
	}
}

func TestRenderQualityRestoresNativeGrid(t *testing.T) {
	r := NewTerminalRenderer(nil, 9, 13)
	w, h := r.RenderQuality(ReducedGrid, func() {
		r.BeginFrame()
		r.Surface[r.Height-1][r.Width-1] = 'X'
	})
	if w != 9 || h != 6 || r.Width != 13 || r.Height != 9 || r.Surface[8][12] != 'X' {
		t.Fatalf("raster %dx%d, output %dx%d", w, h, r.Width, r.Height)
	}
}

func TestLatestFrameSkipsStaleFramesDuringWrite(t *testing.T) {
	started, unblock := make(chan struct{}), make(chan struct{})
	var delivered []int
	queue := NewLatestFrame(func(frame int) {
		if frame == 1 {
			close(started)
			<-unblock
		}
		delivered = append(delivered, frame)
	})
	queue.Submit(1)
	<-started
	queue.Submit(2)
	queue.Submit(3)
	close(unblock)
	queue.Close()
	if !reflect.DeepEqual(delivered, []int{1, 3}) {
		t.Fatal(delivered)
	}
}
