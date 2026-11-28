package terminal

import (
	"sync"
	"time"
)

// LatestFrame runs presentation independently of simulation. Submit replaces
// the pending frame while a write is in progress.
type LatestFrame[T any] struct {
	mu                  sync.Mutex
	wake                chan struct{}
	done                chan struct{}
	pending             T
	hasPending, closed  bool
	queuedAt, startedAt time.Time
	lastTime            time.Duration
	coalesced           uint64
}

func NewLatestFrame[T any](present func(T)) *LatestFrame[T] {
	p := &LatestFrame[T]{wake: make(chan struct{}, 1), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		for range p.wake {
			p.mu.Lock()
			if !p.hasPending {
				closed := p.closed
				p.mu.Unlock()
				if closed {
					return
				}
				continue
			}
			frame := p.pending
			var empty T
			p.pending, p.hasPending = empty, false
			p.startedAt = time.Now()
			p.mu.Unlock()

			present(frame)

			p.mu.Lock()
			p.lastTime = time.Since(p.startedAt)
			p.startedAt = time.Time{}
			closed, pending := p.closed, p.hasPending
			p.mu.Unlock()
			if closed && !pending {
				return
			}
		}
	}()
	return p
}

func (p *LatestFrame[T]) Submit(frame T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	if p.hasPending {
		p.coalesced++
	} else {
		p.queuedAt = time.Now()
	}
	p.pending, p.hasPending = frame, true
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *LatestFrame[T]) Behind(budget time.Duration) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastTime > budget || (!p.startedAt.IsZero() && time.Since(p.startedAt) > budget) ||
		(p.hasPending && time.Since(p.queuedAt) > budget)
}

// Close drains the latest frame and joins the writer before terminal teardown.
func (p *LatestFrame[T]) Close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	select {
	case p.wake <- struct{}{}:
	default:
	}
	<-p.done
}
