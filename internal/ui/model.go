package ui

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gustmrg/lofi/internal/provider"
)

const (
	visualizerBars = 48
	tickInterval   = 80 * time.Millisecond
)

type tickMsg time.Time

type Model struct {
	prov       provider.Provider
	stations   []provider.Station
	activeIdx  int
	track      provider.Track
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
}

func NewModel(p provider.Provider) (*Model, error) {
	stations, err := p.Stations(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load stations: %w", err)
	}
	if len(stations) == 0 {
		return nil, fmt.Errorf("provider returned no stations")
	}

	m := &Model{
		prov:     p,
		stations: stations,
		volume:   72,
		playing:  true,
		keys:     defaultKeys(),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	m.loadTrack(0)
	return m, nil
}

func (m *Model) loadTrack(idx int) {
	if idx < 0 || idx >= len(m.stations) {
		return
	}
	m.activeIdx = idx
	track, err := m.prov.Resolve(context.Background(), m.stations[idx])
	if err != nil {
		track = provider.Track{Title: m.stations[idx].Name, Artist: "unknown", Duration: 4 * time.Minute}
	}
	m.track = track
	m.elapsed = 0
}

func (m *Model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
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

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.PlayPause):
		m.playing = !m.playing
	case key.Matches(msg, m.keys.Next):
		m.loadTrack((m.activeIdx + 1) % len(m.stations))
	case key.Matches(msg, m.keys.Prev):
		m.loadTrack((m.activeIdx - 1 + len(m.stations)) % len(m.stations))
	case key.Matches(msg, m.keys.Shuffle):
		if len(m.stations) > 1 {
			next := m.activeIdx
			for next == m.activeIdx {
				next = m.rng.Intn(len(m.stations))
			}
			m.loadTrack(next)
		}
	case key.Matches(msg, m.keys.VolUp):
		m.muted = false
		m.volume = clampInt(m.volume+5, 0, 100)
	case key.Matches(msg, m.keys.VolDown):
		m.muted = false
		m.volume = clampInt(m.volume-5, 0, 100)
	case key.Matches(msg, m.keys.Mute):
		m.muted = !m.muted
	default:
		for i, b := range m.keys.Stations {
			if i >= len(m.stations) {
				break
			}
			if key.Matches(msg, b) {
				m.loadTrack(i)
				return m, nil
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

