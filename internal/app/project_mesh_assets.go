package app

import (
	"fmt"
	"strings"
)

func (m *projectEditorModel) browseMeshes() {
	paths, err := m.project.MeshAssets()
	if err != nil {
		m.message = err.Error()
		return
	}
	m.meshPaths = paths
	m.meshSelection = 0
	m.modal = "mesh_assets"
}

func (m *projectEditorModel) meshBrowser() string {
	var out strings.Builder
	out.WriteString("\nMesh assets\n")
	if len(m.meshPaths) == 0 {
		out.WriteString("No OBJ or GLB assets\n")
	}
	start := max(0, m.meshSelection-1)
	for i := start; i < min(len(m.meshPaths), start+4); i++ {
		marker := "  "
		if i == m.meshSelection {
			marker = "> "
		}
		fmt.Fprintf(&out, "%s%s\n", marker, m.meshPaths[i])
	}
	out.WriteString("Arrows choose | Enter assign | I import GLB | Esc close\n")
	return out.String()
}

func (m *projectEditorModel) updateMeshBrowser(key string) {
	count := len(m.meshPaths)
	switch key {
	case "i":
		m.modal = "mesh_import"
		m.draft = ""
	case "up":
		if count > 0 {
			m.meshSelection = (m.meshSelection + count - 1) % count
		}
	case "down", "tab":
		if count > 0 {
			m.meshSelection = (m.meshSelection + 1) % count
		}
	case "enter":
		entity, ok := m.selectedEntity()
		if !ok || count == 0 {
			return
		}
		name := m.meshPaths[m.meshSelection]
		entity.MeshAsset = name
		if _, _, err := entity.LoadVisualAssets(m.project.Root); err != nil {
			m.message = err.Error()
			return
		}
		if err := m.document.Set(entity.ID, "mesh_asset", name); err != nil {
			m.message = err.Error()
			return
		}
		m.modal = ""
		m.refresh()
		m.message = "Assigned " + name
	}
}
