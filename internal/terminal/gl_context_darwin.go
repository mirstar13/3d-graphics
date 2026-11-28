//go:build terminalgl && (amd64 || arm64)

package terminal

import (
	"fmt"

	"github.com/ebitengine/purego"
)

type nativeGLContext struct {
	library, context uintptr
	destroyContext   func(uintptr) int32
	setCurrent       func(uintptr) int32
}

func (d *glContext) initialize() error {
	var err error
	d.library, err = purego.Dlopen("/System/Library/Frameworks/OpenGL.framework/OpenGL", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return err
	}
	lookup := func(name string) uintptr {
		addr, _ := purego.Dlsym(d.library, name)
		return addr
	}
	var chooseFormat func(*int32, *uintptr, *int32) int32
	var destroyFormat func(uintptr) int32
	var createContext func(uintptr, uintptr, *uintptr) int32
	if err := bindGLFunctions(lookup, []glBinding{
		{"CGLChoosePixelFormat", &chooseFormat}, {"CGLDestroyPixelFormat", &destroyFormat},
		{"CGLCreateContext", &createContext}, {"CGLDestroyContext", &d.destroyContext},
		{"CGLSetCurrentContext", &d.setCurrent},
	}); err != nil {
		return err
	}

	// CGL's 4.1 core profile supplies GLSL 3.30 and framebuffer objects.
	attrs := [...]int32{99, 0x4100, 0} // kCGLPFAOpenGLProfile, kCGLOGLPVersion_4_1_Core
	var format uintptr
	var count int32
	if status := chooseFormat(&attrs[0], &format, &count); status != 0 || format == 0 {
		return fmt.Errorf("choose CGL 4.1 core pixel format: status %d", status)
	}
	defer destroyFormat(format)
	if status := createContext(format, 0, &d.context); status != 0 || d.context == 0 {
		return fmt.Errorf("create CGL context: status %d", status)
	}
	if status := d.setCurrent(d.context); status != 0 {
		return fmt.Errorf("activate CGL context: status %d", status)
	}
	return d.initializeGL(lookup)
}

func (d *glContext) closeNative() {
	if d.context != 0 {
		d.setCurrent(0)
		d.destroyContext(d.context)
	}
	if d.library != 0 {
		_ = purego.Dlclose(d.library)
	}
}
