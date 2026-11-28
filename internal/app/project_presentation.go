package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
	"golang.org/x/term"
)

type projectView struct {
	text          string
	width, height int
	freeLook      bool
}

func runAdaptivePlayer(m *projectPlayer) error {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, state)
	writer := bufio.NewWriter(os.Stdout)
	console := terminal.NewTerminalRenderer(writer, 1, 1)
	if err := console.Initialize(); err != nil {
		return err
	}
	defer console.Shutdown()
	// Bubble Tea keeps ownership of input, including Windows mouse handling.
	// The presentation worker owns ANSI output and differences between views.
	fmt.Fprint(writer, "\033[?1004h\033[?1006h\033[?1002h\033[?2004h")
	defer func() {
		fmt.Fprint(writer, "\033[?1003l\033[?1002l\033[?1006l\033[?1004l\033[?2004l")
		writer.Flush()
	}()
	var previous projectView
	var rows []string
	presenter := terminal.NewLatestFrame(func(view projectView) {
		if view.width != previous.width || view.height != previous.height {
			fmt.Fprint(writer, "\033[2J")
			rows = nil
		}
		if view.freeLook != previous.freeLook {
			if view.freeLook {
				fmt.Fprint(writer, "\033[?1002l\033[?1003h")
			} else {
				fmt.Fprint(writer, "\033[?1003l\033[?1002h")
			}
		}
		current := strings.Split(view.text, "\n")
		for y, row := range current {
			if y >= len(rows) || row != rows[y] {
				fmt.Fprintf(writer, "\033[%d;1H%s\033[K", y+1, row)
			}
		}
		if len(current) < len(rows) {
			fmt.Fprintf(writer, "\033[%d;1H\033[J", len(current)+1)
		}
		if writer.Flush() == nil {
			previous, rows = view, current
		}
	})
	defer presenter.Close()
	m.adaptive, m.budget, m.behind = true, time.Second/60, presenter.Behind
	m.present = presenter.Submit
	m.checkTerminalSize()
	m.submitView()
	_, err = tea.NewProgram(m, tea.WithoutRenderer(), tea.WithInput(os.Stdin), tea.WithReportFocus(), tea.WithMouseCellMotion()).Run()
	return err
}

func (m *projectPlayer) submitView() {
	if m.present == nil {
		return
	}
	view := projectView{m.View(), m.width, m.height, m.mouseLook && !m.project.Manifest.MouseLook.DragOnly}
	if view != m.previousView {
		m.present(view)
		m.previousView = view
	}
}

type playerResizePoll struct{}

func pollPlayerTerminal() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return playerResizePoll{} })
}

func (m *projectPlayer) checkTerminalSize() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && width > 1 && height > 1 && (width != m.width || height != m.height) {
		m.update(tea.WindowSizeMsg{Width: width, Height: height}, time.Now)
	}
}
