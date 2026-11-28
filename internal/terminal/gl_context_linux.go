//go:build terminalgl && (amd64 || arm64)

package terminal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ebitengine/purego"
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

type nativeGLContext struct {
	egl                                eglAPI
	library, display, surface, context uintptr
	displayReady                       bool
}

func (d *glContext) initialize() error {
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
	selector := os.Getenv("GRAPHICS_EGL_PCI")
	if selector != "" {
		device, err := d.deviceForPCI(selector)
		if err != nil {
			return err
		}
		d.display = e.getPlatformDisplay(0x313F, device, nil) // EGL_PLATFORM_DEVICE_EXT
	} else {
		d.display = e.getPlatformDisplay(0x31DD, 0, nil) // EGL_PLATFORM_SURFACELESS_MESA
	}
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
	return d.initializeGL(e.getProcAddress)
}

func (d *glContext) closeNative() {
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
}

// deviceForPCI resolves the DRM node advertised by EGL to a PCI address.
func (d *glContext) deviceForPCI(address string) (uintptr, error) {
	var queryDevices func(int32, *uintptr, *int32) uint32
	var queryString func(uintptr, int32) string
	if err := bindGLFunctions(d.egl.getProcAddress, []glBinding{
		{"eglQueryDevicesEXT", &queryDevices}, {"eglQueryDeviceStringEXT", &queryString},
	}); err != nil {
		return 0, err
	}
	var count int32
	if queryDevices(0, nil, &count) == 0 || count < 1 {
		return 0, errors.New("no EGL devices available")
	}
	devices := make([]uintptr, count)
	if queryDevices(count, &devices[0], &count) == 0 {
		return 0, errors.New("cannot enumerate EGL devices")
	}
	for _, device := range devices[:count] {
		for _, attr := range []int32{0x3377, 0x3233} { // render node and primary node.
			path := queryString(device, attr)
			if path == "" {
				continue
			}
			sysfs, err := filepath.EvalSymlinks(filepath.Join("/sys/class/drm", filepath.Base(path), "device"))
			if err == nil && strings.EqualFold(filepath.Base(sysfs), address) {
				return device, nil
			}
		}
	}
	return 0, fmt.Errorf("EGL device for PCI %s unavailable", address)
}
