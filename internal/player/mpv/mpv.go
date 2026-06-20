package mpv

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gustmrg/lofi/internal/player"
)

type Player struct {
	cmd  *exec.Cmd
	sock string

	mu   sync.Mutex
	conn net.Conn
	eventConn net.Conn

	events chan player.Event
}

func New() (*Player, error) {
	bin, err := exec.LookPath("mpv")
	if err != nil {
		return nil, fmt.Errorf("mpv not found in PATH: %w", err)
	}

	sock := filepath.Join(os.TempDir(), fmt.Sprintf("lofi-mpv-%d.sock", os.Getpid()))
	_ = os.Remove(sock)

	cmd := exec.Command(bin,
		"--no-video",
		"--idle=yes",
		"--really-quiet",
		"--no-terminal",
		"--input-ipc-server="+sock,
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start mpv: %w", err)
	}

	conn, err := dialWithRetry(sock, 2*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		_ = os.Remove(sock)
		return nil, fmt.Errorf("connect mpv ipc: %w", err)
	}

	eventConn, err := dialWithRetry(sock, 2*time.Second)
	if err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		_ = os.Remove(sock)
		return nil, fmt.Errorf("connect mpv event ipc: %w", err)
	}

	p := &Player{cmd: cmd, sock: sock, conn: conn, eventConn: eventConn, events: make(chan player.Event, 16)}
	go drain(conn)
	go p.readEvents(eventConn)
	_ = writeCommand(eventConn, []any{"observe_property", 1, "paused-for-cache"})

	return p, nil
}

func dialWithRetry(sock string, total time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(total)
	var lastErr error
	for time.Now().Before(deadline) {
		c, err := net.Dial("unix", sock)
		if err == nil {
			return c, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	return nil, lastErr
}

func drain(r io.Reader) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		_ = scanner.Bytes()
	}
}

func (p *Player) readEvents(r io.Reader) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		p.handleEventLine(scanner.Bytes())
	}
	p.emit(player.Event{Kind: player.EventDisconnected, Detail: "mpv IPC closed"})
}

func (p *Player) handleEventLine(line []byte) {
	var msg struct {
		Event  string          `json:"event"`
		Name   string          `json:"name"`
		Data   json.RawMessage `json:"data"`
		Reason string          `json:"reason"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(line, &msg); err != nil || msg.Event == "" {
		return
	}

	switch msg.Event {
	case "file-loaded", "playback-restart":
		p.emit(player.Event{Kind: player.EventHealthy})
	case "start-file":
		p.emit(player.Event{Kind: player.EventReconnecting, Detail: "loading stream"})
	case "end-file":
		detail := msg.Reason
		if detail == "" {
			detail = "stream ended"
		}
		p.emit(player.Event{Kind: player.EventDisconnected, Detail: detail})
	case "property-change":
		if msg.Name != "paused-for-cache" {
			return
		}
		var paused bool
		if err := json.Unmarshal(msg.Data, &paused); err != nil {
			return
		}
		if paused {
			p.emit(player.Event{Kind: player.EventUnstable, Detail: "buffering"})
		} else {
			p.emit(player.Event{Kind: player.EventHealthy})
		}
	}
}

func (p *Player) emit(ev player.Event) {
	select {
	case p.events <- ev:
	default:
	}
}

func (p *Player) send(cmd []any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn == nil {
		return errors.New("mpv connection closed")
	}
	return writeCommand(p.conn, cmd)
}

func writeCommand(conn net.Conn, cmd []any) error {
	payload, err := json.Marshal(map[string]any{"command": cmd})
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	_, err = conn.Write(payload)
	return err
}

func (p *Player) Play(url string) error {
	return p.send([]any{"loadfile", url, "replace"})
}

func (p *Player) Pause(paused bool) error {
	return p.send([]any{"set_property", "pause", paused})
}

func (p *Player) SetVolume(v int) error {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return p.send([]any{"set_property", "volume", v})
}

func (p *Player) Stop() error {
	return p.send([]any{"stop"})
}

func (p *Player) Events() <-chan player.Event {
	return p.events
}

func (p *Player) Close() error {
	_ = p.send([]any{"quit"})

	p.mu.Lock()
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
	if p.eventConn != nil {
		_ = p.eventConn.Close()
		p.eventConn = nil
	}
	p.mu.Unlock()

	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		_ = p.cmd.Process.Kill()
		<-done
	}

	_ = os.Remove(p.sock)
	return nil
}
