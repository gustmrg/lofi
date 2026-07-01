package provider

import (
	"context"
	"errors"
	"time"
)

type ErrorCategory int

const (
	ErrUnknown ErrorCategory = iota
	ErrNetwork
	ErrDecode
	ErrTimeout
)

type Error struct {
	Category ErrorCategory
	Err      error
}

func (e Error) Error() string {
	if e.Err == nil {
		return e.Category.String()
	}
	return e.Err.Error()
}

func (e Error) Unwrap() error { return e.Err }

func (c ErrorCategory) String() string {
	switch c {
	case ErrNetwork:
		return "network"
	case ErrDecode:
		return "decode"
	case ErrTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

func WrapError(category ErrorCategory, err error) error {
	if err == nil {
		return nil
	}
	return Error{Category: category, Err: err}
}

func Category(err error) ErrorCategory {
	var e Error
	if errors.As(err, &e) {
		return e.Category
	}
	return ErrUnknown
}

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

type StationManager interface {
	AddByURL(ctx context.Context, url string) (Station, error)
	Remove(ctx context.Context, id string) error
}
