package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	version "github.com/gustmrg/lofi"
	"github.com/gustmrg/lofi/internal/player"
	"github.com/gustmrg/lofi/internal/provider"
)

func TestVersionCommandSkipsBuild(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			buildCalled := false
			exit := run(args, &stdout, &stderr, func(string) (provider.Provider, player.Player, error) {
				buildCalled = true
				return nil, nil, errors.New("should not build")
			}, func(context.Context, io.Writer) error {
				t.Fatal("update should not be called")
				return nil
			})
			if exit != 0 {
				t.Fatalf("exit = %d, stderr = %q", exit, stderr.String())
			}
			if buildCalled {
				t.Fatal("version command should not build provider/player")
			}
			if got, want := strings.TrimSpace(stdout.String()), version.Display(version.Current()); got != want {
				t.Fatalf("stdout = %q", got)
			}
		})
	}
}

func TestInvalidLogLevelSkipsBuild(t *testing.T) {
	var stdout, stderr bytes.Buffer
	buildCalled := false
	exit := run([]string{"--log-level=nope"}, &stdout, &stderr, func(string) (provider.Provider, player.Player, error) {
		buildCalled = true
		return nil, nil, errors.New("should not build")
	}, func(context.Context, io.Writer) error {
		t.Fatal("update should not be called")
		return nil
	})
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if buildCalled {
		t.Fatal("invalid log level should not build provider/player")
	}
	if !strings.Contains(stderr.String(), "unknown log level") {
		t.Fatalf("stderr = %q, want unknown log level", stderr.String())
	}
}

func TestLogFileReceivesStartupLog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	logPath := filepath.Join(t.TempDir(), "lofi.log")
	exit := run([]string{"--provider=mock", "--log-file", logPath}, &stdout, &stderr, func(string) (provider.Provider, player.Player, error) {
		return nil, nil, errors.New("build failed")
	}, func(context.Context, io.Writer) error {
		t.Fatal("update should not be called")
		return nil
	})
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := string(data); !strings.Contains(got, "starting") || !strings.Contains(got, "build failed") {
		t.Fatalf("log file = %q, want startup and build failure logs", got)
	}
}

func TestUpdateCommandSkipsBuild(t *testing.T) {
	var stdout, stderr bytes.Buffer
	buildCalled := false
	updateCalled := false
	exit := run([]string{"update"}, &stdout, &stderr, func(string) (provider.Provider, player.Player, error) {
		buildCalled = true
		return nil, nil, errors.New("should not build")
	}, func(_ context.Context, out io.Writer) error {
		updateCalled = true
		_, _ = io.WriteString(out, "updated\n")
		return nil
	})
	if exit != 0 {
		t.Fatalf("exit = %d, stderr = %q", exit, stderr.String())
	}
	if buildCalled {
		t.Fatal("update command should not build provider/player")
	}
	if !updateCalled {
		t.Fatal("update command should call updater")
	}
	if got := stdout.String(); got != "updated\n" {
		t.Fatalf("stdout = %q", got)
	}
}
