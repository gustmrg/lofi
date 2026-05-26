package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gustmrg/lofi/internal/provider/mock"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	m, err := NewModel(mock.New())
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	return m
}

func sendKey(t *testing.T, m *Model, r rune, k tea.KeyType) *Model {
	t.Helper()
	msg := tea.KeyMsg{Type: k}
	if k == tea.KeyRunes {
		msg.Runes = []rune{r}
	}
	updated, _ := m.Update(msg)
	return updated.(*Model)
}

func sendRune(t *testing.T, m *Model, r rune) *Model {
	t.Helper()
	return sendKey(t, m, r, tea.KeyRunes)
}

func TestPlayPauseToggle(t *testing.T) {
	m := newTestModel(t)
	if !m.playing {
		t.Fatal("expected initial playing=true")
	}
	m = sendKey(t, m, 0, tea.KeySpace)
	if m.playing {
		t.Fatal("space should toggle to paused")
	}
	m = sendKey(t, m, 0, tea.KeySpace)
	if !m.playing {
		t.Fatal("space should toggle back to playing")
	}
}

func TestNextPrevWrap(t *testing.T) {
	m := newTestModel(t)
	last := len(m.stations) - 1
	m = sendRune(t, m, 'p')
	if m.activeIdx != last {
		t.Fatalf("prev from 0 should wrap to %d, got %d", last, m.activeIdx)
	}
	m = sendRune(t, m, 'n')
	if m.activeIdx != 0 {
		t.Fatalf("next from %d should wrap to 0, got %d", last, m.activeIdx)
	}
}

func TestStationKeys(t *testing.T) {
	m := newTestModel(t)
	m = sendRune(t, m, '3')
	if m.activeIdx != 2 {
		t.Fatalf("key '3' should set activeIdx=2, got %d", m.activeIdx)
	}
}

func TestVolumeClamp(t *testing.T) {
	m := newTestModel(t)
	m.volume = 98
	m = sendKey(t, m, 0, tea.KeyUp)
	if m.volume != 100 {
		t.Fatalf("vol up from 98 should clamp to 100, got %d", m.volume)
	}
	m.volume = 3
	m = sendKey(t, m, 0, tea.KeyDown)
	if m.volume != 0 {
		t.Fatalf("vol down from 3 should clamp to 0, got %d", m.volume)
	}
}

func TestRainToggle(t *testing.T) {
	m := newTestModel(t)
	initial := m.rainOn
	m = sendRune(t, m, 'r')
	if m.rainOn == initial {
		t.Fatal("r should toggle rain")
	}
}

func TestSwitchingStationResetsElapsed(t *testing.T) {
	m := newTestModel(t)
	m.elapsed = 90_000_000_000
	m = sendRune(t, m, 'n')
	if m.elapsed != 0 {
		t.Fatalf("switching station should reset elapsed, got %v", m.elapsed)
	}
}
