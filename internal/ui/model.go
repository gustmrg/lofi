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

	"github.com/gustmrg/lofi/internal/player"
	"github.com/gustmrg/lofi/internal/provider"
)

const (
	visualizerBars = 48
	tickInterval   = 80 * time.Millisecond
	resolveTimeout = 15 * time.Second
)

type uiMode int

const (
	modeNormal uiMode = iota
	modeAddStation
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
	prov       provider.Provider
	manager    provider.StationManager
	player     player.Player
	stations   []provider.Station
	activeIdx  int
	track      provider.Track
	loading    bool
	lastError  string
	elapsed    time.Duration
	playing    bool
	volume     int
	muted      bool
	visualizer [visualizerBars]int
	width      int
	height     int
	keys       keyMap
	rng        *rand.Rand
	lastTick   time.Time

	mode     uiMode
	input    textinput.Model
	adding   bool
	addError string
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

	m := &Model{
		prov:     p,
		manager:  mgr,
		player:   pl,
		stations: stations,
		volume:   72,
		playing:  true,
		loading:  true,
		keys:     defaultKeys(),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		track:    provider.Track{Title: stations[0].Name, Artist: "loading…"},
		input:    ti,
	}
	return m, nil
}

func (m *Model) loadTrack(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.stations) {
		return nil
	}
	m.activeIdx = idx
	m.loading = true
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
		return nil
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

func stopCmd(pl player.Player) tea.Cmd {
	return func() tea.Msg {
		if err := pl.Stop(); err != nil {
			return playerErrorMsg{err: err}
		}
		return nil
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		if !m.lastTick.IsZero() && m.playing {
			delta := now.Sub(m.lastTick)
			m.elapsed += delta
			if m.track.Duration > 0 && m.elapsed >= m.track.Duration {
				m.elapsed = 0
			}
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
		m.elapsed = 0
		return m, playCmd(m.player, msg.track.StreamURL)

	case trackErrorMsg:
		if msg.idx != m.activeIdx {
			return m, nil
		}
		m.loading = false
		m.lastError = msg.err.Error()
		return m, nil

	case playerErrorMsg:
		if msg.err != nil {
			m.lastError = msg.err.Error()
			m.playing = false
		}
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
		m.addError = msg.err.Error()
		return m, nil

	case stationRemovedMsg:
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
		m.lastError = msg.err.Error()
		return m, nil

	case tea.KeyPressMsg:
		if m.mode == modeAddStation {
			return m.handleAddKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
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
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
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
		return m, volumeCmd(m.player, m.volume)
	case key.Matches(msg, m.keys.VolDown):
		m.muted = false
		m.volume = clampInt(m.volume-5, 0, 100)
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
		if m.manager == nil || len(m.stations) <= 1 {
			return m, nil
		}
		return m, removeStationCmd(m.manager, m.stations[m.activeIdx].ID, m.activeIdx)
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
	t := float64(now.UnixNano()) / 1e9
	for i := range m.visualizer {
		if !m.playing {
			m.visualizer[i] = 0
			continue
		}
		base := math.Sin(t*2.0+float64(i)*0.3)*0.5 + 0.5
		wave := math.Sin(t*1.0+float64(i)*0.15)*0.3 + 0.7
		noise := m.rng.Float64() * 0.3
		h := (base*wave + noise) * 7.0
		if h < 0 {
			h = 0
		}
		if h > 7 {
			h = 7
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
