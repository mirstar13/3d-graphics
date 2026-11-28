package app

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

type replayFilter struct {
	driver string
	apply  func(*terminal.TerminalRenderer) (terminal.DeviceStages, int64, error)
	begin  func() error
	end    func() (*benchmarkGC, error)
	close  func()
}

func localReplayFilter(gl bool) (*replayFilter, error) {
	if !gl {
		filter, err := terminal.OpenColorFilter(false)
		if err != nil {
			return nil, err
		}
		return &replayFilter{
			apply: func(r *terminal.TerminalRenderer) (terminal.DeviceStages, int64, error) {
				stages, err := filter.Apply(r)
				return stages, 0, err
			},
			close: filter.Close,
		}, nil
	}
	// Keep the FXAA context on its own OS thread. The calling goroutine
	// owns raster, and lends the color buffer until the filter replies.
	type reply struct {
		driver string
		stages terminal.DeviceStages
		err    error
	}
	frames := make(chan *terminal.TerminalRenderer)
	replies := make(chan reply)
	done := make(chan struct{})
	go func() {
		defer close(done)
		f, err := terminal.OpenColorFilter(true)
		if err != nil {
			replies <- reply{err: err}
			return
		}
		defer f.Close()
		replies <- reply{driver: f.Driver()}
		for r := range frames {
			stages, err := f.Apply(r)
			replies <- reply{stages: stages, err: err}
		}
	}()
	ready := <-replies
	if ready.err != nil {
		return nil, ready.err
	}
	return &replayFilter{
		driver: ready.driver,
		apply: func(r *terminal.TerminalRenderer) (terminal.DeviceStages, int64, error) {
			frames <- r
			result := <-replies
			return result.stages, 0, result.err
		},
		close: func() { close(frames); <-done },
	}, nil
}

type filterRequest struct {
	Operation     string
	Width, Height int
	Pixels        []byte
	Detail        bool
	MemoryBytes   uint64
}

type filterReply struct {
	Driver, Error string
	Pixels        []byte
	Stages        terminal.DeviceStages
	GC            *benchmarkGC
}

func encodeColors(colors [][]engine.Color, pixels []byte) []byte {
	size := len(colors) * len(colors[0]) * 3
	if cap(pixels) < size {
		pixels = make([]byte, size)
	}
	pixels = pixels[:size]
	i := 0
	for _, row := range colors {
		for _, color := range row {
			pixels[i], pixels[i+1], pixels[i+2] = color.R, color.G, color.B
			i += 3
		}
	}
	return pixels
}

func decodeColors(pixels []byte, colors [][]engine.Color) {
	i := 0
	for _, row := range colors {
		for x := range row {
			row[x] = engine.Color{R: pixels[i], G: pixels[i+1], B: pixels[i+2]}
			i += 3
		}
	}
}

func remoteReplayFilter(path, overrides string, memory uint64, tracePath string) (*replayFilter, error) {
	env := map[string]string{}
	if err := json.Unmarshal([]byte(overrides), &env); err != nil {
		return nil, err
	}
	cmd := exec.Command(path, "-worker", "-trace", tracePath)
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for name, value := range env {
		cmd.Env = append(cmd.Env, name+"="+value)
	}
	send, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	recv, err := cmd.StdoutPipe()
	if err != nil {
		send.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		send.Close()
		recv.Close()
		return nil, err
	}
	stop := func() {
		send.Close()
		recv.Close()
		_ = cmd.Wait()
	}
	encoder, decoder := gob.NewEncoder(send), gob.NewDecoder(recv)
	var response filterReply
	exchange := func(request filterRequest) error {
		if err := encoder.Encode(request); err != nil {
			return fmt.Errorf("send FXAA worker request: %w", err)
		}
		response = filterReply{Pixels: response.Pixels[:0]}
		if err := decoder.Decode(&response); err != nil {
			return fmt.Errorf("read FXAA worker response: %w", err)
		}
		if response.Error != "" {
			return fmt.Errorf("FXAA worker: %s", response.Error)
		}
		return nil
	}
	if err := exchange(filterRequest{Operation: "probe"}); err != nil {
		stop()
		return nil, err
	}
	var pixels []byte
	return &replayFilter{
		driver: response.Driver,
		begin:  func() error { return exchange(filterRequest{Operation: "begin"}) },
		end: func() (*benchmarkGC, error) {
			err := exchange(filterRequest{Operation: "end"})
			return response.GC, err
		},
		close: stop,
		apply: func(r *terminal.TerminalRenderer) (terminal.DeviceStages, int64, error) {
			start := time.Now()
			pixels = encodeColors(r.ColorBuffer, pixels)
			err := exchange(filterRequest{Width: r.Width, Height: r.Height, Pixels: pixels,
				Detail: r.Stages != nil, MemoryBytes: memory})
			if err != nil {
				return terminal.DeviceStages{}, 0, err
			}
			if len(response.Pixels) != len(pixels) {
				return terminal.DeviceStages{}, 0, fmt.Errorf("FXAA worker returned %d bytes for %dx%d", len(response.Pixels), r.Width, r.Height)
			}
			decodeColors(response.Pixels, r.ColorBuffer)
			return response.Stages, time.Since(start).Nanoseconds(), nil
		},
	}, nil
}

func runFilterWorker(in io.Reader, out io.Writer) error {
	decoder, encoder := gob.NewDecoder(in), gob.NewEncoder(out)
	filter, openErr := terminal.OpenColorFilter(true)
	if openErr == nil {
		defer filter.Close()
	}
	meter := newGCMeter()
	var request filterRequest
	var colors [][]engine.Color
	var pixels []byte
	for {
		request = filterRequest{Pixels: request.Pixels[:0]}
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		result := filterReply{}
		switch {
		case openErr != nil:
			result.Error = openErr.Error()
		case request.Operation == "probe":
			result.Driver = filter.Driver()
		case request.Operation == "begin":
			meter.begin()
		case request.Operation == "end":
			stats := meter.end()
			result.GC = &stats
		case request.Width < 1 || request.Width > 4096 || request.Height < 1 || request.Height > 4096 ||
			len(request.Pixels) != request.Width*request.Height*3:
			result.Error = "invalid FXAA input dimensions"
		case request.MemoryBytes > 0 && meter.memory() > request.MemoryBytes:
			result.Error = "FXAA worker reached the GC experiment memory ceiling"
		default:
			if len(colors) != request.Height || len(colors[0]) != request.Width {
				colors = make([][]engine.Color, request.Height)
				for y := range colors {
					colors[y] = make([]engine.Color, request.Width)
				}
			}
			decodeColors(request.Pixels, colors)
			var detail *terminal.DeviceStages
			if request.Detail {
				detail = &result.Stages
			}
			if _, err := filter.ApplyColors(colors, detail); err != nil {
				result.Error = err.Error()
			} else {
				pixels = encodeColors(colors, pixels)
				result.Pixels = pixels
			}
		}
		if err := encoder.Encode(result); err != nil {
			return err
		}
	}
}
