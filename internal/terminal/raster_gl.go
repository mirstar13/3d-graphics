//go:build terminalgl && (linux || windows || darwin) && (amd64 || arm64)

package terminal

import (
	"fmt"
	"unsafe"
)

const rasterVertex = `#version 330 core
layout(location = 0) in vec4 clipPosition;
flat out uint triangle;
void main() {
    gl_Position = clipPosition;
    triangle = uint(gl_VertexID / 3) + 1u;
}
`

const rasterFragment = `#version 330 core
flat in uint triangle;
layout(location = 0) out uint cell;
void main() {
    cell = triangle;
}
`

const (
	glArrayBuffer       = 0x8892
	glFloat             = 0x1406
	glRenderbuffer      = 0x8D41
	glRedInteger        = 0x8D94
	glUnsignedInt       = 0x1405
	glR32UI             = 0x8236
	glDepthAttachment   = 0x8D00
	glDepthComponent32F = 0x8CAC
)

type glRaster struct {
	*glContext
	genBuffers              func(int32, *uint32)
	bindBuffer              func(uint32, uint32)
	bufferData              func(uint32, int, unsafe.Pointer, uint32)
	enableVertexAttribArray func(uint32)
	vertexAttribPointer     func(uint32, int32, uint32, uint8, int32, uintptr)
	genRenderbuffers        func(int32, *uint32)
	bindRenderbuffer        func(uint32, uint32)
	renderbufferStorage     func(uint32, uint32, int32, int32)
	framebufferRenderbuffer func(uint32, uint32, uint32, uint32)
	enable                  func(uint32)
	clearBufferuiv          func(uint32, int32, *uint32)
	clear                   func(uint32)

	vertexBuffer, target, depth, framebuffer uint32
	width, height                            int
	ids                                      []uint32
}

func openRasterDevice() (rasterDevice, error) {
	ctx, err := openGLContext()
	if err != nil {
		return nil, err
	}
	d := &glRaster{glContext: ctx}
	if err = d.initializeRaster(); err != nil {
		d.close()
		return nil, err
	}
	return d, nil
}

func (d *glRaster) initializeRaster() error {
	if err := bindGLFunctions(d.lookup, []glBinding{
		{"glGenBuffers", &d.genBuffers}, {"glBindBuffer", &d.bindBuffer}, {"glBufferData", &d.bufferData},
		{"glEnableVertexAttribArray", &d.enableVertexAttribArray}, {"glVertexAttribPointer", &d.vertexAttribPointer},
		{"glGenRenderbuffers", &d.genRenderbuffers}, {"glBindRenderbuffer", &d.bindRenderbuffer},
		{"glRenderbufferStorage", &d.renderbufferStorage}, {"glFramebufferRenderbuffer", &d.framebufferRenderbuffer},
		{"glEnable", &d.enable}, {"glClearBufferuiv", &d.clearBufferuiv}, {"glClear", &d.clear},
	}); err != nil {
		return err
	}
	g := &d.gl
	program, err := g.link(rasterVertex, rasterFragment)
	if err != nil {
		return err
	}
	g.useProgram(program)
	var vao uint32
	g.genVertexArrays(1, &vao)
	g.bindVertexArray(vao)
	d.genBuffers(1, &d.vertexBuffer)
	d.bindBuffer(glArrayBuffer, d.vertexBuffer)
	d.vertexAttribPointer(0, 4, glFloat, 0, 0, 0)
	d.enableVertexAttribArray(0)
	d.enable(0x0B71) // GL_DEPTH_TEST with default GL_LESS.
	g.genFramebuffers(1, &d.framebuffer)
	g.bindFramebuffer(glFramebuffer, d.framebuffer)
	g.genTextures(1, &d.target)
	g.bindTexture(glTexture2D, d.target)
	g.texParameteri(glTexture2D, 0x2801, 0x2600) // GL_NEAREST.
	g.texParameteri(glTexture2D, 0x2800, 0x2600)
	d.genRenderbuffers(1, &d.depth)
	d.bindRenderbuffer(glRenderbuffer, d.depth)
	if code := g.getError(); code != 0 {
		return fmt.Errorf("initialize visibility pass: %#x", code)
	}
	return nil
}

func (d *glRaster) resize(width, height int) error {
	g := &d.gl
	g.texImage2D(glTexture2D, 0, glR32UI, int32(width), int32(height), 0, glRedInteger, glUnsignedInt, nil)
	g.framebufferTexture2D(glFramebuffer, glColorAttachment0, glTexture2D, d.target, 0)
	d.renderbufferStorage(glRenderbuffer, glDepthComponent32F, int32(width), int32(height))
	d.framebufferRenderbuffer(glFramebuffer, glDepthAttachment, glRenderbuffer, d.depth)
	if code := g.checkFramebufferStatus(glFramebuffer); code != glFramebufferComplete {
		return fmt.Errorf("visibility framebuffer incomplete: %#x", code)
	}
	if code := g.getError(); code != 0 {
		return fmt.Errorf("resize visibility pass: %#x", code)
	}
	d.width, d.height = width, height
	count := width * height
	if cap(d.ids) < count {
		d.ids = make([]uint32, count)
	}
	d.ids = d.ids[:count]
	g.viewport(0, 0, int32(width), int32(height))
	return nil
}

func (d *glRaster) visibility(width, height int, vertices []float32) ([]uint32, error) {
	if width != d.width || height != d.height {
		if err := d.resize(width, height); err != nil {
			return nil, err
		}
	}
	d.bufferData(glArrayBuffer, len(vertices)*4, unsafe.Pointer(&vertices[0]), 0x88E0) // GL_STREAM_DRAW.
	zero := [4]uint32{}
	d.clearBufferuiv(0x1800, 0, &zero[0]) // GL_COLOR.
	d.clear(0x0100)                       // GL_DEPTH_BUFFER_BIT.
	d.gl.drawArrays(4, 0, int32(len(vertices)/4))
	d.gl.readPixels(0, 0, int32(width), int32(height), glRedInteger, glUnsignedInt, unsafe.Pointer(&d.ids[0]))
	if code := d.gl.getError(); code != 0 {
		return nil, fmt.Errorf("visibility draw/readback: %#x", code)
	}
	return d.ids, nil
}
