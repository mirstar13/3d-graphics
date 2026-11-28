package app

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/perfclock"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
	"golang.org/x/term"
)

type benchmarkOptions struct {
	workload, profile, raster, filter, color                      string
	viewport, output, workerPath, workerEnv, tracePath, allocPath string
	width, height, frames, warmup, fps, tile                      int
	duration                                                      time.Duration
	paced, live, detail, worker, probe, adaptive                  bool
	memoryBytes                                                   uint64
}

type benchmarkFrame struct {
	Tier                                                                                                                  terminal.QualityTier
	RasterWidth, RasterHeight                                                                                             int
	InputToWriteNS, InputStartNS                                                                                          int64
	Presented                                                                                                             bool
	PresentNS                                                                                                             int64
	Frame, Columns, Rows, Width, Height                                                                                   int
	Tick                                                                                                                  uint64
	InputEvents, Bytes, ChangedCells                                                                                      int
	InputNS, SimulationNS, LastPhysicsStepNS                                                                              int64
	StartNS, IntervalNS, TotalNS, PaceNS, UpdateNS, PrepareNS, RasterNS, ShadeNS, FilterNS, EncodeNS, WriteNS, ExchangeNS int64
	RasterDevice, FilterDevice                                                                                            terminal.DeviceStages
}

type benchmarkDelivery struct {
	Submitted, Completed, Coalesced                    int
	Bytes                                              int64
	CompletedPerSecond, BytesPerSecond, SampleFraction float64
	InputToWriteP50NS, InputToWriteP99NS               int64
}

type benchmarkResult struct {
	Delivery *benchmarkDelivery

	RasterDriver, FilterDriver, Transport, Presentation string
	Frames                                              []benchmarkFrame
	GC                                                  benchmarkGC
	WorkerGC                                            *benchmarkGC
	State                                               benchmarkState
	ElapsedNS                                           int64
	GoVersion, GOGC, GOMEMLIMIT                         string
	CPUs                                                int
}

type benchmarkState struct {
	Tick            uint64
	Camera, Forward engine.Point
	Geometry        string
}

var benchmarkModes = map[string]engine.RenderMode{
	"single": engine.RenderModeSingle, "tiles": engine.RenderModeParallelTiles,
	"jobs": engine.RenderModeParallelJobs, "batched": engine.RenderModeParallelBatched,
	"gl": engine.RenderModeSingle,
}

func RunTerminalBenchmark(args []string) error {
	o := benchmarkOptions{}
	f := flag.NewFlagSet("terminal-benchmark", flag.ContinueOnError)
	f.StringVar(&o.workload, "workload", "kinetic", "kinetic, wave, physics, switch-room, or fxaa")
	f.StringVar(&o.profile, "profile", "render", "render or gameplay")
	f.StringVar(&o.raster, "raster", "batched", "single, tiles, jobs, batched, or gl")
	f.StringVar(&o.filter, "filter", "none", "none, cpu, or gl")
	f.StringVar(&o.color, "color", "truecolor", "terminal color mode")
	f.StringVar(&o.viewport, "viewport", "fixed", "fixed, centered, or full")
	f.StringVar(&o.output, "output", "", "JSON result file")
	f.StringVar(&o.workerPath, "filter-worker", "", "adapter-specific FXAA executable")
	f.StringVar(&o.workerEnv, "filter-env", "{}", "environment overrides for the FXAA worker")
	f.StringVar(&o.tracePath, "trace", "", "Go execution trace file")
	f.StringVar(&o.allocPath, "alloc-profile", "", "Go allocation profile file")
	f.IntVar(&o.width, "columns", 160, "columns or terminal grid width")
	f.IntVar(&o.height, "rows", 60, "rows or terminal grid height")
	f.IntVar(&o.frames, "frames", 0, "fixed frame count, or 0 to use duration")
	f.IntVar(&o.warmup, "warmup", 8, "warmup frames")
	f.IntVar(&o.fps, "fps", 60, "target frame rate")
	f.IntVar(&o.tile, "tile", 32, "CPU raster tile size")
	f.DurationVar(&o.duration, "duration", 300*time.Millisecond, "minimum timed duration")
	f.BoolVar(&o.adaptive, "adaptive", false, "adapt scene quality during gameplay replay")
	f.BoolVar(&o.paced, "paced", false, "pace frame starts to target FPS")
	f.BoolVar(&o.live, "live", false, "submit gameplay frames to the current terminal")
	f.BoolVar(&o.detail, "detail", false, "record raster, transfer, and GPU timings")
	f.BoolVar(&o.worker, "worker", false, "run the FXAA worker protocol")
	inventory := f.Bool("inventory", false, "list native OpenGL renderer IDs when exposed")
	f.BoolVar(&o.probe, "probe", false, "verify requested OpenGL stages and print their drivers")
	f.Uint64Var(&o.memoryBytes, "memory-bytes", 0, "per-process runtime memory ceiling")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *inventory {
		ids, err := terminal.OpenGLRendererIDs()
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(ids)
	}
	if o.worker {
		return withBenchmarkTrace(o.tracePath, func() error { return runFilterWorker(os.Stdin, os.Stdout) })
	}
	if o.probe {
		return benchmarkProbe(o)
	}
	_, validMode := benchmarkModes[o.raster]
	if !validMode || (o.filter != "none" && o.filter != "cpu" && o.filter != "gl") ||
		(o.profile != "render" && o.profile != "gameplay") ||
		(o.viewport != "fixed" && o.viewport != "centered" && o.viewport != "full") ||
		o.width < 3 || o.width > 4096 || o.height < 3 || o.height > 4096 ||
		o.fps < 1 || o.fps > 1000 || o.tile < 1 || o.warmup < 0 || o.frames < 0 ||
		(o.frames == 0 && o.duration <= 0) || o.output == "" {
		return errors.New("invalid benchmark settings; see -help")
	}
	if o.adaptive && (o.profile != "gameplay" || o.workload == "fxaa") {
		return errors.New("adaptive quality requires a scene gameplay replay")
	}
	if o.live && (o.profile != "gameplay" || !term.IsTerminal(int(os.Stdout.Fd()))) {
		return errors.New("live output requires a gameplay replay and an interactive terminal")
	}
	mode, err := terminal.ResolveColorMode(o.color)
	if err != nil {
		return err
	}
	o.color = mode
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var result benchmarkResult
	if err := withBenchmarkTrace(o.tracePath, func() error {
		var err error
		result, err = runTerminalReplay(ctx, o)
		return err
	}); err != nil {
		return err
	}
	if o.allocPath != "" {
		file, err := os.Create(o.allocPath)
		if err != nil {
			return err
		}
		err = pprof.Lookup("allocs").WriteTo(file, 0)
		if err := errors.Join(err, file.Close()); err != nil {
			return err
		}
	}
	file, err := os.Create(o.output)
	if err != nil {
		return err
	}
	return errors.Join(json.NewEncoder(file).Encode(result), file.Close())
}

func withBenchmarkTrace(path string, run func() error) error {
	if path == "" {
		return run()
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := trace.Start(file); err != nil {
		return err
	}
	defer trace.Stop()
	return run()
}

func benchmarkProbe(o benchmarkOptions) error {
	driver := ""
	if o.filter == "gl" {
		f, err := terminal.OpenColorFilter(true)
		if err != nil {
			return err
		}
		defer f.Close()
		driver = f.Driver()
		cpu, _ := terminal.OpenColorFilter(false)
		colors := [][]engine.Color{
			{{}, {}, {R: 255}},
			{{}, {R: 255}, {R: 255}},
			{{}, {R: 255}, {R: 255}},
		}
		expected := make([][]engine.Color, len(colors))
		for i, row := range colors {
			expected[i] = append([]engine.Color(nil), row...)
		}
		_, _ = cpu.ApplyColors(expected, nil)
		if _, err := f.ApplyColors(colors, nil); err != nil {
			return err
		}
		if colors[1][1] != expected[1][1] {
			return errors.New("OpenGL FXAA probe differs from the CPU filter")
		}
	} else {
		r := terminal.NewTerminalRenderer(nil, 40, 120)
		r.UseOpenGLRaster = true
		r.InitializeOffscreen()
		defer r.Shutdown()
		driver = r.OpenGLRasterDriver()
		scene := engine.NewScene()
		demo.ConfigureCamera(scene.Camera, 0, engine.OrientationTerminal)
		demo.BuildScene(scene, 0)
		r.RenderScene(scene)
		if !r.UsedOpenGLRaster() {
			return errors.New("OpenGL raster probe fell back to CPU")
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"driver": driver})
}

type timedBenchmarkWriter struct {
	writer   io.Writer
	bytes    int
	elapsed  time.Duration
	finished time.Time
	err      error
}

func (w *timedBenchmarkWriter) Write(p []byte) (int, error) {
	start := perfclock.Now()
	n, err := w.writer.Write(p)
	w.finished = perfclock.Now()
	w.elapsed += w.finished.Sub(start)
	w.bytes += n
	if err != nil {
		w.err = err
	}
	return n, err
}

func benchmarkGrid(o benchmarkOptions) (int, int, error) {
	if !o.live {
		return o.width, o.height, nil
	}
	columns, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, 0, err
	}
	if columns < 3 || columns > 4096 || rows < 3 || rows > 4096 {
		return 0, 0, fmt.Errorf("terminal grid %dx%d outside usable range", columns, rows)
	}
	return columns, rows, nil
}

func runTerminalReplay(ctx context.Context, o benchmarkOptions) (benchmarkResult, error) {
	out := benchmarkResult{Transport: "in-process", GoVersion: runtime.Version(), CPUs: runtime.GOMAXPROCS(0),
		GOGC: os.Getenv("GOGC"), GOMEMLIMIT: os.Getenv("GOMEMLIMIT")}
	plannedCapacity := 1024
	if o.frames > 0 {
		plannedCapacity = min(o.frames, 10000)
	}
	out.Frames = make([]benchmarkFrame, 0, plannedCapacity)
	columns, rows, err := benchmarkGrid(o)
	if err != nil {
		return out, err
	}
	writer := &timedBenchmarkWriter{writer: io.Discard}
	if o.live {
		writer.writer = os.Stdout
	}
	r := terminal.NewTerminalRenderer(bufio.NewWriter(writer), o.height, o.width)
	r.UseOpenGLRaster = o.raster == "gl"
	r.ColorMode, r.UseColor = o.color, o.color != "none"
	r.ShowDebugInfo = false
	r.DiffPresentation = true
	useTea := o.workload == "switch-room" && o.paced
	if o.live && !useTea {
		if err := r.Initialize(); err != nil {
			return out, err
		}
	} else {
		r.InitializeOffscreen()
	}
	defer r.Shutdown()
	out.RasterDriver = r.OpenGLRasterDriver()
	if r.UseOpenGLRaster && out.RasterDriver == "" {
		return out, errors.New("requested OpenGL raster device unavailable")
	}

	var filter *replayFilter
	if o.workerPath != "" {
		childTrace := ""
		if o.tracePath != "" {
			childTrace = o.tracePath + ".fxaa"
		}
		filter, err = remoteReplayFilter(o.workerPath, o.workerEnv, o.memoryBytes, childTrace)
		out.Transport = "host RGB round trip over worker pipes"
	} else if o.filter != "none" {
		filter, err = localReplayFilter(o.filter == "gl")
	}
	if err != nil {
		return out, err
	}
	if filter != nil {
		defer filter.close()
		out.FilterDriver = filter.driver
	}

	var presentation *replayPresentation
	if useTea {
		presentation = startReplayPresentation(ctx, o.live, o.fps, columns, rows)
		defer presentation.close()
		out.Presentation = "Bubble Tea"
	} else if o.workload == "switch-room" {
		out.Presentation = "synchronous project view"
	} else if o.profile == "gameplay" {
		out.Presentation = "terminal renderer"
	}
	var player *projectPlayer
	var sim *simulation
	var scene *engine.Scene
	var controller *engine.CameraController
	var tick uint64
	var clock time.Time
	var stages terminal.FrameStages
	var renderErr error
	var previous benchmarkCells
	var frame *benchmarkFrame
	var quality terminal.AdaptiveQuality
	var priorWork, priorWrite time.Duration
	var hasLayout bool
	dt := time.Second / time.Duration(o.fps)
	draw := terminal.ComposeRenderer(r, engine.EngineConfig{
		RenderMode: benchmarkModes[o.raster], NumWorkers: runtime.GOMAXPROCS(0), TileSize: o.tile,
	})
	drawFrame := func(r *terminal.TerminalRenderer) {
		if o.detail {
			r.Stages = &stages
		}
		if o.workload != "fxaa" {
			start := perfclock.Now()
			draw.RenderScene(scene)
			frame.RasterNS = perfclock.Since(start).Nanoseconds()
			if r.UseOpenGLRaster && !r.UsedOpenGLRaster() {
				renderErr = errors.New("requested OpenGL raster fell back to CPU")
				return
			}
		}
		if filter != nil && !r.ReducedShadingActive() {
			start := perfclock.Now()
			stages.FXAA, frame.ExchangeNS, renderErr = filter.apply(r)
			frame.FilterNS = perfclock.Since(start).Nanoseconds()
		}
	}
	if o.workload == "switch-room" {
		path := sampleProjectPath("switch-room")
		if path == "" {
			return out, errors.New("cannot find examples/switch-room from the current directory")
		}
		p, err := project.Open(path)
		if err != nil {
			return out, err
		}
		player = &projectPlayer{project: p, room: &switchRoom{}, playerID: "player",
			options: physics.DefaultOptions(), colorMode: o.color, renderer: r, manualTicks: true,
			drawFrame: drawFrame, frameClock: func() time.Time { return clock }}
		if err := player.reset(); err != nil {
			return out, err
		}
		defer player.session.Close()
		player.last = time.Unix(0, 0)
		player.adaptive, player.budget = o.adaptive, dt
		player.behind = func(budget time.Duration) bool {
			if presentation != nil {
				return presentation.pressure.behind(budget)
			}
			return priorWrite > budget
		}
		scene = player.session.Scene
		r.UseColor, r.ShadowRenderer = false, nil
	} else if o.workload != "fxaa" {
		index, exists := map[string]int{"kinetic": 0, "wave": 5, "physics": 6}[o.workload]
		if !exists {
			return out, fmt.Errorf("unknown workload %q", o.workload)
		}
		scene = engine.NewScene()
		demo.ConfigureCamera(scene.Camera, index, engine.OrientationTerminal)
		demo.BuildScene(scene, index)
		lighting := demo.SetupLighting(scene.Camera, index)
		r.SetCamera(scene.Camera)
		r.SetLightingSystem(lighting)
		if !demo.PresetAt(index).Shadows {
			r.ShadowRenderer = nil
		}
		controller = engine.NewCameraController(scene.Camera)
		demo.ConfigureCameraController(controller, index)
		if o.profile == "gameplay" {
			controller.EnableAutoOrbit(true)
			sim, err = newSimulation(scene, lighting, index)
			if err != nil {
				return out, err
			}
		} else {
			controller.Update(engine.InputState{}, engine.OrientationTerminal, 1.25)
			demo.AnimateSceneDemo(scene, index, 1.25, 1.0/30)
			scene.Update(1.0 / 30)
			demo.AnimateLighting(lighting, index, 1.25)
		}
	}

	frameStep := func(i int, sample *benchmarkFrame) error {
		frame, stages = sample, terminal.FrameStages{}
		clock = time.Unix(0, 0).Add(time.Duration(i+1) * dt)
		start := perfclock.Now()
		newColumns, newRows, err := benchmarkGrid(o)
		if err != nil {
			return err
		}
		if !hasLayout || columns != newColumns || rows != newRows {
			quality = terminal.AdaptiveQuality{}
			hasLayout = true
		}
		sample.InputStartNS = perfclock.Since(start).Nanoseconds()
		if player != nil {
			if player.width == 0 || newColumns != columns || newRows != rows {
				player.Update(tea.WindowSizeMsg{Width: newColumns, Height: newRows})
				if presentation != nil {
					presentation.resize(newColumns, newRows)
				}
			}
			if o.viewport != "centered" {
				player.requestedWidth, player.requestedHeight = math.MaxInt, math.MaxInt
			}
			inputStart := perfclock.Now()
			sample.InputStartNS = inputStart.Sub(start).Nanoseconds()
			if i%48 < 24 {
				player.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
				sample.InputEvents++
			}
			if i%60 == 0 {
				player.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, X: 10, Y: 10})
				sample.InputEvents++
			}
			if i%60 < 30 {
				player.Update(tea.MouseMsg{Action: tea.MouseActionMotion, X: 10 + (i%60)/4, Y: 10})
				sample.InputEvents++
			} else if i%60 == 30 {
				player.Update(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonRight})
				sample.InputEvents++
			}
			sample.InputNS = perfclock.Since(inputStart).Nanoseconds()
			simulationStart := perfclock.Now()
			player.Update(editorTick(clock))
			sample.SimulationNS = perfclock.Since(simulationStart).Nanoseconds()
			if player.err != nil {
				return player.err
			}
			nextTick := player.session.World.Stats().Tick
			if nextTick != tick {
				sample.LastPhysicsStepNS = player.session.World.SolverStats().Duration.Nanoseconds()
			}
			tick = nextTick
		} else {
			w, h := newColumns, newRows
			if o.viewport != "fixed" || o.live {
				w, h = w-1, h-1
			}
			if o.viewport == "centered" {
				w, h = terminal.FitViewport(0, 0, w, h)
			}
			if o.profile == "gameplay" {
				r.FitTerminal(w, h, newColumns, newRows)
			} else {
				r.Resize(w, h)
			}
			if sim != nil {
				cameraStart := perfclock.Now()
				sample.InputStartNS = cameraStart.Sub(start).Nanoseconds()
				if o.workload != "physics" || i%120 >= 20 && i%120 < 90 {
					controller.Update(engine.InputState{}, engine.OrientationTerminal, dt.Seconds())
					sample.InputEvents++
				}
				sample.InputNS = perfclock.Since(cameraStart).Nanoseconds()
				simulationStart := perfclock.Now()
				if err := sim.advance(dt, engine.SimulationCommands{}); err != nil {
					return err
				}
				sample.SimulationNS = perfclock.Since(simulationStart).Nanoseconds()
				r.SetSimulationStatus(sim.status())
				nextTick := sim.world.Stats().Tick
				if nextTick != tick {
					sample.LastPhysicsStepNS = sim.world.SolverStats().Duration.Nanoseconds()
				}
				tick = nextTick
			}
		}
		columns, rows = newColumns, newRows
		sample.UpdateNS = perfclock.Since(start).Nanoseconds()
		writer.bytes, writer.elapsed, writer.err, writer.finished = 0, 0, nil, time.Time{}
		if player != nil {
			viewStart := perfclock.Now()
			view := player.View()
			sample.Tier, sample.RasterWidth, sample.RasterHeight = player.quality.Tier, player.rasterWidth, player.rasterHeight
			sample.EncodeNS = perfclock.Since(viewStart).Nanoseconds() - sample.RasterNS - sample.FilterNS
			if presentation != nil {
				presentation.submit(i+1, view)
			} else {
				if _, err := io.WriteString(writer, "\033[H"+view+"\033[J"); err != nil {
					return err
				}
				sample.Presented = true
			}
		} else {
			if sim != nil {
				prep := perfclock.Now()
				sim.prepareRender()
				sample.PrepareNS = perfclock.Since(prep).Nanoseconds()
			}
			if o.adaptive {
				sample.Tier = quality.Update(scene.Camera, priorWork, priorWrite > dt, dt)
			}
			sample.RasterWidth, sample.RasterHeight = r.RenderQuality(sample.Tier, func() { drawFrame(r) })
			if o.profile == "gameplay" {
				present := perfclock.Now()
				draw.EndFrame()
				draw.Present()
				sample.Presented = true
				if writer.err != nil {
					return writer.err
				}
				sample.EncodeNS = perfclock.Since(present).Nanoseconds() - writer.elapsed.Nanoseconds()
			}
			if sim != nil {
				sim.bindings.synchronize()
			}
		}
		sample.Columns, sample.Rows, sample.Width, sample.Height = columns, rows, r.Width, r.Height
		sample.Tick, sample.Bytes, sample.WriteNS = tick, writer.bytes, writer.elapsed.Nanoseconds()
		sample.PrepareNS += stages.PrepareNS
		sample.RasterNS -= stages.PrepareNS
		sample.ShadeNS = stages.ShadeNS
		sample.RasterDevice, sample.FilterDevice = stages.Raster, stages.FXAA
		if o.detail && o.profile == "gameplay" {
			sample.ChangedCells = previous.observe(r, o.color != "none")
		}
		if sample.Presented {
			if writer.finished.IsZero() {
				writer.finished = perfclock.Now()
			}
			sample.PresentNS = writer.finished.Sub(start).Nanoseconds()
			sample.InputToWriteNS = sample.PresentNS - sample.InputStartNS
		}
		priorWork, priorWrite = perfclock.Since(start), writer.elapsed
		return renderErr
	}

	var filterSeed []byte
	if o.workload == "fxaa" {
		for y, row := range r.ColorBuffer {
			for x := range row {
				row[x] = engine.Color{R: uint8(x*37 ^ y*11), G: uint8(x*3 + y*171), B: uint8(x ^ y*9)}
			}
		}
		// Reuse the same source image throughout the standalone filter pass.
		filterSeed = encodeColors(r.ColorBuffer, nil)
		apply := drawFrame
		drawFrame = func(r *terminal.TerminalRenderer) {
			decodeColors(filterSeed, r.ColorBuffer)
			apply(r)
		}
	}
	for i := 0; i < o.warmup; i++ {
		if err := frameStep(i, &benchmarkFrame{}); err != nil {
			return out, err
		}
	}
	if o.warmup > 0 && !o.adaptive && scene != nil && (r.UseOpenGLRaster || out.FilterDriver != "") {
		if err := validateBenchmarkFrame(r, scene, filter != nil, sim); err != nil {
			return out, err
		}
	}
	meter := newGCMeter()
	meter.begin()
	if filter != nil && filter.begin != nil {
		if err := filter.begin(); err != nil {
			return out, err
		}
	}
	started := perfclock.Now()
	var last time.Time
	var workNS int64
	for i := 0; ; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if o.memoryBytes > 0 && meter.memory() > o.memoryBytes {
			return out, errors.New("replay reached the GC experiment memory ceiling")
		}
		waitStart := perfclock.Now()
		if o.paced && i > 0 {
			if wait := started.Add(time.Duration(i) * dt).Sub(perfclock.Now()); wait > 0 {
				time.Sleep(wait)
			}
		}
		start := perfclock.Now()
		sample := benchmarkFrame{Frame: i + 1, StartNS: start.Sub(started).Nanoseconds(), PaceNS: start.Sub(waitStart).Nanoseconds()}
		region := trace.StartRegion(ctx, "game frame")
		err := frameStep(i+o.warmup, &sample)
		region.End()
		if err != nil {
			return out, err
		}
		finished := perfclock.Now()
		sample.TotalNS = finished.Sub(start).Nanoseconds()

		if !last.IsZero() {
			sample.IntervalNS = finished.Sub(last).Nanoseconds()
		}
		last = finished
		workNS += sample.TotalNS
		out.Frames = append(out.Frames, sample)
		if (o.frames > 0 && len(out.Frames) == o.frames) || (o.frames == 0 && workNS >= o.duration.Nanoseconds()) {
			break
		}
	}
	out.ElapsedNS = last.Sub(started).Nanoseconds()
	if presentation != nil {
		if err := presentation.collect(ctx, o.warmup, started, out.Frames); err != nil {
			return out, err
		}
	}
	if o.profile == "gameplay" {
		out.Delivery = summarizeBenchmarkDelivery(out.Frames, out.ElapsedNS)
	}
	out.GC = meter.end()
	if filter != nil && filter.end != nil {
		out.WorkerGC, err = filter.end()
		if err != nil {
			return out, err
		}
	}
	if filterSeed != nil && out.FilterDriver != "" {
		reference := terminal.NewTerminalRenderer(nil, r.Height, r.Width)
		decodeColors(filterSeed, reference.ColorBuffer)
		cpu, _ := terminal.OpenColorFilter(false)
		_, _ = cpu.Apply(reference)
		if changes := countBenchmarkDifferences(r, reference); changes > 0 {
			return out, fmt.Errorf("isolated OpenGL FXAA differs from CPU on %d cells", changes)
		}
	}
	out.State.Tick = tick
	if !o.adaptive && scene != nil && (r.UseOpenGLRaster || out.FilterDriver != "") {
		if err := validateBenchmarkFrame(r, scene, filter != nil, sim); err != nil {
			return out, err
		}
	}
	if scene != nil {
		out.State.Camera = scene.Camera.GetPosition()
		out.State.Forward = scene.Camera.GetForwardVectorPoint()
		digest := sha256.New()
		encode := json.NewEncoder(digest)
		for _, node := range scene.GetRenderableNodes() {
			if err := encode.Encode(node.Transform.GetWorldMatrix()); err != nil {
				return out, err
			}
		}
		out.State.Geometry = fmt.Sprintf("%x", digest.Sum(nil))
	}
	return out, nil
}

func validateBenchmarkFrame(actual *terminal.TerminalRenderer, scene *engine.Scene, filtered bool, sim *simulation) error {
	if sim != nil {
		sim.prepareRender()
		defer sim.bindings.synchronize()
	}
	reference := terminal.NewTerminalRenderer(nil, actual.Height, actual.Width)
	reference.UseColor = actual.UseColor
	reference.SetLightingSystem(actual.LightingSystem)
	if actual.ShadowRenderer == nil {
		reference.ShadowRenderer = nil
	}
	reference.RenderScene(scene)
	if filtered {
		cpu, _ := terminal.OpenColorFilter(false)
		_, _ = cpu.Apply(reference)
	}
	var occupied, differences int
	for y, row := range reference.ZBuffer {
		for x, depth := range row {
			wantEmpty, gotEmpty := math.IsInf(depth, 1), math.IsInf(actual.ZBuffer[y][x], 1)
			if !wantEmpty || !gotEmpty {
				occupied++
			}
			if wantEmpty != gotEmpty || (!wantEmpty && math.Abs(depth-actual.ZBuffer[y][x]) > 1e-5) ||
				reference.ColorBuffer[y][x] != actual.ColorBuffer[y][x] {
				differences++
			}
		}
	}
	if differences > max(2, occupied/50) {
		return fmt.Errorf("CPU parity failed at %dx%d: %d differing cells (%d occupied)", actual.Width, actual.Height, differences, occupied)
	}
	return nil
}

type benchmarkCell struct {
	glyph rune
	color engine.Color
}

type benchmarkCells struct {
	width, height int
	cells         []benchmarkCell
}

func (p *benchmarkCells) observe(r *terminal.TerminalRenderer, color bool) int {
	resized := p.width != r.Width || p.height != r.Height
	if resized {
		p.width, p.height = r.Width, r.Height
		p.cells = make([]benchmarkCell, r.Width*r.Height)
	}
	changed := 0
	for y, row := range r.Surface {
		for x, glyph := range row {
			cell := benchmarkCell{glyph: glyph}
			if color {
				cell.color = r.ColorBuffer[y][x]
			}
			index := y*r.Width + x
			if resized || p.cells[index] != cell {
				changed++
			}
			p.cells[index] = cell
		}
	}
	return changed
}

func countBenchmarkDifferences(a, b *terminal.TerminalRenderer) int {
	diff := 0
	for y, row := range a.ColorBuffer {
		for x, value := range row {
			if value != b.ColorBuffer[y][x] {
				diff++
			}
		}
	}
	return diff
}

func summarizeBenchmarkDelivery(frames []benchmarkFrame, elapsed int64) *benchmarkDelivery {
	out := &benchmarkDelivery{Submitted: len(frames)}
	var samples []int64
	for _, f := range frames {
		if !f.Presented {
			continue
		}
		out.Completed++
		out.Bytes += int64(f.Bytes)
		elapsed = max(elapsed, f.StartNS+f.PresentNS)
		samples = append(samples, f.InputToWriteNS)
	}
	out.Coalesced = out.Submitted - out.Completed
	if out.Submitted > 0 {
		out.SampleFraction = float64(out.Completed) / float64(out.Submitted)
	}
	if elapsed > 0 {
		seconds := time.Duration(elapsed).Seconds()
		out.CompletedPerSecond, out.BytesPerSecond = float64(out.Completed)/seconds, float64(out.Bytes)/seconds
	}
	if len(samples) > 0 {
		slices.Sort(samples)
		out.InputToWriteP50NS = samples[(len(samples)*50+99)/100-1]
		out.InputToWriteP99NS = samples[(len(samples)*99+99)/100-1]
	}
	return out
}
