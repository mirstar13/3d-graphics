package main

import "math"

type Drawable interface {
	Draw(renderer *Renderer, camera *Camera)
	DrawFilled(renderer *Renderer, camera *Camera)
	Project(renderer *Renderer, camera *Camera)
	RotateLocal(axis byte, angle float64)
	RotateGlobal(axis byte, angle float64)
}

type Point struct {
	X, Y, Z float64
	char    byte
}

type Line struct {
	Start, End Point
}

type Circle struct {
	Center Point
	Radius float64
	Points []Point
}

type Rect struct {
	PointA Point
	PointB Point
}

func NewPoint(x, y, z float64) *Point {
	return &Point{x, y, z, 'o'}
}

func (p *Point) Draw(renderer *Renderer, camera *Camera) {
	normalizedX, normalizedY := normalize(renderer.Height, renderer.Width, int(p.X), int(p.Y))
	if normalizedY >= 0 && normalizedY < renderer.Height && normalizedX >= 0 && normalizedX < renderer.Width {
		renderer.Surface[normalizedY][normalizedX] = renderer.Charset[7]
	}
}

func (p *Point) DrawFilled(renderer *Renderer, camera *Camera) {
	// Cannot fill a single pixel
}

func (p *Point) projectPointRaw(renderer *Renderer, camera *Camera) (int, int, float64) {
	return camera.ProjectPoint(*p, renderer.Height, renderer.Width)
}

func (p *Point) Project(renderer *Renderer, camera *Camera) {
	normalizedX, normalizedY, zDepth := camera.ProjectPoint(*p, renderer.Height, renderer.Width)

	if normalizedX == -1 {
		return
	}

	if normalizedX >= 0 && normalizedX < renderer.Width && normalizedY >= 0 && normalizedY < renderer.Height {
		if zDepth < renderer.ZBuffer[normalizedY][normalizedX] {
			renderer.Surface[normalizedY][normalizedX] = renderer.Charset[7]
			renderer.ZBuffer[normalizedY][normalizedX] = zDepth
		}
	}
}

func (p *Point) Rotate(axis byte, angle float64) {
	c := math.Cos(angle)
	s := math.Sin(angle)

	switch axis {
	case 'x':
		y, z := p.Y, p.Z
		p.Y = y*c - z*s
		p.Z = y*s + z*c
	case 'y':
		x, z := p.X, p.Z
		p.X = x*c - z*s
		p.Z = x*s + z*c
	case 'z':
		x, y := p.X, p.Y
		p.X = x*c - y*s
		p.Y = x*s + y*c
	}
}

func (p *Point) RotateGlobal(axis byte, angle float64) {
	p.Rotate(axis, angle)
}

func (p *Point) RotateLocal(axis byte, angle float64) {
	// A point cannot spin around itself visibly
}

func (p *Point) SetChar(char byte) *Point {
	p.char = char
	return p
}

func NewLine(start, end Point) *Line {
	return &Line{start, end}
}

func (l *Line) Draw(renderer *Renderer, camera *Camera) {
	sx0, sy0 := normalize(renderer.Height, renderer.Width, int(l.Start.X), int(l.Start.Y))
	sx1, sy1 := normalize(renderer.Height, renderer.Width, int(l.End.X), int(l.End.Y))

	drawLineOnSurface(renderer, sx0, sy0, sx1, sy1)
}

func (l *Line) DrawFilled(renderer *Renderer, camera *Camera) {
	// Cannot fill line
}

func (l *Line) Project(renderer *Renderer, camera *Camera) {
	sx0, sy0, zStart := camera.ProjectPoint(l.Start, renderer.Height, renderer.Width)
	sx1, sy1, zEnd := camera.ProjectPoint(l.End, renderer.Height, renderer.Width)

	// If both points are behind camera, skip
	if sx0 == -1 && sx1 == -1 {
		return
	}

	// If one point is behind camera, clip the line at the near plane
	if sx0 == -1 || sx1 == -1 {
		// Calculate z-depth for both points
		z0 := l.Start.Z + camera.DZ
		z1 := l.End.Z + camera.DZ

		// If one is behind near plane, clip it
		if z0 <= camera.Near || z1 <= camera.Near {
			// Calculate intersection with near plane
			nearPlane := camera.Near + 0.01 // Small epsilon

			if z0 <= camera.Near {
				// Clip start point
				t := (nearPlane - z0) / (z1 - z0)
				clippedStart := Point{
					X: l.Start.X + t*(l.End.X-l.Start.X),
					Y: l.Start.Y + t*(l.End.Y-l.Start.Y),
					Z: l.Start.Z + t*(l.End.Z-l.Start.Z),
				}
				sx0, sy0, zStart = camera.ProjectPoint(clippedStart, renderer.Height, renderer.Width)
			}

			if z1 <= camera.Near {
				// Clip end point
				t := (nearPlane - z0) / (z1 - z0)
				clippedEnd := Point{
					X: l.Start.X + t*(l.End.X-l.Start.X),
					Y: l.Start.Y + t*(l.End.Y-l.Start.Y),
					Z: l.Start.Z + t*(l.End.Z-l.Start.Z),
				}
				sx1, sy1, zEnd = camera.ProjectPoint(clippedEnd, renderer.Height, renderer.Width)
			}

			// If clipping still fails, skip line
			if sx0 == -1 || sx1 == -1 {
				return
			}
		}
	}

	drawLineOnSurfaceWithZ(renderer, sx0, sy0, sx1, sy1, zStart, zEnd)
}

func (l *Line) RotateGlobal(axis byte, angle float64) {
	l.Start.RotateGlobal(axis, angle)
	l.End.RotateGlobal(axis, angle)
}

func (l *Line) RotateLocal(axis byte, angle float64) {
	l.RotateGlobal(axis, angle)
}

func NewCircle(x, y, z, r float64, segments int) *Circle {
	if segments < 3 {
		segments = 3
	}

	points := make([]Point, segments)
	step := 2 * math.Pi / float64(segments)

	for i := 0; i < segments; i++ {
		theta := float64(i) * step

		px := x + r*math.Cos(theta)
		py := y + r*math.Sin(theta)
		pz := z

		points[i] = *NewPoint(px, py, pz)
	}

	return &Circle{
		Center: Point{X: x, Y: y, Z: z},
		Radius: r,
		Points: points,
	}
}

func (c *Circle) Draw(renderer *Renderer, camera *Camera) {
	for _, point := range c.Points {
		point.Draw(renderer, camera)
	}
}

func (c *Circle) DrawFilled(renderer *Renderer, camera *Camera) {
	// Not implemented
}

func (c *Circle) Project(renderer *Renderer, camera *Camera) {
	if len(c.Points) == 0 {
		return
	}

	for i := 0; i < len(c.Points); i++ {
		p1 := c.Points[i]
		p2 := c.Points[(i+1)%len(c.Points)]

		line := NewLine(p1, p2)
		line.Project(renderer, camera)
	}
}

func (c *Circle) RotateGlobal(axis byte, angle float64) {
	for i := range c.Points {
		c.Points[i].RotateGlobal(axis, angle)
	}
	c.Center.RotateGlobal(axis, angle)
}

func (c *Circle) RotateLocal(axis byte, angle float64) {
	for i := range c.Points {
		c.Points[i].X -= c.Center.X
		c.Points[i].Y -= c.Center.Y
		c.Points[i].Z -= c.Center.Z

		c.Points[i].RotateGlobal(axis, angle)

		c.Points[i].X += c.Center.X
		c.Points[i].Y += c.Center.Y
		c.Points[i].Z += c.Center.Z
	}
}
