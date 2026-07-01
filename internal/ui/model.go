package ui

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	version "github.com/gustmrg/lofi"
	"github.com/gustmrg/lofi/internal/player"
	"github.com/gustmrg/lofi/internal/provider"
	"github.com/gustmrg/lofi/internal/store"
	"github.com/gustmrg/lofi/internal/updater"
)

const (
	visualizerMaxBars  = 256
	visualizerMaxLevel = 7.0
	visualizerGravity  = 0.92
	visualizerEase     = 0.3
	visualizerImpulse  = 0.35
	tickInterval       = 80 * time.Millisecond
	resolveTimeout     = 15 * time.Second
	streamStartTimeout = 20 * time.Second
	defaultVolume      = 70
)

type uiMode int

const (
	modeNormal uiMode = iota
	modeAddStation
	modeConfirmDelete
	modeNotice
)

type connectionHealth int

const (
	healthHealthy connectionHealth = iota
	healthUnstable
	healthReconnecting
	healthDisconnected
)

type tickMsg time.Time

type trackResolvedMsg struct {
	idx   int
	track provider.Track
}

type trackErrorMsg struct {
	idx int
	err error
}

type playerErrorMsg struct {
	err error
}

type playerStartedMsg struct{}

type playerEventMsg struct {
	event player.Event
}

type updateNoticeMsg struct {
	notice string
}

type stationAddedMsg struct {
	station provider.Station
}

type stationRemovedMsg struct {
	removedIdx int
}

type addErrorMsg struct {
	err error
}

type removeErrorMsg struct {
	err error
}

type Model struct {
	prov          provider.Provider
	manager       provider.StationManager
	player        player.Player
	stations      []provider.Station
	activeIdx     int
	track         provider.Track
	loading       bool
	lastError     string
	elapsed       time.Duration
	playing       bool
	volume        int
	muted         bool
	visualizer    [visualizerMaxBars]int
	width         int
	height        int
	keys          keyMap
	rng           *rand.Rand
	lastTick      time.Time
	updateInfo    string
	checkUpdate   func(context.Context) (string, error)
	streamStarted bool
	health        connectionHealth
	healthSince   time.Time
	beatPhase     float64
	visCurr       [visualizerMaxBars]float64
	visTarget     [visualizerMaxBars]float64

	mode     uiMode
	input    textinput.Model
	adding   bool
	addError string

	removing    bool
	removeError string
	noticeTitle string
	noticeText  string
}

func NewModel(p provider.Provider, pl player.Player) (*Model, error) {
	stations, err := p.Stations(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load stations: %w", err)
	}
	if len(stations) == 0 {
		return nil, fmt.Errorf("provider returned no stations")
	}

	mgr, _ := p.(provider.StationManager)

	ti := textinput.New()
	ti.Placeholder = "https://youtube.com/watch?v=..."
	ti.CharLimit = 256
	ti.SetWidth(56)
	checker := updater.DefaultNoticeChecker(version.Current())

	volume, _ := loadVolume()

	m := &Model{
		prov:        p,
		manager:     mgr,
		player:      pl,
		stations:    stations,
		volume:      volume,
		playing:     true,
		loading:     true,
		health:      healthReconnecting,
		healthSince: time.Now(),
		keys:        defaultKeys(),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		track:       provider.Track{Title: stations[0].Name, Artist: "loading…"},
		checkUpdate: checker.Notice,
		input:       ti,
	}
	return m, nil
}

func loadVolume() (int, error) {
	cfg, err := store.LoadConfig()
	if err != nil || cfg == nil {
		return defaultVolume, store.SaveConfig(store.Config{Volume: defaultVolume})
	}

	volume := clampInt(cfg.Volume, 0, 100)
	if volume != cfg.Volume {
		return volume, store.SaveConfig(store.Config{Volume: volume})
	}
	return volume, nil
}

func (m *Model) loadTrack(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.stations) {
		return nil
	}
	m.activeIdx = idx
	m.loading = true
	m.streamStarted = false
	m.health = healthReconnecting
	m.healthSince = time.Now()
	m.lastError = ""
	m.elapsed = 0
	m.track = provider.Track{Title: m.stations[idx].Name, Artist: "loading…"}
	return tea.Batch(
		stopCmd(m.player),
		resolveCmd(m.prov, idx, m.stations[idx]),
	)
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		volumeCmd(m.player, m.volume),
		resolveCmd(m.prov, 0, m.stations[0]),
		updateNoticeCmd(m.checkUpdate),
	)
}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func resolveCmd(p provider.Provider, idx int, s provider.Station) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
		defer cancel()
		t, err := p.Resolve(ctx, s)
		if err != nil {
			return trackErrorMsg{idx: idx, err: err}
		}
		return trackResolvedMsg{idx: idx, track: t}
	}
}

func addStationCmd(sm provider.StationManager, url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
		defer cancel()
		s, err := sm.AddByURL(ctx, url)
		if err != nil {
			return addErrorMsg{err: err}
		}
		return stationAddedMsg{station: s}
	}
}

func removeStationCmd(sm provider.StationManager, id string, idx int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := sm.Remove(ctx, id); err != nil {
			return removeErrorMsg{err: err}
		}
		return stationRemovedMsg{removedIdx: idx}
	}
}

func playCmd(pl player.Player, url string) tea.Cmd {
	return func() tea.Msg {
		if err := pl.Play(url); err != nil {
			return playerErrorMsg{err: err}
		}
		return playerStartedMsg{}
	}
}

func pauseCmd(pl player.Player, paused bool) tea.Cmd {
	return func() tea.Msg {
		if err := pl.Pause(paused); err != nil {
			return playerErrorMsg{err: err}
		}
		return nil
	}
}

func volumeCmd(pl player.Player, v int) tea.Cmd {
	return func() tea.Msg {
		if err := pl.SetVolume(v); err != nil {
			return playerErrorMsg{err: err}
		}
		return nil
	}
}

func playerEventCmd(pl player.Player) tea.Cmd {
	return func() tea.Msg {
		ch := pl.Events()
		if ch == nil {
			return nil
		}
		event, ok := <-ch
		if !ok {
			return nil
		}
		return playerEventMsg{event: event}
	}
}

func (m *Model) persistVolume() {
	if err := store.SaveConfig(store.Config{Volume: m.volume}); err != nil {
		m.lastError = fmt.Sprintf("save config: %v", err)
	}
}

func (m *Model) applyPlayerEvent(event player.Event) {
	switch event.Kind {
	case player.EventHealthy:
		if m.playing {
			if !m.streamStarted {
				m.streamStarted = true
				m.elapsed = 0
			}
			m.health = healthHealthy
			m.healthSince = time.Now()
		}
	case player.EventUnstable:
		if m.playing {
			m.health = healthUnstable
			m.healthSince = time.Now()
		}
	case player.EventReconnecting:
		m.health = healthReconnecting
		m.healthSince = time.Now()
	case player.EventDisconnected:
		if m.loading {
			m.health = healthReconnecting
			return
		}
		m.streamStarted = false
		m.health = healthDisconnected
		m.healthSince = time.Now()
		if msg := playerEventMessage(event); msg != "" {
			m.lastError = msg
		}
	}
}

func stopCmd(pl player.Player) tea.Cmd {
	return func() tea.Msg {
		if err := pl.Stop(); err != nil {
			return playerErrorMsg{err: err}
		}
		return nil
	}
}

func updateNoticeCmd(check func(context.Context) (string, error)) tea.Cmd {
	if check == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		notice, err := check(ctx)
		if err != nil || notice == "" {
			return nil
		}
		return updateNoticeMsg{notice: notice}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		if !m.lastTick.IsZero() && m.playing && m.streamStarted {
			delta := now.Sub(m.lastTick)
			m.elapsed += delta
			if m.track.Duration > 0 && m.elapsed >= m.track.Duration {
				m.elapsed = 0
			}
		}
		if m.playing && !m.loading && m.health == healthReconnecting && !m.healthSince.IsZero() && now.Sub(m.healthSince) > streamStartTimeout {
			m.health = healthDisconnected
			m.healthSince = now
			m.lastError = "stream did not start; mpv never reported playback"
		}
		m.lastTick = now
		m.stepVisualizer(now)
		return m, tick()

	case trackResolvedMsg:
		if msg.idx != m.activeIdx {
			return m, nil
		}
		m.track = msg.track
		m.loading = false
		m.playing = true
		m.streamStarted = false
		m.health = healthReconnecting
		m.healthSince = time.Now()
		m.elapsed = 0
		return m, playCmd(m.player, msg.track.StreamURL)

	case trackErrorMsg:
		if msg.idx != m.activeIdx {
			return m, nil
		}
		m.loading = false
		m.health = healthDisconnected
		m.healthSince = time.Now()
		m.lastError = providerErrorMessage(msg.err)
		return m, nil

	case playerStartedMsg:
		return m, playerEventCmd(m.player)

	case playerEventMsg:
		m.applyPlayerEvent(msg.event)
		return m, playerEventCmd(m.player)

	case playerErrorMsg:
		if msg.err != nil {
			m.health = healthDisconnected
			m.healthSince = time.Now()
			m.lastError = playerErrorMessage(msg.err)
			m.playing = false
		}
		return m, nil

	case updateNoticeMsg:
		m.updateInfo = msg.notice
		return m, nil

	case stationAddedMsg:
		m.adding = false
		m.addError = ""
		m.mode = modeNormal
		m.input.Reset()
		m.input.Blur()
		stations, err := m.prov.Stations(context.Background())
		if err == nil {
			m.stations = stations
		} else {
			m.stations = append(m.stations, msg.station)
		}
		return m, m.loadTrack(len(m.stations) - 1)

	case addErrorMsg:
		m.adding = false
		m.addError = providerErrorMessage(msg.err)
		return m, nil

	case stationRemovedMsg:
		m.removing = false
		m.removeError = ""
		m.mode = modeNormal
		stations, err := m.prov.Stations(context.Background())
		if err == nil {
			m.stations = stations
		}
		if len(m.stations) == 0 {
			return m, nil
		}
		newIdx := msg.removedIdx
		if newIdx >= len(m.stations) {
			newIdx = len(m.stations) - 1
		}
		return m, m.loadTrack(newIdx)

	case removeErrorMsg:
		m.removing = false
		m.removeError = msg.err.Error()
		return m, nil

	case tea.KeyPressMsg:
		if m.mode == modeAddStation {
			return m.handleAddKey(msg)
		}
		if m.mode == modeConfirmDelete {
			return m.handleDeleteConfirmKey(msg)
		}
		if m.mode == modeNotice {
			return m.handleNoticeKey(msg)
		}
		return m.handleKey(msg)

	case tea.PasteMsg:
		if m.mode == modeAddStation {
			return m.updateAddInput(msg)
		}
		return m, nil
	}
	if m.mode == modeAddStation {
		return m.updateAddInput(msg)
	}
	return m, nil
}

func (m *Model) updateAddInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Err != nil {
		m.addError = m.input.Err.Error()
	}
	return m, cmd
}

func (m *Model) handleAddKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = modeNormal
		m.adding = false
		m.addError = ""
		m.input.Reset()
		m.input.Blur()
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		if m.adding {
			return m, nil
		}
		url := m.input.Value()
		if url == "" {
			m.addError = "url is empty"
			return m, nil
		}
		m.adding = true
		m.addError = ""
		return m, addStationCmd(m.manager, url)
	}
	return m.updateAddInput(msg)
}

func (m *Model) handleDeleteConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = modeNormal
		m.removing = false
		m.removeError = ""
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		if m.removing || m.manager == nil || len(m.stations) <= 1 {
			return m, nil
		}
		m.removing = true
		m.removeError = ""
		return m, removeStationCmd(m.manager, m.stations[m.activeIdx].ID, m.activeIdx)
	}
	return m, nil
}

func (m *Model) handleNoticeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Confirm) {
		m.mode = modeNormal
		m.noticeTitle = ""
		m.noticeText = ""
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.PlayPause):
		m.playing = !m.playing
		return m, pauseCmd(m.player, !m.playing)
	case key.Matches(msg, m.keys.Next):
		return m, m.loadTrack((m.activeIdx + 1) % len(m.stations))
	case key.Matches(msg, m.keys.Prev):
		return m, m.loadTrack((m.activeIdx - 1 + len(m.stations)) % len(m.stations))
	case key.Matches(msg, m.keys.Shuffle):
		if len(m.stations) > 1 {
			next := m.activeIdx
			for next == m.activeIdx {
				next = m.rng.Intn(len(m.stations))
			}
			return m, m.loadTrack(next)
		}
	case key.Matches(msg, m.keys.VolUp):
		m.muted = false
		m.volume = clampInt(m.volume+5, 0, 100)
		m.persistVolume()
		return m, volumeCmd(m.player, m.volume)
	case key.Matches(msg, m.keys.VolDown):
		m.muted = false
		m.volume = clampInt(m.volume-5, 0, 100)
		m.persistVolume()
		return m, volumeCmd(m.player, m.volume)
	case key.Matches(msg, m.keys.Mute):
		m.muted = !m.muted
		target := m.volume
		if m.muted {
			target = 0
		}
		return m, volumeCmd(m.player, target)
	case key.Matches(msg, m.keys.Add):
		if m.manager == nil {
			return m, nil
		}
		m.mode = modeAddStation
		m.addError = ""
		m.input.Reset()
		return m, m.input.Focus()
	case key.Matches(msg, m.keys.Delete):
		if m.manager == nil {
			return m, nil
		}
		if len(m.stations) <= 1 {
			m.mode = modeNotice
			m.noticeTitle = "CANNOT DELETE"
			m.noticeText = "At least one station must exist."
			return m, nil
		}
		m.mode = modeConfirmDelete
		m.removeError = ""
		return m, nil
	default:
		for i, b := range m.keys.Stations {
			if i >= len(m.stations) {
				break
			}
			if key.Matches(msg, b) {
				return m, m.loadTrack(i)
			}
		}
	}
	return m, nil
}

func (m *Model) stepVisualizer(now time.Time) {
	if !m.playing || !m.streamStarted {
		for i := range m.visualizer {
			m.visualizer[i] = 0
			m.visCurr[i] = 0
			m.visTarget[i] = 0
		}
		return
	}

	m.beatPhase += tickInterval.Seconds()
	beat := math.Sin(m.beatPhase*1.6)*0.5 + 0.5

	for i := range m.visualizer {
		m.visTarget[i] *= visualizerGravity

		if m.rng.Float64() < visualizerImpulse*(0.3+0.7*beat) {
			m.visTarget[i] = m.rng.Float64() * visualizerMaxLevel
		}

		m.visCurr[i] += (m.visTarget[i] - m.visCurr[i]) * visualizerEase

		h := m.visCurr[i]
		if h < 0 {
			h = 0
		}
		if h > visualizerMaxLevel {
			h = visualizerMaxLevel
		}
		m.visualizer[i] = int(h)
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func providerErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	switch provider.Category(err) {
	case provider.ErrNetwork:
		return "Connection failed while opening the stream"
	case provider.ErrDecode:
		return "Could not decode the stream"
	case provider.ErrTimeout:
		return "Timed out connecting to the stream"
	default:
		return err.Error()
	}
}

func playerErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if msg := playerCategoryMessage(player.Category(err)); msg != "" {
		return msg
	}
	return err.Error()
}

func playerEventMessage(event player.Event) string {
	if msg := playerCategoryMessage(event.Category); msg != "" {
		return msg
	}
	if event.Err != nil {
		return event.Err.Error()
	}
	return event.Detail
}

func playerCategoryMessage(category player.ErrorCategory) string {
	switch category {
	case player.ErrAudioOutput:
		return "Audio output unavailable; check PulseAudio/WSLg"
	case player.ErrNetwork:
		return "Connection failed while playing the stream"
	case player.ErrDecode:
		return "Could not decode the stream"
	case player.ErrTimeout:
		return "Timed out connecting to the stream"
	case player.ErrIPC:
		return "Audio player connection closed"
	default:
		return ""
	}
}
