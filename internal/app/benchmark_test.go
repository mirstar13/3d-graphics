package app

import (
	"context"
	"testing"
)

func TestSwitchRoomBenchmarkReplay(t *testing.T) {
	t.Chdir("../..")
	options := benchmarkOptions{
		workload: "switch-room", profile: "gameplay", viewport: "full",
		raster: "single", filter: "none", color: "none",
		width: 180, height: 65, frames: 18, warmup: 3, fps: 60, tile: 32,
	}
	single, err := runTerminalReplay(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	options.raster, options.filter, options.viewport = "batched", "cpu", "centered"
	batched, err := runTerminalReplay(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if single.State != batched.State || single.State.Tick == 0 {
		t.Fatalf("replay state differs by pipeline or viewport: %+v versus %+v", single.State, batched.State)
	}
	last := single.Frames[len(single.Frames)-1]
	if last.Width != options.width-1 || last.Height >= options.height || !last.Presented || last.Bytes == 0 {
		t.Fatalf("invalid full-grid presentation: %+v", last)
	}
	centered := batched.Frames[len(batched.Frames)-1]
	if centered.Width > 160 || centered.Height > 60 {
		t.Fatalf("centered viewport exceeded its budget: %+v", centered)
	}
}
