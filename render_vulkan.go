package main

import (
	"fmt"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

type VulkanRenderer struct {
	window        *glfw.Window
	width         int
	height        int
	renderContext *RenderContext

	// Vulkan Handles
	instance       vk.Instance
	surface        vk.Surface
	physicalDevice vk.PhysicalDevice
	device         vk.Device

	// Queue families and queues
	graphicsFamily uint32
	presentFamily  uint32
	graphicsQueue  vk.Queue
	presentQueue   vk.Queue

	swapchain  vk.Swapchain
	images     []vk.Image
	imageViews []vk.ImageView

	renderPass        vk.RenderPass
	framebuffers      []vk.Framebuffer
	commandPool       vk.CommandPool
	commandBuffers    []vk.CommandBuffer
	imageAvailableSem vk.Semaphore
	renderFinishedSem vk.Semaphore
	inFlightFence     vk.Fence

	initialized bool
	frameCount  int
}

func NewVulkanRenderer(width, height int) *VulkanRenderer {
	return &VulkanRenderer{
		width:  width,
		height: height,
		renderContext: &RenderContext{
			ViewFrustum: &ViewFrustum{},
		},
	}
}

// initWindow initializes GLFW and creates the window
func (r *VulkanRenderer) initWindow() error {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		return fmt.Errorf("failed to initialize GLFW: %v", err)
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI) // Tell GLFW not to create an OpenGL context
	window, err := glfw.CreateWindow(r.width, r.height, "Go 3D Engine (Vulkan)", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}

	r.window = window
	return nil
}

func (r *VulkanRenderer) Initialize() error {
	if r.initialized {
		return nil
	}

	fmt.Println("[Vulkan] Initializing Window and Vulkan...")

	// 1. Create Window
	if err := r.initWindow(); err != nil {
		return err
	}

	// 2. Init Vulkan Loader
	glfw.Init()
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	if err := vk.Init(); err != nil {
		return fmt.Errorf("failed to init vulkan loader: %v", err)
	}

	// 3. Create Instance
	appInfo := vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		ApiVersion:         vk.MakeVersion(1, 0, 0),
		PApplicationName:   "Go 3D Engine\x00",
		ApplicationVersion: vk.MakeVersion(1, 0, 0),
		PEngineName:        "No Engine\x00",
		EngineVersion:      vk.MakeVersion(1, 0, 0),
	}

	requiredExtensions := r.window.GetRequiredInstanceExtensions()
	instanceInfo := vk.InstanceCreateInfo{
		SType:                   vk.StructureTypeInstanceCreateInfo,
		PApplicationInfo:        &appInfo,
		EnabledExtensionCount:   uint32(len(requiredExtensions)),
		PpEnabledExtensionNames: requiredExtensions,
	}

	var instance vk.Instance
	if res := vk.CreateInstance(&instanceInfo, nil, &instance); res != vk.Success {
		return fmt.Errorf("failed to create instance: %v", res)
	}
	r.instance = instance

	// 4. Create Surface
	surfacePtr, err := r.window.CreateWindowSurface(r.instance, nil)
	if err != nil {
		return fmt.Errorf("failed to create window surface: %v", err)
	}
	r.surface = vk.SurfaceFromPointer(surfacePtr)

	// 5 & 6. Select Physical Device and Queue Families
	var deviceCount uint32
	vk.EnumeratePhysicalDevices(r.instance, &deviceCount, nil)
	if deviceCount == 0 {
		return fmt.Errorf("no GPU with Vulkan support found")
	}
	devices := make([]vk.PhysicalDevice, deviceCount)
	vk.EnumeratePhysicalDevices(r.instance, &deviceCount, devices)

	fmt.Printf("[Vulkan] Found %d physical devices.\n", len(devices))

	var chosenDevice vk.PhysicalDevice
	var graphicsIdx, presentIdx int = -1, -1
	found := false

	for i, device := range devices {
		var props vk.PhysicalDeviceProperties
		vk.GetPhysicalDeviceProperties(device, &props)

		// Debug Name
		var name []byte
		for _, b := range props.DeviceName {
			if b == 0 {
				break
			}
			name = append(name, byte(b))
		}
		fmt.Printf("[Vulkan] Device %d: %s (Type: %d)\n", i, string(name), props.DeviceType)

		var queueFamilyCount uint32
		vk.GetPhysicalDeviceQueueFamilyProperties(device, &queueFamilyCount, nil)
		if queueFamilyCount == 0 {
			fmt.Printf("[Vulkan] Warning: Device %d reported 0 queue families.\n", i)
			continue
		}

		queueProps := make([]vk.QueueFamilyProperties, queueFamilyCount)
		vk.GetPhysicalDeviceQueueFamilyProperties(device, &queueFamilyCount, queueProps)

		currGraphicsIdx := -1
		currPresentIdx := -1

		for j, qProp := range queueProps {
			// Check Graphics
			if qProp.QueueFlags&vk.QueueFlags(vk.QueueGraphicsBit) != 0 {
				currGraphicsIdx = j
			}
			// Check Present
			var supportsPresent vk.Bool32
			vk.GetPhysicalDeviceSurfaceSupport(device, uint32(j), r.surface, &supportsPresent)
			if supportsPresent.B() {
				currPresentIdx = j
			}

			if currGraphicsIdx != -1 && currPresentIdx != -1 {
				// Optimization: Stop if we found a queue that does both (ideal)
				if currGraphicsIdx == currPresentIdx {
					break
				}
			}
		}

		if currGraphicsIdx != -1 && currPresentIdx != -1 {
			chosenDevice = device
			graphicsIdx = currGraphicsIdx
			presentIdx = currPresentIdx
			found = true

			// Prefer Discrete GPU
			if props.DeviceType == vk.PhysicalDeviceTypeDiscreteGpu {
				fmt.Println("[Vulkan] > Selected Discrete GPU.")
				break
			}
		}
	}

	// FALLBACK: If smart selection failed (e.g. properties still zeroed), force Device 0
	if !found && len(devices) > 0 {
		fmt.Println("[Vulkan] Warning: No suitable device found via properties. Attempting fallback to Device 0...")
		chosenDevice = devices[0]
		graphicsIdx = 0
		presentIdx = 0 // Assumption
		found = true
	}

	if !found {
		return fmt.Errorf("failed to find suitable queue families on any device")
	}

	r.physicalDevice = chosenDevice
	r.graphicsFamily = uint32(graphicsIdx)
	r.presentFamily = uint32(presentIdx)

	fmt.Printf("[Vulkan] Using Queue Families -> Graphics: %d, Present: %d\n", r.graphicsFamily, r.presentFamily)

	// 7. Create Logical Device
	uniqueQueueFamilies := make(map[uint32]bool)
	uniqueQueueFamilies[r.graphicsFamily] = true
	uniqueQueueFamilies[r.presentFamily] = true

	var queueCreateInfos []vk.DeviceQueueCreateInfo
	queuePriority := []float32{1.0}

	for family := range uniqueQueueFamilies {
		queueCreateInfos = append(queueCreateInfos, vk.DeviceQueueCreateInfo{
			SType:            vk.StructureTypeDeviceQueueCreateInfo,
			QueueFamilyIndex: family,
			QueueCount:       1,
			PQueuePriorities: queuePriority,
		})
	}

	deviceInfo := vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueCreateInfos)),
		PQueueCreateInfos:       queueCreateInfos,
		EnabledExtensionCount:   1,
		PpEnabledExtensionNames: []string{"VK_KHR_swapchain\x00"},
	}

	var device vk.Device
	if res := vk.CreateDevice(r.physicalDevice, &deviceInfo, nil, &device); res != vk.Success {
		return fmt.Errorf("failed to create logical device: %v", res)
	}
	r.device = device

	var gQueue, pQueue vk.Queue
	vk.GetDeviceQueue(r.device, r.graphicsFamily, 0, &gQueue)
	vk.GetDeviceQueue(r.device, r.presentFamily, 0, &pQueue)
	r.graphicsQueue = gQueue
	r.presentQueue = pQueue

	// 8. Create Swapchain
	if err := r.createSwapchain(); err != nil {
		return err
	}

	// --- NEW INITIALIZATION STEPS ---
	// 9. Create Render Pass
	if err := r.createRenderPass(); err != nil {
		return err
	}

	// 10. Create Framebuffers
	if err := r.createFramebuffers(); err != nil {
		return err
	}

	// 11. Create Command Pool
	if err := r.createCommandPool(); err != nil {
		return err
	}

	// 12. Create Sync Objects
	if err := r.createSyncObjects(); err != nil {
		return err
	}

	// 13. Allocate and Record Command Buffers (Basic Clear Screen)
	if err := r.createCommandBuffers(); err != nil {
		return err
	}
	// --------------------------------

	fmt.Println("[Vulkan] Window created and Vulkan initialized successfully.")
	r.initialized = true
	return nil
}

func (r *VulkanRenderer) createSwapchain() error {
	var caps vk.SurfaceCapabilities
	vk.GetPhysicalDeviceSurfaceCapabilities(r.physicalDevice, r.surface, &caps)

	format := vk.FormatB8g8r8a8Srgb
	colorSpace := vk.ColorSpaceSrgbNonlinear

	swapchainInfo := vk.SwapchainCreateInfo{
		SType:            vk.StructureTypeSwapchainCreateInfo,
		Surface:          r.surface,
		MinImageCount:    caps.MinImageCount + 1,
		ImageFormat:      format,
		ImageColorSpace:  colorSpace,
		ImageExtent:      caps.CurrentExtent,
		ImageArrayLayers: 1,
		ImageUsage:       vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit),
		PreTransform:     caps.CurrentTransform,
		CompositeAlpha:   vk.CompositeAlphaOpaqueBit,
		PresentMode:      vk.PresentModeFifo,
		Clipped:          vk.True,
	}

	// Handle Queue Family Sharing
	if r.graphicsFamily != r.presentFamily {
		swapchainInfo.ImageSharingMode = vk.SharingModeConcurrent
		swapchainInfo.QueueFamilyIndexCount = 2
		swapchainInfo.PQueueFamilyIndices = []uint32{r.graphicsFamily, r.presentFamily}
	} else {
		swapchainInfo.ImageSharingMode = vk.SharingModeExclusive
	}

	var swapchain vk.Swapchain
	if res := vk.CreateSwapchain(r.device, &swapchainInfo, nil, &swapchain); res != vk.Success {
		return fmt.Errorf("failed to create swapchain: %v", res)
	}
	r.swapchain = swapchain

	// Get Images
	var imageCount uint32
	vk.GetSwapchainImages(r.device, r.swapchain, &imageCount, nil)
	r.images = make([]vk.Image, imageCount)
	vk.GetSwapchainImages(r.device, r.swapchain, &imageCount, r.images)

	// Create Views
	r.imageViews = make([]vk.ImageView, len(r.images))
	for i, img := range r.images {
		viewInfo := vk.ImageViewCreateInfo{
			SType:    vk.StructureTypeImageViewCreateInfo,
			Image:    img,
			ViewType: vk.ImageViewType2d,
			Format:   format,
			Components: vk.ComponentMapping{
				R: vk.ComponentSwizzleIdentity,
				G: vk.ComponentSwizzleIdentity,
				B: vk.ComponentSwizzleIdentity,
				A: vk.ComponentSwizzleIdentity,
			},
			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask:     vk.ImageAspectFlags(vk.ImageAspectColorBit),
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		}
		var view vk.ImageView
		if res := vk.CreateImageView(r.device, &viewInfo, nil, &view); res != vk.Success {
			return fmt.Errorf("failed to create image view: %v", res)
		}
		r.imageViews[i] = view
	}
	return nil
}

func (r *VulkanRenderer) createRenderPass() error {
	attachment := vk.AttachmentDescription{
		Format:         vk.FormatB8g8r8a8Srgb, // Must match swapchain format
		Samples:        vk.SampleCount1Bit,
		LoadOp:         vk.AttachmentLoadOpClear, // Clear screen on start
		StoreOp:        vk.AttachmentStoreOpStore,
		StencilLoadOp:  vk.AttachmentLoadOpDontCare,
		StencilStoreOp: vk.AttachmentStoreOpDontCare,
		InitialLayout:  vk.ImageLayoutUndefined,
		FinalLayout:    vk.ImageLayoutPresentSrc,
	}

	subpass := vk.SubpassDescription{
		PipelineBindPoint:    vk.PipelineBindPointGraphics,
		ColorAttachmentCount: 1,
		PColorAttachments: []vk.AttachmentReference{
			{
				Attachment: 0,
				Layout:     vk.ImageLayoutColorAttachmentOptimal,
			},
		},
	}

	// Subpass dependency to synchronize layout transitions
	dependency := vk.SubpassDependency{
		SrcSubpass:    vk.SubpassExternal,
		DstSubpass:    0,
		SrcStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		SrcAccessMask: 0,
		DstStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		DstAccessMask: vk.AccessFlags(vk.AccessColorAttachmentWriteBit),
	}

	renderPassInfo := vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		AttachmentCount: 1,
		PAttachments:    []vk.AttachmentDescription{attachment},
		SubpassCount:    1,
		PSubpasses:      []vk.SubpassDescription{subpass},
		DependencyCount: 1,
		PDependencies:   []vk.SubpassDependency{dependency},
	}

	var renderPass vk.RenderPass
	if res := vk.CreateRenderPass(r.device, &renderPassInfo, nil, &renderPass); res != vk.Success {
		return fmt.Errorf("failed to create render pass: %v", res)
	}
	r.renderPass = renderPass
	return nil
}

func (r *VulkanRenderer) createFramebuffers() error {
	r.framebuffers = make([]vk.Framebuffer, len(r.imageViews))

	for i, imageView := range r.imageViews {
		attachments := []vk.ImageView{imageView}

		fbInfo := vk.FramebufferCreateInfo{
			SType:           vk.StructureTypeFramebufferCreateInfo,
			RenderPass:      r.renderPass,
			AttachmentCount: 1,
			PAttachments:    attachments,
			Width:           uint32(r.width),
			Height:          uint32(r.height),
			Layers:          1,
		}

		var fb vk.Framebuffer
		if res := vk.CreateFramebuffer(r.device, &fbInfo, nil, &fb); res != vk.Success {
			return fmt.Errorf("failed to create framebuffer: %v", res)
		}
		r.framebuffers[i] = fb
	}
	return nil
}

func (r *VulkanRenderer) createCommandPool() error {
	poolInfo := vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: r.graphicsFamily,
		Flags:            0,
	}
	var pool vk.CommandPool
	if res := vk.CreateCommandPool(r.device, &poolInfo, nil, &pool); res != vk.Success {
		return fmt.Errorf("failed to create command pool: %v", res)
	}
	r.commandPool = pool
	return nil
}

func (r *VulkanRenderer) createSyncObjects() error {
	semInfo := vk.SemaphoreCreateInfo{SType: vk.StructureTypeSemaphoreCreateInfo}
	fenceInfo := vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
		Flags: vk.FenceCreateFlags(vk.FenceCreateSignaledBit), // Start signaled
	}

	var ias, rfs vk.Semaphore
	var iff vk.Fence

	if vk.CreateSemaphore(r.device, &semInfo, nil, &ias) != vk.Success ||
		vk.CreateSemaphore(r.device, &semInfo, nil, &rfs) != vk.Success ||
		vk.CreateFence(r.device, &fenceInfo, nil, &iff) != vk.Success {
		return fmt.Errorf("failed to create synchronization objects")
	}

	r.imageAvailableSem = ias
	r.renderFinishedSem = rfs
	r.inFlightFence = iff
	return nil
}

func (r *VulkanRenderer) createCommandBuffers() error {
	// Allocation
	allocInfo := vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        r.commandPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: uint32(len(r.framebuffers)),
	}

	r.commandBuffers = make([]vk.CommandBuffer, len(r.framebuffers))
	if res := vk.AllocateCommandBuffers(r.device, &allocInfo, r.commandBuffers); res != vk.Success {
		return fmt.Errorf("failed to allocate command buffers: %v", res)
	}

	// Recording (Static commands for clearing screen)
	for i, cmdBuffer := range r.commandBuffers {
		beginInfo := vk.CommandBufferBeginInfo{
			SType: vk.StructureTypeCommandBufferBeginInfo,
		}
		vk.BeginCommandBuffer(cmdBuffer, &beginInfo)

		clearColor := vk.ClearValue{}
		// Set clear color to Black (0,0,0,1)
		clearColor.SetColor([]float32{0.0, 0.0, 0.0, 1.0})

		renderPassInfo := vk.RenderPassBeginInfo{
			SType:       vk.StructureTypeRenderPassBeginInfo,
			RenderPass:  r.renderPass,
			Framebuffer: r.framebuffers[i],
			RenderArea: vk.Rect2D{
				Offset: vk.Offset2D{X: 0, Y: 0},
				Extent: vk.Extent2D{Width: uint32(r.width), Height: uint32(r.height)},
			},
			ClearValueCount: 1,
			PClearValues:    []vk.ClearValue{clearColor},
		}

		vk.CmdBeginRenderPass(cmdBuffer, &renderPassInfo, vk.SubpassContentsInline)
		// Draw commands would go here (empty for now, just clearing)
		vk.CmdEndRenderPass(cmdBuffer)

		if res := vk.EndCommandBuffer(cmdBuffer); res != vk.Success {
			return fmt.Errorf("failed to record command buffer: %v", res)
		}
	}
	return nil
}

func (r *VulkanRenderer) Shutdown() {
	if !r.initialized {
		return
	}
	fmt.Println("[Vulkan] Shutting down...")
	vk.DeviceWaitIdle(r.device)

	// Cleanup Sync Objects
	vk.DestroySemaphore(r.device, r.renderFinishedSem, nil)
	vk.DestroySemaphore(r.device, r.imageAvailableSem, nil)
	vk.DestroyFence(r.device, r.inFlightFence, nil)

	// Cleanup Command Pool
	vk.DestroyCommandPool(r.device, r.commandPool, nil)

	// Cleanup Framebuffers
	for _, fb := range r.framebuffers {
		vk.DestroyFramebuffer(r.device, fb, nil)
	}

	// Cleanup RenderPass
	vk.DestroyRenderPass(r.device, r.renderPass, nil)

	for _, view := range r.imageViews {
		vk.DestroyImageView(r.device, view, nil)
	}
	vk.DestroySwapchain(r.device, r.swapchain, nil)
	vk.DestroyDevice(r.device, nil)
	vk.DestroySurface(r.instance, r.surface, nil)
	vk.DestroyInstance(r.instance, nil)

	r.window.Destroy()
	glfw.Terminate()
	r.initialized = false
}

func (r *VulkanRenderer) BeginFrame() {
	if !r.initialized {
		return
	}

	// Poll window events (keyboard/mouse for the window)
	glfw.PollEvents()

	if r.window.ShouldClose() {
		// Handle exit
	}
}

func (r *VulkanRenderer) EndFrame() {
	// Not implemented: Rendering commands
}

func (r *VulkanRenderer) Present() {
	if !r.initialized {
		return
	}

	// 1. Wait for previous frame to finish
	// We wait, but we DO NOT reset yet.
	vk.WaitForFences(r.device, 1, []vk.Fence{r.inFlightFence}, vk.True, ^uint64(0))

	// 2. Acquire image
	var imageIndex uint32
	// Use the corrected function (AcquireNextImage without KHR suffix)
	res := vk.AcquireNextImage(r.device, r.swapchain, ^uint64(0), r.imageAvailableSem, vk.NullFence, &imageIndex)

	// Handle specific return codes to avoid crashing or deadlocking
	if res == vk.NotReady {
		// Window system not ready yet, try again next frame
		return
	}
	if res == vk.Timeout {
		return
	}
	// Note: vk.Suboptimal might be 1000001003, but typically we can proceed.
	// We only strictly fail on error codes (negative values usually, but checking != Success is safer provided we handle special cases)
	if res != vk.Success && res != vk.Suboptimal {
		fmt.Printf("Failed to acquire next image: %v\n", res)
		return
	}

	// 3. NOW it is safe to reset the fence because we are guaranteed to submit work
	vk.ResetFences(r.device, 1, []vk.Fence{r.inFlightFence})

	// 4. Submit Commands
	dstStageMask := vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit)
	submitInfo := vk.SubmitInfo{
		SType:                vk.StructureTypeSubmitInfo,
		WaitSemaphoreCount:   1,
		PWaitSemaphores:      []vk.Semaphore{r.imageAvailableSem},
		PWaitDstStageMask:    []vk.PipelineStageFlags{dstStageMask},
		CommandBufferCount:   1,
		PCommandBuffers:      []vk.CommandBuffer{r.commandBuffers[imageIndex]},
		SignalSemaphoreCount: 1,
		PSignalSemaphores:    []vk.Semaphore{r.renderFinishedSem},
	}

	if res := vk.QueueSubmit(r.graphicsQueue, 1, []vk.SubmitInfo{submitInfo}, r.inFlightFence); res != vk.Success {
		fmt.Printf("Failed to submit draw command buffer: %v\n", res)
		// If submit fails, the fence won't be signaled. We might need to handle this,
		// but usually this is a fatal error.
		return
	}

	// 5. Present
	presentInfo := vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{r.renderFinishedSem},
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{r.swapchain},
		PImageIndices:      []uint32{imageIndex},
	}

	// We ignore the result of Present for now (it might be Suboptimal or OutOfDate during resize)
	vk.QueuePresent(r.presentQueue, &presentInfo)

	// Maintenance
	r.frameCount++
	if r.frameCount%60 == 0 {
		r.window.SetTitle(fmt.Sprintf("Go 3D Engine (Vulkan) - Frame %d", r.frameCount))
	}
}

// Stub methods to satisfy interface
func (r *VulkanRenderer) RenderScene(scene *Scene)                                            {}
func (r *VulkanRenderer) RenderTriangle(tri *Triangle, worldMatrix Matrix4x4, camera *Camera) {}
func (r *VulkanRenderer) RenderLine(line *Line, worldMatrix Matrix4x4, camera *Camera)        {}
func (r *VulkanRenderer) RenderPoint(point *Point, worldMatrix Matrix4x4, camera *Camera)     {}
func (r *VulkanRenderer) RenderMesh(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera)        {}
func (r *VulkanRenderer) SetLightingSystem(ls *LightingSystem)                                {}
func (r *VulkanRenderer) SetCamera(camera *Camera)                                            {}
func (r *VulkanRenderer) GetDimensions() (int, int)                                           { return r.width, r.height }
func (r *VulkanRenderer) SetUseColor(useColor bool)                                           {}
func (r *VulkanRenderer) SetShowDebugInfo(show bool)                                          {}
func (r *VulkanRenderer) SetClipBounds(minX, minY, maxX, maxY int)                            {}
func (r *VulkanRenderer) GetRenderContext() *RenderContext                                    { return r.renderContext }
