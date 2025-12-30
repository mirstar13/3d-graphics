package main

import "math"

func drawLineOnSurface(r *Renderer, x0, y0, x1, y1 int) {
	dx := x1 - x0
	dy := y1 - y0

	absDx := abs(dx)
	absDy := abs(dy)

	var char rune = '/'
	if absDy*2 < absDx {
		char = '-'
	} else if absDx*2 < absDy {
		char = '|'
	} else if (dx > 0 && dy > 0) || (dx < 0 && dy < 0) {
		char = '\\'
	}

	dx = abs(x1 - x0)
	dy = -abs(y1 - y0)

	sx := -1
	if x0 < x1 {
		sx = 1
	}

	sy := -1
	if y0 < y1 {
		sy = 1
	}

	err := dx + dy

	for {
		if x0 >= 0 && x0 < r.Width && y0 >= 0 && y0 < r.Height {
			r.Surface[y0][x0] = char
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// drawLineOnSurfaceWithZ draws a line with Z-buffer checking
func drawLineOnSurfaceWithZ(r *Renderer, x0, y0, x1, y1 int, z0, z1 float64) {
	dx := x1 - x0
	dy := y1 - y0

	absDx := abs(dx)
	absDy := abs(dy)

	var char rune = '/'
	if absDy*2 < absDx {
		char = '-'
	} else if absDx*2 < absDy {
		char = '|'
	} else if (dx > 0 && dy > 0) || (dx < 0 && dy < 0) {
		char = '\\'
	}

	// Calculate total distance for Z interpolation
	totalDist := math.Sqrt(float64(dx*dx + dy*dy))
	if totalDist == 0 {
		totalDist = 1
	}

	dx = abs(x1 - x0)
	dy = -abs(y1 - y0)

	sx := -1
	if x0 < x1 {
		sx = 1
	}

	sy := -1
	if y0 < y1 {
		sy = 1
	}

	err := dx + dy
	origX0, origY0 := x0, y0

	for {
		if x0 >= 0 && x0 < r.Width && y0 >= 0 && y0 < r.Height {
			// Calculate current distance from start
			distX := float64(x0 - origX0)
			distY := float64(y0 - origY0)
			currentDist := math.Sqrt(distX*distX + distY*distY)

			// Interpolate Z value
			t := currentDist / totalDist
			z := z0 + t*(z1-z0)

			// Z-buffer check
			if z < r.ZBuffer[y0][x0] {
				if r.UseColor {
					// In color mode, use FILLED_CHAR for consistency
					r.Surface[y0][x0] = FILLED_CHAR
					r.ColorBuffer[y0][x0] = ColorWhite
				} else {
					// In ASCII mode, use directional characters
					r.Surface[y0][x0] = char
				}
				r.ZBuffer[y0][x0] = z
			}
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func removeLast[T any](arr []T) []T {
	res := make([]T, len(arr)-1)

	for i, item := range arr {
		if i == len(arr)-1 {
			break
		}

		res[i] = item
	}

	return res
}
