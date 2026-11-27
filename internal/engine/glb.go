package engine

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"

	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"
)

const MaxGLBBytes = 32 << 20
const maxGLBVertices = 250_000
const maxGLBIndices = 750_000

// LoadProjectGLB opens a self-contained static GLB beneath the project root.
func LoadProjectGLB(root, name string) (*Mesh, error) {
	file, err := os.OpenInRoot(root, name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if info, err := file.Stat(); err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("GLB must be a regular file")
	}
	return ReadGLB(file)
}

// ReadGLB flattens the default glTF scene into one visual mesh.
func ReadGLB(file io.Reader) (*Mesh, error) {
	data, err := io.ReadAll(io.LimitReader(file, MaxGLBBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxGLBBytes {
		return nil, fmt.Errorf("GLB exceeds %d MiB", MaxGLBBytes>>20)
	}
	doc, err := glbDocument(data)
	if err != nil {
		return nil, err
	}
	importer := glbImporter{doc: doc, materials: make(map[int]IMaterial), mesh: &Mesh{}, fallback: NewMaterial()}
	scene := 0
	if doc.Scene != nil {
		scene = *doc.Scene
	}
	root := glbItem(doc.Scenes, scene)
	if root == nil {
		return nil, fmt.Errorf("GLB has no default scene")
	}
	for _, node := range root.Nodes {
		if err := importer.node(node, IdentityMatrix(), 0); err != nil {
			return nil, err
		}
	}
	if len(importer.mesh.Indices) == 0 {
		return nil, fmt.Errorf("GLB contains no triangle faces")
	}
	return importer.mesh, nil
}

// Validate the container before decoding metadata. Keep the embedded BIN data
// as a bounded slice, independent of untrusted byte lengths in the JSON.
func glbDocument(data []byte) (*gltf.Document, error) {
	if len(data) < 20 || string(data[:4]) != "glTF" || binary.LittleEndian.Uint32(data[4:]) != 2 ||
		int64(binary.LittleEndian.Uint32(data[8:])) != int64(len(data)) || string(data[16:20]) != "JSON" {
		return nil, fmt.Errorf("invalid GLB 2.0 header")
	}
	jsonSize := int64(binary.LittleEndian.Uint32(data[12:]))
	if jsonSize > int64(len(data)-20) || jsonSize%4 != 0 {
		return nil, fmt.Errorf("invalid GLB JSON chunk")
	}
	end := 20 + int(jsonSize)
	var doc gltf.Document
	if err := json.Unmarshal(data[20:end], &doc); err != nil {
		return nil, fmt.Errorf("decode GLB scene: %w", err)
	}
	if doc.Asset.Version != "2.0" || len(doc.ExtensionsRequired) > 0 || len(doc.Animations) > 0 || len(doc.Skins) > 0 {
		return nil, fmt.Errorf("GLB requires static glTF 2.0 geometry without required extensions")
	}
	buffer := glbItem(doc.Buffers, 0)
	if len(doc.Buffers) != 1 || buffer == nil || buffer.URI != "" ||
		len(data)-end < 8 || string(data[end+4:end+8]) != "BIN\x00" {
		return nil, fmt.Errorf("GLB requires one embedded binary buffer")
	}
	chunk := int64(binary.LittleEndian.Uint32(data[end:]))
	if chunk != int64(len(data)-end-8) || chunk%4 != 0 || buffer.ByteLength <= 0 ||
		int64(buffer.ByteLength) > chunk || chunk-int64(buffer.ByteLength) > 3 {
		return nil, fmt.Errorf("invalid GLB binary chunk")
	}
	buffer.Data = data[end+8 : end+8+buffer.ByteLength]
	return &doc, nil
}

func glbItem[T any](items []*T, index int) *T {
	if index < 0 || index >= len(items) {
		return nil
	}
	return items[index]
}

type glbImporter struct {
	doc                  *gltf.Document
	mesh                 *Mesh
	materials            map[int]IMaterial
	fallback             Material
	nodes, texturePixels int
}

func (g *glbImporter) node(index int, parent Matrix4x4, depth int) error {
	n := glbItem(g.doc.Nodes, index)
	g.nodes++
	if n == nil || depth >= 64 || g.nodes > 4096 || n.Skin != nil {
		return fmt.Errorf("invalid, skinned or too deeply nested GLB node %d", index)
	}
	// glTF stores its affine matrices column-major.
	local := glbTransform(n)
	world := parent.Multiply(local)
	if n.Mesh != nil {
		mesh := glbItem(g.doc.Meshes, *n.Mesh)
		if mesh == nil {
			return fmt.Errorf("invalid GLB mesh %d", *n.Mesh)
		}
		for _, primitive := range mesh.Primitives {
			if err := g.primitive(primitive, world); err != nil {
				return fmt.Errorf("GLB node %d: %w", index, err)
			}
		}
	}
	for _, child := range n.Children {
		if err := g.node(child, world, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func glbTransform(n *gltf.Node) Matrix4x4 {
	if n.Matrix != gltf.DefaultMatrix {
		var matrix Matrix4x4
		for r := range 4 {
			for c := range 4 {
				matrix.M[r*4+c] = n.Matrix[c*4+r]
			}
		}
		return matrix
	}
	p, q, s := n.Translation, n.Rotation, n.Scale
	return ComposeMatrix(Point{X: p[0], Y: p[1], Z: p[2]}, Quaternion{X: q[0], Y: q[1], Z: q[2], W: q[3]}.Normalize(), Point{X: s[0], Y: s[1], Z: s[2]})
}

func (g *glbImporter) view(index int) *gltf.BufferView {
	v := glbItem(g.doc.BufferViews, index)
	size := g.doc.Buffers[0].ByteLength
	if v == nil || v.Buffer != 0 || v.ByteOffset < 0 || v.ByteOffset > size || v.ByteLength <= 0 || v.ByteLength > size-v.ByteOffset {
		return nil
	}
	return v
}

func (g *glbImporter) accessor(index, width, limit int) (*gltf.Accessor, error) {
	a := glbItem(g.doc.Accessors, index)
	if a == nil || a.BufferView == nil || a.Sparse != nil || a.Count <= 0 || a.Count > limit || a.ByteOffset < 0 {
		return nil, fmt.Errorf("invalid or sparse GLB accessor %d", index)
	}
	component := 0
	switch a.ComponentType {
	case gltf.ComponentFloat, gltf.ComponentUint:
		component = 4
	case gltf.ComponentUshort:
		component = 2
	case gltf.ComponentUbyte:
		component = 1
	default:
		return nil, fmt.Errorf("unsupported GLB accessor component")
	}
	v := g.view(*a.BufferView)
	if v == nil {
		return nil, fmt.Errorf("invalid GLB buffer view")
	}
	element := width * component
	stride := v.ByteStride
	if stride == 0 {
		stride = element
	}
	if stride < element || stride > 252 || a.ByteOffset > v.ByteLength ||
		int64(stride)*int64(a.Count-1)+int64(element) > int64(v.ByteLength-a.ByteOffset) {
		return nil, fmt.Errorf("GLB accessor spans beyond its buffer view")
	}
	return a, nil
}

func (g *glbImporter) primitive(p *gltf.Primitive, matrix Matrix4x4) error {
	if p == nil || p.Mode != gltf.PrimitiveTriangles || len(p.Targets) > 0 {
		return fmt.Errorf("expected a static triangle primitive")
	}
	position, ok := p.Attributes["POSITION"]
	if !ok {
		return fmt.Errorf("missing POSITION")
	}
	mesh := g.mesh
	a, err := g.accessor(position, 3, maxGLBVertices-len(mesh.Vertices))
	if err != nil {
		return err
	}
	positions, err := modeler.ReadPosition(g.doc, a, nil)
	if err != nil {
		return err
	}

	var indices []uint32
	if p.Indices != nil {
		a, err = g.accessor(*p.Indices, 1, maxGLBIndices-len(mesh.Indices))
		if err != nil {
			return err
		}
		indices, err = modeler.ReadIndices(g.doc, a, nil)
		if err != nil {
			return err
		}
	} else {
		indices = make([]uint32, len(positions))
		for i := range indices {
			indices[i] = uint32(i)
		}
	}
	if len(indices)%3 != 0 || len(indices) > maxGLBIndices-len(mesh.Indices) {
		return fmt.Errorf("invalid triangle index count")
	}
	for _, index := range indices {
		if uint64(index) >= uint64(len(positions)) {
			return fmt.Errorf("triangle index outside positions")
		}
	}

	var uvs [][2]float32
	if uv, ok := p.Attributes["TEXCOORD_0"]; ok {
		a, err = g.accessor(uv, 2, len(positions))
		if err != nil {
			return err
		}
		uvs, err = modeler.ReadTextureCoord(g.doc, a, nil)
		if err != nil {
			return err
		}
		if len(uvs) != len(positions) {
			return fmt.Errorf("mismatched GLB UV count")
		}
	}
	material, err := g.material(p.Material, len(uvs) > 0)
	if err != nil {
		return err
	}
	base := len(mesh.Vertices)
	for i, v := range positions {
		point := matrix.TransformPoint(Point{X: float64(v[0]), Y: float64(v[1]), Z: float64(v[2])})
		point.Z = -point.Z // Convert to the terminal camera's +Z-forward convention.
		for _, axis := range []float64{point.X, point.Y, point.Z} {
			if math.IsNaN(axis) || math.IsInf(axis, 0) {
				return fmt.Errorf("nonfinite GLB position")
			}
		}
		mesh.Vertices = append(mesh.Vertices, point)
		var uv TextureCoord
		if len(uvs) > 0 {
			uv = TextureCoord{U: float64(uvs[i][0]), V: float64(uvs[i][1])}
			if math.IsNaN(uv.U) || math.IsNaN(uv.V) || math.IsInf(uv.U, 0) || math.IsInf(uv.V, 0) {
				return fmt.Errorf("nonfinite GLB UV")
			}
		}
		mesh.UVs = append(mesh.UVs, uv)
	}
	m := matrix.M
	determinant := m[0]*(m[5]*m[10]-m[6]*m[9]) - m[1]*(m[4]*m[10]-m[6]*m[8]) + m[2]*(m[4]*m[9]-m[5]*m[8])
	for i := 0; i < len(indices); i += 3 {
		a, b, c := indices[i], indices[i+1], indices[i+2]
		if determinant >= 0 {
			b, c = c, b
		}
		mesh.Indices = append(mesh.Indices, base+int(a), base+int(b), base+int(c))
		mesh.FaceMaterials = append(mesh.FaceMaterials, material)
	}
	return nil
}

func (g *glbImporter) material(index *int, hasUV bool) (IMaterial, error) {
	if index == nil {
		return &g.fallback, nil
	}
	m := glbItem(g.doc.Materials, *index)
	if m == nil || m.AlphaMode != gltf.AlphaOpaque {
		return nil, fmt.Errorf("GLB material must be opaque")
	}
	pbr := m.PBRMetallicRoughness
	if pbr != nil && pbr.BaseColorTexture != nil && !hasUV {
		return nil, fmt.Errorf("textured GLB primitive needs TEXCOORD_0")
	}
	if existing := g.materials[*index]; existing != nil {
		return existing, nil
	}
	result := NewPBRMaterial()
	if pbr != nil {
		p := result
		base := pbr.BaseColorFactorOrDefault()
		p.Albedo = Color{R: glbSRGB(base[0]), G: glbSRGB(base[1]), B: glbSRGB(base[2])}
		p.Metallic = Clamp(pbr.MetallicFactorOrDefault(), 0, 1)
		p.Roughness = Clamp(pbr.RoughnessFactorOrDefault(), 0, 1)
		if pbr.BaseColorTexture != nil {
			tex, err := g.texture(pbr.BaseColorTexture, base)
			if err != nil {
				return nil, err
			}
			p.AlbedoMap, p.UseTextures = tex, true
		}
	}
	g.materials[*index] = result
	return result, nil
}

func glbSRGB(value float64) uint8 {
	value = Clamp(value, 0, 1)
	if value <= .0031308 {
		return uint8(math.Round(255 * 12.92 * value))
	}
	return uint8(math.Round(255 * (1.055*math.Pow(value, 1/2.4) - .055)))
}

func (g *glbImporter) texture(info *gltf.TextureInfo, factor [4]float64) (*Texture, error) {
	t := glbItem(g.doc.Textures, info.Index)
	if t == nil || t.Source == nil || info.TexCoord != 0 {
		return nil, fmt.Errorf("GLB base color texture requires TEXCOORD_0")
	}
	img := glbItem(g.doc.Images, *t.Source)
	if img == nil || img.BufferView == nil || img.URI != "" || (img.MimeType != "image/png" && img.MimeType != "image/jpeg") {
		return nil, fmt.Errorf("GLB base color texture must be embedded PNG or JPEG")
	}
	view := g.view(*img.BufferView)
	if view == nil {
		return nil, fmt.Errorf("invalid GLB image buffer view")
	}
	data, err := modeler.ReadBufferView(g.doc, view)
	if err != nil {
		return nil, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 4096 || cfg.Height > 4096 || cfg.Width*cfg.Height > 8_388_608-g.texturePixels {
		return nil, fmt.Errorf("GLB exceeds decoded texture budget")
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	g.texturePixels += cfg.Width * cfg.Height
	texture := NewTextureFromImage(decoded)
	for i, pixel := range texture.Data {
		texture.Data[i] = Color{R: tintGLB(pixel.R, factor[0]), G: tintGLB(pixel.G, factor[1]), B: tintGLB(pixel.B, factor[2])}
	}
	return texture, nil
}

func tintGLB(value uint8, factor float64) uint8 {
	channel := float64(value) / 255
	if channel <= .04045 {
		channel /= 12.92
	} else {
		channel = math.Pow((channel+.055)/1.055, 2.4)
	}
	return glbSRGB(channel * factor)
}
