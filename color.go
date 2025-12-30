package main

import "fmt"

type Color struct {
	R, G, B uint8
}

// Predefined color palette
var (
	ColorBlack   = Color{0, 0, 0}
	ColorRed     = Color{255, 0, 0}
	ColorGreen   = Color{0, 255, 0}
	ColorBlue    = Color{0, 0, 255}
	ColorYellow  = Color{255, 255, 0}
	ColorCyan    = Color{0, 255, 255}
	ColorMagenta = Color{255, 0, 255}
	ColorWhite   = Color{255, 255, 255}
	ColorOrange  = Color{255, 165, 0}
	ColorPurple  = Color{128, 0, 128}
)

// NewColor creates a new RGB color
func NewColor(r, g, b uint8) Color {
	return Color{r, g, b}
}

// ToANSI converts color to ANSI escape code for foreground
func (c Color) ToANSI() string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

// ToANSIBackground converts color to ANSI escape code for background
func (c Color) ToANSIBackground() string {
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", c.R, c.G, c.B)
}

// Reset returns ANSI reset code
func ColorReset() string {
	return "\033[0m"
}

// Lerp linearly interpolates between two colors
func (c Color) Lerp(other Color, t float64) Color {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	// Convert to float64 BEFORE subtraction to prevent uint8 wraparound
	r := float64(c.R) + t*(float64(other.R)-float64(c.R))
	g := float64(c.G) + t*(float64(other.G)-float64(c.G))
	b := float64(c.B) + t*(float64(other.B)-float64(c.B))

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

	return Color{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
	}
}

// IntensityToColor maps lighting intensity (0.0 to 1.0) to a color gradient
// Dark blue -> cyan -> green -> yellow -> orange -> red -> white
func IntensityToColor(intensity float64) Color {
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}

	// Define gradient stops
	if intensity < 0.2 {
		// Dark blue to blue
		t := intensity / 0.2
		return Color{0, 0, 100}.Lerp(ColorBlue, t)
	} else if intensity < 0.4 {
		// Blue to cyan
		t := (intensity - 0.2) / 0.2
		return ColorBlue.Lerp(ColorCyan, t)
	} else if intensity < 0.6 {
		// Cyan to green
		t := (intensity - 0.4) / 0.2
		return ColorCyan.Lerp(ColorGreen, t)
	} else if intensity < 0.8 {
		// Green to yellow
		t := (intensity - 0.6) / 0.2
		return ColorGreen.Lerp(ColorYellow, t)
	} else {
		// Yellow to white
		t := (intensity - 0.8) / 0.2
		return ColorYellow.Lerp(ColorWhite, t)
	}
}

// IntensityToWarmColor maps intensity to warm colors (better for single objects)
// Black -> red -> orange -> yellow -> white
func IntensityToWarmColor(intensity float64) Color {
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}

	if intensity < 0.25 {
		// Black to dark red
		t := intensity / 0.25
		return ColorBlack.Lerp(Color{100, 0, 0}, t)
	} else if intensity < 0.5 {
		// Dark red to red
		t := (intensity - 0.25) / 0.25
		return Color{100, 0, 0}.Lerp(ColorRed, t)
	} else if intensity < 0.75 {
		// Red to orange
		t := (intensity - 0.5) / 0.25
		return ColorRed.Lerp(ColorOrange, t)
	} else {
		// Orange to yellow to white
		t := (intensity - 0.75) / 0.25
		return ColorOrange.Lerp(ColorWhite, t)
	}
}
