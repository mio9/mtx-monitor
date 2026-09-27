package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"mio9/mtx-monitor/internal/config"
	"mio9/mtx-monitor/internal/poll"
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

func TestSnapshotUpdatesDashboard(t *testing.T) {
	app, err := NewModel(config.Config{PollIntervalMs: 3000})
	if err != nil {
		t.Fatal(err)
	}

	updated, cmd := app.Update(poll.PollSnapshot{
		Enforced:  []poll.SessionRow{{Name: "live/cam"}},
		PollError: "paths/list failed: 401 Unauthorized",
	})
	if cmd == nil {
		t.Fatal("snapshot did not schedule the next poll")
	}

	next := updated.(*model)
	if next.state.LastUpdatedMs == nil {
		t.Fatal("snapshot did not mark the dashboard updated")
	}
	if len(next.state.Snapshot.Enforced) != 1 || next.state.Snapshot.Enforced[0].Name != "live/cam" {
		t.Fatal("snapshot rows were not applied")
	}
	if next.state.Snapshot.PollError == "" {
		t.Fatal("poll error was not applied")
	}

	_, cmd = next.Update(pollTick{})
	if cmd == nil {
		t.Fatal("poll tick did not start a poll")
	}
}
