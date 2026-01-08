package main

import (
	"bufio"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"
)

// LoadOBJ loads a Wavefront OBJ file and returns a Mesh
func LoadOBJ(filepath string) (*Mesh, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	mesh := NewMesh()
	scanner := bufio.NewScanner(file)

	var vertices []Point
	var normals []Point
	var uvs []TextureCoord
	var materialLib *MaterialLibrary
	var currentMaterial *Material

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

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
				return nil, fmt.Errorf("line %d: invalid vertex definition", lineNum)
			}
			x, err1 := strconv.ParseFloat(parts[1], 64)
			y, err2 := strconv.ParseFloat(parts[2], 64)
			z, err3 := strconv.ParseFloat(parts[3], 64)
			if err1 != nil || err2 != nil || err3 != nil {
				return nil, fmt.Errorf("line %d: invalid vertex coordinates", lineNum)
			}
			vertices = append(vertices, Point{X: x, Y: y, Z: z})

		case "vn": // Vertex normal
			if len(parts) < 4 {
				return nil, fmt.Errorf("line %d: invalid normal definition", lineNum)
			}
			x, err1 := strconv.ParseFloat(parts[1], 64)
			y, err2 := strconv.ParseFloat(parts[2], 64)
			z, err3 := strconv.ParseFloat(parts[3], 64)
			if err1 != nil || err2 != nil || err3 != nil {
				return nil, fmt.Errorf("line %d: invalid normal coordinates", lineNum)
			}
			normals = append(normals, Point{X: x, Y: y, Z: z})

		case "vt": // Texture coordinate
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid texture coordinate", lineNum)
			}
			u, err1 := strconv.ParseFloat(parts[1], 64)
			v, err2 := strconv.ParseFloat(parts[2], 64)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("line %d: invalid UV coordinates", lineNum)
			}
			uvs = append(uvs, TextureCoord{U: u, V: v})

		case "f": // Face
			if len(parts) < 4 {
				return nil, fmt.Errorf("line %d: face must have at least 3 vertices", lineNum)
			}

			// Parse face indices (supports v, v/vt, v/vt/vn, v//vn formats)
			faceVertices := make([]int, 0, len(parts)-1)

			for i := 1; i < len(parts); i++ {
				indices := parseFaceVertex(parts[i])
				if indices[0] == 0 {
					return nil, fmt.Errorf("line %d: invalid face index", lineNum)
				}

				// OBJ indices are 1-based, convert to 0-based
				vertexIdx := indices[0] - 1
				if vertexIdx < 0 || vertexIdx >= len(vertices) {
					return nil, fmt.Errorf("line %d: vertex index out of range", lineNum)
				}

				// Add vertex to mesh if not already added
				// For simplicity, we duplicate vertices (could optimize with index mapping)
				meshVertexIdx := mesh.AddVertex(vertices[vertexIdx].X, vertices[vertexIdx].Y, vertices[vertexIdx].Z)
				faceVertices = append(faceVertices, meshVertexIdx)
			}

			// Triangulate face (fan triangulation for n-gons)
			for i := 1; i < len(faceVertices)-1; i++ {
				mesh.AddTriangleIndices(faceVertices[0], faceVertices[i], faceVertices[i+1])
			}

		case "mtllib": // Material library
			if len(parts) >= 2 {
				// Load material library from the same directory as the OBJ file
				dir := filepath[:strings.LastIndex(filepath, string(os.PathSeparator))+1]
				mtlPath := dir + parts[1]
				lib, err := LoadMTL(mtlPath)
				if err == nil {
					materialLib = lib
				}
			}
			continue

		case "usemtl": // Use material
			if len(parts) >= 2 && materialLib != nil {
				if mat, exists := materialLib.Materials[parts[1]]; exists {
					currentMaterial = mat
					// Apply material to mesh
					mesh.Material = *currentMaterial
				}
			}
			continue

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
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	if len(mesh.Vertices) == 0 {
		return nil, fmt.Errorf("no vertices found in OBJ file")
	}

	return mesh, nil
}

// parseFaceVertex parses a face vertex string (v, v/vt, v/vt/vn, v//vn)
// Returns [vertexIdx, texCoordIdx, normalIdx] (0 means not present)
func parseFaceVertex(s string) [3]int {
	result := [3]int{0, 0, 0}

	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return result
	}

	// Vertex index
	if parts[0] != "" {
		if idx, err := strconv.Atoi(parts[0]); err == nil {
			result[0] = idx
		}
	}

	// Texture coordinate index
	if len(parts) > 1 && parts[1] != "" {
		if idx, err := strconv.Atoi(parts[1]); err == nil {
			result[1] = idx
		}
	}

	// Normal index
	if len(parts) > 2 && parts[2] != "" {
		if idx, err := strconv.Atoi(parts[2]); err == nil {
			result[2] = idx
		}
	}

	return result
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
	// Map face indices to material names (not fully implemented)
}

// LoadOBJWithMaterials loads an OBJ file and attempts to load its materials
func LoadOBJWithMaterials(filepath string) (*OBJWithMaterials, error) {
	mesh, err := LoadOBJ(filepath)
	if err != nil {
		return nil, err
	}

	result := &OBJWithMaterials{
		Mesh: mesh,
	}

	// Try to find and load MTL file (same directory, same name)
	mtlPath := strings.TrimSuffix(filepath, ".obj") + ".mtl"
	if _, err := os.Stat(mtlPath); err == nil {
		materials, err := LoadMTL(mtlPath)
		if err == nil {
			result.Materials = materials
		}
	}

	return result, nil
}
