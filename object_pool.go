package main

import "sync"

// TrianglePool manages a pool of Triangle objects to reduce allocations
type TrianglePool struct {
	pool  []*Triangle
	index int
	mutex sync.Mutex
}

// NewTrianglePool creates a triangle pool with initial capacity
func NewTrianglePool(capacity int) *TrianglePool {
	pool := make([]*Triangle, capacity)
	for i := 0; i < capacity; i++ {
		pool[i] = &Triangle{}
	}
	return &TrianglePool{
		pool:  pool,
		index: 0,
	}
}

// Get retrieves a triangle from the pool
func (p *TrianglePool) Get() *Triangle {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.index < len(p.pool) {
		tri := p.pool[p.index]
		p.index++
		return tri
	}

	// Pool exhausted, allocate new
	return &Triangle{}
}

// Reset resets the pool for reuse
func (p *TrianglePool) Reset() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.index = 0
}

// QuadPool manages a pool of Quad objects
type QuadPool struct {
	pool  []*Quad
	index int
	mutex sync.Mutex
}

func NewQuadPool(capacity int) *QuadPool {
	pool := make([]*Quad, capacity)
	for i := 0; i < capacity; i++ {
		pool[i] = &Quad{}
	}
	return &QuadPool{
		pool:  pool,
		index: 0,
	}
}

func (p *QuadPool) Get() *Quad {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.index < len(p.pool) {
		quad := p.pool[p.index]
		p.index++
		return quad
	}
	return &Quad{}
}

func (p *QuadPool) Reset() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.index = 0
}

// PointPool manages Point allocations
type PointPool struct {
	pool  []Point
	index int
	mutex sync.Mutex
}

func NewPointPool(capacity int) *PointPool {
	return &PointPool{
		pool:  make([]Point, capacity),
		index: 0,
	}
}

func (p *PointPool) Get() *Point {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.index < len(p.pool) {
		point := &p.pool[p.index]
		p.index++
		return point
	}
	return &Point{}
}

func (p *PointPool) Reset() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.index = 0
}

// MatrixPool for temporary matrix calculations
type MatrixPool struct {
	pool  []Matrix4x4
	index int
	mutex sync.Mutex
}

func NewMatrixPool(capacity int) *MatrixPool {
	return &MatrixPool{
		pool:  make([]Matrix4x4, capacity),
		index: 0,
	}
}

func (p *MatrixPool) Get() *Matrix4x4 {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.index < len(p.pool) {
		mat := &p.pool[p.index]
		p.index++
		return mat
	}
	return &Matrix4x4{}
}

func (p *MatrixPool) Reset() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.index = 0
}

// RenderPools holds all pools for rendering
type RenderPools struct {
	Triangles *TrianglePool
	Quads     *QuadPool
	Points    *PointPool
	Matrices  *MatrixPool
}

// NewRenderPools creates all render pools
func NewRenderPools(triangleCapacity, quadCapacity, pointCapacity, matrixCapacity int) *RenderPools {
	return &RenderPools{
		Triangles: NewTrianglePool(triangleCapacity),
		Quads:     NewQuadPool(quadCapacity),
		Points:    NewPointPool(pointCapacity),
		Matrices:  NewMatrixPool(matrixCapacity),
	}
}

// ResetAll resets all pools
func (rp *RenderPools) ResetAll() {
	rp.Triangles.Reset()
	rp.Quads.Reset()
	rp.Points.Reset()
	rp.Matrices.Reset()
}

// CopyTriangle copies data into a pooled triangle
func CopyTriangle(dst, src *Triangle) {
	dst.P0 = src.P0
	dst.P1 = src.P1
	dst.P2 = src.P2
	dst.char = src.char
	dst.Material = src.Material
	dst.UseSetNormal = src.UseSetNormal
	if src.Normal != nil {
		if dst.Normal == nil {
			n := *src.Normal
			dst.Normal = &n
		} else {
			*dst.Normal = *src.Normal
		}
	}
}

// CopyQuad copies data into a pooled quad
func CopyQuad(dst, src *Quad) {
	dst.P0 = src.P0
	dst.P1 = src.P1
	dst.P2 = src.P2
	dst.P3 = src.P3
	dst.Material = src.Material
	dst.UseSetNormal = src.UseSetNormal
	if src.Normal != nil {
		if dst.Normal == nil {
			n := *src.Normal
			dst.Normal = &n
		} else {
			*dst.Normal = *src.Normal
		}
	}
}
