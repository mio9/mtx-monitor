package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"mio9/mtx-monitor/internal/config"
)

func TestQuitOnSingleQ(t *testing.T) {
	app, err := NewModel(config.Config{})
	if err != nil {
		t.Fatal(err)
	}

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q did not quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q did not return quit")
	}

	_, cmd = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("second q did not quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("second q did not return quit")
	}

	app.Update(tea.QuitMsg{})
}
