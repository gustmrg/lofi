package youtube

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestNew_RequiresYtDlp(t *testing.T) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		t.Skip("yt-dlp not installed; skipping")
	}
	t.Setenv("HOME", t.TempDir())
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.ID() != "youtube" {
		t.Fatalf("ID = %q, want youtube", p.ID())
	}
	stations, err := p.Stations(context.Background())
	if err != nil {
		t.Fatalf("Stations: %v", err)
	}
	if len(stations) == 0 {
		t.Fatal("expected at least one default station")
	}
}

func TestResolve_LiveStream(t *testing.T) {
	if testing.Short() {
		t.Skip("network test; skipping in -short mode")
	}
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		t.Skip("yt-dlp not installed; skipping")
	}
	t.Setenv("HOME", t.TempDir())

	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	stations, err := p.Stations(context.Background())
	if err != nil {
		t.Fatalf("Stations: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	track, err := p.Resolve(ctx, stations[0])
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if track.StreamURL == "" {
		t.Fatal("expected non-empty StreamURL")
	}
}
