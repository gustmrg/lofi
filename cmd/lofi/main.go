package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	version "github.com/gustmrg/lofi"
	"github.com/gustmrg/lofi/internal/player"
	"github.com/gustmrg/lofi/internal/player/mpv"
	"github.com/gustmrg/lofi/internal/provider"
	"github.com/gustmrg/lofi/internal/provider/mock"
	"github.com/gustmrg/lofi/internal/provider/youtube"
	"github.com/gustmrg/lofi/internal/ui"
	"github.com/gustmrg/lofi/internal/updater"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, build, runUpdate))
}

type buildFunc func(string) (provider.Provider, player.Player, error)
type updateFunc func(context.Context, io.Writer) error

func run(args []string, stdout, stderr io.Writer, build buildFunc, update updateFunc) int {
	fs := flag.NewFlagSet("lofi", flag.ContinueOnError)
	fs.SetOutput(stderr)
	providerFlag := fs.String("provider", "youtube", "provider: youtube|mock")
	versionFlag := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *versionFlag {
		fmt.Fprintln(stdout, version.Display(version.Current()))
		return 0
	}

	switch fs.NArg() {
	case 0:
	case 1:
		switch fs.Arg(0) {
		case "version":
			fmt.Fprintln(stdout, version.Display(version.Current()))
			return 0
		case "update":
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if err := update(ctx, stdout); err != nil {
				fmt.Fprintf(stderr, "lofi: update: %v\n", err)
				return 1
			}
			return 0
		default:
			fmt.Fprintf(stderr, "lofi: unknown command %q\n", fs.Arg(0))
			return 2
		}
	default:
		fmt.Fprintf(stderr, "lofi: too many arguments\n")
		return 2
	}

	prov, pl, err := build(*providerFlag)
	if err != nil {
		fmt.Fprintf(stderr, "lofi: %v\n", err)
		return 1
	}
	defer pl.Close()

	model, err := ui.NewModel(prov, pl)
	if err != nil {
		fmt.Fprintf(stderr, "lofi: %v\n", err)
		return 1
	}

	prog := tea.NewProgram(model)
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(stderr, "lofi: %v\n", err)
		return 1
	}
	return 0
}

func runUpdate(ctx context.Context, stdout io.Writer) error {
	res, err := updater.DefaultUpdater(version.Current()).Run(ctx)
	if errors.Is(err, updater.ErrAlreadyCurrent) {
		fmt.Fprintf(stdout, "LoFi is already up to date (%s).\n", version.Tag(res.Current))
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "LoFi updated to %s.\n", version.Tag(res.Latest))
	return nil
}

func build(name string) (provider.Provider, player.Player, error) {
	switch name {
	case "mock":
		return mock.New(), player.Noop{}, nil
	case "youtube":
		yp, err := youtube.New()
		if err != nil {
			return nil, nil, fmt.Errorf("youtube provider: %w", err)
		}
		mp, err := mpv.New()
		if err != nil {
			return nil, nil, fmt.Errorf("mpv player: %w", err)
		}
		return yp, mp, nil
	default:
		return nil, nil, fmt.Errorf("unknown provider %q (want: youtube|mock)", name)
	}
}
