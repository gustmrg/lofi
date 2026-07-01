package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	version "github.com/gustmrg/lofi"
	appLog "github.com/gustmrg/lofi/internal/log"
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
	logFileFlag := fs.String("log-file", "", "write diagnostic logs to file")
	logLevelFlag := fs.String("log-level", "info", "log level: debug|info|warn|error")
	versionFlag := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	level, err := parseLogLevel(*logLevelFlag)
	if err != nil {
		fmt.Fprintf(stderr, "lofi: %v\n", err)
		return 2
	}
	var logFile *os.File
	if *logFileFlag != "" {
		f, err := os.OpenFile(*logFileFlag, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintf(stderr, "lofi: open log file %q: %v\n", *logFileFlag, err)
		} else {
			logFile = f
			defer logFile.Close()
		}
	}
	appLog.Init(appLog.Config{Level: level, File: logFile, Stderr: stderr})
	logger := appLog.For("main")

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
				logger.Error("command failed", "op", "update", "err", err, "err_type", fmt.Sprintf("%T", err))
				fmt.Fprintf(stderr, "lofi: update: %v\n", err)
				return 1
			}
			return 0
		default:
			logger.Error("unknown command", "op", "parse_args", "command", fs.Arg(0))
			fmt.Fprintf(stderr, "lofi: unknown command %q\n", fs.Arg(0))
			return 2
		}
	default:
		logger.Error("too many arguments", "op", "parse_args", "count", fs.NArg())
		fmt.Fprintf(stderr, "lofi: too many arguments\n")
		return 2
	}

	logger.Info("starting", "version", version.Display(version.Current()), "provider", *providerFlag, "log_level", *logLevelFlag, "log_file", *logFileFlag != "")
	prov, pl, err := build(*providerFlag)
	if err != nil {
		logger.Error("build failed", "op", "build", "provider", *providerFlag, "err", err, "err_type", fmt.Sprintf("%T", err))
		fmt.Fprintf(stderr, "lofi: %v\n", err)
		return 1
	}
	defer pl.Close()

	model, err := ui.NewModel(prov, pl)
	if err != nil {
		logger.Error("model initialization failed", "op", "ui_new_model", "err", err, "err_type", fmt.Sprintf("%T", err))
		fmt.Fprintf(stderr, "lofi: %v\n", err)
		return 1
	}

	prog := tea.NewProgram(model)
	if _, err := prog.Run(); err != nil {
		logger.Error("program failed", "op", "tea_run", "err", err, "err_type", fmt.Sprintf("%T", err))
		fmt.Fprintf(stderr, "lofi: %v\n", err)
		return 1
	}
	return 0
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch raw {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level %q (want: debug|info|warn|error)", raw)
	}
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
