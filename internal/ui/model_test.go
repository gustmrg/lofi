package ui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/gustmrg/lofi/internal/player"
	"github.com/gustmrg/lofi/internal/provider"
	"github.com/gustmrg/lofi/internal/provider/mock"
)

type fakeStationManager struct{}

func (fakeStationManager) AddByURL(context.Context, string) (provider.Station, error) {
	return provider.Station{}, nil
}

func (fakeStationManager) Remove(context.Context, string) error {
	return nil
}

func newTestModel(t *testing.T) *Model {
	t.Helper()
	m, err := NewModel(mock.New(), player.Noop{})
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	return m
}

func sendString(t *testing.T, m *Model, s string) *Model {
	t.Helper()
	msg := tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	updated, _ := m.Update(msg)
	return updated.(*Model)
}

func sendSpecial(t *testing.T, m *Model, code rune) *Model {
	t.Helper()
	msg := tea.KeyPressMsg{Code: code}
	updated, _ := m.Update(msg)
	return updated.(*Model)
}

func enterAddMode(t *testing.T, m *Model) *Model {
	t.Helper()
	m.manager = fakeStationManager{}
	m = sendString(t, m, "a")
	if m.mode != modeAddStation {
		t.Fatal("expected add station mode")
	}
	return m
}

func TestPlayPauseToggle(t *testing.T) {
	m := newTestModel(t)
	if !m.playing {
		t.Fatal("expected initial playing=true")
	}
	m = sendSpecial(t, m, tea.KeySpace)
	if m.playing {
		t.Fatal("space should toggle to paused")
	}
	m = sendSpecial(t, m, tea.KeySpace)
	if !m.playing {
		t.Fatal("space should toggle back to playing")
	}
}

func TestBrowseStationsWrap(t *testing.T) {
	m := newTestModel(t)
	last := len(m.stations) - 1
	m = sendSpecial(t, m, tea.KeyUp)
	if m.activeIdx != last {
		t.Fatalf("up from 0 should wrap to %d, got %d", last, m.activeIdx)
	}
	m = sendSpecial(t, m, tea.KeyDown)
	if m.activeIdx != 0 {
		t.Fatalf("down from %d should wrap to 0, got %d", last, m.activeIdx)
	}
}

func TestStationKeys(t *testing.T) {
	m := newTestModel(t)
	m = sendString(t, m, "3")
	if m.activeIdx != 2 {
		t.Fatalf("key '3' should set activeIdx=2, got %d", m.activeIdx)
	}
}

func TestVolumeClamp(t *testing.T) {
	m := newTestModel(t)
	m.volume = 98
	m = sendSpecial(t, m, tea.KeyRight)
	if m.volume != 100 {
		t.Fatalf("vol up from 98 should clamp to 100, got %d", m.volume)
	}
	m.volume = 3
	m = sendSpecial(t, m, tea.KeyLeft)
	if m.volume != 0 {
		t.Fatalf("vol down from 3 should clamp to 0, got %d", m.volume)
	}
}

func TestMuteToggle(t *testing.T) {
	m := newTestModel(t)
	if m.muted {
		t.Fatal("expected initial muted=false")
	}
	m = sendString(t, m, "m")
	if !m.muted {
		t.Fatal("m should mute")
	}
	m = sendString(t, m, "m")
	if m.muted {
		t.Fatal("m should unmute")
	}
}

func TestVolumeKeyUnmutes(t *testing.T) {
	m := newTestModel(t)
	m.muted = true
	m = sendSpecial(t, m, tea.KeyRight)
	if m.muted {
		t.Fatal("volume key should unmute")
	}
}

func TestSwitchingStationResetsElapsed(t *testing.T) {
	m := newTestModel(t)
	m.elapsed = 90_000_000_000
	m = sendSpecial(t, m, tea.KeyDown)
	if m.elapsed != 0 {
		t.Fatalf("switching station should reset elapsed, got %v", m.elapsed)
	}
}

func TestAddStationTypingUpdatesInput(t *testing.T) {
	m := enterAddMode(t, newTestModel(t))
	m = sendString(t, m, "h")
	m = sendString(t, m, "t")
	if got, want := m.input.Value(), "ht"; got != want {
		t.Fatalf("input value = %q, want %q", got, want)
	}
}

func TestAddStationBracketedPasteUpdatesInput(t *testing.T) {
	m := enterAddMode(t, newTestModel(t))
	url := "https://www.youtube.com/watch?v=jfKfPfyJRdk"
	updated, _ := m.Update(tea.PasteMsg{Content: url})
	m = updated.(*Model)
	if got := m.input.Value(); got != url {
		t.Fatalf("input value = %q, want %q", got, url)
	}
}

func TestAddStationCancelResetsInput(t *testing.T) {
	m := enterAddMode(t, newTestModel(t))
	m = sendString(t, m, "x")
	m = sendSpecial(t, m, tea.KeyEsc)
	if m.mode != modeNormal {
		t.Fatal("expected normal mode after cancel")
	}
	if got := m.input.Value(); got != "" {
		t.Fatalf("input value after cancel = %q, want empty", got)
	}
}
