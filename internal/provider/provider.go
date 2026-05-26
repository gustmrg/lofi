package provider

import (
	"context"
	"time"
)

type Station struct {
	ID          string
	Name        string
	Description string
	Listeners   int
	Bitrate     string
	Source      string
	SourceRef   string
}

type Track struct {
	Title     string
	Artist    string
	Duration  time.Duration
	StreamURL string
}

type Provider interface {
	ID() string
	Stations(ctx context.Context) ([]Station, error)
	Resolve(ctx context.Context, s Station) (Track, error)
}
