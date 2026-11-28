//go:build terminalgl && (linux || windows || darwin) && (amd64 || arm64)

package terminal

import (
	_ "embed"
	"fmt"
	"unsafe"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/perfclock"
)

//go:embed fxaa.frag
var fxaaFragment string

const fxaaVertex = `#version 330 core
void main() {
    vec2 corner = vec2(float((gl_VertexID << 1) & 2), float(gl_VertexID & 2));
    gl_Position = vec4(corner * 2.0 - 1.0, 0.0, 1.0);
}
`

type glFXAA struct {
	*glContext
	source, target, framebuffer uint32
	width, height               int
	pixels                      []byte
}

func openFXAADevice() (fxaaDevice, error) {
	ctx, err := openGLContext()
	if err != nil {
		return nil, err
	}
	d := &glFXAA{glContext: ctx}
	if err = d.initializeFXAA(); err != nil {
		d.close()
		return nil, err
	}
	return d, nil
}

func (d *glFXAA) initializeFXAA() error {
	g := &d.gl
	program, err := g.link(fxaaVertex, fxaaFragment)
	if err != nil {
		return err
	}
	g.useProgram(program)
	var vao uint32
	g.genVertexArrays(1, &vao)
	g.bindVertexArray(vao)
	g.genTextures(1, &d.source)
	g.genTextures(1, &d.target)
	g.genFramebuffers(1, &d.framebuffer)
	g.bindFramebuffer(glFramebuffer, d.framebuffer)
	return nil
}

func (d *glFXAA) resize(width, height int) error {
	g := &d.gl
	for _, texture := range []uint32{d.source, d.target} {
		g.bindTexture(glTexture2D, texture)
		g.texParameteri(glTexture2D, 0x2801, 0x2600) // GL_TEXTURE_MIN_FILTER, GL_NEAREST.
		g.texParameteri(glTexture2D, 0x2800, 0x2600) // GL_TEXTURE_MAG_FILTER, GL_NEAREST.
		g.texImage2D(glTexture2D, 0, glRGBA8UI, int32(width), int32(height), 0, glRGBAInteger, glUnsignedByte, nil)
	}
	g.framebufferTexture2D(glFramebuffer, glColorAttachment0, glTexture2D, d.target, 0)
	if status := g.checkFramebufferStatus(glFramebuffer); status != glFramebufferComplete {
		return fmt.Errorf("OpenGL FXAA framebuffer incomplete: %#x", status)
	}
	if status := g.getError(); status != 0 {
		return fmt.Errorf("allocate OpenGL FXAA textures: %#x", status)
	}
	d.width, d.height = width, height
	size := width * height * 4
	if cap(d.pixels) < size {
		d.pixels = make([]byte, size)
	}
	d.pixels = d.pixels[:size]
	g.viewport(0, 0, int32(width), int32(height))
	return nil
}

func (d *glFXAA) apply(colors [][]engine.Color) error {
	stages := d.stages
	d.stages = nil
	start := stageStart(stages)
	height, width := len(colors), len(colors[0])
	if width != d.width || height != d.height {
		if err := d.resize(width, height); err != nil {
			return err
		}
	}
	i := 0
	for _, row := range colors {
		for _, c := range row {
			d.pixels[i], d.pixels[i+1], d.pixels[i+2] = c.R, c.G, c.B
			i += 4
		}
	}
	if stages != nil {
		stages.PackNS = perfclock.Since(start).Nanoseconds()
		start = perfclock.Now()
	}
	g := &d.gl
	g.bindTexture(glTexture2D, d.source)
	g.texSubImage2D(glTexture2D, 0, 0, 0, int32(width), int32(height), glRGBAInteger, glUnsignedByte, unsafe.Pointer(&d.pixels[0]))
	if stages != nil {
		stages.UploadNS = perfclock.Since(start).Nanoseconds()
		stages.UploadBytes = len(d.pixels)
		start = perfclock.Now()
		d.beginTiming(true)
	}
	g.drawArrays(4, 0, 3) // GL_TRIANGLES.
	if stages != nil {
		d.endTiming()
		stages.SubmitNS = perfclock.Since(start).Nanoseconds()
		start = perfclock.Now()
	}
	g.readPixels(0, 0, int32(width), int32(height), glRGBAInteger, glUnsignedByte, unsafe.Pointer(&d.pixels[0]))
	if stages != nil {
		stages.ReadbackNS = perfclock.Since(start).Nanoseconds()
		stages.ReadbackBytes = len(d.pixels)
		stages.GPUNS = d.gpuTime()
		start = perfclock.Now()
	}
	if status := g.getError(); status != 0 {
		return fmt.Errorf("OpenGL FXAA filter/readback: %#x", status)
	}
	i = 0
	for _, row := range colors {
		for x := range row {
			row[x] = engine.Color{R: d.pixels[i], G: d.pixels[i+1], B: d.pixels[i+2]}
			i += 4
		}
	}
	if stages != nil {
		stages.PackNS += perfclock.Since(start).Nanoseconds()
	}
	return nil
}
