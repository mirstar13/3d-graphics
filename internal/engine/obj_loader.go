package engine

import (
	"bufio"
	"fmt"
	"image"
	"os"
	path "path/filepath"
	"strconv"
	"strings"
)

// LoadOBJ reads indexed Wavefront faces, including negative indices and UV seams.
// Faces are triangulated as fans. Vertex normals are validated; shading uses
// geometric face normals because Mesh has no per-vertex normal storage.
// usemtl assignments are stored per triangulated face.
func LoadOBJ(filepath string) (*Mesh, error) {
	mesh, _, err := loadOBJ(filepath)
	return mesh, err
}

func loadOBJ(filepath string) (*Mesh, *MaterialLibrary, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	mesh := NewMesh()
	scanner := bufio.NewScanner(file)

	var vertices []Point
	normalCount := 0
	hasUVs := false
	var uvs []TextureCoord
	materialLib := NewMaterialLibrary()
	var currentMaterial *Material

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "v": // Vertex position
			if len(parts) < 4 {
				return nil, nil, fmt.Errorf("line %d: invalid vertex definition", lineNum)
			}
			x, err1 := strconv.ParseFloat(parts[1], 64)
			y, err2 := strconv.ParseFloat(parts[2], 64)
			z, err3 := strconv.ParseFloat(parts[3], 64)
			if err1 != nil || err2 != nil || err3 != nil {
				return nil, nil, fmt.Errorf("line %d: invalid vertex coordinates", lineNum)
			}
			vertices = append(vertices, Point{X: x, Y: y, Z: z})

		case "vn":
			if len(parts) < 4 {
				return nil, nil, fmt.Errorf("line %d: invalid normal definition", lineNum)
			}
			for _, component := range parts[1:4] {
				if _, err := strconv.ParseFloat(component, 64); err != nil {
					return nil, nil, fmt.Errorf("line %d: invalid normal coordinates", lineNum)
				}
			}
			normalCount++

		case "vt": // Texture coordinate
			if len(parts) < 3 {
				return nil, nil, fmt.Errorf("line %d: invalid texture coordinate", lineNum)
			}
			u, err1 := strconv.ParseFloat(parts[1], 64)
			v, err2 := strconv.ParseFloat(parts[2], 64)
			if err1 != nil || err2 != nil {
				return nil, nil, fmt.Errorf("line %d: invalid UV coordinates", lineNum)
			}
			uvs = append(uvs, TextureCoord{U: u, V: v})

		case "f": // Face
			if len(parts) < 4 {
				return nil, nil, fmt.Errorf("line %d: face must have at least 3 vertices", lineNum)
			}

			// Parse face indices (supports v, v/vt, v/vt/vn, v//vn formats)
			faceVertices := make([]int, 0, len(parts)-1)

			for i := 1; i < len(parts); i++ {
				indices, err := parseFaceVertex(parts[i])
				if err != nil {
					return nil, nil, fmt.Errorf("line %d: %w", lineNum, err)
				}
				vertexIdx, err := resolveOBJIndex(indices[0], len(vertices))
				if err != nil {
					return nil, nil, fmt.Errorf("line %d: vertex %w", lineNum, err)
				}
				var uv TextureCoord
				if indices[1] != 0 {
					uvIndex, err := resolveOBJIndex(indices[1], len(uvs))
					if err != nil {
						return nil, nil, fmt.Errorf("line %d: texture %w", lineNum, err)
					}
					uv = uvs[uvIndex]
					hasUVs = true
				}
				if indices[2] != 0 {
					if _, err := resolveOBJIndex(indices[2], normalCount); err != nil {
						return nil, nil, fmt.Errorf("line %d: normal %w", lineNum, err)
					}
				}
				// Separate face vertices preserve seams where one position has multiple UVs.
				vertex := vertices[vertexIdx]
				meshVertexIdx := mesh.AddVertexWithUV(vertex.X, vertex.Y, vertex.Z, uv.U, uv.V)
				faceVertices = append(faceVertices, meshVertexIdx)
			}

			// Triangulate face (fan triangulation for n-gons)
			for i := 1; i < len(faceVertices)-1; i++ {
				mesh.AddTriangleIndices(faceVertices[0], faceVertices[i], faceVertices[i+1])
				var assigned IMaterial
				if currentMaterial != nil {
					assigned = currentMaterial
				}
				mesh.FaceMaterials = append(mesh.FaceMaterials, assigned)
			}

		case "mtllib":
			for _, name := range parts[1:] {
				lib, err := LoadMTL(path.Join(path.Dir(filepath), name))
				if err != nil {
					return nil, nil, fmt.Errorf("line %d: %w", lineNum, err)
				}
				for name, mat := range lib.Materials {
					materialLib.Materials[name] = mat
				}
			}
		case "usemtl":
			if len(parts) != 2 {
				return nil, nil, fmt.Errorf("line %d: missing material name", lineNum)
			}
			currentMaterial = materialLib.Materials[parts[1]]
			if currentMaterial == nil {
				return nil, nil, fmt.Errorf("line %d: unknown material %q", lineNum, parts[1])
			}

		case "o": // Object name
			// mesh.Name = parts[1] if we add Name field
			continue

		case "g": // Group name
			// Could be used for sub-meshes
			continue

		case "s": // Smooth shading (on/off)
			// Could affect normal calculation
			continue

		default:
			// Unknown directive, skip
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading file: %w", err)
	}

	if len(mesh.Vertices) == 0 {
		return nil, nil, fmt.Errorf("no vertices found in OBJ file")
	}

	if !hasUVs {
		mesh.UVs = nil
	}
	return mesh, materialLib, nil
}

// parseFaceVertex accepts v, v/vt, v//vn, and v/vt/vn index tuples.
func parseFaceVertex(value string) ([3]int, error) {
	var result [3]int
	parts := strings.Split(value, "/")
	if len(parts) > 3 || parts[0] == "" {
		return result, fmt.Errorf("invalid face index %q", value)
	}
	for i, part := range parts {
		if part == "" {
			continue
		}
		index, err := strconv.Atoi(part)
		if err != nil || index == 0 {
			return result, fmt.Errorf("invalid face index %q", value)
		}
		result[i] = index
	}
	return result, nil
}

func resolveOBJIndex(index, count int) (int, error) {
	if index < 0 {
		index += count
	} else {
		index--
	}
	if index < 0 || index >= count {
		return 0, fmt.Errorf("index out of range")
	}
	return index, nil
}

// SaveOBJ saves a mesh to an OBJ file
func SaveOBJ(mesh *Mesh, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.WriteString("# Generated by Go 3D Graphics Engine\n")
	writer.WriteString(fmt.Sprintf("# Vertices: %d\n", len(mesh.Vertices)))
	writer.WriteString(fmt.Sprintf("# Triangles: %d\n\n", len(mesh.Indices)/3))

	// Write vertices
	for _, v := range mesh.Vertices {
		writer.WriteString(fmt.Sprintf("v %.6f %.6f %.6f\n", v.X, v.Y, v.Z))
	}

	writer.WriteString("\n")

	// Write faces (triangles)
	for i := 0; i < len(mesh.Indices); i += 3 {
		// OBJ uses 1-based indexing
		writer.WriteString(fmt.Sprintf("f %d %d %d\n",
			mesh.Indices[i]+1,
			mesh.Indices[i+1]+1,
			mesh.Indices[i+2]+1))
	}

	return nil
}

// LoadTextureFromFile loads a texture from an image file
func LoadTextureFromFile(filepath string) (*Texture, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return NewTextureFromImage(img), nil
}

// OBJStats holds statistics about a loaded OBJ file
type OBJStats struct {
	Vertices  int
	Triangles int
	Quads     int
	Normals   int
	UVs       int
}

// GetOBJStats returns statistics about an OBJ file without fully loading it
func GetOBJStats(filepath string) (*OBJStats, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := &OBJStats{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "v":
			stats.Vertices++
		case "vn":
			stats.Normals++
		case "vt":
			stats.UVs++
		case "f":
			// Count face complexity
			numVertices := len(parts) - 1
			if numVertices == 3 {
				stats.Triangles++
			} else if numVertices == 4 {
				stats.Quads++
				stats.Triangles += 2 // Quads become 2 triangles
			} else if numVertices > 4 {
				stats.Triangles += numVertices - 2 // N-gon triangulation
			}
		}
	}

	return stats, scanner.Err()
}

func (s *OBJStats) String() string {
	return fmt.Sprintf("Vertices: %d | Triangles: %d | Normals: %d | UVs: %d",
		s.Vertices, s.Triangles, s.Normals, s.UVs)
}

// MaterialLibrary holds materials from an MTL file
type MaterialLibrary struct {
	Materials map[string]*Material
}

// NewMaterialLibrary creates a new material library
func NewMaterialLibrary() *MaterialLibrary {
	return &MaterialLibrary{
		Materials: make(map[string]*Material),
	}
}

// LoadMTL loads a Wavefront MTL material library file
func LoadMTL(filepath string) (*MaterialLibrary, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open material file: %w", err)
	}
	defer file.Close()

	lib := NewMaterialLibrary()
	scanner := bufio.NewScanner(file)
	var currentMaterial *Material
	var currentName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "newmtl": // New material
			if len(parts) < 2 {
				continue
			}
			currentName = parts[1]
			mat := NewMaterial()
			currentMaterial = &mat
			lib.Materials[currentName] = currentMaterial

		case "Kd": // Diffuse color
			if currentMaterial != nil && len(parts) >= 4 {
				r, _ := strconv.ParseFloat(parts[1], 64)
				g, _ := strconv.ParseFloat(parts[2], 64)
				b, _ := strconv.ParseFloat(parts[3], 64)
				currentMaterial.DiffuseColor = Color{
					R: uint8(r * 255),
					G: uint8(g * 255),
					B: uint8(b * 255),
				}
			}

		case "Ks": // Specular color
			if currentMaterial != nil && len(parts) >= 4 {
				r, _ := strconv.ParseFloat(parts[1], 64)
				g, _ := strconv.ParseFloat(parts[2], 64)
				b, _ := strconv.ParseFloat(parts[3], 64)
				currentMaterial.SpecularColor = Color{
					R: uint8(r * 255),
					G: uint8(g * 255),
					B: uint8(b * 255),
				}
			}

		case "Ka": // Ambient color (affects ambient strength)
			if currentMaterial != nil && len(parts) >= 4 {
				r, _ := strconv.ParseFloat(parts[1], 64)
				g, _ := strconv.ParseFloat(parts[2], 64)
				b, _ := strconv.ParseFloat(parts[3], 64)
				avg := (r + g + b) / 3.0
				currentMaterial.AmbientStrength = avg
			}

		case "Ns": // Shininess
			if currentMaterial != nil && len(parts) >= 2 {
				ns, _ := strconv.ParseFloat(parts[1], 64)
				currentMaterial.Shininess = ns
			}

		case "d": // Dissolve (transparency, not implemented)
		case "Tr": // Transparency (not implemented)
		case "illum": // Illumination model (not implemented)
		case "map_Kd": // Diffuse texture map (not implemented here)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lib, nil
}

// OBJWithMaterials represents a mesh with material information
type OBJWithMaterials struct {
	Mesh      *Mesh
	Materials *MaterialLibrary
}

// LoadOBJWithMaterials loads an OBJ file and attempts to load its materials
func LoadOBJWithMaterials(filepath string) (*OBJWithMaterials, error) {
	mesh, materials, err := loadOBJ(filepath)
	if err != nil {
		return nil, err
	}
	return &OBJWithMaterials{Mesh: mesh, Materials: materials}, nil
}
