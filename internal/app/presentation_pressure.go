package app

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/perfclock"
)

type presentationPressure struct {
	start    atomic.Pointer[time.Time]
	duration atomic.Int64
}

func (p *presentationPressure) write(dst io.Writer, data []byte) (int, error) {
	start := perfclock.Now()
	p.start.Store(&start)
	n, err := dst.Write(data)
	p.duration.Store(perfclock.Since(start).Nanoseconds())
	p.start.Store(nil)
	return n, err
}

func (p *presentationPressure) behind(budget time.Duration) bool {
	start := p.start.Load()
	return time.Duration(p.duration.Load()) > budget || (start != nil && perfclock.Since(*start) > budget)
}
