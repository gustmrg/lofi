package log

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

type Config struct {
	Level  slog.Level
	File   io.Writer
	Stderr io.Writer
}

type branchHandler struct {
	min     slog.Level
	handler slog.Handler
}

type fanoutHandler struct {
	branches []branchHandler
}

var (
	mu     sync.RWMutex
	logger = slog.New(slog.NewTextHandler(io.Discard, nil))
)

func Init(cfg Config) {
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	var branches []branchHandler
	if cfg.File != nil {
		branches = append(branches, branchHandler{
			min:     cfg.Level,
			handler: slog.NewTextHandler(cfg.File, &slog.HandlerOptions{Level: cfg.Level}),
		})
		branches = append(branches, branchHandler{
			min:     slog.LevelWarn,
			handler: slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelWarn}),
		})
	} else {
		branches = append(branches, branchHandler{
			min:     cfg.Level,
			handler: slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: cfg.Level}),
		})
	}

	mu.Lock()
	logger = slog.New(fanoutHandler{branches: branches})
	mu.Unlock()
}

func Default() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return logger
}

func For(subsystem string) *slog.Logger {
	return Default().With(slog.String("subsystem", subsystem))
}

func (h fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, b := range h.branches {
		if level >= b.min && b.handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	var first error
	for _, b := range h.branches {
		if record.Level < b.min || !b.handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := b.handler.Handle(ctx, record.Clone()); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (h fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := fanoutHandler{branches: make([]branchHandler, len(h.branches))}
	for i, b := range h.branches {
		out.branches[i] = branchHandler{min: b.min, handler: b.handler.WithAttrs(attrs)}
	}
	return out
}

func (h fanoutHandler) WithGroup(name string) slog.Handler {
	out := fanoutHandler{branches: make([]branchHandler, len(h.branches))}
	for i, b := range h.branches {
		out.branches[i] = branchHandler{min: b.min, handler: b.handler.WithGroup(name)}
	}
	return out
}
