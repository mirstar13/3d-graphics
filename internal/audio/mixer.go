package audio

import (
	"context"
	"encoding/binary"
	"io"
	"sync"
	"time"
)

const Voices = 32
const BlockFrames = 512

// Output consumes signed 16-bit little-endian stereo frames at Rate.
// It owns the output stream until cancellation or device failure.
type Output interface {
	Run(context.Context, io.Reader) error
}

// Manager owns the mix and, optionally, one lazily opened native stream.
// Control operations share only the mixer lock with the output reader.
type Manager struct {
	mu                             sync.Mutex
	voices                         [Voices]voice
	scope                          uint64
	dropped                        uint64
	muted, paused, closed, started bool
	output                         Output
	cancel                         context.CancelFunc
	done                           chan struct{}
	err                            error
	finished                       chan struct{}
	finish                         sync.Once
}

type voice struct {
	samples []int16
	cursor  int
}

// New creates a manager. With nil output the caller can pull PCM using Read.
func New(output Output) *Manager { return &Manager{output: output, finished: make(chan struct{})} }

func NewNative() *Manager { return New(nativeOutput{}) }

// Scope replaces all voices and invalidates previous playback handles.
func (m *Manager) Scope() *Scope {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scope++
	clear(m.voices[:])
	m.paused = false
	return &Scope{manager: m, id: m.scope}
}

// Scope identifies the currently active session or audition.
type Scope struct {
	manager *Manager
	id      uint64
}

func (s *Scope) Play(clip *Clip) {
	if s == nil || clip == nil {
		return
	}
	m := s.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.err != nil || m.scope != s.id || m.paused {
		return
	}
	for i := range m.voices {
		if m.voices[i].samples == nil {
			m.voices[i] = voice{samples: clip.samples}
			m.start()
			return
		}
	}
	m.dropped++
}

func (m *Manager) start() {
	if m.output == nil || m.started {
		return
	}
	m.started = true
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel, m.done = cancel, make(chan struct{})
	go func() {
		defer close(m.done)
		if err := m.output.Run(ctx, m); err != nil && ctx.Err() == nil {
			m.mu.Lock()
			m.err = err
			m.finish.Do(func() { close(m.finished) })
			clear(m.voices[:])
			m.mu.Unlock()
		}
	}()
}

func (s *Scope) Pause(paused bool) {
	if s == nil {
		return
	}
	m := s.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.scope == s.id {
		m.paused = paused
	}
}

func (s *Scope) Close() {
	if s == nil {
		return
	}
	m := s.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.scope == s.id {
		m.scope++
		clear(m.voices[:])
	}
}

func (m *Manager) ToggleMute() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.muted = !m.muted
	return m.muted
}

// Finished closes after a device error or when the host closes the manager.
func (m *Manager) Finished() <-chan struct{} { return m.finished }

func (m *Manager) Err() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

func (m *Manager) Dropped() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dropped
}

func (m *Manager) Close() {
	m.mu.Lock()
	m.closed = true
	m.finish.Do(func() { close(m.finished) })
	clear(m.voices[:])
	cancel, done := m.cancel, m.done
	m.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

// Read writes complete stereo frames. The native player requests aligned buffers.
func (m *Manager) Read(p []byte) (int, error) {
	n := len(p) - len(p)%4
	if len(p) > 0 && n == 0 {
		return 0, io.ErrShortBuffer
	}
	for offset := 0; offset < n; offset += BlockFrames * 4 {
		m.mix(p[offset:min(n, offset+BlockFrames*4)])
	}
	return n, nil
}

func (m *Manager) mix(p []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(p)
	if m.paused || m.closed {
		return
	}
	var sums [BlockFrames * 2]int32
	active := false
	for i := range m.voices {
		v := &m.voices[i]
		count := min(len(p)/2, len(v.samples)-v.cursor)
		if count == 0 {
			continue
		}
		active = true
		if !m.muted {
			for j, sample := range v.samples[v.cursor : v.cursor+count] {
				sums[j] += int32(sample)
			}
		}
		v.cursor += count
		if v.cursor == len(v.samples) {
			*v = voice{}
		}
	}
	if active && !m.muted {
		for i, sum := range sums[:len(p)/2] {
			binary.LittleEndian.PutUint16(p[2*i:], uint16(int16(max(-32768, min(32767, sum)))))
		}
	}
}

const deviceBuffer = 40 * time.Millisecond
