package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestInit_FileGetsSelectedLevelsAndStderrGetsWarnPlus(t *testing.T) {
	var file, stderr bytes.Buffer
	Init(Config{Level: slog.LevelDebug, File: &file, Stderr: &stderr})

	logger := For("test")
	logger.Debug("debug message")
	logger.Warn("warn message")

	if got := file.String(); !strings.Contains(got, "debug message") || !strings.Contains(got, "warn message") {
		t.Fatalf("file log = %q, want debug and warn messages", got)
	}
	if got := stderr.String(); strings.Contains(got, "debug message") || !strings.Contains(got, "warn message") {
		t.Fatalf("stderr log = %q, want warn only", got)
	}
}

func TestInit_FileOnlyWhenStderrNil(t *testing.T) {
	var file bytes.Buffer
	Init(Config{Level: slog.LevelInfo, File: &file})

	logger := For("test")
	logger.Warn("warn message")

	if got := file.String(); !strings.Contains(got, "warn message") {
		t.Fatalf("file log = %q, want warn message", got)
	}
}

func TestInit_StderrOnlyHonorsLevel(t *testing.T) {
	var stderr bytes.Buffer
	Init(Config{Level: slog.LevelWarn, Stderr: &stderr})

	logger := For("test")
	logger.Info("info message")
	logger.Error("error message")

	if got := stderr.String(); strings.Contains(got, "info message") || !strings.Contains(got, "error message") {
		t.Fatalf("stderr log = %q, want error only", got)
	}
}
