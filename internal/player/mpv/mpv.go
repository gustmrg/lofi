package mpv

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appLog "github.com/gustmrg/lofi/internal/log"
	"github.com/gustmrg/lofi/internal/player"
)

const maxStderrTailLines = 64

type Player struct {
	cmd  *exec.Cmd
	sock string

	mu        sync.Mutex
	conn      net.Conn
	eventConn net.Conn

	events chan player.Event
	logger *slog.Logger

	stderrMu   sync.Mutex
	stderrTail []string
}

func New() (*Player, error) {
	logger := appLog.For("mpv")
	bin, err := exec.LookPath("mpv")
	if err != nil {
		return nil, fmt.Errorf("mpv not found in PATH: %w", err)
	}

	sock := filepath.Join(os.TempDir(), fmt.Sprintf("lofi-mpv-%d.sock", os.Getpid()))
	_ = os.Remove(sock)

	cmd := exec.Command(bin,
		"--no-video",
		"--idle=yes",
		"--no-terminal",
		"--input-ipc-server="+sock,
	)
	cmd.Stdout = io.Discard
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = os.Remove(sock)
		return nil, fmt.Errorf("mpv stderr pipe: %w", err)
	}

	logger.Debug("starting mpv", "op", "start", "binary", bin, "socket", sock)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start mpv: %w", err)
	}

	p := &Player{cmd: cmd, sock: sock, events: make(chan player.Event, 16), logger: logger}
	go p.captureStderr(stderr)

	logger.Debug("dialing mpv ipc", "op", "dial_ipc", "socket", sock)
	conn, err := dialWithRetry(sock, 2*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		_ = os.Remove(sock)
		return nil, fmt.Errorf("connect mpv ipc: %w", err)
	}

	logger.Debug("dialing mpv event ipc", "op", "dial_event_ipc", "socket", sock)
	eventConn, err := dialWithRetry(sock, 2*time.Second)
	if err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		_ = os.Remove(sock)
		return nil, fmt.Errorf("connect mpv event ipc: %w", err)
	}

	p.conn = conn
	p.eventConn = eventConn
	go drain(conn)
	go p.readEvents(eventConn)
	logger.Debug("observing mpv property", "op", "observe_property", "property", "paused-for-cache")
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
	if err := scanner.Err(); err != nil {
		p.logger.Warn("mpv event scanner exited", "op", "read_events", "err", err)
	} else {
		p.logger.Warn("mpv event scanner closed", "op", "read_events")
	}
	p.emit(player.Event{Kind: player.EventDisconnected, Detail: "mpv IPC closed", Category: player.ErrIPC})
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
		category := classifyEndFile(msg.Reason, p.stderrText())
		p.logger.Warn("mpv ended file", "op", "event_end_file", "reason", msg.Reason, "category", category.String())
		p.emit(player.Event{Kind: player.EventDisconnected, Detail: detail, Category: category})
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
		err := errors.New("mpv connection closed")
		p.logger.Error("mpv command failed", "op", "send", "command", commandName(cmd), "category", player.ErrIPC.String(), "err", err)
		return player.WrapError(player.ErrIPC, err)
	}
	p.logger.Debug("sending mpv command", "op", "send", "command", commandName(cmd), "url", commandURL(cmd))
	if err := writeCommand(p.conn, cmd); err != nil {
		p.logger.Error("mpv command failed", "op", "send", "command", commandName(cmd), "category", player.ErrIPC.String(), "err", err, "err_type", fmt.Sprintf("%T", err))
		return player.WrapError(player.ErrIPC, err)
	}
	return nil
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
	p.logger.Debug("play", "op", "play", "url", truncate(url, 160))
	return p.send([]any{"loadfile", url, "replace"})
}

func (p *Player) Pause(paused bool) error {
	p.logger.Debug("pause", "op", "pause", "paused", paused)
	return p.send([]any{"set_property", "pause", paused})
}

func (p *Player) SetVolume(v int) error {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	p.logger.Debug("set volume", "op", "set_volume", "volume", v)
	return p.send([]any{"set_property", "volume", v})
}

func (p *Player) Stop() error {
	p.logger.Debug("stop", "op", "stop")
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

func (p *Player) captureStderr(r io.Reader) {
	lines := make(chan string, 64)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for line := range lines {
			p.logger.Log(context.Background(), stderrLogLevel(line), "mpv stderr", "op", "stderr", "line", line)
		}
	}()

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		p.appendStderrLine(line)
		select {
		case lines <- line:
		default:
		}
	}
	if err := scanner.Err(); err != nil {
		p.logger.Warn("mpv stderr scanner exited", "op", "stderr", "err", err)
	}
	close(lines)
	<-done
}

func (p *Player) appendStderrLine(line string) {
	p.stderrMu.Lock()
	defer p.stderrMu.Unlock()
	p.stderrTail = append(p.stderrTail, line)
	if len(p.stderrTail) > maxStderrTailLines {
		copy(p.stderrTail, p.stderrTail[len(p.stderrTail)-maxStderrTailLines:])
		p.stderrTail = p.stderrTail[:maxStderrTailLines]
	}
}

func (p *Player) stderrText() string {
	p.stderrMu.Lock()
	defer p.stderrMu.Unlock()
	return strings.Join(p.stderrTail, "\n")
}

func classifyEndFile(reason, stderrTail string) player.ErrorCategory {
	lower := strings.ToLower(reason + " " + stderrTail)
	switch {
	case strings.Contains(lower, "could not open/initialize audio device"),
		strings.Contains(lower, "failed to initialize audio driver"),
		strings.Contains(lower, "audio output"),
		strings.Contains(lower, "ao/init"):
		return player.ErrAudioOutput
	case strings.Contains(lower, "network"),
		strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "connection reset"),
		strings.Contains(lower, "timed out"),
		strings.Contains(lower, "failed to resolve"),
		strings.Contains(lower, "no route to host"):
		return player.ErrNetwork
	case strings.Contains(lower, "decoder"),
		strings.Contains(lower, "decoding"),
		strings.Contains(lower, "ffmpeg"),
		strings.Contains(lower, "demuxer"),
		strings.Contains(lower, "invalid data"):
		return player.ErrDecode
	default:
		return player.ErrUnknown
	}
}

func stderrLogLevel(line string) slog.Level {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error"),
		strings.Contains(lower, "failed"),
		strings.Contains(lower, "could not"),
		strings.Contains(lower, "no route to host"),
		strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "connection reset"),
		strings.Contains(lower, "timed out"):
		return slog.LevelWarn
	default:
		return slog.LevelDebug
	}
}

func commandName(cmd []any) string {
	if len(cmd) == 0 {
		return ""
	}
	name, _ := cmd[0].(string)
	return name
}

func commandURL(cmd []any) string {
	if commandName(cmd) != "loadfile" || len(cmd) < 2 {
		return ""
	}
	url, _ := cmd[1].(string)
	return truncate(url, 160)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
