package terminal

import (
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/perfclock"
)

// FrameStages records host durations in nanoseconds. GPUNS overlaps the host
// path. A nil GPU duration means the driver did not supply a ready result.
type FrameStages struct {
	PrepareNS int64
	ShadeNS   int64
	Raster    DeviceStages
	FXAA      DeviceStages
}

type DeviceStages struct {
	PackNS        int64
	UploadNS      int64
	SubmitNS      int64
	ReadbackNS    int64
	GPUNS         *uint64
	UploadBytes   int
	ReadbackBytes int
}

func stageStart[T any](stages *T) time.Time {
	if stages == nil {
		return time.Time{}
	}
	return perfclock.Now()
}

// ColorFilter applies FXAA to a terminal color buffer. The OpenGL variant
// must be opened, applied and closed on the same goroutine.
type ColorFilter struct {
	device  fxaaDevice
	scratch []engine.Color
}

func OpenColorFilter(openGL bool) (*ColorFilter, error) {
	f := new(ColorFilter)
	if openGL {
		var err error
		f.device, err = openFXAADevice()
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

func (f *ColorFilter) Driver() string {
	if f.device == nil {
		return ""
	}
	return f.device.name()
}

func (f *ColorFilter) Apply(r *TerminalRenderer) (DeviceStages, error) {
	var stages *DeviceStages
	if r.Stages != nil {
		stages = &r.Stages.FXAA
	}
	return f.ApplyColors(r.ColorBuffer, stages)
}

func (f *ColorFilter) ApplyColors(colors [][]engine.Color, stages *DeviceStages) (DeviceStages, error) {
	if len(colors) < 3 || len(colors[0]) < 3 {
		return DeviceStages{}, nil
	}
	if f.device == nil {
		f.scratch = filterFXAA(colors, f.scratch)
		return DeviceStages{}, nil
	}
	if profiled, ok := f.device.(interface{ measure(*DeviceStages) }); ok {
		profiled.measure(stages)
	}
	err := f.device.apply(colors)
	if stages != nil {
		return *stages, err
	}
	return DeviceStages{}, err
}

func (f *ColorFilter) Close() {
	if f.device != nil {
		f.device.close()
	}
}
