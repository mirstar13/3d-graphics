//go:build terminalgl && (amd64 || arm64)

package terminal

import (
	_ "embed"
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

//go:embed fxaa.frag
var fxaaFragment string

const fxaaVertex = `#version 330 core
void main() {
    vec2 corner = vec2(float((gl_VertexID << 1) & 2), float(gl_VertexID & 2));
    gl_Position = vec4(corner * 2.0 - 1.0, 0.0, 1.0);
}
`

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

type eglAPI struct {
	getPlatformDisplay   func(uint32, uintptr, *int) uintptr
	initialize           func(uintptr, *int32, *int32) uint32
	terminate            func(uintptr) uint32
	bindAPI              func(uint32) uint32
	chooseConfig         func(uintptr, *int32, *uintptr, int32, *int32) uint32
	createPbufferSurface func(uintptr, uintptr, *int32) uintptr
	createContext        func(uintptr, uintptr, uintptr, *int32) uintptr
	makeCurrent          func(uintptr, uintptr, uintptr, uintptr) uint32
	destroyContext       func(uintptr, uintptr) uint32
	destroySurface       func(uintptr, uintptr) uint32
	getProcAddress       func(string) uintptr
	getError             func() uint32
}

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

type glFXAA struct {
	egl                                eglAPI
	gl                                 glAPI
	library, display, surface, context uintptr
	displayReady                       bool
	driver                             string
	source, target, framebuffer        uint32
	width, height                      int
	pixels                             []byte
}

func openFXAADevice() (fxaaDevice, error) {
	device := new(glFXAA)
	// The EGL context must stay on the thread that owns it until close.
	runtime.LockOSThread()
	if err := device.initialize(); err != nil {
		device.close()
		return nil, err
	}
	return device, nil
}

func (d *glFXAA) initialize() error {
	var err error
	d.library, err = purego.Dlopen("libEGL.so.1", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return err
	}
	e := &d.egl
	if err = bindGLFunctions(func(name string) uintptr { addr, _ := purego.Dlsym(d.library, name); return addr }, []glBinding{
		{"eglGetPlatformDisplay", &e.getPlatformDisplay}, {"eglInitialize", &e.initialize}, {"eglTerminate", &e.terminate},
		{"eglBindAPI", &e.bindAPI}, {"eglChooseConfig", &e.chooseConfig}, {"eglCreatePbufferSurface", &e.createPbufferSurface},
		{"eglCreateContext", &e.createContext}, {"eglMakeCurrent", &e.makeCurrent}, {"eglDestroyContext", &e.destroyContext},
		{"eglDestroySurface", &e.destroySurface}, {"eglGetProcAddress", &e.getProcAddress}, {"eglGetError", &e.getError},
	}); err != nil {
		return err
	}

	const eglNone = 0x3038
	d.display = e.getPlatformDisplay(0x31DD, 0, nil) // EGL_PLATFORM_SURFACELESS_MESA
	if d.display == 0 || e.initialize(d.display, nil, nil) == 0 {
		return fmt.Errorf("initialize surfaceless EGL: %#x", e.getError())
	}
	d.displayReady = true
	if e.bindAPI(0x30A2) == 0 {
		return fmt.Errorf("bind EGL OpenGL API: %#x", e.getError())
	}
	configAttrs := [...]int32{0x3033, 1, 0x3040, 8, eglNone} // Pbuffer surface, desktop OpenGL.
	var config uintptr
	var count int32
	if e.chooseConfig(d.display, &configAttrs[0], &config, 1, &count) == 0 || count == 0 {
		return errors.New("EGL has no desktop OpenGL pbuffer configuration")
	}
	surfaceAttrs := [...]int32{0x3057, 1, 0x3056, 1, eglNone} // Width and height.
	d.surface = e.createPbufferSurface(d.display, config, &surfaceAttrs[0])
	contextAttrs := [...]int32{0x3098, 3, 0x30FB, 3, 0x30FD, 1, eglNone} // OpenGL 3.3 core.
	d.context = e.createContext(d.display, config, 0, &contextAttrs[0])
	if d.surface == 0 || d.context == 0 || e.makeCurrent(d.display, d.surface, d.surface, d.context) == 0 {
		return fmt.Errorf("create OpenGL 3.3 context: %#x", e.getError())
	}
	return d.initializeGL()
}

func (d *glFXAA) initializeGL() error {
	g := &d.gl
	if err := bindGLFunctions(d.egl.getProcAddress, []glBinding{
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
	vertex, err := g.compile(glVertexShader, fxaaVertex)
	if err != nil {
		return err
	}
	defer g.deleteShader(vertex)
	fragment, err := g.compile(glFragmentShader, fxaaFragment)
	if err != nil {
		return err
	}
	defer g.deleteShader(fragment)
	program := g.createProgram()
	g.attachShader(program, vertex)
	g.attachShader(program, fragment)
	g.linkProgram(program)
	var linked int32
	g.getProgramiv(program, glLinkStatus, &linked)
	if linked == 0 {
		return fmt.Errorf("link FXAA program: %s", shaderLog(program, g.getProgramiv, g.getProgramInfoLog))
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
		return 0, fmt.Errorf("compile FXAA shader: %s", log)
	}
	return shader, nil
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
	g := &d.gl
	g.bindTexture(glTexture2D, d.source)
	g.texSubImage2D(glTexture2D, 0, 0, 0, int32(width), int32(height), glRGBAInteger, glUnsignedByte, unsafe.Pointer(&d.pixels[0]))
	g.drawArrays(4, 0, 3) // GL_TRIANGLES.
	g.readPixels(0, 0, int32(width), int32(height), glRGBAInteger, glUnsignedByte, unsafe.Pointer(&d.pixels[0]))
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
	return nil
}

func (d *glFXAA) name() string { return d.driver }

func (d *glFXAA) close() {
	if d.displayReady {
		if d.context != 0 {
			d.egl.makeCurrent(d.display, 0, 0, 0)
			d.egl.destroyContext(d.display, d.context)
		}
		if d.surface != 0 {
			d.egl.destroySurface(d.display, d.surface)
		}
		d.egl.terminate(d.display)
	}
	if d.library != 0 {
		_ = purego.Dlclose(d.library)
	}
	runtime.UnlockOSThread()
}
