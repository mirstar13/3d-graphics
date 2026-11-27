package app

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func examplePlayer(t *testing.T, name string) *projectPlayer {
	t.Helper()
	p, err := project.Open("../../examples/" + name)
	if err != nil {
		t.Fatal(err)
	}
	profile := exampleProfiles[name]
	m := &projectPlayer{project: p, playerID: profile.player, width: 100, height: 36,
		options: physics.DefaultOptions(), newControl: profile.controls, screens: true, menu: true}
	if profile.firstPerson {
		m.room = &switchRoom{}
	}
	if err := m.reset(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.session.Close() })
	return m
}

func clickGame(t *testing.T, m *projectPlayer, label string) {
	t.Helper()
	for row, line := range strings.Split(m.View(), "\n") {
		if col := strings.Index(line, "["+label+"]"); col >= 0 {
			m.Update(tea.MouseMsg{X: col, Y: row, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			return
		}
	}
	t.Fatalf("missing %q in %s", label, m.View())
}

func TestBankShotLossReplayAndWin(t *testing.T) {
	m := examplePlayer(t, "bank-shot")
	m.Update(editorTick(m.last.Add(time.Second)))
	if m.session.World.Stats().Tick != 0 {
		t.Fatal("main menu advanced simulation")
	}
	clickGame(t, m, "Play")
	try := func(keys []string, want game.Outcome) {
		for _, key := range keys {
			if _, err := m.control(m.session, key); err != nil {
				t.Fatal(err)
			}
		}
		for m.session.Objective().Outcome == game.Playing {
			if err := m.session.Advance(m.options.FixedStep); err != nil {
				t.Fatal(err)
			}
		}
		if m.session.Objective().Outcome != want {
			t.Fatalf("shot %v ended %s, want %s", keys, m.session.Objective().Outcome, want)
		}
	}
	try([]string{" "}, game.Lost)
	if !strings.Contains(m.View(), "Game over") {
		t.Fatal(m.View())
	}
	clickGame(t, m, "Play again")
	try([]string{"left", " "}, game.Won)
	if !strings.Contains(m.View(), "You win") {
		t.Fatal(m.View())
	}
	clickGame(t, m, "Main menu")
	if !m.menu {
		t.Fatal("missing main menu")
	}
}

func TestLiftRunLaunchPadReachesLanding(t *testing.T) {
	m := examplePlayer(t, "lift-run")
	clickGame(t, m, "Play")
	if _, err := m.control(m.session, "right"); err != nil {
		t.Fatal(err)
	}
	for m.session.Objective().Outcome == game.Playing {
		if err := m.session.Advance(m.options.FixedStep); err != nil {
			t.Fatal(err)
		}
	}
	if m.session.Objective().Outcome != game.Won {
		t.Fatal("the authored launch trajectory missed the landing")
	}
}

func TestSwitchRoomRightDragStartsOnClickAndTurns(t *testing.T) {
	m := examplePlayer(t, "switch-room")
	clickGame(t, m, "Play")
	m.Update(tea.MouseMsg{X: 15, Y: 6, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})
	m.Update(tea.MouseMsg{X: 20, Y: 4, Button: tea.MouseButtonRight, Action: tea.MouseActionMotion})
	m.Update(tea.MouseMsg{X: 20, Y: 4, Button: tea.MouseButtonRight, Action: tea.MouseActionRelease})
	m.Update(editorTick(m.last.Add(m.options.FixedStep)))
	if m.room.yaw <= 0 || m.room.pitch <= 0 || m.pointer.Captured {
		t.Fatalf("drag yaw=%g pitch=%g captured=%v", m.room.yaw, m.room.pitch, m.pointer.Captured)
	}
	m.Update(tea.MouseMsg{X: 25, Y: 9, Action: tea.MouseActionMotion})
	yaw, pitch := m.room.yaw, m.room.pitch
	m.Update(editorTick(m.last.Add(m.options.FixedStep)))
	if yaw != m.room.yaw || pitch != m.room.pitch {
		t.Fatal("movement leaked after releasing the right button")
	}
}
