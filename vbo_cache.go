package main

import "sync"

// MeshBuffer represents a cached GPU buffer for a mesh
type MeshBuffer struct {
	VBO       uint32 // Vertex Buffer Object (OpenGL/Vulkan handle)
	EBO       uint32 // Element Buffer Object
	VAO       uint32 // Vertex Array Object (OpenGL only)
	VertCount int
	IndCount  int
	Dirty     bool
	LastUsed  uint64 // Frame number
}

// MeshBufferCache caches mesh buffers for reuse
type MeshBufferCache struct {
	buffers map[*Mesh]*MeshBuffer
	mu      sync.RWMutex
	frame   uint64
}

// NewMeshBufferCache creates a new buffer cache
func NewMeshBufferCache() *MeshBufferCache {
	return &MeshBufferCache{
		buffers: make(map[*Mesh]*MeshBuffer),
		frame:   0,
	}
}

// Get retrieves or creates a buffer for a mesh
func (mbc *MeshBufferCache) Get(mesh *Mesh) (*MeshBuffer, bool) {
	mbc.mu.RLock()
	buffer, exists := mbc.buffers[mesh]
	mbc.mu.RUnlock()

	if exists && !buffer.Dirty {
		buffer.LastUsed = mbc.frame
		return buffer, true
	}

	return buffer, false
}

// Set stores a buffer for a mesh
func (mbc *MeshBufferCache) Set(mesh *Mesh, buffer *MeshBuffer) {
	mbc.mu.Lock()
	defer mbc.mu.Unlock()
	buffer.LastUsed = mbc.frame
	mbc.buffers[mesh] = buffer
}

// MarkDirty marks a mesh's buffer as needing update
func (mbc *MeshBufferCache) MarkDirty(mesh *Mesh) {
	mbc.mu.Lock()
	defer mbc.mu.Unlock()
	if buffer, ok := mbc.buffers[mesh]; ok {
		buffer.Dirty = true
	}
}

// Remove removes a buffer from cache
func (mbc *MeshBufferCache) Remove(mesh *Mesh) (*MeshBuffer, bool) {
	mbc.mu.Lock()
	defer mbc.mu.Unlock()
	buffer, ok := mbc.buffers[mesh]
	if ok {
		delete(mbc.buffers, mesh)
	}
	return buffer, ok
}

// NextFrame increments the frame counter
func (mbc *MeshBufferCache) NextFrame() {
	mbc.frame++
}

// CleanUnused removes buffers not used in N frames
func (mbc *MeshBufferCache) CleanUnused(maxAge uint64, cleanup func(*MeshBuffer)) {
	mbc.mu.Lock()
	defer mbc.mu.Unlock()

	toRemove := make([]*Mesh, 0)
	for mesh, buffer := range mbc.buffers {
		if mbc.frame-buffer.LastUsed > maxAge {
			toRemove = append(toRemove, mesh)
			if cleanup != nil {
				cleanup(buffer)
			}
		}
	}

	for _, mesh := range toRemove {
		delete(mbc.buffers, mesh)
	}
}

// Clear removes all cached buffers
func (mbc *MeshBufferCache) Clear(cleanup func(*MeshBuffer)) {
	mbc.mu.Lock()
	defer mbc.mu.Unlock()

	if cleanup != nil {
		for _, buffer := range mbc.buffers {
			cleanup(buffer)
		}
	}

	mbc.buffers = make(map[*Mesh]*MeshBuffer)
}

// GetStats returns cache statistics
func (mbc *MeshBufferCache) GetStats() MeshBufferCacheStats {
	mbc.mu.RLock()
	defer mbc.mu.RUnlock()

	return MeshBufferCacheStats{
		CachedBuffers: len(mbc.buffers),
		CurrentFrame:  mbc.frame,
	}
}

// MeshBufferCacheStats holds cache statistics
type MeshBufferCacheStats struct {
	CachedBuffers int
	CurrentFrame  uint64
}

// TextureCache caches texture GPU handles
type TextureCache struct {
	textures map[*Texture]uint32 // Texture -> GPU handle
	mu       sync.RWMutex
}

// NewTextureCache creates a new texture cache
func NewTextureCache() *TextureCache {
	return &TextureCache{
		textures: make(map[*Texture]uint32),
	}
}

// Get retrieves a cached texture handle
func (tc *TextureCache) Get(tex *Texture) (uint32, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	handle, ok := tc.textures[tex]
	return handle, ok
}

// Set stores a texture handle
func (tc *TextureCache) Set(tex *Texture, handle uint32) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.textures[tex] = handle
}

// Remove removes a texture from cache
func (tc *TextureCache) Remove(tex *Texture) (uint32, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	handle, ok := tc.textures[tex]
	if ok {
		delete(tc.textures, tex)
	}
	return handle, ok
}

// Clear removes all cached textures
func (tc *TextureCache) Clear(cleanup func(uint32)) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if cleanup != nil {
		for _, handle := range tc.textures {
			cleanup(handle)
		}
	}

	tc.textures = make(map[*Texture]uint32)
}

// ShaderCache caches compiled shaders
type ShaderCache struct {
	programs map[string]uint32 // Shader name -> program handle
	mu       sync.RWMutex
}

// NewShaderCache creates a new shader cache
func NewShaderCache() *ShaderCache {
	return &ShaderCache{
		programs: make(map[string]uint32),
	}
}

// Get retrieves a cached shader program
func (sc *ShaderCache) Get(name string) (uint32, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	handle, ok := sc.programs[name]
	return handle, ok
}

// Set stores a shader program
func (sc *ShaderCache) Set(name string, handle uint32) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.programs[name] = handle
}

// Remove removes a shader from cache
func (sc *ShaderCache) Remove(name string) (uint32, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	handle, ok := sc.programs[name]
	if ok {
		delete(sc.programs, name)
	}
	return handle, ok
}

// Clear removes all cached shaders
func (sc *ShaderCache) Clear(cleanup func(uint32)) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if cleanup != nil {
		for _, handle := range sc.programs {
			cleanup(handle)
		}
	}

	sc.programs = make(map[string]uint32)
}

// GPUResourceManager manages all GPU resource caches
type GPUResourceManager struct {
	MeshBuffers *MeshBufferCache
	Textures    *TextureCache
	Shaders     *ShaderCache
}

// NewGPUResourceManager creates a new GPU resource manager
func NewGPUResourceManager() *GPUResourceManager {
	return &GPUResourceManager{
		MeshBuffers: NewMeshBufferCache(),
		Textures:    NewTextureCache(),
		Shaders:     NewShaderCache(),
	}
}

// NextFrame should be called each frame
func (grm *GPUResourceManager) NextFrame() {
	grm.MeshBuffers.NextFrame()
}

// CleanUnusedResources removes old unused resources
func (grm *GPUResourceManager) CleanUnusedResources(
	maxAge uint64,
	meshCleanup func(*MeshBuffer),
) {
	grm.MeshBuffers.CleanUnused(maxAge, meshCleanup)
}

// ClearAll removes all cached resources
func (grm *GPUResourceManager) ClearAll(
	meshCleanup func(*MeshBuffer),
	texCleanup func(uint32),
	shaderCleanup func(uint32),
) {
	grm.MeshBuffers.Clear(meshCleanup)
	grm.Textures.Clear(texCleanup)
	grm.Shaders.Clear(shaderCleanup)
}
