package mock

import (
	"context"
	"time"

	"github.com/gustmrg/lofi/internal/provider"
)

type Provider struct {
	stations []provider.Station
	tracks   map[string]provider.Track
}

func New() *Provider {
	stations := []provider.Station{
		{
			ID:          "midnight-drift",
			Name:        "midnight drift",
			Description: "slow beats . rain ambience . tape hiss",
			Listeners:   342,
			Bitrate:     "256k",
			Source:      "mock",
			SourceRef:   "midnight-drift",
		},
		{
			ID:          "cafe-morning",
			Name:        "cafe morning",
			Description: "jazzy chords . warm keys . coffee shop",
			Listeners:   518,
			Bitrate:     "256k",
			Source:      "mock",
			SourceRef:   "cafe-morning",
		},
		{
			ID:          "code-focus",
			Name:        "code focus",
			Description: "minimal . deep . no vocals",
			Listeners:   891,
			Bitrate:     "320k",
			Source:      "mock",
			SourceRef:   "code-focus",
		},
		{
			ID:          "winter-walk",
			Name:        "winter walk",
			Description: "ambient pads . soft piano . snowfall",
			Listeners:   127,
			Bitrate:     "192k",
			Source:      "mock",
			SourceRef:   "winter-walk",
		},
		{
			ID:          "vinyl-sunset",
			Name:        "vinyl sunset",
			Description: "soul samples . warm distortion . analog",
			Listeners:   263,
			Bitrate:     "256k",
			Source:      "mock",
			SourceRef:   "vinyl-sunset",
		},
	}

	tracks := map[string]provider.Track{
		"midnight-drift": {Title: "Moonlit Sidewalk", Artist: "sleepytapes", Duration: 4*time.Minute + 53*time.Second},
		"cafe-morning":   {Title: "Espresso Window", Artist: "bluekey trio", Duration: 3*time.Minute + 41*time.Second},
		"code-focus":     {Title: "Deep Buffer", Artist: "null pointer", Duration: 6*time.Minute + 12*time.Second},
		"winter-walk":    {Title: "First Snow", Artist: "atrium", Duration: 5*time.Minute + 4*time.Second},
		"vinyl-sunset":   {Title: "Crate Dust", Artist: "midbar", Duration: 4*time.Minute + 28*time.Second},
	}

	return &Provider{stations: stations, tracks: tracks}
}

func (p *Provider) ID() string { return "mock" }

func (p *Provider) Stations(_ context.Context) ([]provider.Station, error) {
	out := make([]provider.Station, len(p.stations))
	copy(out, p.stations)
	return out, nil
}

func (p *Provider) Resolve(_ context.Context, s provider.Station) (provider.Track, error) {
	if t, ok := p.tracks[s.ID]; ok {
		return t, nil
	}
	return provider.Track{Title: s.Name, Artist: "unknown", Duration: 4 * time.Minute}, nil
}
