//go:build terminalgl && (amd64 || arm64)

package terminal

import (
	"fmt"
	"os"
	"strconv"

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
	attrs := []int32{99, 0x4100} // kCGLPFAOpenGLProfile, kCGLOGLPVersion_4_1_Core
	if requested := os.Getenv("GRAPHICS_CGL_RENDERER"); requested != "" {
		id, err := strconv.ParseUint(requested, 0, 32)
		if err != nil {
			return fmt.Errorf("invalid CGL renderer ID: %w", err)
		}
		attrs = append(attrs, 70, int32(id)) // kCGLPFARendererID
	}
	attrs = append(attrs, 0)
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

// OpenGLRendererIDs reports accelerated CGL renderer IDs visible on the host.
func OpenGLRendererIDs() ([]string, error) {
	library, err := purego.Dlopen("/System/Library/Frameworks/OpenGL.framework/OpenGL", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, err
	}
	defer purego.Dlclose(library)
	var query func(uint32, *uintptr, *int32) int32
	var describe func(uintptr, int32, int32, *int32) int32
	var destroy func(uintptr) int32
	if err := bindGLFunctions(func(name string) uintptr { addr, _ := purego.Dlsym(library, name); return addr }, []glBinding{
		{"CGLQueryRendererInfo", &query}, {"CGLDescribeRenderer", &describe}, {"CGLDestroyRendererInfo", &destroy},
	}); err != nil {
		return nil, err
	}
	var info uintptr
	var count int32
	if status := query(^uint32(0), &info, &count); status != 0 || info == 0 {
		return nil, fmt.Errorf("query CGL renderers: status %d", status)
	}
	defer destroy(info)
	ids := []string{}
	seen := map[int32]bool{}
	for i := int32(0); i < count; i++ {
		var id, accelerated int32
		if describe(info, i, 73, &accelerated) != 0 || accelerated == 0 ||
			describe(info, i, 70, &id) != 0 || id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, fmt.Sprintf("%#x", uint32(id)))
	}
	return ids, nil
}
