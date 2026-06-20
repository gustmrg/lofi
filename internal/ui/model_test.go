package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
	if m.streamStarted {
		t.Fatal("switching station should reset streamStarted to false")
	}
}

func TestTimerDoesNotAdvanceBeforePlaybackStarts(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(trackResolvedMsg{idx: m.activeIdx, track: provider.Track{Title: "t", Artist: "a", Duration: 4 * time.Minute}})
	m = updated.(*Model)
	if m.streamStarted {
		t.Fatal("expected streamStarted=false after resolve")
	}
	if !m.playing {
		t.Fatal("expected playing=true after resolve")
	}

	t0 := time.Now()
	updated, _ = m.Update(tickMsg(t0))
	m = updated.(*Model)
	updated, _ = m.Update(tickMsg(t0.Add(tickInterval)))
	m = updated.(*Model)
	if m.elapsed != 0 {
		t.Fatalf("elapsed = %v, want 0 before playback starts", m.elapsed)
	}
}

func TestEventHealthyStartsTimerAndResetsElapsed(t *testing.T) {
	m := newTestModel(t)
	m.playing = true
	m.streamStarted = false
	m.elapsed = 5 * time.Second

	updated, _ := m.Update(playerEventMsg{event: player.Event{Kind: player.EventHealthy}})
	m = updated.(*Model)
	if !m.streamStarted {
		t.Fatal("expected streamStarted=true after healthy event")
	}
	if m.elapsed != 0 {
		t.Fatalf("elapsed = %v, want 0 reset on stream start", m.elapsed)
	}

	t0 := time.Now()
	updated, _ = m.Update(tickMsg(t0))
	m = updated.(*Model)
	updated, _ = m.Update(tickMsg(t0.Add(tickInterval)))
	m = updated.(*Model)
	if m.elapsed <= 0 {
		t.Fatalf("elapsed = %v, want >0 after tick once started", m.elapsed)
	}
}

func TestSubsequentHealthyDoesNotResetElapsed(t *testing.T) {
	m := newTestModel(t)
	m.playing = true
	m.streamStarted = true
	m.elapsed = 10 * time.Second

	updated, _ := m.Update(playerEventMsg{event: player.Event{Kind: player.EventHealthy}})
	m = updated.(*Model)
	if m.elapsed != 10*time.Second {
		t.Fatalf("elapsed = %v, want 10s (not reset by subsequent healthy)", m.elapsed)
	}
}

func TestDisconnectMidPlaybackFreezesTimer(t *testing.T) {
	m := newTestModel(t)
	m.loading = false
	m.playing = true
	m.streamStarted = true
	m.elapsed = 30 * time.Second

	updated, _ := m.Update(playerEventMsg{event: player.Event{Kind: player.EventDisconnected, Detail: "stream ended"}})
	m = updated.(*Model)
	if m.streamStarted {
		t.Fatal("expected streamStarted=false after disconnect")
	}

	t0 := time.Now()
	updated, _ = m.Update(tickMsg(t0))
	m = updated.(*Model)
	updated, _ = m.Update(tickMsg(t0.Add(tickInterval)))
	m = updated.(*Model)
	if m.elapsed != 30*time.Second {
		t.Fatalf("elapsed = %v, want 30s (frozen after disconnect)", m.elapsed)
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

func TestDeleteKeyOpensConfirmation(t *testing.T) {
	m := newTestModel(t)
	fm := &fakeStationManager{}
	m = enterDeleteConfirmMode(t, m, fm)
	if fm.removeCalls != 0 {
		t.Fatalf("remove calls = %d, want 0 before confirmation", fm.removeCalls)
	}
}

func TestDeleteConfirmationCancelDoesNotRemove(t *testing.T) {
	m := newTestModel(t)
	fm := &fakeStationManager{}
	m = enterDeleteConfirmMode(t, m, fm)
	m = sendSpecial(t, m, tea.KeyEsc)
	if m.mode != modeNormal {
		t.Fatal("expected normal mode after cancel")
	}
	if fm.removeCalls != 0 {
		t.Fatalf("remove calls = %d, want 0", fm.removeCalls)
	}
}

func TestDeleteConfirmationEnterRemovesActiveStation(t *testing.T) {
	m := newTestModel(t)
	fm := &fakeStationManager{}
	m = enterDeleteConfirmMode(t, m, fm)
	wantID := m.stations[m.activeIdx].ID
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected remove command")
	}
	msg := cmd()
	if _, ok := msg.(stationRemovedMsg); !ok {
		t.Fatalf("remove command returned %T, want stationRemovedMsg", msg)
	}
	if fm.removeCalls != 1 {
		t.Fatalf("remove calls = %d, want 1", fm.removeCalls)
	}
	if fm.removedID != wantID {
		t.Fatalf("removed id = %q, want %q", fm.removedID, wantID)
	}
}

func TestDeleteDisabledWithSingleStation(t *testing.T) {
	m := newTestModel(t)
	fm := &fakeStationManager{}
	m.manager = fm
	m.stations = m.stations[:1]
	m = sendString(t, m, "d")
	if m.mode != modeNotice {
		t.Fatal("expected delete notice with one station")
	}
	if m.noticeTitle != "CANNOT DELETE" {
		t.Fatalf("notice title = %q, want CANNOT DELETE", m.noticeTitle)
	}
	if m.noticeText != "At least one station must exist." {
		t.Fatalf("notice text = %q, want last-station message", m.noticeText)
	}
	if fm.removeCalls != 0 {
		t.Fatalf("remove calls = %d, want 0", fm.removeCalls)
	}
}

func TestDeleteNoticeCanBeClosed(t *testing.T) {
	m := newTestModel(t)
	m.manager = &fakeStationManager{}
	m.stations = m.stations[:1]
	m = sendString(t, m, "d")
	m = sendSpecial(t, m, tea.KeyEsc)
	if m.mode != modeNormal {
		t.Fatal("expected normal mode after closing notice")
	}
	if m.noticeTitle != "" || m.noticeText != "" {
		t.Fatalf("notice state not cleared: title=%q text=%q", m.noticeTitle, m.noticeText)
	}
}

func TestDeleteErrorKeepsConfirmationOpen(t *testing.T) {
	m := newTestModel(t)
	fm := &fakeStationManager{removeErr: errors.New("remove failed")}
	m = enterDeleteConfirmMode(t, m, fm)
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatal("expected remove command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(*Model)
	if m.mode != modeConfirmDelete {
		t.Fatal("expected confirmation mode after remove error")
	}
	if m.removeError != "remove failed" {
		t.Fatalf("remove error = %q, want remove failed", m.removeError)
	}
}

func TestVisualizerStaysInRange(t *testing.T) {
	m := newTestModel(t)
	m.playing = true
	m.streamStarted = true
	t0 := time.Now()
	for i := 0; i < 100; i++ {
		updated, _ := m.Update(tickMsg(t0.Add(time.Duration(i) * tickInterval)))
		m = updated.(*Model)
	}
	for i, h := range m.visualizer {
		if h < 0 || h > 7 {
			t.Fatalf("visualizer[%d] = %d, want 0..7", i, h)
		}
	}
}

func TestVisualizerFlatWhenNotPlaying(t *testing.T) {
	m := newTestModel(t)
	m.playing = false
	m.streamStarted = true
	updated, _ := m.Update(tickMsg(time.Now()))
	m = updated.(*Model)
	for i, h := range m.visualizer {
		if h != 0 {
			t.Fatalf("visualizer[%d] = %d, want 0 when not playing", i, h)
		}
	}
}

func TestVisualizerFlatBeforeStreamStart(t *testing.T) {
	m := newTestModel(t)
	m.playing = true
	m.streamStarted = false
	updated, _ := m.Update(tickMsg(time.Now()))
	m = updated.(*Model)
	for i, h := range m.visualizer {
		if h != 0 {
			t.Fatalf("visualizer[%d] = %d, want 0 before stream start", i, h)
		}
	}
}
