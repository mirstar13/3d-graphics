package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/mirstar13/go-3d-graphics/internal/input"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

type editorHit struct {
	x, y, width int
	button      editorButton
	row         int // One-based absolute list index, when this is a choice.
}

type editorDialog struct {
	body                string
	buttons             []editorButton
	listRow, first, end int
}

func (m projectEditorModel) dialog() editorDialog {
	close := editorButton{"Close", "esc"}
	cancel := editorButton{"Cancel", "esc"}
	choose := []editorButton{{"Previous", "up"}, {"Next", "down"}}
	form := func(body, action string) editorDialog {
		buttons := []editorButton{{action, "enter"}, cancel}
		if m.modal == "open" || m.modal == "new" || m.modal == "mesh_import" || m.modal == "sound_import" {
			buttons = append(buttons, editorButton{"Browse", "ui.files"})
		}
		buttons = append(buttons, editorButton{"Keys", "ui.keyboard"}, editorButton{"Clear", "ui.clear_draft"})
		return editorDialog{body: body, buttons: buttons}
	}
	switch m.modal {
	case "commands":
		var body strings.Builder
		body.WriteString("Editor actions\n")
		commands := m.commands()
		visible := max(1, m.height-5)
		first := max(0, m.actionSelection-visible+1)
		for i, command := range commands[first:min(first+visible, len(commands))] {
			writeChoice(&body, command.label, first+i == m.actionSelection)
		}
		return editorDialog{body: body.String(), buttons: append([]editorButton{close}, choose...), listRow: 1, first: first, end: min(len(commands), first+visible)}
	case "files":
		return m.fileDialog()
	case "discard":
		return editorDialog{body: "Unsaved edits will be discarded.", buttons: []editorButton{{"Discard", "d"}, cancel}}
	case "tools":
		first := max(0, m.toolField-1)
		return editorDialog{body: m.toolsView(), listRow: 1, first: first, end: min(len(toolLabels), first+3),
			buttons: []editorButton{{"Edit", "enter"}, close, {"Impulse", "i"}, {"Spawn", "g"}, {"Delete runtime", "d"}, {"Reset", "r"}, {"Previous", "up"}, {"Next", "down"}}}
	case "tool_value":
		return form(fmt.Sprintf("Edit %s: %s_", toolLabels[m.toolField], printableEditorText(m.draft)), "Apply")
	case "spawn_confirm":
		e := m.spawnPreview
		return editorDialog{body: fmt.Sprintf("Spawn preview: %s\nPosition %.2f %.2f %.2f\nSize %.2f m | Mass %.2f kg",
			e.Shape, e.Position.X, e.Position.Y, e.Position.Z, e.Size.X, e.Mass),
			buttons: []editorButton{{"Queue spawn", "enter"}, cancel}}
	case "bindings":
		first := max(0, m.bindingSelected-1)
		return editorDialog{body: m.bindingsView(), listRow: 1, first: first, end: min(len(input.Definitions()), first+3),
			buttons: []editorButton{{"Edit", "enter"}, close, {"Defaults", "d"}, {"Previous", "up"}, {"Next", "down"}}}
	case "binding":
		return form(fmt.Sprintf("Bind %s: %s_\nSeparate keys with commas.", input.Definitions()[m.bindingSelected].Label, printableEditorText(m.draft)), "Save")
	case "sound_assets":
		first := max(0, m.soundSelection-1)
		return editorDialog{body: m.soundBrowser(), listRow: 1, first: first, end: min(len(m.project.Manifest.Sounds), first+4),
			buttons: []editorButton{{"Import", "i"}, {"Audition", "a"}, {"Stop", "s"}, {"Assign", "enter"}, close, {"Mute", "ctrl+y"}, {"Previous", "up"}, {"Next", "down"},
				{"Use on win", "win"}, {"Use on loss", "loss"}, {"Clear win", "clear-win"}, {"Clear loss", "clear-loss"}}}
	case "sound_import":
		return form("WAV file path  "+printableEditorText(m.draft)+"_", "Import")
	case "mesh_assets":
		first := max(0, m.meshSelection-1)
		return editorDialog{body: m.meshBrowser(), listRow: 1, first: first, end: min(len(m.meshPaths), first+4),
			buttons: []editorButton{{"Import GLB", "i"}, {"Assign", "enter"}, close, {"Previous", "up"}, {"Next", "down"}}}
	case "mesh_import":
		return form("Import GLB file: "+printableEditorText(m.draft)+"_", "Import")
	case "assets":
		first := max(0, m.assetSelected-1)
		return editorDialog{body: m.resourceBrowser(), listRow: 1, first: first, end: min(len(m.assets)+1, first+3),
			buttons: []editorButton{{"Add / open", "enter"}, {"Edit source", "e"}, close, {"Previous", "up"}, {"Next", "down"}}}
	case "extract":
		return form("Resource name: "+printableEditorText(m.draft)+"_", "Create")
	case "parent":
		return m.parentDialog()
	case "property":
		p := project.Properties[m.property]
		d := form(fmt.Sprintf("Edit %s %s %v: %s_", p.Label, p.Units, p.Choices, printableEditorText(m.draft)), "Apply")
		for _, choice := range p.Choices {
			d.buttons = append(d.buttons, editorButton{choice, "choice:" + choice})
		}
		return d
	case "build", "child", "export":
		return editorDialog{body: m.buildView(), buttons: m.buildControls()}
	case "scene":
		return editorDialog{body: m.scenePanel(), buttons: []editorButton{
			{"Edit", "enter"}, close, {"Previous", "up"}, {"Next", "down"}, {"Add light", "l"}, {"Delete light", "d"},
			{"Reset camera", "c"}, {"Reset lights", "r"}, {"Inherit simulation", "s"}, {"Clear objective", "o"},
		}}
	case "scene_edit":
		return form(m.scenePanel(), "Apply")
	case "shortcuts":
		return editorDialog{body: m.shortcutView(), buttons: []editorButton{close}}
	default:
		return form(fmt.Sprintf("%s project directory: %s_", m.modal, printableEditorText(m.draft)), "Confirm")
	}
}

func writeChoice(body *strings.Builder, label string, selected bool) {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	fmt.Fprintln(body, prefix+label)
}

func (m projectEditorModel) parentDialog() editorDialog {
	var body strings.Builder
	body.WriteString("Parent picker (keeps LOCAL pose)\n")
	e, ok := m.selectedEntity()
	if !ok {
		return editorDialog{body: body.String(), buttons: []editorButton{{"Close", "esc"}}}
	}
	choices := m.document.Document().ParentChoices(e.ID)
	first := max(0, m.parentChoice-1)
	end := min(len(choices)+1, first+3)
	for i := first; i < end; i++ {
		label := "Scene root"
		if i > 0 {
			label = choices[i-1].Name + " [" + choices[i-1].ID + "]"
		}
		writeChoice(&body, label, i == m.parentChoice)
	}
	body.WriteString("Enter apply | Esc cancel")
	return editorDialog{body: body.String(), listRow: 1, first: first, end: end,
		buttons: []editorButton{{"Apply", "enter"}, {"Cancel", "esc"}, {"Previous", "up"}, {"Next", "down"}}}
}

func (m projectEditorModel) buildControls() []editorButton {
	if m.build.cancel != nil {
		buttons := []editorButton{{"Cancel", "esc"}}
		if m.modal == "child" {
			buttons = []editorButton{{"Stop", "esc"}, {"Restart", "f6"}}
		}
		return buttons
	}
	if m.modal == "export" {
		return []editorButton{{"Export again", "f9"}, {"Close", "esc"}}
	}
	return []editorButton{{"Build", "ctrl+b"}, {"Run", "f6"}, {"Close", "esc"}}
}

func (m projectEditorModel) dialogLayout() (string, []editorHit) {
	d := m.dialog()
	width := max(1, m.width-1)
	height := max(1, m.height)
	var lines []string
	var hits []editorHit
	addButtons := func(buttons []editorButton, maxRows int) {
		var line string
		start := len(lines)
		for _, b := range buttons {
			label := "[" + b.label + "]"
			size := runewidth.StringWidth(label)
			if size > width {
				continue
			}
			x := runewidth.StringWidth(line)
			if x != 0 && x+1+size > width {
				lines = append(lines, line)
				line, x = "", 0
			}
			if len(lines)-start >= maxRows {
				break
			}
			if x != 0 {
				line += " "
				x++
			}
			hits = append(hits, editorHit{x: x, y: len(lines), width: size, button: b})
			line += label
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	// Controls are above the content so Cancel stays reachable on a short screen.
	addButtons(d.buttons, max(1, height/2))
	body := strings.Split(strings.Trim(d.body, "\n"), "\n")
	if m.keyboard && hasEditorButton(d.buttons, "ui.keyboard") {
		// Show the draft first, followed by a paged ASCII keyboard. Hardware
		// input and pasted text also support Unicode in authored values.
		for _, line := range body[:min(2, len(body))] {
			lines = append(lines, line)
		}
		chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
		capacity := max(1, width/4) * max(1, height-len(lines)-3)
		pages := (len(chars) + capacity - 1) / capacity
		page := m.keyPage % pages
		addButtons([]editorButton{{"More keys", "ui.next_keys"}, {"Backspace", "backspace"}, {"Space", "type: "}}, 1)
		var keys []editorButton
		for _, r := range chars[page*capacity : min(len(chars), (page+1)*capacity)] {
			keys = append(keys, editorButton{string(r), "type:" + string(r)})
		}
		addButtons(keys, max(0, height-len(lines)-1))
	} else {
		offset := len(lines)
		lines = append(lines, body...)
		for i := d.first; i < d.end && d.listRow+i-d.first < len(body); i++ {
			row := d.listRow + i - d.first
			hits = append(hits, editorHit{y: offset + row, width: min(width, runewidth.StringWidth(body[row])), row: i + 1})
		}
	}
	// Keep asynchronous build errors and validation messages visible.
	if m.modal != "build" && m.modal != "child" && m.modal != "export" && m.message != "" {
		lines = append(lines[:min(len(lines), max(0, height-1))], printableEditorText(m.message))
		hits = removeHiddenHits(hits, height-1)
	}
	return fitView(strings.Join(lines, "\n"), m.width, m.height), hits
}

func removeHiddenHits(hits []editorHit, row int) []editorHit {
	visible := hits[:0]
	for _, hit := range hits {
		if hit.y < row {
			visible = append(visible, hit)
		}
	}
	return visible
}

func hasEditorButton(buttons []editorButton, key string) bool {
	for _, b := range buttons {
		if b.key == key {
			return true
		}
	}
	return false
}

func (m projectEditorModel) dialogMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.X < 0 || msg.Y < 0 || msg.X >= m.width-1 || msg.Y >= m.height {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.handleEditorKey(tea.KeyMsg{}, "up")
	case tea.MouseButtonWheelDown:
		return m.handleEditorKey(tea.KeyMsg{}, "down")
	case tea.MouseButtonLeft:
		_, hits := m.dialogLayout()
		for _, hit := range hits {
			if msg.Y != hit.y || msg.X < hit.x || msg.X >= hit.x+hit.width {
				continue
			}
			if hit.row != 0 {
				return m.selectDialogRow(hit.row - 1)
			}
			key := hit.button.key
			if value, ok := strings.CutPrefix(key, "choice:"); ok {
				m.draft = value
				key = "enter"
			}
			if value, ok := strings.CutPrefix(key, "type:"); ok {
				// Pass the rune as text, even for a printable key that also
				// names a command in another part of the editor.
				return m.handleEditorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}, "ui.type")
			}
			return m.handleEditorKey(tea.KeyMsg{}, key)
		}
	}
	return m, nil
}

func (m projectEditorModel) selectDialogRow(index int) (tea.Model, tea.Cmd) {
	switch m.modal {
	case "commands":
		m.actionSelection = index
		return m.commandKey("enter")
	case "files":
		m.fileSelection = index
		return m.fileKey("enter")
	case "parent":
		m.parentChoice = index
	case "bindings":
		m.bindingSelected = index
	case "tools":
		m.toolField = index
	case "assets":
		m.assetSelected = index
	case "mesh_assets":
		m.meshSelection = index
	case "sound_assets":
		m.soundSelection = index
	}
	return m, nil
}
