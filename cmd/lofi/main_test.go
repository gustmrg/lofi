package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestDefaultLogFileReceivesStartupLog(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fixedNow := time.Date(2026, 7, 1, 12, 30, 45, 0, time.UTC)
	oldNow := nowFunc
	nowFunc = func() time.Time { return fixedNow }
	t.Cleanup(func() { nowFunc = oldNow })

	var stdout, stderr bytes.Buffer
	exit := run([]string{"--provider=mock"}, &stdout, &stderr, func(string) (provider.Provider, player.Player, error) {
		return nil, nil, errors.New("build failed")
	}, func(context.Context, io.Writer) error {
		t.Fatal("update should not be called")
		return nil
	})
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}

	logPath := filepath.Join(os.Getenv("HOME"), ".lofi", "logs", "lofi-20260701-123045-"+strconv.Itoa(os.Getpid())+".log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", logPath, err)
	}
	if got := string(data); !strings.Contains(got, "starting") || !strings.Contains(got, "build failed") {
		t.Fatalf("log file = %q, want startup and build failure logs", got)
	}

	if target, err := os.Readlink(filepath.Join(filepath.Dir(logPath), "latest.log")); err == nil {
		if target != filepath.Base(logPath) {
			t.Fatalf("latest.log target = %q, want %q", target, filepath.Base(logPath))
		}
		return
	}
	data, err = os.ReadFile(filepath.Join(filepath.Dir(logPath), "latest.txt"))
	if err != nil {
		t.Fatalf("expected latest.log symlink or latest.txt fallback: %v", err)
	}
	if strings.TrimSpace(string(data)) != filepath.Base(logPath) {
		t.Fatalf("latest.txt = %q, want %q", data, filepath.Base(logPath))
	}
}

func TestExplicitLogFileSkipsManagedLatest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	logPath := filepath.Join(t.TempDir(), "custom.log")
	exit := run([]string{"--provider=mock", "--log-file", logPath}, &stdout, &stderr, func(string) (provider.Provider, player.Player, error) {
		return nil, nil, errors.New("build failed")
	}, func(context.Context, io.Writer) error {
		t.Fatal("update should not be called")
		return nil
	})
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("Stat(%s): %v", logPath, err)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".lofi", "logs", "latest.log")); !os.IsNotExist(err) {
		t.Fatalf("latest.log should not be managed for explicit log file, err=%v", err)
	}
}

func TestPruneManagedLogs(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "lofi-20260601-120000-1.log")
	newPath := filepath.Join(dir, "lofi-20260701-120000-1.log")
	customPath := filepath.Join(dir, "custom.log")
	for _, path := range []string{oldPath, newPath, customPath} {
		if err := os.WriteFile(path, []byte("log"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}
	oldTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes old: %v", err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatalf("Chtimes new: %v", err)
	}
	if err := os.Chtimes(customPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes custom: %v", err)
	}

	pruneManagedLogs(dir, newTime, managedLogRetention)

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old managed log should be pruned, err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new managed log should remain: %v", err)
	}
	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("custom log should remain: %v", err)
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
