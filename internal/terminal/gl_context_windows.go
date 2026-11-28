//go:build terminalgl && (amd64 || arm64)

package terminal

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	openGL32          = windows.NewLazySystemDLL("opengl32.dll")
	wglGetProcAddress = openGL32.NewProc("wglGetProcAddress")
	wglCreateContext  = openGL32.NewProc("wglCreateContext")
	wglDeleteContext  = openGL32.NewProc("wglDeleteContext")
	wglMakeCurrent    = openGL32.NewProc("wglMakeCurrent")

	glUser32            = windows.NewLazySystemDLL("user32.dll")
	registerGLWindow    = glUser32.NewProc("RegisterClassExW")
	defaultGLWindowProc = glUser32.NewProc("DefWindowProcW")
	createWindow        = glUser32.NewProc("CreateWindowExW")
	destroyWindow       = glUser32.NewProc("DestroyWindow")
	getWindowDC         = glUser32.NewProc("GetDC")
	releaseWindowDC     = glUser32.NewProc("ReleaseDC")

	glGDI32           = windows.NewLazySystemDLL("gdi32.dll")
	choosePixelFormat = glGDI32.NewProc("ChoosePixelFormat")
	setPixelFormat    = glGDI32.NewProc("SetPixelFormat")
)

type nativeGLContext struct {
	window, dc, context uintptr
}

type glWindowClass struct {
	size, style                        uint32
	windowProc                         uintptr
	classExtra, windowExtra            int32
	instance, icon, cursor, background uintptr
	menuName, className                *uint16
	smallIcon                          uintptr
}

var glWindowClassName = windows.StringToUTF16Ptr("TerminalGraphicsOpenGLWindow")

var registerGLWindowClass = sync.OnceValues(func() (windows.Handle, error) {
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(windows.GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, nil, &instance); err != nil {
		return 0, err
	}
	class := glWindowClass{
		style:      0x20, // CS_OWNDC
		windowProc: defaultGLWindowProc.Addr(),
		instance:   uintptr(instance),
		className:  glWindowClassName,
	}
	class.size = uint32(unsafe.Sizeof(class))
	if atom, _, err := registerGLWindow.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return 0, fmt.Errorf("register WGL window class: %w", err)
	}
	return instance, nil
})

type pixelFormatDescriptor struct {
	size, version                                                          uint16
	flags                                                                  uint32
	pixelType, colorBits                                                   uint8
	redBits, redShift, greenBits, greenShift, blueBits, blueShift          uint8
	alphaBits, alphaShift                                                  uint8
	accumBits, accumRedBits, accumGreenBits, accumBlueBits, accumAlphaBits uint8
	depthBits, stencilBits, auxBuffers, layerType, reserved                uint8
	layerMask, visibleMask, damageMask                                     uint32
}

func wglLookup(name string) uintptr {
	cname, err := windows.BytePtrFromString(name)
	if err != nil {
		return 0
	}
	addr, _, _ := wglGetProcAddress.Call(uintptr(unsafe.Pointer(cname)))
	// Some drivers return these sentinel values for missing functions.
	if addr > 3 && addr != ^uintptr(0) {
		return addr
	}
	proc := openGL32.NewProc(name)
	if err := proc.Find(); err != nil {
		return 0
	}
	return proc.Addr()
}

func (d *glContext) initialize() error {
	if err := openGL32.Load(); err != nil {
		return err
	}
	// A hidden window provides a pixel format and device context for WGL.
	// Rendering stays in framebuffer objects and never presents to the window.
	instance, err := registerGLWindowClass()
	if err != nil {
		return err
	}
	const windowStyle = 0x80000000 | 0x02000000 | 0x04000000 // WS_POPUP | WS_CLIPCHILDREN | WS_CLIPSIBLINGS
	window, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(glWindowClassName)), 0, windowStyle,
		0, 0, 1, 1, 0, 0, uintptr(instance), 0)
	if window == 0 {
		return fmt.Errorf("create hidden WGL window: %w", err)
	}
	d.window = window
	dc, _, err := getWindowDC.Call(window)
	if dc == 0 {
		return fmt.Errorf("get WGL window device context: %w", err)
	}
	d.dc = dc
	pfd := pixelFormatDescriptor{version: 1, flags: 0x24, colorBits: 24} // PFD_DRAW_TO_WINDOW | PFD_SUPPORT_OPENGL
	pfd.size = uint16(unsafe.Sizeof(pfd))
	format, _, err := choosePixelFormat.Call(dc, uintptr(unsafe.Pointer(&pfd)))
	if format == 0 {
		return fmt.Errorf("choose WGL pixel format: %w", err)
	}
	if ok, _, err := setPixelFormat.Call(dc, format, uintptr(unsafe.Pointer(&pfd))); ok == 0 {
		return fmt.Errorf("set WGL pixel format: %w", err)
	}
	legacy, _, err := wglCreateContext.Call(dc)
	if legacy == 0 {
		return fmt.Errorf("create bootstrap WGL context: %w", err)
	}
	// Keep the bootstrap context in d until the core context is current.
	d.context = legacy
	if ok, _, err := wglMakeCurrent.Call(dc, legacy); ok == 0 {
		return fmt.Errorf("activate bootstrap WGL context: %w", err)
	}
	var createCore func(uintptr, uintptr, *int32) uintptr
	if err := bindGLFunctions(wglLookup, []glBinding{{"wglCreateContextAttribsARB", &createCore}}); err != nil {
		return fmt.Errorf("WGL driver does not support OpenGL 3.3 core: %w", err)
	}
	attrs := [...]int32{0x2091, 3, 0x2092, 3, 0x9126, 1, 0}
	core := createCore(dc, 0, &attrs[0])
	if core == 0 {
		return fmt.Errorf("WGL driver could not create an OpenGL 3.3 core context")
	}
	if ok, _, err := wglMakeCurrent.Call(dc, core); ok == 0 {
		wglDeleteContext.Call(core)
		return fmt.Errorf("activate core WGL context: %w", err)
	}
	d.context = core
	wglDeleteContext.Call(legacy)
	return d.initializeGL(wglLookup)
}

func (d *glContext) closeNative() {
	if d.context != 0 {
		wglMakeCurrent.Call(0, 0)
		wglDeleteContext.Call(d.context)
	}
	if d.dc != 0 {
		releaseWindowDC.Call(d.window, d.dc)
	}
	if d.window != 0 {
		destroyWindow.Call(d.window)
	}
}
