package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

// finishAction runs only after the unsaved-change guard has been satisfied.
func (m *projectEditorModel) finishAction(action string) tea.Cmd {
	if action == "quit" {
		return tea.Quit
	}
	if action == "asset" {
		m.modal = ""
		if err := m.openAsset(m.assetTarget); err != nil {
			m.message = err.Error()
		}
		return nil
	}
	m.modal = action
	m.draft = ""
	return nil
}

func (m *projectEditorModel) openAsset(path string) error {
	if path == "" {
		p, err := project.Open(m.project.Root)
		if err != nil {
			return err
		}
		return m.install(p)
	}
	r, err := m.project.LoadResource(path)
	if err != nil {
		return err
	}
	// install stages validation and preview before swapping the editor. The project
	// identity remains unchanged; this document has an independent history.
	original := m.project
	staging := *original
	staging.Scene = r.Scene
	if err = m.install(&staging); err != nil {
		return err
	}
	m.project = original
	m.activeResource = &r
	m.resourcePath = path
	m.message = "Editing source: " + r.Scene.Name + "; A opens entry scene"
	return nil
}
func (m *projectEditorModel) browseResources() {
	assets, err := m.project.Resources()
	if err != nil {
		m.message = err.Error()
		return
	}
	m.assets = assets
	m.assetSelected = 0
	m.modal = "assets"
	m.message = "Entry scene: Enter open | Resource: Enter instantiate, E edit source"
}
func (m *projectEditorModel) updateAssets(key string) tea.Cmd {
	count := len(m.assets) + 1
	switch key {
	case "up":
		m.assetSelected = (m.assetSelected + count - 1) % count
	case "down", "tab":
		m.assetSelected = (m.assetSelected + 1) % count
	case "enter", "e":
		if m.assetSelected == 0 {
			m.assetTarget = ""
			return m.begin("asset")
		}
		info := m.assets[m.assetSelected-1]
		if info.Error != "" {
			m.message = info.Error
			return nil
		}
		if key == "e" {
			m.assetTarget = info.Path
			return m.begin("asset")
		}
		if m.activeResource != nil {
			m.message = "Nested linked resources are unsupported; open the entry scene to instantiate"
			return nil
		}
		id, err := m.document.Instantiate(m.project, info.Path)
		if err != nil {
			m.message = err.Error()
			return nil
		}
		m.modal = ""
		m.selectID(id)
		m.refresh()
		m.message = "Instantiated " + info.Name
	}
	return nil
}
func (m projectEditorModel) resourceBrowser() string {
	var out strings.Builder
	out.WriteString("\nScene resources\n")
	first := max(0, m.assetSelected-1)
	for i := first; i < min(len(m.assets)+1, first+3); i++ {
		label := "Entry scene"
		if i > 0 {
			info := m.assets[i-1]
			label = info.Name + " [" + info.ID + "]"
			if info.Error != "" {
				label += " [invalid]"
			}
		}
		prefix := "  "
		if i == m.assetSelected {
			prefix = "> "
		}
		fmt.Fprintf(&out, "%s%s\n", prefix, label)
	}
	out.WriteString("Arrows choose | Enter add/open | E edit source | Esc cancel\n")
	return out.String()
}
func (m *projectEditorModel) resourceCommand(key string) {
	selected, ok := m.selectedEntity()
	var err error
	switch key {
	case "a":
		m.browseResources()
		return
	case "r":
		if ok {
			m.modal = "extract"
			m.draft = selected.Name
			m.message = "Save selection as a new resource; original stays unchanged"
		}
		return
	case "ctrl+r":
		if m.activeResource != nil {
			m.message = "Source documents contain no links to refresh"
			return
		}
		err = m.document.RefreshLinks(m.project)
		m.message = "Links refreshed; overrides preserved"
	case "ctrl+u":
		if ok {
			err = m.document.Unpack(selected.ID)
			m.message = "Instance unpacked; values and IDs preserved"
		}
	case "ctrl+t":
		if ok {
			err = m.document.ResetOverride(m.project, selected.ID, project.Properties[m.property].Key)
			m.message = "Property override reset from source"
		}
	}
	if err != nil {
		m.message = err.Error()
		return
	}
	if ok {
		m.selectID(selected.ID)
	}
	m.refresh()
}
func (m projectEditorModel) inspectorProperty(index int, e project.Entity) project.Property {
	p := project.Properties[index]
	if i, source, ok := m.document.Document().Link(e.ID); ok {
		if _, overridden := m.document.Document().Instances[i].Overrides[source][p.Key]; overridden {
			p.Label += " *"
		}
	}
	return p
}
