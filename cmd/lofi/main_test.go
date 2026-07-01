package main

import (
	"bytes"
	"context"
	"errors"
	"io"
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
