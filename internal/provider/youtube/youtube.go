package youtube

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gustmrg/lofi/internal/provider"
	"github.com/gustmrg/lofi/internal/store"
)

const id = "youtube"

type Provider struct {
	binary string

	mu       sync.Mutex
	stations []provider.Station
}

func New() (*Provider, error) {
	bin, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("yt-dlp not found in PATH: %w", err)
	}

	p := &Provider{binary: bin}

	saved, err := store.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lofi: stations.json unreadable, falling back to defaults: %v\n", err)
		p.stations = defaultStations()
		return p, nil
	}

	if saved == nil {
		p.stations = defaultStations()
		if err := store.Save(toSaved(p.stations)); err != nil {
			fmt.Fprintf(os.Stderr, "lofi: seed stations.json: %v\n", err)
		}
		return p, nil
	}

	p.stations = fromSaved(saved)
	if refreshDefaultStationRefs(p.stations) {
		if err := store.Save(toSaved(p.stations)); err != nil {
			fmt.Fprintf(os.Stderr, "lofi: update stations.json: %v\n", err)
		}
	}
	return p, nil
}

func (p *Provider) ID() string { return id }

func (p *Provider) Stations(_ context.Context) ([]provider.Station, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]provider.Station, len(p.stations))
	copy(out, p.stations)
	return out, nil
}

type meta struct {
	title, uploader string
	durSecs         float64
	streamURL       string // empty when not requested
}

func (p *Provider) runYtDlp(ctx context.Context, link string, withStreamURL bool) (meta, error) {
	args := []string{
		"--no-playlist",
		"--skip-download",
		"-f", "bestaudio/best",
		"--print", "%(title)s",
		"--print", "%(uploader)s",
		"--print", "%(duration)s",
	}
	if withStreamURL {
		args = append(args, "--print", "%(urls)s")
	}
	args = append(args, link)

	cmd := exec.CommandContext(ctx, p.binary, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return meta{}, fmt.Errorf("yt-dlp: %w: %s", err, strings.TrimSpace(errBuf.String()))
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	want := 3
	if withStreamURL {
		want = 4
	}
	if len(lines) < want {
		return meta{}, fmt.Errorf("unexpected yt-dlp output: %q", out.String())
	}

	dur, err := strconv.ParseFloat(lines[2], 64)
	if err != nil {
		dur = 0
	}
	m := meta{title: lines[0], uploader: lines[1], durSecs: dur}
	if withStreamURL {
		m.streamURL = strings.TrimSpace(lines[3])
	}
	return m, nil
}

func (p *Provider) Resolve(ctx context.Context, s provider.Station) (provider.Track, error) {
	if s.SourceRef == "" {
		return provider.Track{}, errors.New("station has no source ref")
	}

	m, err := p.runYtDlp(ctx, normalizeURL(s.SourceRef), true)
	if err != nil {
		return provider.Track{}, err
	}
	if m.streamURL == "" {
		return provider.Track{}, errors.New("yt-dlp returned empty stream url")
	}

	return provider.Track{
		Title:     m.title,
		Artist:    m.uploader,
		Duration:  time.Duration(m.durSecs * float64(time.Second)),
		StreamURL: m.streamURL,
	}, nil
}

func (p *Provider) AddByURL(ctx context.Context, raw string) (provider.Station, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return provider.Station{}, errors.New("url is empty")
	}

	vid := extractVideoID(raw)
	if vid == "" {
		return provider.Station{}, fmt.Errorf("could not extract YouTube video id from %q", raw)
	}

	p.mu.Lock()
	for _, s := range p.stations {
		if s.SourceRef == vid {
			p.mu.Unlock()
			return provider.Station{}, fmt.Errorf("station already exists: %s", s.Name)
		}
	}
	p.mu.Unlock()

	m, err := p.runYtDlp(ctx, "https://www.youtube.com/watch?v="+vid, false)
	if err != nil {
		return provider.Station{}, err
	}

	name := strings.TrimSpace(m.title)
	if name == "" {
		name = vid
	}
	desc := ""
	if m.uploader != "" {
		desc = "by " + m.uploader
	}

	s := provider.Station{
		ID:          vid,
		Name:        name,
		Description: desc,
		Bitrate:     "128k",
		Source:      id,
		SourceRef:   vid,
	}

	p.mu.Lock()
	p.stations = append(p.stations, s)
	snapshot := toSaved(p.stations)
	p.mu.Unlock()

	if err := store.Save(snapshot); err != nil {
		return provider.Station{}, fmt.Errorf("persist stations: %w", err)
	}
	return s, nil
}

func (p *Provider) Remove(_ context.Context, sid string) error {
	p.mu.Lock()
	idx := -1
	for i, s := range p.stations {
		if s.ID == sid {
			idx = i
			break
		}
	}
	if idx < 0 {
		p.mu.Unlock()
		return fmt.Errorf("station %q not found", sid)
	}
	if len(p.stations) == 1 {
		p.mu.Unlock()
		return errors.New("cannot remove the last station")
	}
	p.stations = append(p.stations[:idx], p.stations[idx+1:]...)
	snapshot := toSaved(p.stations)
	p.mu.Unlock()

	if err := store.Save(snapshot); err != nil {
		return fmt.Errorf("persist stations: %w", err)
	}
	return nil
}

func normalizeURL(ref string) string {
	if strings.Contains(ref, "://") {
		return ref
	}
	return "https://www.youtube.com/watch?v=" + ref
}

func extractVideoID(raw string) string {
	if !strings.Contains(raw, "://") {
		// Already a bare video ID.
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Host)
	switch {
	case strings.HasSuffix(host, "youtu.be"):
		return strings.Trim(u.Path, "/")
	case strings.Contains(host, "youtube.com"):
		if v := u.Query().Get("v"); v != "" {
			return v
		}
		// /live/<id> or /embed/<id>
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 && (parts[0] == "live" || parts[0] == "embed" || parts[0] == "shorts") {
			return parts[1]
		}
	}
	return ""
}

func toSaved(stations []provider.Station) []store.Saved {
	out := make([]store.Saved, len(stations))
	for i, s := range stations {
		out[i] = store.Saved{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			SourceRef:   s.SourceRef,
		}
	}
	return out
}

func fromSaved(saved []store.Saved) []provider.Station {
	out := make([]provider.Station, len(saved))
	for i, s := range saved {
		out[i] = provider.Station{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Bitrate:     "128k",
			Source:      id,
			SourceRef:   s.SourceRef,
		}
	}
	return out
}

func refreshDefaultStationRefs(stations []provider.Station) bool {
	changed := false
	for i := range stations {
		switch stations[i].ID {
		case "lofi-girl-beats":
			if stations[i].SourceRef == "jfKfPfyJRdk" {
				stations[i].SourceRef = "X4VbdwhkE10"
				changed = true
			}
		case "lofi-girl-sleep":
			if stations[i].SourceRef == "rUxyKA_-grg" {
				stations[i].SourceRef = "JD-kMIpDfnY"
				changed = true
			}
		}
	}
	return changed
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
			SourceRef:   "X4VbdwhkE10",
		},
		{
			ID:          "lofi-girl-sleep",
			Name:        "lofi girl - sleep",
			Description: "lofi hip hop radio . beats to sleep/chill to",
			Listeners:   0,
			Bitrate:     "128k",
			Source:      id,
			SourceRef:   "JD-kMIpDfnY",
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
