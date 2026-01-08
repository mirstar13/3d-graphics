package main

import (
	"fmt"
	"sync"
)

// AssetManager manages loading and caching of assets
type AssetManager struct {
	meshes    map[string]*Mesh
	textures  map[string]*Texture
	materials map[string]IMaterial
	mu        sync.RWMutex

	// Statistics
	loadedMeshes   int
	loadedTextures int
	cacheHits      int
	cacheMisses    int
}

// NewAssetManager creates a new asset manager
func NewAssetManager() *AssetManager {
	return &AssetManager{
		meshes:    make(map[string]*Mesh),
		textures:  make(map[string]*Texture),
		materials: make(map[string]IMaterial),
	}
}

// LoadMesh loads or retrieves a cached mesh
func (am *AssetManager) LoadMesh(path string) (*Mesh, error) {
	am.mu.RLock()
	if mesh, ok := am.meshes[path]; ok {
		am.cacheHits++
		am.mu.RUnlock()
		return mesh, nil
	}
	am.mu.RUnlock()

	am.cacheMisses++

	// Load the mesh (currently only supports OBJ)
	mesh, err := LoadOBJ(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load mesh %s: %w", path, err)
	}

	am.mu.Lock()
	am.meshes[path] = mesh
	am.loadedMeshes++
	am.mu.Unlock()

	return mesh, nil
}

// LoadMeshAsync loads a mesh asynchronously
func (am *AssetManager) LoadMeshAsync(path string, callback func(*Mesh, error)) {
	go func() {
		mesh, err := am.LoadMesh(path)
		callback(mesh, err)
	}()
}

// LoadTexture loads or retrieves a cached texture
func (am *AssetManager) LoadTexture(path string) (*Texture, error) {
	am.mu.RLock()
	if tex, ok := am.textures[path]; ok {
		am.cacheHits++
		am.mu.RUnlock()
		return tex, nil
	}
	am.mu.RUnlock()

	am.cacheMisses++

	// Load texture from file
	tex, err := LoadTextureFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load texture %s: %w", path, err)
	}

	am.mu.Lock()
	am.textures[path] = tex
	am.loadedTextures++
	am.mu.Unlock()

	return tex, nil
}

// LoadTextureAsync loads a texture asynchronously
func (am *AssetManager) LoadTextureAsync(path string, callback func(*Texture, error)) {
	go func() {
		tex, err := am.LoadTexture(path)
		callback(tex, err)
	}()
}

// RegisterMaterial registers a material with a name
func (am *AssetManager) RegisterMaterial(name string, material IMaterial) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.materials[name] = material
}

// GetMaterial retrieves a registered material
func (am *AssetManager) GetMaterial(name string) (IMaterial, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	mat, ok := am.materials[name]
	return mat, ok
}

// UnloadMesh removes a mesh from cache
func (am *AssetManager) UnloadMesh(path string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	if _, ok := am.meshes[path]; ok {
		delete(am.meshes, path)
		am.loadedMeshes--
	}
}

// UnloadTexture removes a texture from cache
func (am *AssetManager) UnloadTexture(path string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	if _, ok := am.textures[path]; ok {
		delete(am.textures, path)
		am.loadedTextures--
	}
}

// Clear removes all cached assets
func (am *AssetManager) Clear() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.meshes = make(map[string]*Mesh)
	am.textures = make(map[string]*Texture)
	am.materials = make(map[string]IMaterial)
	am.loadedMeshes = 0
	am.loadedTextures = 0
}

// GetStats returns asset manager statistics
func (am *AssetManager) GetStats() AssetManagerStats {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return AssetManagerStats{
		LoadedMeshes:   am.loadedMeshes,
		LoadedTextures: am.loadedTextures,
		CacheHits:      am.cacheHits,
		CacheMisses:    am.cacheMisses,
		CacheHitRate:   float64(am.cacheHits) / float64(am.cacheHits+am.cacheMisses),
	}
}

// AssetManagerStats holds statistics
type AssetManagerStats struct {
	LoadedMeshes   int
	LoadedTextures int
	CacheHits      int
	CacheMisses    int
	CacheHitRate   float64
}

func (s AssetManagerStats) String() string {
	return fmt.Sprintf("Assets: %d meshes, %d textures | Cache: %.1f%% hit rate (%d hits, %d misses)",
		s.LoadedMeshes, s.LoadedTextures, s.CacheHitRate*100, s.CacheHits, s.CacheMisses)
}

// GetCachedMesh retrieves a mesh without loading
func (am *AssetManager) GetCachedMesh(path string) (*Mesh, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	mesh, ok := am.meshes[path]
	return mesh, ok
}

// GetCachedTexture retrieves a texture without loading
func (am *AssetManager) GetCachedTexture(path string) (*Texture, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	tex, ok := am.textures[path]
	return tex, ok
}

// PreloadAssets loads multiple assets at once
func (am *AssetManager) PreloadAssets(meshPaths []string, texturePaths []string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(meshPaths)+len(texturePaths))

	// Load meshes in parallel
	for _, path := range meshPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			_, err := am.LoadMesh(p)
			if err != nil {
				errChan <- err
			}
		}(path)
	}

	// Load textures in parallel
	for _, path := range texturePaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			_, err := am.LoadTexture(p)
			if err != nil {
				errChan <- err
			}
		}(path)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// Global asset manager instance
var globalAssetManager *AssetManager

// GetGlobalAssetManager returns the global asset manager
func GetGlobalAssetManager() *AssetManager {
	if globalAssetManager == nil {
		globalAssetManager = NewAssetManager()
	}
	return globalAssetManager
}
