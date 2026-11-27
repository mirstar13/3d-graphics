package input

// Pointer accumulates terminal cell motion between updates. After capture or
// resize the first position establishes an origin without moving the camera.
type Pointer struct {
	Captured     bool
	known        bool
	x, y, dx, dy int
}

func (p *Pointer) Reset(captured bool) { *p = Pointer{Captured: captured} }

func (p *Pointer) Motion(x, y int) {
	if !p.Captured {
		return
	}
	if p.known {
		p.dx += x - p.x
		p.dy += y - p.y
	}
	p.x, p.y, p.known = x, y, true
}

func (p *Pointer) Take() (int, int) {
	x, y := p.dx, p.dy
	p.dx, p.dy = 0, 0
	return x, y
}

// Reanchor forgets the last cell, preserving motion awaiting the next tick.
func (p *Pointer) Reanchor() { p.known = false }

// EndDrag stops collecting motion and keeps the latest delta for the next tick.
func (p *Pointer) EndDrag() { p.Captured, p.known = false, false }
