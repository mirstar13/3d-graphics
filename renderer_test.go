package main

import (
	"bufio"
	"bytes"
	"testing"
)

func TestTerminalRenderer_RenderMesh(t *testing.T) {
	// Setup
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	renderer := NewTerminalRenderer(writer, 50, 50)

	// Initialize (sets up buffers)
	renderer.Initialize()
	renderer.BeginFrame()

	// Create a camera
	camera := NewCameraAt(0, 0, -10)
	renderer.SetCamera(camera)

	// Create a simple mesh (triangle)
	mesh := NewMesh()
	mesh.AddVertex(-1, -1, 0)
	mesh.AddVertex(1, -1, 0)
	mesh.AddVertex(0, 1, 0)
	mesh.AddTriangleIndices(0, 1, 2)
	mat := NewMaterial()
	mat.DiffuseColor = ColorRed
	mesh.Material = &mat

	// Render
	worldMatrix := IdentityMatrix()
	renderer.RenderMesh(mesh, worldMatrix, camera)

	// Check if any pixels were drawn (ZBuffer should be updated)
	drawn := false
	for y := 0; y < renderer.Height; y++ {
		for x := 0; x < renderer.Width; x++ {
			if renderer.ZBuffer[y][x] != 1.7976931348623157e+308 { // math.Inf(1)
				drawn = true
				break
			}
		}
		if drawn {
			break
		}
	}

	if !drawn {
		t.Error("RenderMesh failed to draw anything to ZBuffer")
	}
}

func TestTerminalRenderer_RenderMesh_ManyAllocations(t *testing.T) {
	// This test simulates many mesh renders to verify our optimization doesn't break things
	// and potentially to benchmark (though hard in functional test).

	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	renderer := NewTerminalRenderer(writer, 50, 50)
	renderer.Initialize()
	renderer.BeginFrame()

	camera := NewCameraAt(0, 0, -10)
	renderer.SetCamera(camera)

	mesh := NewMesh()
	mesh.AddVertex(-1, -1, 0)
	mesh.AddVertex(1, -1, 0)
	mesh.AddVertex(0, 1, 0)
	mesh.AddTriangleIndices(0, 1, 2)

	worldMatrix := IdentityMatrix()

	// Render 100 times
	for i := 0; i < 100; i++ {
		renderer.RenderMesh(mesh, worldMatrix, camera)
	}
}
