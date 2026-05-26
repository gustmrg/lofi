package player

type Player interface {
	Play(url string) error
	Pause(paused bool) error
	SetVolume(v int) error
	Stop() error
	Close() error
}

type Noop struct{}

func (Noop) Play(string) error   { return nil }
func (Noop) Pause(bool) error    { return nil }
func (Noop) SetVolume(int) error { return nil }
func (Noop) Stop() error         { return nil }
func (Noop) Close() error        { return nil }
