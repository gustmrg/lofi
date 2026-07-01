package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gustmrg/lofi/internal/player"
	"github.com/gustmrg/lofi/internal/provider"
	"github.com/gustmrg/lofi/internal/provider/mock"
	"github.com/gustmrg/lofi/internal/store"
)

type fakeStationManager struct {
	removeErr   error
	removeCalls int
	removedID   string
}

func (*fakeStationManager) AddByURL(context.Context, string) (provider.Station, error) {
	return provider.Station{}, nil
}

func (f *fakeStationManager) Remove(_ context.Context, id string) error {
	f.removeCalls++
	f.removedID = id
	return f.removeErr
}

func newTestModel(t *testing.T) *Model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return newTestModelFromEnv(t)
}

func newTestModelFromEnv(t *testing.T) *Model {
	t.Helper()
	m, err := NewModel(mock.New(), player.Noop{})
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	return m
}

func TestDefaultVolume(t *testing.T) {
	m := newTestModel(t)
	if m.volume != defaultVolume {
		t.Fatalf("default volume = %d, want %d", m.volume, defaultVolume)
	}
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected missing config to be created")
	}
	if cfg.Volume != defaultVolume {
		t.Fatalf("created config volume = %d, want %d", cfg.Volume, defaultVolume)
	}
}

func TestSavedVolumeLoaded(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := store.SaveConfig(store.Config{Volume: 45}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	m := newTestModelFromEnv(t)
	if m.volume != 45 {
		t.Fatalf("loaded volume = %d, want 45", m.volume)
	}
}

func TestSavedVolumeClamped(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := store.SaveConfig(store.Config{Volume: 145}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	m := newTestModelFromEnv(t)
	if m.volume != 100 {
		t.Fatalf("loaded volume = %d, want 100", m.volume)
	}
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config to exist")
	}
	if cfg.Volume != 100 {
		t.Fatalf("rewritten config volume = %d, want 100", cfg.Volume)
	}
}

func TestMalformedConfigFallsBackToDefaultVolume(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".lofi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	m := newTestModelFromEnv(t)
	if m.volume != defaultVolume {
		t.Fatalf("volume after malformed config = %d, want %d", m.volume, defaultVolume)
	}
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after rewrite: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected malformed config to be rewritten")
	}
	if cfg.Volume != defaultVolume {
		t.Fatalf("rewritten config volume = %d, want %d", cfg.Volume, defaultVolume)
	}
}

func TestVolumeChangePersists(t *testing.T) {
	m := newTestModel(t)
	m = sendSpecial(t, m, tea.KeyRight)
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config to be saved")
	}
	if cfg.Volume != defaultVolume+5 {
		t.Fatalf("saved volume = %d, want %d", cfg.Volume, defaultVolume+5)
	}
}

func TestPlayerEventsUpdateConnectionHealth(t *testing.T) {
	m := newTestModel(t)
	m.loading = false

	updated, _ := m.Update(playerEventMsg{event: player.Event{Kind: player.EventHealthy}})
	m = updated.(*Model)
	if m.health != healthHealthy {
		t.Fatalf("healthy event health = %v, want %v", m.health, healthHealthy)
	}

	updated, _ = m.Update(playerEventMsg{event: player.Event{Kind: player.EventUnstable, Detail: "buffering"}})
	m = updated.(*Model)
	if m.health != healthUnstable {
		t.Fatalf("unstable event health = %v, want %v", m.health, healthUnstable)
	}

	updated, _ = m.Update(playerEventMsg{event: player.Event{Kind: player.EventReconnecting}})
	m = updated.(*Model)
	if m.health != healthReconnecting {
		t.Fatalf("reconnecting event health = %v, want %v", m.health, healthReconnecting)
	}

	updated, _ = m.Update(playerEventMsg{event: player.Event{Kind: player.EventDisconnected, Detail: "stream ended"}})
	m = updated.(*Model)
	if m.health != healthDisconnected {
		t.Fatalf("disconnected event health = %v, want %v", m.health, healthDisconnected)
	}
	if m.lastError != "stream ended" {
		t.Fatalf("lastError = %q, want stream ended", m.lastError)
	}
}

func TestPlayerStartedKeepsConnecting(t *testing.T) {
	m := newTestModel(t)
	m.health = healthReconnecting
	updated, _ := m.Update(playerStartedMsg{})
	m = updated.(*Model)
	if m.health != healthReconnecting {
		t.Fatalf("health = %v, want %v", m.health, healthReconnecting)
	}
}

func TestConnectingTimesOutWhenPlaybackNeverStarts(t *testing.T) {
	m := newTestModel(t)
	m.loading = false
	m.playing = true
	m.health = healthReconnecting
	now := time.Now()
	m.healthSince = now.Add(-streamStartTimeout - time.Second)
	updated, _ := m.Update(tickMsg(now))
	m = updated.(*Model)
	if m.health != healthDisconnected {
		t.Fatalf("health = %v, want %v", m.health, healthDisconnected)
	}
	if m.lastError != "stream did not start; mpv never reported playback" {
		t.Fatalf("lastError = %q, want stream timeout", m.lastError)
	}
}

func TestDisconnectedPlayerEventDuringLoadKeepsReconnecting(t *testing.T) {
	m := newTestModel(t)
	m.loading = true
	m.health = healthReconnecting
	updated, _ := m.Update(playerEventMsg{event: player.Event{Kind: player.EventDisconnected, Detail: "stop"}})
	m = updated.(*Model)
	if m.health != healthReconnecting {
		t.Fatalf("health = %v, want %v", m.health, healthReconnecting)
	}
	if m.lastError != "" {
		t.Fatalf("lastError = %q, want empty", m.lastError)
	}
}

func TestStatusBadgeTextSameLength(t *testing.T) {
	badges := []string{
		statusBadgeText("ok healthy"),
		statusBadgeText("~ unstable"),
		statusBadgeText("^ unstable"),
		statusBadgeText(".   connecting"),
		statusBadgeText("..  connecting"),
		statusBadgeText("... connecting"),
		statusBadgeText("! disconnected"),
		statusBadgeText("- paused"),
	}
	want := len(badges[0])
	for _, badge := range badges[1:] {
		if len(badge) != want {
			t.Fatalf("badge %q length = %d, want %d", badge, len(badge), want)
		}
	}
}

func TestConnectingStatusBadgeLabelsAnimate(t *testing.T) {
	cases := []struct {
		frame int
		want  string
	}{
		{0, ".   connecting"},
		{1, "..  connecting"},
		{2, "... connecting"},
		{3, ".   connecting"},
	}
	for _, tc := range cases {
		got := statusBadgeLabel(healthReconnecting, tc.frame)
		if got != tc.want {
			t.Fatalf("frame %d label = %q, want %q", tc.frame, got, tc.want)
		}
	}
}

func TestUnstableStatusBadgeLabelsAnimate(t *testing.T) {
	cases := []struct {
		frame int
		want  string
	}{
		{0, "~ unstable"},
		{1, "^ unstable"},
		{2, "~ unstable"},
	}
	for _, tc := range cases {
		got := statusBadgeLabel(healthUnstable, tc.frame)
		if got != tc.want {
			t.Fatalf("frame %d label = %q, want %q", tc.frame, got, tc.want)
		}
	}
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
	m.manager = &fakeStationManager{}
	m = sendString(t, m, "a")
	if m.mode != modeAddStation {
		t.Fatal("expected add station mode")
	}
	return m
}

func enterDeleteConfirmMode(t *testing.T, m *Model, fm *fakeStationManager) *Model {
	t.Helper()
	m.manager = fm
	m = sendString(t, m, "d")
	if m.mode != modeConfirmDelete {
		t.Fatal("expected delete confirmation mode")
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
	m.streamStarted = true
	m = sendSpecial(t, m, tea.KeyDown)
	if m.elapsed != 0 {
		t.Fatalf("switching station should reset elapsed, got %v", m.elapsed)
	}
}

func TestUpdateNoticeMessage(t *testing.T) {
	m := newTestModel(t)
	cmd := updateNoticeCmd(func(context.Context) (string, error) {
		return "New version available: v0.2.0 (you are using v0.1.0). Run 'lofi update' to update.", nil
	})
	updated, _ := m.Update(cmd())
	m = updated.(*Model)
	if !strings.Contains(m.updateInfo, "v0.2.0") {
		t.Fatalf("updateInfo = %q", m.updateInfo)
	}
}

func TestPlaybackErrorTakesPrecedenceOverUpdateNotice(t *testing.T) {
	m := newTestModel(t)
	m.updateInfo = "New version available: v0.2.0 (you are using v0.1.0). Run 'lofi update' to update."
	m.lastError = "player failed"
	view := m.renderBackground()
	if !strings.Contains(view, "player failed") {
		t.Fatalf("view should contain playback error:\n%s", view)
	}
	if strings.Contains(view, "New version available") {
		t.Fatalf("view should hide update notice while an error is present:\n%s", view)
	}
}
