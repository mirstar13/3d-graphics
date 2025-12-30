package main

type Triangle struct {
	P0           Point
	P1           Point
	P2           Point
	char         byte
	Material     Material
	Normal       *Point
	UseSetNormal bool
}

func NewTriangle(p0, p1, p2 Point, char byte) *Triangle {
	return &Triangle{
		P0:           p0,
		P1:           p1,
		P2:           p2,
		char:         char,
		Material:     NewMaterial(),
		Normal:       nil,
		UseSetNormal: false,
	}
}

func (t *Triangle) SetMaterial(material Material) *Triangle {
	t.Material = material
	return t
}

func (t *Triangle) SetColor(color Color) *Triangle {
	t.Material.DiffuseColor = color
	return t
}

func (t *Triangle) SetNormal(normal Point) *Triangle {
	t.Normal = &normal
	t.UseSetNormal = true
	return t
}

func (t *Triangle) Draw(renderer *Renderer, camera *Camera) {
	NewLine(t.P0, t.P1).Draw(renderer, camera)
	NewLine(t.P1, t.P2).Draw(renderer, camera)
	NewLine(t.P2, t.P0).Draw(renderer, camera)
}

func (t *Triangle) Project(renderer *Renderer, camera *Camera) {
	NewLine(t.P0, t.P1).Project(renderer, camera)
	NewLine(t.P1, t.P2).Project(renderer, camera)
	NewLine(t.P2, t.P0).Project(renderer, camera)
}

func (t *Triangle) RotateGlobal(axis byte, angle float64) {
	t.P0.RotateGlobal(axis, angle)
	t.P1.RotateGlobal(axis, angle)
	t.P2.RotateGlobal(axis, angle)
}

func (t *Triangle) RotateLocal(axis byte, angle float64) {
	centerX := (t.P0.X + t.P1.X + t.P2.X) / 3.0
	centerY := (t.P0.Y + t.P1.Y + t.P2.Y) / 3.0
	centerZ := (t.P0.Z + t.P1.Z + t.P2.Z) / 3.0

	t.P0.X -= centerX
	t.P0.Y -= centerY
	t.P0.Z -= centerZ

	t.P1.X -= centerX
	t.P1.Y -= centerY
	t.P1.Z -= centerZ

	t.P2.X -= centerX
	t.P2.Y -= centerY
	t.P2.Z -= centerZ

	t.P0.RotateGlobal(axis, angle)
	t.P1.RotateGlobal(axis, angle)
	t.P2.RotateGlobal(axis, angle)

	t.P0.X += centerX
	t.P0.Y += centerY
	t.P0.Z += centerZ

	t.P1.X += centerX
	t.P1.Y += centerY
	t.P1.Z += centerZ

	t.P2.X += centerX
	t.P2.Y += centerY
	t.P2.Z += centerZ

	if t.UseSetNormal && t.Normal != nil {
		t.Normal.Rotate(axis, angle)
	}
}

func (t *Triangle) DrawFilled(renderer *Renderer, camera *Camera) {
	// Use the clipping system
	clippedTriangles := ClipTriangleToNearPlane(t, camera)

	// If completely clipped, return early
	if len(clippedTriangles) == 0 {
		return
	}

	// Calculate normal ONCE for lighting (shared by all clipped triangles)
	normal := CalculateSurfaceNormal(&t.P0, &t.P1, &t.P2, t.Normal, t.UseSetNormal)

	// Calculate surface center ONCE
	surfacePoint := Point{
		X: (t.P0.X + t.P1.X + t.P2.X) / 3.0,
		Y: (t.P0.Y + t.P1.Y + t.P2.Y) / 3.0,
		Z: (t.P0.Z + t.P1.Z + t.P2.Z) / 3.0,
	}

	// Backface culling
	cameraDirX, cameraDirY, cameraDirZ := camera.GetCameraDirection(surfacePoint)
	facing := dotProduct(normal.X, normal.Y, normal.Z, cameraDirX, cameraDirY, cameraDirZ)
	if facing < 0 {
		return // Back-facing, cull it
	}

	// Calculate lighting ONCE
	var pixelColor Color
	var fillChar rune

	if renderer.LightingSystem != nil {
		ao := CalculateSimpleAO(normal)
		pixelColor = renderer.LightingSystem.CalculateLighting(
			surfacePoint,
			normal,
			t.Material,
			ao,
		)
	} else {
		// Fallback simple lighting
		ao := CalculateSimpleAO(normal)
		lx, ly, lz := -1.0, 1.0, -1.0
		lx, ly, lz = normalizeVector(lx, ly, lz)
		intensity := dotProduct(normal.X, normal.Y, normal.Z, lx, ly, lz)
		if intensity < 0 {
			intensity = 0
		}
		intensity *= ao

		// Apply material color
		r := float64(t.Material.DiffuseColor.R) * intensity
		g := float64(t.Material.DiffuseColor.G) * intensity
		b := float64(t.Material.DiffuseColor.B) * intensity

		// Clamp BEFORE converting to uint8
		if r < 0 {
			r = 0
		}
		if r > 255 {
			r = 255
		}
		if g < 0 {
			g = 0
		}
		if g > 255 {
			g = 255
		}
		if b < 0 {
			b = 0
		}
		if b > 255 {
			b = 255
		}

		pixelColor = Color{R: uint8(r), G: uint8(g), B: uint8(b)}
	}

	// Calculate fill character based on brightness
	brightness := (float64(pixelColor.R) + float64(pixelColor.G) + float64(pixelColor.B)) / (3.0 * 255.0)
	index := int(brightness * float64(len(SHADING_RAMP)-1))
	if index < 0 {
		index = 0
	}
	if index >= len(SHADING_RAMP) {
		index = len(SHADING_RAMP) - 1
	}
	fillChar = rune(SHADING_RAMP[index])

	// Render each clipped triangle with the same lighting
	for _, clipped := range clippedTriangles {
		clipped.drawFilledNoClip(renderer, camera, pixelColor, fillChar)
	}
}

// drawFilledNoClip is the internal rendering method that assumes no clipping is needed
func (t *Triangle) drawFilledNoClip(renderer *Renderer, camera *Camera, pixelColor Color, fillChar rune) {
	// Project vertices
	x0, y0, zDepth0 := camera.ProjectPoint(t.P0, renderer.Height, renderer.Width)
	x1, y1, zDepth1 := camera.ProjectPoint(t.P1, renderer.Height, renderer.Width)
	x2, y2, zDepth2 := camera.ProjectPoint(t.P2, renderer.Height, renderer.Width)

	// After clipping, these should not be -1, but check anyway
	if x0 == -1 || x1 == -1 || x2 == -1 {
		return
	}

	// Guard against bad depths
	if zDepth0 <= 0 {
		zDepth0 = 0.001
	}
	if zDepth1 <= 0 {
		zDepth1 = 0.001
	}
	if zDepth2 <= 0 {
		zDepth2 = 0.001
	}

	// Sort vertices by Y coordinate
	if y1 < y0 {
		x0, y0, zDepth0, x1, y1, zDepth1 = x1, y1, zDepth1, x0, y0, zDepth0
	}
	if y2 < y0 {
		x0, y0, zDepth0, x2, y2, zDepth2 = x2, y2, zDepth2, x0, y0, zDepth0
	}
	if y2 < y1 {
		x1, y1, zDepth1, x2, y2, zDepth2 = x2, y2, zDepth2, x1, y1, zDepth1
	}

	// Now y0 <= y1 <= y2
	totalHeight := y2 - y0
	if totalHeight == 0 {
		return
	}

	// Rasterize the triangle
	for y := y0; y <= y2; y++ {
		if y < 0 || y >= renderer.Height {
			continue
		}

		// Determine which segment we're in
		secondHalf := y > y1 || y1 == y0

		// Calculate alpha (long edge 0->2)
		alpha := float64(y-y0) / float64(totalHeight)

		// Calculate beta (short edge)
		beta := 0.0
		if secondHalf {
			if y2 != y1 {
				beta = float64(y-y1) / float64(y2-y1)
			}
		} else {
			if y1 != y0 {
				beta = float64(y-y0) / float64(y1-y0)
			}
		}

		// Interpolate X on long edge (0->2)
		ax := int(float64(x0) + alpha*float64(x2-x0) + 0.5)
		az := zDepth0 + alpha*(zDepth2-zDepth0)

		// Interpolate X on short edge
		var bx int
		var bz float64
		if secondHalf {
			bx = int(float64(x1) + beta*float64(x2-x1) + 0.5)
			bz = zDepth1 + beta*(zDepth2-zDepth1)
		} else {
			bx = int(float64(x0) + beta*float64(x1-x0) + 0.5)
			bz = zDepth0 + beta*(zDepth1-zDepth0)
		}

		// Draw scanline
		if ax > bx {
			ax, bx = bx, ax
			az, bz = bz, az
		}

		for x := ax; x <= bx; x++ {
			if x < 0 || x >= renderer.Width {
				continue
			}

			// Interpolate depth
			t := 0.0
			if bx != ax {
				t = float64(x-ax) / float64(bx-ax)
			}
			z := az + t*(bz-az)

			if z > 0 && z < renderer.ZBuffer[y][x] {
				if renderer.UseColor {
					renderer.Surface[y][x] = FILLED_CHAR
					renderer.ColorBuffer[y][x] = pixelColor
				} else {
					renderer.Surface[y][x] = fillChar
				}
				renderer.ZBuffer[y][x] = z
			}
		}
	}
}

// ConvertQuadToTriangles converts a quad into two triangles
// Handles both solid and wireframe materials
func ConvertQuadToTriangles(q *Quad) []*Triangle {
	// Split quad into two triangles: (P0, P1, P2) and (P0, P2, P3)
	t1 := NewTriangle(q.P0, q.P1, q.P2, 'x')
	t1.Material = q.Material
	if q.UseSetNormal {
		t1.SetNormal(*q.Normal)
	}

	t2 := NewTriangle(q.P0, q.P2, q.P3, 'x')
	t2.Material = q.Material
	if q.UseSetNormal {
		t2.SetNormal(*q.Normal)
	}

	return []*Triangle{t1, t2}
}
