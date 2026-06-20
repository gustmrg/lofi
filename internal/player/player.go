package player

type EventKind int

const (
	EventHealthy EventKind = iota
	EventUnstable
	EventReconnecting
	EventDisconnected
)

type Event struct {
	Kind   EventKind
	Detail string
	Err    error
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

func (Noop) Play(string) error   { return nil }
func (Noop) Pause(bool) error    { return nil }
func (Noop) SetVolume(int) error { return nil }
func (Noop) Stop() error         { return nil }
func (Noop) Events() <-chan Event { return nil }
func (Noop) Close() error        { return nil }
