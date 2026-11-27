package terminal

import (
	"fmt"
	"math"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

type rasterDevice interface {
	visibility(width, height int, clipVertices []float32) ([]uint32, error)
	name() string
	close()
}

type projectedTriangle struct {
	triangle engine.Triangle
	normal   engine.Point
	x, y, z  [3]float64
	area     float64
}

type triangleBatch struct {
	triangles   []projectedTriangle
	vertices    []float32
	unsupported bool
}

func (r *TerminalRenderer) tryOpenGLRaster(nodes []*engine.SceneNode, camera *engine.Camera) bool {
	r.usedOpenGLRaster = r.drawWithOpenGL(nodes, camera)
	return r.usedOpenGLRaster
}

func (r *TerminalRenderer) drawWithOpenGL(nodes []*engine.SceneNode, camera *engine.Camera) bool {
	if r.rasterDevice == nil || r.sampleCount > 1 || camera.Near <= 0 || camera.Far <= camera.Near {
		return false
	}
	// Appended line/point geometry can use the CPU depth test after the GL
	// result. Interleaved primitives require CPU traversal to keep draw order.
	prefix := len(nodes)
	for i, node := range nodes {
		if triangleNode(node) {
			if i >= prefix {
				return false
			}
		} else {
			prefix = min(prefix, i)
		}
	}

	batch := &r.rasterBatch
	batch.unsupported = false
	r.collectTriangles = batch
	defer func() {
		r.collectTriangles = nil
		clear(batch.triangles)
		batch.triangles = batch.triangles[:0]
		batch.vertices = batch.vertices[:0]
	}()
	for _, node := range nodes[:prefix] {
		r.renderNode(node, node.Transform.GetWorldMatrix(), camera)
		if batch.unsupported {
			return false
		}
	}
	r.collectTriangles = nil
	if len(batch.triangles) == 0 {
		return false
	}
	ids, err := r.rasterDevice.visibility(r.Width, r.Height, batch.vertices)
	if err == nil {
		err = validateVisibility(ids, r.Width, r.Height, len(batch.triangles))
	}
	if err != nil {
		r.rasterError = err
		r.rasterDevice.close()
		r.rasterDevice = nil
		return false
	}

	for y := 0; y < r.Height; y++ {
		for x, id := range ids[y*r.Width : (y+1)*r.Width] {
			if id == 0 {
				continue
			}
			p := &batch.triangles[id-1]
			px, py := float64(x)+.5, float64(y)+.5
			a := edge(p.x[1], p.y[1], p.x[2], p.y[2], px, py) / p.area
			b := edge(p.x[2], p.y[2], p.x[0], p.y[0], px, py) / p.area
			c := 1 - a - b
			z := 1 / (a/p.z[0] + b/p.z[1] + c/p.z[2])
			// Screen clipping and coverage run on the device. Use the CPU
			// projection for interpolants and the shared overlay depth buffer.
			if z < camera.Near || z > camera.Far {
				continue
			}
			color := r.shadeTriangleAtWeights(&p.triangle, camera, p.normal, p.z[0], p.z[1], p.z[2], a, b, c)
			r.ZBuffer[y][x] = z
			r.ColorBuffer[y][x] = color
			r.Surface[y][x] = r.resolvedGlyph(color, z)
		}
	}
	for _, node := range nodes[prefix:] {
		r.renderNode(node, node.Transform.GetWorldMatrix(), camera)
	}
	return true
}

func triangleNode(node *engine.SceneNode) bool {
	switch node.Object.(type) {
	case *engine.Triangle, *engine.Quad, *engine.Mesh, *engine.InstancedMesh, *engine.LODGroup, *engine.LODGroupWithTransitions:
		return true
	default:
		return false
	}
}

func (r *TerminalRenderer) collectProjectedTriangle(t *engine.Triangle, camera *engine.Camera, normal engine.Point, x, y, z [3]float64, area float64) {
	batch := r.collectTriangles
	if batch.unsupported || r.lodMask {
		// Preserve complementary screen-door coverage on the CPU.
		batch.unsupported = true
		return
	}
	near, far := camera.Near, camera.Far
	scaleZ, offsetZ := (far+near)/(far-near), -2*far*near/(far-near)
	var clip [12]float32
	for i := range z {
		// Upload row zero at the bottom of the GL surface. Readback has the
		// same row order, so terminal Y increases without a copy or flip.
		j := i * 4
		clip[j] = float32((2*x[i]/float64(r.Width) - 1) * z[i])
		clip[j+1] = float32((2*y[i]/float64(r.Height) - 1) * z[i])
		clip[j+2] = float32(scaleZ*z[i] + offsetZ)
		clip[j+3] = float32(z[i])
	}
	for _, value := range clip {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			batch.unsupported = true
			return
		}
	}
	batch.vertices = append(batch.vertices, clip[:]...)
	batch.triangles = append(batch.triangles, projectedTriangle{triangle: *t, normal: normal, x: x, y: y, z: z, area: area})
}

// UsedOpenGLRaster reports whether the last scene frame used the visibility pass.
func (r *TerminalRenderer) UsedOpenGLRaster() bool { return r.usedOpenGLRaster }

func validateVisibility(ids []uint32, width, height, triangles int) error {
	if len(ids) != width*height {
		return fmt.Errorf("invalid visibility buffer size: %d", len(ids))
	}
	for _, id := range ids {
		if uint64(id) > uint64(triangles) {
			return fmt.Errorf("visibility returned invalid triangle ID %d", id)
		}
	}
	return nil
}
