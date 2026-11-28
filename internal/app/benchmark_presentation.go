package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/perfclock"
)

type replayView string

func (v replayView) Init() tea.Cmd { return nil }
func (v replayView) View() string  { return string(v) }
func (v replayView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if next, ok := msg.(replayView); ok {
		return next, nil
	}
	return v, nil
}

const replayMarker = "\033]graphics-replay="
const replayMarkerEnd = "\033\\"

type replaySubmission struct {
	finished time.Time
	write    time.Duration
	size     int
}

type replayPresentation struct {
	*os.File
	program     *tea.Program
	done        chan struct{}
	notify      chan struct{}
	mu          sync.Mutex
	submissions map[int]replaySubmission
	err         error
	buffer      bytes.Buffer
}

func startReplayPresentation(ctx context.Context, live bool, fps, columns, rows int) *replayPresentation {
	p := &replayPresentation{done: make(chan struct{}), notify: make(chan struct{}, 1), submissions: make(map[int]replaySubmission)}
	if live {
		p.File = os.Stdout
	}
	p.program = tea.NewProgram(replayView(""), tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(p),
		tea.WithFPS(fps), tea.WithAltScreen(), tea.WithoutSignalHandler())
	go func() {
		defer close(p.done)
		_, err := p.program.Run()
		if err != nil {
			p.mu.Lock()
			p.err = err
			p.mu.Unlock()
		}
	}()
	p.resize(columns, rows)
	return p
}

func (p *replayPresentation) resize(columns, rows int) {
	p.program.Send(tea.WindowSizeMsg{Width: columns, Height: rows})
}

func (p *replayPresentation) Fd() uintptr {
	if p.File == nil {
		return ^uintptr(0)
	}
	return p.File.Fd()
}

func (p *replayPresentation) submit(id int, frame string) {
	p.program.Send(replayView(replayMarker + strconv.Itoa(id) + replayMarkerEnd + frame))
}

func (p *replayPresentation) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	actual := data
	var id int
	if start := bytes.Index(data, []byte(replayMarker)); start >= 0 {
		idStart := start + len(replayMarker)
		if length := bytes.Index(data[idStart:], []byte(replayMarkerEnd)); length >= 0 {
			idEnd := idStart + length
			id, _ = strconv.Atoi(string(data[idStart:idEnd]))
			p.buffer.Reset()
			p.buffer.Write(data[:start])
			p.buffer.Write(data[idEnd+len(replayMarkerEnd):])
			actual = p.buffer.Bytes()
		}
	}
	var sink io.Writer = io.Discard
	if p.File != nil {
		sink = p.File
	}
	start := perfclock.Now()
	n, err := sink.Write(actual)
	finished := perfclock.Now()
	if err == nil && n != len(actual) {
		err = io.ErrShortWrite
	}
	if err != nil {
		p.err = err
		return 0, err
	}
	if id > 0 {
		p.submissions[id] = replaySubmission{finished, finished.Sub(start), n}
		select {
		case p.notify <- struct{}{}:
		default:
		}
	}
	return len(data), nil
}

func (p *replayPresentation) collect(ctx context.Context, warmup int, started time.Time, frames []benchmarkFrame) error {
	end := warmup + len(frames)
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for {
		p.mu.Lock()
		_, complete := p.submissions[end]
		err := p.err
		if complete {
			var last time.Time
			for i := range frames {
				f := &frames[i]
				f.IntervalNS = 0
				if record, ok := p.submissions[warmup+i+1]; ok {
					f.Presented = true
					f.PresentNS = record.finished.Sub(started).Nanoseconds() - f.StartNS
					f.Bytes, f.WriteNS = record.size, record.write.Nanoseconds()
					if !last.IsZero() {
						f.IntervalNS = record.finished.Sub(last).Nanoseconds()
					}
					last = record.finished
				}
			}
		}
		p.mu.Unlock()
		if complete || err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.done:
			return fmt.Errorf("project presentation stopped before its final frame")
		case <-p.notify:
		case <-deadline.C:
			return fmt.Errorf("project presentation did not submit its final frame")
		}
	}
}

func (p *replayPresentation) close() {
	p.program.Quit()
	<-p.done
}
