package player

import "errors"

type EventKind int

type ErrorCategory int

const (
	EventHealthy EventKind = iota
	EventUnstable
	EventReconnecting
	EventDisconnected
)

const (
	ErrUnknown ErrorCategory = iota
	ErrAudioOutput
	ErrNetwork
	ErrDecode
	ErrTimeout
	ErrIPC
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
	case ErrAudioOutput:
		return "audio_output"
	case ErrNetwork:
		return "network"
	case ErrDecode:
		return "decode"
	case ErrTimeout:
		return "timeout"
	case ErrIPC:
		return "ipc"
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

type Event struct {
	Kind     EventKind
	Detail   string
	Err      error
	Category ErrorCategory
}

type Player interface {
	Play(url string) error
	Pause(paused bool) error
	SetVolume(v int) error
	Stop() error
	Events() <-chan Event
	Close() error
}

type Noop struct{}

func (Noop) Play(string) error    { return nil }
func (Noop) Pause(bool) error     { return nil }
func (Noop) SetVolume(int) error  { return nil }
func (Noop) Stop() error          { return nil }
func (Noop) Events() <-chan Event { return nil }
func (Noop) Close() error         { return nil }
