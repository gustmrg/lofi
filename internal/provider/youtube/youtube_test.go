package youtube

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/gustmrg/lofi/internal/provider"
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

func TestRefreshDefaultStationRefs(t *testing.T) {
	stations := []provider.Station{
		{ID: "lofi-girl-beats", SourceRef: "jfKfPfyJRdk"},
		{ID: "lofi-girl-sleep", SourceRef: "rUxyKA_-grg"},
		{ID: "custom", SourceRef: "jfKfPfyJRdk"},
	}
	if !refreshDefaultStationRefs(stations) {
		t.Fatal("expected default station refs to be updated")
	}
	if stations[0].SourceRef != "X4VbdwhkE10" {
		t.Fatalf("beats SourceRef = %q, want X4VbdwhkE10", stations[0].SourceRef)
	}
	if stations[1].SourceRef != "JD-kMIpDfnY" {
		t.Fatalf("sleep SourceRef = %q, want JD-kMIpDfnY", stations[1].SourceRef)
	}
	if stations[2].SourceRef != "jfKfPfyJRdk" {
		t.Fatalf("custom SourceRef = %q, want unchanged", stations[2].SourceRef)
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
