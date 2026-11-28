//go:build terminalgl && (linux || windows || darwin) && (amd64 || arm64)

package terminal

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	glTexture2D           = 0x0DE1
	glRGBA8UI             = 0x8D7C
	glRGBAInteger         = 0x8D99
	glUnsignedByte        = 0x1401
	glFramebuffer         = 0x8D40
	glColorAttachment0    = 0x8CE0
	glFramebufferComplete = 0x8CD5
	glVertexShader        = 0x8B31
	glFragmentShader      = 0x8B30
	glCompileStatus       = 0x8B81
	glLinkStatus          = 0x8B82
	glInfoLogLength       = 0x8B84
)

type glAPI struct {
	getString              func(uint32) string
	getError               func() uint32
	createShader           func(uint32) uint32
	shaderSource           func(uint32, int32, **byte, *int32)
	compileShader          func(uint32)
	getShaderiv            func(uint32, uint32, *int32)
	getShaderInfoLog       func(uint32, int32, *int32, *byte)
	deleteShader           func(uint32)
	createProgram          func() uint32
	attachShader           func(uint32, uint32)
	linkProgram            func(uint32)
	getProgramiv           func(uint32, uint32, *int32)
	getProgramInfoLog      func(uint32, int32, *int32, *byte)
	useProgram             func(uint32)
	genVertexArrays        func(int32, *uint32)
	bindVertexArray        func(uint32)
	genTextures            func(int32, *uint32)
	bindTexture            func(uint32, uint32)
	texParameteri          func(uint32, uint32, int32)
	texImage2D             func(uint32, int32, int32, int32, int32, int32, uint32, uint32, unsafe.Pointer)
	texSubImage2D          func(uint32, int32, int32, int32, int32, int32, uint32, uint32, unsafe.Pointer)
	genFramebuffers        func(int32, *uint32)
	bindFramebuffer        func(uint32, uint32)
	framebufferTexture2D   func(uint32, uint32, uint32, uint32, int32)
	checkFramebufferStatus func(uint32) uint32
	viewport               func(int32, int32, int32, int32)
	drawArrays             func(uint32, int32, int32)
	readPixels             func(int32, int32, int32, int32, uint32, uint32, unsafe.Pointer)
}

type glBinding struct {
	name   string
	target any
}

func bindGLFunctions(lookup func(string) uintptr, bindings []glBinding) error {
	for _, binding := range bindings {
		addr := lookup(binding.name)
		if addr == 0 {
			return fmt.Errorf("missing %s", binding.name)
		}
		purego.RegisterFunc(binding.target, addr)
	}
	return nil
}

type glContext struct {
	nativeGLContext
	gl     glAPI
	driver string
	lookup func(string) uintptr
	timer  gpuTimer
	stages *DeviceStages
}

func (d *glContext) measure(stages *DeviceStages) { d.stages = stages }

func openGLContext() (*glContext, error) {
	d := new(glContext)
	// Context creation, rendering, and destruction use the owning OS thread.
	runtime.LockOSThread()
	if err := d.initialize(); err != nil {
		d.close()
		return nil, err
	}
	return d, nil
}

func (d *glContext) initializeGL(lookup func(string) uintptr) error {
	d.lookup = lookup
	g := &d.gl
	if err := bindGLFunctions(lookup, []glBinding{
		{"glGetString", &g.getString}, {"glGetError", &g.getError},
		{"glCreateShader", &g.createShader}, {"glShaderSource", &g.shaderSource}, {"glCompileShader", &g.compileShader},
		{"glGetShaderiv", &g.getShaderiv}, {"glGetShaderInfoLog", &g.getShaderInfoLog}, {"glDeleteShader", &g.deleteShader},
		{"glCreateProgram", &g.createProgram}, {"glAttachShader", &g.attachShader}, {"glLinkProgram", &g.linkProgram},
		{"glGetProgramiv", &g.getProgramiv}, {"glGetProgramInfoLog", &g.getProgramInfoLog}, {"glUseProgram", &g.useProgram},
		{"glGenVertexArrays", &g.genVertexArrays}, {"glBindVertexArray", &g.bindVertexArray},
		{"glGenTextures", &g.genTextures}, {"glBindTexture", &g.bindTexture}, {"glTexParameteri", &g.texParameteri},
		{"glTexImage2D", &g.texImage2D}, {"glTexSubImage2D", &g.texSubImage2D},
		{"glGenFramebuffers", &g.genFramebuffers}, {"glBindFramebuffer", &g.bindFramebuffer},
		{"glFramebufferTexture2D", &g.framebufferTexture2D}, {"glCheckFramebufferStatus", &g.checkFramebufferStatus},
		{"glViewport", &g.viewport}, {"glDrawArrays", &g.drawArrays}, {"glReadPixels", &g.readPixels},
	}); err != nil {
		return err
	}
	d.driver = g.getString(0x1F01) // GL_RENDERER
	return nil
}

func (g *glAPI) link(vertexSource, fragmentSource string) (uint32, error) {
	vertex, err := g.compile(glVertexShader, vertexSource)
	if err != nil {
		return 0, err
	}
	defer g.deleteShader(vertex)
	fragment, err := g.compile(glFragmentShader, fragmentSource)
	if err != nil {
		return 0, err
	}
	defer g.deleteShader(fragment)
	program := g.createProgram()
	g.attachShader(program, vertex)
	g.attachShader(program, fragment)
	g.linkProgram(program)
	var linked int32
	g.getProgramiv(program, glLinkStatus, &linked)
	if linked == 0 {
		return 0, fmt.Errorf("link OpenGL program: %s", shaderLog(program, g.getProgramiv, g.getProgramInfoLog))
	}
	return program, nil
}

func shaderLog(object uint32, size func(uint32, uint32, *int32), get func(uint32, int32, *int32, *byte)) string {
	var length int32
	size(object, glInfoLogLength, &length)
	if length <= 0 {
		return "(no log)"
	}
	log := make([]byte, length)
	get(object, length, &length, &log[0])
	return string(log[:length])
}

func (g *glAPI) compile(kind uint32, text string) (uint32, error) {
	shader := g.createShader(kind)
	source := []byte(text)
	ptr := &source[0]
	length := int32(len(source))
	g.shaderSource(shader, 1, &ptr, &length)
	runtime.KeepAlive(source)
	g.compileShader(shader)
	var compiled int32
	g.getShaderiv(shader, glCompileStatus, &compiled)
	if compiled == 0 {
		log := shaderLog(shader, g.getShaderiv, g.getShaderInfoLog)
		g.deleteShader(shader)
		return 0, fmt.Errorf("compile OpenGL shader: %s", log)
	}
	return shader, nil
}

func (d *glContext) name() string { return d.driver }

func (d *glContext) close() {
	d.closeNative()
	runtime.UnlockOSThread()
}

type gpuTimer struct {
	id      uint32
	begin   func(uint32, uint32)
	end     func(uint32)
	status  func(uint32, uint32, *uint32)
	result  func(uint32, uint32, *uint64)
	checked bool
}

func (d *glContext) beginTiming(enabled bool) {
	if !enabled {
		return
	}
	t := &d.timer
	if !t.checked {
		t.checked = true
		var generate func(int32, *uint32)
		if bindGLFunctions(d.lookup, []glBinding{
			{"glGenQueries", &generate}, {"glBeginQuery", &t.begin},
			{"glEndQuery", &t.end}, {"glGetQueryObjectuiv", &t.status},
			{"glGetQueryObjectui64v", &t.result},
		}) == nil {
			generate(1, &t.id)
		}
	}
	if t.id != 0 {
		t.begin(0x88BF, t.id) // GL_TIME_ELAPSED.
	}
}

func (d *glContext) endTiming() {
	if d.timer.id != 0 {
		d.timer.end(0x88BF)
	}
}

// Read the timer after the required framebuffer readback. Leave the sample
// empty when the driver has not made the query available yet.
func (d *glContext) gpuTime() *uint64 {
	t := &d.timer
	if t.id == 0 {
		return nil
	}
	var ready uint32
	t.status(t.id, 0x8867, &ready) // GL_QUERY_RESULT_AVAILABLE.
	if ready == 0 {
		return nil
	}
	var elapsed uint64
	t.result(t.id, 0x8866, &elapsed) // GL_QUERY_RESULT.
	return &elapsed
}
