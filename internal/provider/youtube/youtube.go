package youtube

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gustmrg/lofi/internal/provider"
)

const id = "youtube"

type Provider struct {
	binary   string
	stations []provider.Station
}

func New() (*Provider, error) {
	bin, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("yt-dlp not found in PATH: %w", err)
	}

	return &Provider{
		binary:   bin,
		stations: defaultStations(),
	}, nil
}

func (p *Provider) ID() string { return id }

func (p *Provider) Stations(_ context.Context) ([]provider.Station, error) {
	out := make([]provider.Station, len(p.stations))
	copy(out, p.stations)
	return out, nil
}

func (p *Provider) Resolve(ctx context.Context, s provider.Station) (provider.Track, error) {
	if s.SourceRef == "" {
		return provider.Track{}, errors.New("station has no source ref")
	}

	url := s.SourceRef
	if !strings.Contains(url, "://") {
		url = "https://www.youtube.com/watch?v=" + url
	}

	cmd := exec.CommandContext(ctx, p.binary,
		"--no-playlist",
		"--skip-download",
		"-f", "bestaudio/best",
		"--print", "%(title)s",
		"--print", "%(uploader)s",
		"--print", "%(duration)s",
		"--print", "%(urls)s",
		url,
	)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return provider.Track{}, fmt.Errorf("yt-dlp: %w: %s", err, errBuf.String())
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) < 4 {
		return provider.Track{}, fmt.Errorf("unexpected yt-dlp output: %q", out.String())
	}
	title, uploader, durStr, streamURL := lines[0], lines[1], lines[2], strings.TrimSpace(lines[3])

	durSecs, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		durSecs = 0
	}

	if streamURL == "" {
		return provider.Track{}, errors.New("yt-dlp returned empty stream url")
	}

	return provider.Track{
		Title:     title,
		Artist:    uploader,
		Duration:  time.Duration(durSecs * float64(time.Second)),
		StreamURL: streamURL,
	}, nil
}

func defaultStations() []provider.Station {
	return []provider.Station{
		{
			ID:          "lofi-girl-beats",
			Name:        "lofi girl - beats to study",
			Description: "lofi hip hop radio . beats to relax/study to",
			Listeners:   0,
			Bitrate:     "128k",
			Source:      id,
			SourceRef:   "jfKfPfyJRdk",
		},
		{
			ID:          "lofi-girl-sleep",
			Name:        "lofi girl - sleep",
			Description: "lofi hip hop radio . beats to sleep/chill to",
			Listeners:   0,
			Bitrate:     "128k",
			Source:      id,
			SourceRef:   "rUxyKA_-grg",
		},
		{
			ID:          "chillhop-radio",
			Name:        "chillhop radio",
			Description: "jazzy and lofi hip hop beats",
			Listeners:   0,
			Bitrate:     "128k",
			Source:      id,
			SourceRef:   "5yx6BWlEVcY",
		},
		{
			ID:          "lofi-daily",
			Name:        "lofi daily",
			Description: "the daily driver",
			Listeners:   0,
			Bitrate:     "128k",
			Source:      id,
			SourceRef:   "E2vONfzoyRI",
		},
	}
}
