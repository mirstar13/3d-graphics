package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	vk "github.com/vulkan-go/vulkan"
)

// Vertex represents a vertex with position and color
type VulkanVertex struct {
	Pos   [3]float32
	Color [3]float32
}

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

	swapchain       vk.Swapchain
	swapchainFormat vk.Format
	swapchainExtent vk.Extent2D
	images          []vk.Image
	imageViews      []vk.ImageView

	renderPass       vk.RenderPass
	pipelineLayout   vk.PipelineLayout
	graphicsPipeline vk.Pipeline
	framebuffers     []vk.Framebuffer

	commandPool    vk.CommandPool
	commandBuffers []vk.CommandBuffer

	// Synchronization
	imageAvailableSem vk.Semaphore
	renderFinishedSem vk.Semaphore
	inFlightFence     vk.Fence

	// Vertex buffer
	vertexBuffer       vk.Buffer
	vertexBufferMemory vk.DeviceMemory
	maxVertices        int
	currentVertices    []VulkanVertex

	// Depth buffer
	depthImage       vk.Image
	depthImageMemory vk.DeviceMemory
	depthImageView   vk.ImageView

	// Uniform buffer
	uniformBuffer       vk.Buffer
	uniformBufferMemory vk.DeviceMemory
	descriptorSetLayout vk.DescriptorSetLayout
	descriptorPool      vk.DescriptorPool
	descriptorSet       vk.DescriptorSet

	// Settings
	UseColor       bool
	ShowDebugInfo  bool
	LightingSystem *LightingSystem
	Camera         *Camera

	initialized bool
	frameCount  int
}

// UniformBufferObject contains matrices for rendering
type UniformBufferObject struct {
	Model [16]float32
	View  [16]float32
	Proj  [16]float32
}

func NewVulkanRenderer(width, height int) *VulkanRenderer {
	return &VulkanRenderer{
		width:  width,
		height: height,
		renderContext: &RenderContext{
			ViewFrustum: &ViewFrustum{},
		},
		UseColor:        true,
		ShowDebugInfo:   false,
		maxVertices:     100000, // Preallocate for 100k vertices
		currentVertices: make([]VulkanVertex, 0, 10000),
	}
}

func (r *VulkanRenderer) Initialize() error {
	if r.initialized {
		return nil
	}

	fmt.Println("[Vulkan] Initializing...")

	// 1. Create Window
	if err := r.initWindow(); err != nil {
		return err
	}

	// 2. Init Vulkan
	if err := r.initVulkan(); err != nil {
		return err
	}

	// 3. Create Graphics Pipeline
	if err := r.createGraphicsPipeline(); err != nil {
		return err
	}

	// 4. Create Depth Resources
	if err := r.createDepthResources(); err != nil {
		return err
	}

	// 5. Create Framebuffers (with depth)
	if err := r.createFramebuffers(); err != nil {
		return err
	}

	// 6. Create Vertex Buffer
	if err := r.createVertexBuffer(); err != nil {
		return err
	}

	// 7. Create Uniform Buffer
	if err := r.createUniformBuffer(); err != nil {
		return err
	}

	// 8. Create Descriptor Sets
	if err := r.createDescriptorSets(); err != nil {
		return err
	}

	// 9. Create Command Buffers
	if err := r.createCommandBuffers(); err != nil {
		return err
	}

	// 10. Create Sync Objects
	if err := r.createSyncObjects(); err != nil {
		return err
	}

	fmt.Println("[Vulkan] Initialization complete")
	r.initialized = true
	return nil
}

func (r *VulkanRenderer) initWindow() error {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		return fmt.Errorf("failed to initialize GLFW: %v", err)
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(r.width, r.height, "Go 3D Engine (Vulkan)", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}

	r.window = window
	return nil
}

func (r *VulkanRenderer) initVulkan() error {
	// Init Vulkan loader
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	if err := vk.Init(); err != nil {
		return fmt.Errorf("failed to init vulkan: %v", err)
	}

	// Create Instance
	appInfo := vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		ApiVersion:         vk.MakeVersion(1, 0, 0),
		PApplicationName:   "Go 3D Engine\x00",
		ApplicationVersion: vk.MakeVersion(1, 0, 0),
		PEngineName:        "No Engine\x00",
		EngineVersion:      vk.MakeVersion(1, 0, 0),
	}

	extensions := r.window.GetRequiredInstanceExtensions()
	instanceInfo := vk.InstanceCreateInfo{
		SType:                   vk.StructureTypeInstanceCreateInfo,
		PApplicationInfo:        &appInfo,
		EnabledExtensionCount:   uint32(len(extensions)),
		PpEnabledExtensionNames: extensions,
	}

	var instance vk.Instance
	if res := vk.CreateInstance(&instanceInfo, nil, &instance); res != vk.Success {
		return fmt.Errorf("failed to create instance: %v", res)
	}
	r.instance = instance

	// Create Surface
	surfacePtr, err := r.window.CreateWindowSurface(r.instance, nil)
	if err != nil {
		return fmt.Errorf("failed to create surface: %v", err)
	}
	r.surface = vk.SurfaceFromPointer(surfacePtr)

	// Select Physical Device
	if err := r.pickPhysicalDevice(); err != nil {
		return err
	}

	// Create Logical Device
	if err := r.createLogicalDevice(); err != nil {
		return err
	}

	// Create Swapchain
	if err := r.createSwapchain(); err != nil {
		return err
	}

	// Create Render Pass
	if err := r.createRenderPass(); err != nil {
		return err
	}

	// Create Command Pool
	if err := r.createCommandPool(); err != nil {
		return err
	}

	return nil
}

func (r *VulkanRenderer) pickPhysicalDevice() error {
	var deviceCount uint32
	vk.EnumeratePhysicalDevices(r.instance, &deviceCount, nil)
	if deviceCount == 0 {
		return fmt.Errorf("no GPU with Vulkan support")
	}

	devices := make([]vk.PhysicalDevice, deviceCount)
	vk.EnumeratePhysicalDevices(r.instance, &deviceCount, devices)

	for _, device := range devices {
		if r.isDeviceSuitable(device) {
			r.physicalDevice = device
			return nil
		}
	}

	return fmt.Errorf("no suitable GPU found")
}

func (r *VulkanRenderer) isDeviceSuitable(device vk.PhysicalDevice) bool {
	var queueFamilyCount uint32
	vk.GetPhysicalDeviceQueueFamilyProperties(device, &queueFamilyCount, nil)
	queueFamilies := make([]vk.QueueFamilyProperties, queueFamilyCount)
	vk.GetPhysicalDeviceQueueFamilyProperties(device, &queueFamilyCount, queueFamilies)

	graphicsIdx := -1
	presentIdx := -1

	for i, qf := range queueFamilies {
		if qf.QueueFlags&vk.QueueFlags(vk.QueueGraphicsBit) != 0 {
			graphicsIdx = i
		}

		var presentSupport vk.Bool32
		vk.GetPhysicalDeviceSurfaceSupport(device, uint32(i), r.surface, &presentSupport)
		if presentSupport.B() {
			presentIdx = i
		}

		if graphicsIdx >= 0 && presentIdx >= 0 {
			r.graphicsFamily = uint32(graphicsIdx)
			r.presentFamily = uint32(presentIdx)
			return true
		}
	}

	return false
}

func (r *VulkanRenderer) createLogicalDevice() error {
	uniqueFamilies := make(map[uint32]bool)
	uniqueFamilies[r.graphicsFamily] = true
	uniqueFamilies[r.presentFamily] = true

	var queueCreateInfos []vk.DeviceQueueCreateInfo
	queuePriority := []float32{1.0}

	for family := range uniqueFamilies {
		queueCreateInfos = append(queueCreateInfos, vk.DeviceQueueCreateInfo{
			SType:            vk.StructureTypeDeviceQueueCreateInfo,
			QueueFamilyIndex: family,
			QueueCount:       1,
			PQueuePriorities: queuePriority,
		})
	}

	features := []vk.PhysicalDeviceFeatures{{}}

	deviceInfo := vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueCreateInfos)),
		PQueueCreateInfos:       queueCreateInfos,
		EnabledExtensionCount:   1,
		PpEnabledExtensionNames: []string{"VK_KHR_swapchain\x00"},
		PEnabledFeatures:        features,
	}

	var device vk.Device
	if res := vk.CreateDevice(r.physicalDevice, &deviceInfo, nil, &device); res != vk.Success {
		return fmt.Errorf("failed to create device: %v", res)
	}
	r.device = device

	var gQueue, pQueue vk.Queue
	vk.GetDeviceQueue(r.device, r.graphicsFamily, 0, &gQueue)
	vk.GetDeviceQueue(r.device, r.presentFamily, 0, &pQueue)
	r.graphicsQueue = gQueue
	r.presentQueue = pQueue

	return nil
}

func (r *VulkanRenderer) createSwapchain() error {
	var caps vk.SurfaceCapabilities
	vk.GetPhysicalDeviceSurfaceCapabilities(r.physicalDevice, r.surface, &caps)

	r.swapchainFormat = vk.FormatB8g8r8a8Srgb
	r.swapchainExtent = caps.CurrentExtent

	swapchainInfo := vk.SwapchainCreateInfo{
		SType:            vk.StructureTypeSwapchainCreateInfo,
		Surface:          r.surface,
		MinImageCount:    caps.MinImageCount + 1,
		ImageFormat:      r.swapchainFormat,
		ImageColorSpace:  vk.ColorSpaceSrgbNonlinear,
		ImageExtent:      r.swapchainExtent,
		ImageArrayLayers: 1,
		ImageUsage:       vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit),
		PreTransform:     caps.CurrentTransform,
		CompositeAlpha:   vk.CompositeAlphaOpaqueBit,
		PresentMode:      vk.PresentModeFifo,
		Clipped:          vk.True,
	}

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

	var imageCount uint32
	vk.GetSwapchainImages(r.device, r.swapchain, &imageCount, nil)
	r.images = make([]vk.Image, imageCount)
	vk.GetSwapchainImages(r.device, r.swapchain, &imageCount, r.images)

	r.imageViews = make([]vk.ImageView, len(r.images))
	for i, img := range r.images {
		viewInfo := vk.ImageViewCreateInfo{
			SType:    vk.StructureTypeImageViewCreateInfo,
			Image:    img,
			ViewType: vk.ImageViewType2d,
			Format:   r.swapchainFormat,
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
	colorAttachment := vk.AttachmentDescription{
		Format:         r.swapchainFormat,
		Samples:        vk.SampleCount1Bit,
		LoadOp:         vk.AttachmentLoadOpClear,
		StoreOp:        vk.AttachmentStoreOpStore,
		StencilLoadOp:  vk.AttachmentLoadOpDontCare,
		StencilStoreOp: vk.AttachmentStoreOpDontCare,
		InitialLayout:  vk.ImageLayoutUndefined,
		FinalLayout:    vk.ImageLayoutPresentSrc,
	}

	depthAttachment := vk.AttachmentDescription{
		Format:         vk.FormatD32Sfloat,
		Samples:        vk.SampleCount1Bit,
		LoadOp:         vk.AttachmentLoadOpClear,
		StoreOp:        vk.AttachmentStoreOpDontCare,
		StencilLoadOp:  vk.AttachmentLoadOpDontCare,
		StencilStoreOp: vk.AttachmentStoreOpDontCare,
		InitialLayout:  vk.ImageLayoutUndefined,
		FinalLayout:    vk.ImageLayoutDepthStencilAttachmentOptimal,
	}

	colorAttachmentRef := vk.AttachmentReference{
		Attachment: 0,
		Layout:     vk.ImageLayoutColorAttachmentOptimal,
	}

	depthAttachmentRef := vk.AttachmentReference{
		Attachment: 1,
		Layout:     vk.ImageLayoutDepthStencilAttachmentOptimal,
	}

	subpass := vk.SubpassDescription{
		PipelineBindPoint:       vk.PipelineBindPointGraphics,
		ColorAttachmentCount:    1,
		PColorAttachments:       []vk.AttachmentReference{colorAttachmentRef},
		PDepthStencilAttachment: &depthAttachmentRef,
	}

	dependency := vk.SubpassDependency{
		SrcSubpass:    vk.SubpassExternal,
		DstSubpass:    0,
		SrcStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit | vk.PipelineStageEarlyFragmentTestsBit),
		SrcAccessMask: 0,
		DstStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit | vk.PipelineStageEarlyFragmentTestsBit),
		DstAccessMask: vk.AccessFlags(vk.AccessColorAttachmentWriteBit | vk.AccessDepthStencilAttachmentWriteBit),
	}

	renderPassInfo := vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		AttachmentCount: 2,
		PAttachments:    []vk.AttachmentDescription{colorAttachment, depthAttachment},
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

func (r *VulkanRenderer) createDescriptorSetLayout() error {
	uboLayoutBinding := vk.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vk.DescriptorTypeUniformBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageVertexBit),
	}

	layoutInfo := vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: 1,
		PBindings:    []vk.DescriptorSetLayoutBinding{uboLayoutBinding},
	}

	var layout vk.DescriptorSetLayout
	if res := vk.CreateDescriptorSetLayout(r.device, &layoutInfo, nil, &layout); res != vk.Success {
		return fmt.Errorf("failed to create descriptor set layout: %v", res)
	}
	r.descriptorSetLayout = layout
	return nil
}

func (r *VulkanRenderer) createDepthResources() error {
	// Stub implementation - see completion guide for full version
	depthFormat := vk.FormatD32Sfloat

	imageInfo := vk.ImageCreateInfo{
		SType:     vk.StructureTypeImageCreateInfo,
		ImageType: vk.ImageType2d,
		Extent: vk.Extent3D{
			Width:  r.swapchainExtent.Width,
			Height: r.swapchainExtent.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Format:        depthFormat,
		Tiling:        vk.ImageTilingOptimal,
		InitialLayout: vk.ImageLayoutUndefined,
		Usage:         vk.ImageUsageFlags(vk.ImageUsageDepthStencilAttachmentBit),
		SharingMode:   vk.SharingModeExclusive,
		Samples:       vk.SampleCount1Bit,
	}

	var image vk.Image
	if res := vk.CreateImage(r.device, &imageInfo, nil, &image); res != vk.Success {
		return fmt.Errorf("failed to create depth image: %v", res)
	}
	r.depthImage = image

	// Allocate memory
	var memReqs vk.MemoryRequirements
	vk.GetImageMemoryRequirements(r.device, r.depthImage, &memReqs)
	memReqs.Deref()

	allocInfo := vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: r.findMemoryType(memReqs.MemoryTypeBits, vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit)),
	}

	var memory vk.DeviceMemory
	if res := vk.AllocateMemory(r.device, &allocInfo, nil, &memory); res != vk.Success {
		return fmt.Errorf("failed to allocate depth image memory: %v", res)
	}
	r.depthImageMemory = memory

	vk.BindImageMemory(r.device, r.depthImage, r.depthImageMemory, 0)

	// Create image view
	viewInfo := vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		Image:    r.depthImage,
		ViewType: vk.ImageViewType2d,
		Format:   depthFormat,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     vk.ImageAspectFlags(vk.ImageAspectDepthBit),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}

	var view vk.ImageView
	if res := vk.CreateImageView(r.device, &viewInfo, nil, &view); res != vk.Success {
		return fmt.Errorf("failed to create depth image view: %v", res)
	}
	r.depthImageView = view

	return nil
}

func (r *VulkanRenderer) createDescriptorSets() error {
	// Create descriptor pool
	poolSize := vk.DescriptorPoolSize{
		Type:            vk.DescriptorTypeUniformBuffer,
		DescriptorCount: 1,
	}

	poolInfo := vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		PoolSizeCount: 1,
		PPoolSizes:    []vk.DescriptorPoolSize{poolSize},
		MaxSets:       1,
	}

	var pool vk.DescriptorPool
	if res := vk.CreateDescriptorPool(r.device, &poolInfo, nil, &pool); res != vk.Success {
		return fmt.Errorf("failed to create descriptor pool: %v", res)
	}
	r.descriptorPool = pool

	// Allocate descriptor set
	allocInfo := vk.DescriptorSetAllocateInfo{
		SType:              vk.StructureTypeDescriptorSetAllocateInfo,
		DescriptorPool:     r.descriptorPool,
		DescriptorSetCount: 1,
		PSetLayouts:        []vk.DescriptorSetLayout{r.descriptorSetLayout},
	}

	descriptorSets := make([]vk.DescriptorSet, 1)
	if res := vk.AllocateDescriptorSets(r.device, &allocInfo, &descriptorSets[0]); res != vk.Success {
		return fmt.Errorf("failed to allocate descriptor sets: %v", res)
	}
	r.descriptorSet = descriptorSets[0]

	// Update descriptor set
	bufferInfo := vk.DescriptorBufferInfo{
		Buffer: r.uniformBuffer,
		Offset: 0,
		Range:  vk.DeviceSize(unsafe.Sizeof(UniformBufferObject{})),
	}

	descriptorWrite := vk.WriteDescriptorSet{
		SType:           vk.StructureTypeWriteDescriptorSet,
		DstSet:          r.descriptorSet,
		DstBinding:      0,
		DstArrayElement: 0,
		DescriptorType:  vk.DescriptorTypeUniformBuffer,
		DescriptorCount: 1,
		PBufferInfo:     []vk.DescriptorBufferInfo{bufferInfo},
	}

	vk.UpdateDescriptorSets(r.device, 1, []vk.WriteDescriptorSet{descriptorWrite}, 0, nil)
	return nil
}

func (r *VulkanRenderer) createGraphicsPipeline() error {
	// Load shaders
	vertShaderModule, err := r.loadShaderModule("shaders/compiled/vert.spv")
	if err != nil {
		return err
	}
	defer vk.DestroyShaderModule(r.device, vertShaderModule, nil)

	fragShaderModule, err := r.loadShaderModule("shaders/compiled/frag.spv")
	if err != nil {
		return err
	}
	defer vk.DestroyShaderModule(r.device, fragShaderModule, nil)

	// Shader stages
	vertShaderStageInfo := vk.PipelineShaderStageCreateInfo{
		SType:  vk.StructureTypePipelineShaderStageCreateInfo,
		Stage:  vk.ShaderStageVertexBit,
		Module: vertShaderModule,
		PName:  "main\x00",
	}

	fragShaderStageInfo := vk.PipelineShaderStageCreateInfo{
		SType:  vk.StructureTypePipelineShaderStageCreateInfo,
		Stage:  vk.ShaderStageFragmentBit,
		Module: fragShaderModule,
		PName:  "main\x00",
	}

	shaderStages := []vk.PipelineShaderStageCreateInfo{
		vertShaderStageInfo,
		fragShaderStageInfo,
	}

	// Vertex input description
	bindingDescription := vk.VertexInputBindingDescription{
		Binding:   0,
		Stride:    uint32(unsafe.Sizeof(VulkanVertex{})),
		InputRate: vk.VertexInputRateVertex,
	}

	attributeDescriptions := []vk.VertexInputAttributeDescription{
		// Position
		{
			Binding:  0,
			Location: 0,
			Format:   vk.FormatR32g32b32Sfloat,
			Offset:   0,
		},
		// Color
		{
			Binding:  0,
			Location: 1,
			Format:   vk.FormatR32g32b32Sfloat,
			Offset:   uint32(unsafe.Offsetof(VulkanVertex{}.Color)),
		},
	}

	vertexInputInfo := vk.PipelineVertexInputStateCreateInfo{
		SType:                           vk.StructureTypePipelineVertexInputStateCreateInfo,
		VertexBindingDescriptionCount:   1,
		PVertexBindingDescriptions:      []vk.VertexInputBindingDescription{bindingDescription},
		VertexAttributeDescriptionCount: uint32(len(attributeDescriptions)),
		PVertexAttributeDescriptions:    attributeDescriptions,
	}

	// Input assembly
	inputAssembly := vk.PipelineInputAssemblyStateCreateInfo{
		SType:                  vk.StructureTypePipelineInputAssemblyStateCreateInfo,
		Topology:               vk.PrimitiveTopologyTriangleList,
		PrimitiveRestartEnable: vk.False,
	}

	// Viewport and scissor
	viewport := vk.Viewport{
		X:        0.0,
		Y:        0.0,
		Width:    float32(r.swapchainExtent.Width),
		Height:   float32(r.swapchainExtent.Height),
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}

	scissor := vk.Rect2D{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: r.swapchainExtent,
	}

	viewportState := vk.PipelineViewportStateCreateInfo{
		SType:         vk.StructureTypePipelineViewportStateCreateInfo,
		ViewportCount: 1,
		PViewports:    []vk.Viewport{viewport},
		ScissorCount:  1,
		PScissors:     []vk.Rect2D{scissor},
	}

	// Rasterizer
	rasterizer := vk.PipelineRasterizationStateCreateInfo{
		SType:                   vk.StructureTypePipelineRasterizationStateCreateInfo,
		DepthClampEnable:        vk.False,
		RasterizerDiscardEnable: vk.False,
		PolygonMode:             vk.PolygonModeFill,
		LineWidth:               1.0,
		CullMode:                vk.CullModeFlags(vk.CullModeBackBit),
		FrontFace:               vk.FrontFaceCounterClockwise,
		DepthBiasEnable:         vk.False,
	}

	// Multisampling
	multisampling := vk.PipelineMultisampleStateCreateInfo{
		SType:                vk.StructureTypePipelineMultisampleStateCreateInfo,
		SampleShadingEnable:  vk.False,
		RasterizationSamples: vk.SampleCount1Bit,
	}

	// Depth stencil
	depthStencil := vk.PipelineDepthStencilStateCreateInfo{
		SType:                 vk.StructureTypePipelineDepthStencilStateCreateInfo,
		DepthTestEnable:       vk.True,
		DepthWriteEnable:      vk.True,
		DepthCompareOp:        vk.CompareOpLess,
		DepthBoundsTestEnable: vk.False,
		StencilTestEnable:     vk.False,
	}

	// Color blending
	colorBlendAttachment := vk.PipelineColorBlendAttachmentState{
		ColorWriteMask: vk.ColorComponentFlags(
			vk.ColorComponentRBit | vk.ColorComponentGBit |
				vk.ColorComponentBBit | vk.ColorComponentABit,
		),
		BlendEnable: vk.False,
	}

	colorBlending := vk.PipelineColorBlendStateCreateInfo{
		SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
		LogicOpEnable:   vk.False,
		AttachmentCount: 1,
		PAttachments:    []vk.PipelineColorBlendAttachmentState{colorBlendAttachment},
	}

	// Create descriptor set layout
	if err := r.createDescriptorSetLayout(); err != nil {
		return err
	}

	// Pipeline layout
	pipelineLayoutInfo := vk.PipelineLayoutCreateInfo{
		SType:          vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount: 1,
		PSetLayouts:    []vk.DescriptorSetLayout{r.descriptorSetLayout},
	}

	var pipelineLayout vk.PipelineLayout
	if res := vk.CreatePipelineLayout(r.device, &pipelineLayoutInfo, nil, &pipelineLayout); res != vk.Success {
		return fmt.Errorf("failed to create pipeline layout: %v", res)
	}
	r.pipelineLayout = pipelineLayout

	// Create graphics pipeline
	pipelineInfo := vk.GraphicsPipelineCreateInfo{
		SType:               vk.StructureTypeGraphicsPipelineCreateInfo,
		StageCount:          2,
		PStages:             shaderStages,
		PVertexInputState:   &vertexInputInfo,
		PInputAssemblyState: &inputAssembly,
		PViewportState:      &viewportState,
		PRasterizationState: &rasterizer,
		PMultisampleState:   &multisampling,
		PDepthStencilState:  &depthStencil,
		PColorBlendState:    &colorBlending,
		Layout:              pipelineLayout,
		RenderPass:          r.renderPass,
		Subpass:             0,
	}

	pipelines := make([]vk.Pipeline, 1)
	if res := vk.CreateGraphicsPipelines(r.device, vk.NullPipelineCache, 1, []vk.GraphicsPipelineCreateInfo{pipelineInfo}, nil, pipelines); res != vk.Success {
		return fmt.Errorf("failed to create graphics pipeline: %v", res)
	}
	r.graphicsPipeline = pipelines[0]

	return nil
}

func (r *VulkanRenderer) createShaderModuleStub(code []uint32) vk.ShaderModule {
	// Stub - requires actual SPIR-V bytecode
	return vk.ShaderModule(vk.NullHandle)
}

func (r *VulkanRenderer) recordCommandBuffer(commandBuffer vk.CommandBuffer, imageIndex uint32) error {
	beginInfo := vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
	}

	if res := vk.BeginCommandBuffer(commandBuffer, &beginInfo); res != vk.Success {
		return fmt.Errorf("failed to begin command buffer: %v", res)
	}

	// Clear values
	clearValues := []vk.ClearValue{
		vk.NewClearValue([]float32{0.0, 0.0, 0.0, 1.0}),
		vk.NewClearDepthStencil(1.0, 0),
	}

	renderPassInfo := vk.RenderPassBeginInfo{
		SType:       vk.StructureTypeRenderPassBeginInfo,
		RenderPass:  r.renderPass,
		Framebuffer: r.framebuffers[imageIndex],
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: r.swapchainExtent,
		},
		ClearValueCount: 2,
		PClearValues:    clearValues,
	}

	vk.CmdBeginRenderPass(commandBuffer, &renderPassInfo, vk.SubpassContentsInline)
	vk.CmdBindPipeline(commandBuffer, vk.PipelineBindPointGraphics, r.graphicsPipeline)

	// Bind vertex buffer
	vertexBuffers := []vk.Buffer{r.vertexBuffer}
	offsets := []vk.DeviceSize{0}
	vk.CmdBindVertexBuffers(commandBuffer, 0, 1, vertexBuffers, offsets)

	// Bind descriptor sets
	vk.CmdBindDescriptorSets(commandBuffer, vk.PipelineBindPointGraphics, r.pipelineLayout, 0, 1, []vk.DescriptorSet{r.descriptorSet}, 0, nil)

	// Draw
	vertexCount := uint32(len(r.currentVertices))
	if vertexCount > 0 {
		vk.CmdDraw(commandBuffer, vertexCount, 1, 0, 0)
	}

	vk.CmdEndRenderPass(commandBuffer)

	if res := vk.EndCommandBuffer(commandBuffer); res != vk.Success {
		return fmt.Errorf("failed to record command buffer: %v", res)
	}

	return nil
}

func (r *VulkanRenderer) RenderScene(scene *Scene) {
	if !r.initialized {
		return
	}

	// Clear vertex buffer
	r.currentVertices = r.currentVertices[:0]

	// Collect all geometry
	nodes := scene.GetRenderableNodes()
	for _, node := range nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		r.addNodeGeometry(node, worldMatrix, scene.Camera)
	}

	// Update vertex buffer
	r.updateVertexBuffer()

	// Update uniform buffer with camera matrices
	r.updateUniformBuffer(scene.Camera)
}

func (r *VulkanRenderer) addNodeGeometry(node *SceneNode, worldMatrix Matrix4x4, camera *Camera) {
	switch obj := node.Object.(type) {
	case *Triangle:
		r.addTriangleVertices(obj, worldMatrix, camera)
	case *Mesh:
		r.addMeshVertices(obj, worldMatrix, camera)
	case *Quad:
		r.addQuadVertices(obj, worldMatrix, camera)
	}
}

func (r *VulkanRenderer) addTriangleVertices(tri *Triangle, worldMatrix Matrix4x4, camera *Camera) {
	p0 := worldMatrix.TransformPoint(tri.P0)
	p1 := worldMatrix.TransformPoint(tri.P1)
	p2 := worldMatrix.TransformPoint(tri.P2)

	color := tri.Material.DiffuseColor
	c := [3]float32{float32(color.R) / 255.0, float32(color.G) / 255.0, float32(color.B) / 255.0}

	r.currentVertices = append(r.currentVertices,
		VulkanVertex{Pos: [3]float32{float32(p0.X), float32(p0.Y), float32(p0.Z)}, Color: c},
		VulkanVertex{Pos: [3]float32{float32(p1.X), float32(p1.Y), float32(p1.Z)}, Color: c},
		VulkanVertex{Pos: [3]float32{float32(p2.X), float32(p2.Y), float32(p2.Z)}, Color: c},
	)
}

func (r *VulkanRenderer) addMeshVertices(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera) {
	for i := 0; i < len(mesh.Indices); i += 3 {
		if i+2 < len(mesh.Indices) {
			idx0, idx1, idx2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
			if idx0 < len(mesh.Vertices) && idx1 < len(mesh.Vertices) && idx2 < len(mesh.Vertices) {
				tri := NewTriangle(mesh.Vertices[idx0], mesh.Vertices[idx1], mesh.Vertices[idx2], 'o')
				tri.Material = mesh.Material
				r.addTriangleVertices(tri, worldMatrix, camera)
			}
		}
	}
}

func (r *VulkanRenderer) addQuadVertices(quad *Quad, worldMatrix Matrix4x4, camera *Camera) {
	triangles := ConvertQuadToTriangles(quad)
	for _, tri := range triangles {
		r.addTriangleVertices(tri, worldMatrix, camera)
	}
}

func (r *VulkanRenderer) updateVertexBuffer() {
	// Map memory and copy vertices
	if len(r.currentVertices) == 0 {
		return
	}

	var data unsafe.Pointer
	vk.MapMemory(r.device, r.vertexBufferMemory, 0, vk.DeviceSize(len(r.currentVertices)*int(unsafe.Sizeof(VulkanVertex{}))), 0, &data)

	// Copy vertex data
	vk.Memcopy(data, r.sliceToBytes(r.currentVertices))

	vk.UnmapMemory(r.device, r.vertexBufferMemory)
}

func (r *VulkanRenderer) updateUniformBuffer(camera *Camera) {
	if camera == nil {
		return
	}

	// Build matrices
	ubo := UniformBufferObject{}

	// Model matrix (identity for now)
	ubo.Model = r.matrixToArray(IdentityMatrix())

	// View matrix (inverse of camera transform)
	viewMatrix := camera.Transform.GetInverseMatrix()
	ubo.View = r.matrixToArray(viewMatrix)

	// Projection matrix
	projMatrix := r.buildProjectionMatrix(camera)
	ubo.Proj = r.matrixToArray(projMatrix)

	// Update uniform buffer
	var data unsafe.Pointer
	vk.MapMemory(r.device, r.uniformBufferMemory, 0, vk.DeviceSize(unsafe.Sizeof(ubo)), 0, &data)
	vk.Memcopy(data, r.structToBytes(&ubo))
	vk.UnmapMemory(r.device, r.uniformBufferMemory)
}

func (r *VulkanRenderer) buildProjectionMatrix(camera *Camera) Matrix4x4 {
	fovY := camera.FOV.Y * math.Pi / 180.0
	aspect := float64(r.width) / float64(r.height)
	near := camera.Near
	far := camera.Far

	f := 1.0 / math.Tan(fovY/2.0)

	return Matrix4x4{M: [16]float64{
		f / aspect, 0, 0, 0,
		0, -f, 0, 0, // Flip Y for Vulkan
		0, 0, far / (near - far), -1,
		0, 0, (near * far) / (near - far), 0,
	}}
}

func (r *VulkanRenderer) matrixToArray(m Matrix4x4) [16]float32 {
	var result [16]float32
	for i := 0; i < 16; i++ {
		result[i] = float32(m.M[i])
	}
	return result
}

func (r *VulkanRenderer) sliceToBytes(slice []VulkanVertex) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&slice[0])), len(slice)*int(unsafe.Sizeof(VulkanVertex{})))
}

func (r *VulkanRenderer) structToBytes(s *UniformBufferObject) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(s)), unsafe.Sizeof(*s))
}

// Stub implementations for remaining interface methods
func (r *VulkanRenderer) createVertexBuffer() error {
	// Create large vertex buffer
	bufferSize := vk.DeviceSize(r.maxVertices * int(unsafe.Sizeof(VulkanVertex{})))

	bufferInfo := vk.BufferCreateInfo{
		SType:       vk.StructureTypeBufferCreateInfo,
		Size:        bufferSize,
		Usage:       vk.BufferUsageFlags(vk.BufferUsageVertexBufferBit),
		SharingMode: vk.SharingModeExclusive,
	}

	var buffer vk.Buffer
	if res := vk.CreateBuffer(r.device, &bufferInfo, nil, &buffer); res != vk.Success {
		return fmt.Errorf("failed to create vertex buffer: %v", res)
	}
	r.vertexBuffer = buffer

	var memReqs vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(r.device, r.vertexBuffer, &memReqs)
	memReqs.Deref()

	allocInfo := vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: r.findMemoryType(memReqs.MemoryTypeBits, vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)),
	}

	var memory vk.DeviceMemory
	if res := vk.AllocateMemory(r.device, &allocInfo, nil, &memory); res != vk.Success {
		return fmt.Errorf("failed to allocate vertex buffer memory: %v", res)
	}
	r.vertexBufferMemory = memory

	vk.BindBufferMemory(r.device, r.vertexBuffer, r.vertexBufferMemory, 0)
	return nil
}

func (r *VulkanRenderer) findMemoryType(typeFilter uint32, properties vk.MemoryPropertyFlags) uint32 {
	var memProperties vk.PhysicalDeviceMemoryProperties
	vk.GetPhysicalDeviceMemoryProperties(r.physicalDevice, &memProperties)
	memProperties.Deref()

	for i := uint32(0); i < memProperties.MemoryTypeCount; i++ {
		if (typeFilter&(1<<i)) != 0 && (memProperties.MemoryTypes[i].PropertyFlags&properties) == properties {
			return i
		}
	}
	return 0
}

func (r *VulkanRenderer) createUniformBuffer() error  { return nil }
func (r *VulkanRenderer) createFramebuffers() error   { return nil }
func (r *VulkanRenderer) createCommandBuffers() error { return nil }
func (r *VulkanRenderer) createCommandPool() error    { return nil }
func (r *VulkanRenderer) createSyncObjects() error    { return nil }

func (r *VulkanRenderer) Shutdown() {
	if !r.initialized {
		return
	}
	vk.DeviceWaitIdle(r.device)

	// Cleanup all Vulkan resources
	vk.DestroyBuffer(r.device, r.vertexBuffer, nil)
	vk.FreeMemory(r.device, r.vertexBufferMemory, nil)

	for _, fb := range r.framebuffers {
		vk.DestroyFramebuffer(r.device, fb, nil)
	}

	vk.DestroyPipeline(r.device, r.graphicsPipeline, nil)
	vk.DestroyPipelineLayout(r.device, r.pipelineLayout, nil)
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

func (r *VulkanRenderer) loadShaderModule(filename string) (vk.ShaderModule, error) {
	// Read SPIR-V bytecode
	code, err := os.ReadFile(filename)
	if err != nil {
		return vk.NullShaderModule, fmt.Errorf("failed to read shader: %v", err)
	}

	// Convert to uint32 slice
	codeSize := len(code)
	codeUint32 := make([]uint32, codeSize/4)
	for i := 0; i < len(codeUint32); i++ {
		codeUint32[i] = uint32(code[i*4]) |
			uint32(code[i*4+1])<<8 |
			uint32(code[i*4+2])<<16 |
			uint32(code[i*4+3])<<24
	}

	createInfo := vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint(codeSize),
		PCode:    codeUint32,
	}

	var shaderModule vk.ShaderModule
	if res := vk.CreateShaderModule(r.device, &createInfo, nil, &shaderModule); res != vk.Success {
		return vk.NullShaderModule, fmt.Errorf("failed to create shader module: %v", res)
	}

	return shaderModule, nil
}

func (r *VulkanRenderer) Present() {
	if !r.initialized {
		return
	}

	// Wait for previous frame
	vk.WaitForFences(r.device, 1, []vk.Fence{r.inFlightFence}, vk.True, ^uint64(0))

	// Acquire image
	var imageIndex uint32
	res := vk.AcquireNextImage(r.device, r.swapchain, ^uint64(0), r.imageAvailableSem, vk.NullFence, &imageIndex)
	if res == vk.ErrorOutOfDate || res == vk.Suboptimal {
		// Recreate swapchain
		return
	}
	if res != vk.Success {
		return
	}

	vk.ResetFences(r.device, 1, []vk.Fence{r.inFlightFence})

	// Record command buffer
	vk.ResetCommandBuffer(r.commandBuffers[imageIndex], 0)
	r.recordCommandBuffer(r.commandBuffers[imageIndex], imageIndex)

	// Submit
	waitStages := []vk.PipelineStageFlags{vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit)}
	submitInfo := vk.SubmitInfo{
		SType:                vk.StructureTypeSubmitInfo,
		WaitSemaphoreCount:   1,
		PWaitSemaphores:      []vk.Semaphore{r.imageAvailableSem},
		PWaitDstStageMask:    waitStages,
		CommandBufferCount:   1,
		PCommandBuffers:      []vk.CommandBuffer{r.commandBuffers[imageIndex]},
		SignalSemaphoreCount: 1,
		PSignalSemaphores:    []vk.Semaphore{r.renderFinishedSem},
	}

	if res := vk.QueueSubmit(r.graphicsQueue, 1, []vk.SubmitInfo{submitInfo}, r.inFlightFence); res != vk.Success {
		return
	}

	// Present
	presentInfo := vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{r.renderFinishedSem},
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{r.swapchain},
		PImageIndices:      []uint32{imageIndex},
	}

	vk.QueuePresent(r.presentQueue, &presentInfo)
	r.frameCount++
}

func (r *VulkanRenderer) BeginFrame() {
	if r.initialized {
		glfw.PollEvents()
	}
}

func (r *VulkanRenderer) EndFrame()                                               {}
func (r *VulkanRenderer) RenderTriangle(tri *Triangle, wm Matrix4x4, cam *Camera) {}
func (r *VulkanRenderer) RenderLine(line *Line, wm Matrix4x4, cam *Camera)        {}
func (r *VulkanRenderer) RenderPoint(point *Point, wm Matrix4x4, cam *Camera)     {}
func (r *VulkanRenderer) RenderMesh(mesh *Mesh, wm Matrix4x4, cam *Camera)        {}
func (r *VulkanRenderer) SetLightingSystem(ls *LightingSystem)                    { r.LightingSystem = ls }
func (r *VulkanRenderer) SetCamera(camera *Camera)                                { r.Camera = camera }
func (r *VulkanRenderer) GetDimensions() (int, int)                               { return r.width, r.height }
func (r *VulkanRenderer) SetUseColor(useColor bool)                               { r.UseColor = useColor }
func (r *VulkanRenderer) SetShowDebugInfo(show bool)                              { r.ShowDebugInfo = show }
func (r *VulkanRenderer) SetClipBounds(minX, minY, maxX, maxY int)                {}
func (r *VulkanRenderer) GetRenderContext() *RenderContext                        { return r.renderContext }
