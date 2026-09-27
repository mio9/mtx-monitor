package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mio9/mtx-monitor/internal/bitrate"
	"mio9/mtx-monitor/internal/config"
	"mio9/mtx-monitor/internal/mediamtx"
	"mio9/mtx-monitor/internal/poll"
)

// RunCLI runs the headless CLI poll loop until interrupted.
func RunCLI(cfg *config.Config) error {
	client := mediamtx.NewClient(cfg.APIURL, cfg.ApiAuth)
	tracker := bitrate.NewTracker()

	fmt.Printf("mtx-watcher started: api=%s %s %s poll=%dms limit=%s\n",
		cfg.APIURL,
		authLabel(cfg),
		pathFilterLabel(cfg),
		cfg.PollIntervalMs,
		bitrate.FormatBitrate(cfg.MaxBitrateBps),
	)

	// Set up signal handling.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	opts := poll.RunPollCycleOptions{
		Client:           client,
		Tracker:          tracker,
		MaxBitrateBps:    cfg.MaxBitrateBps,
		PathIncludeRegex: cfg.PathIncludeRegex,
	}

	for {
		select {
		case <-sigCh:
			fmt.Println("\nmtx-watcher stopped")
			return nil
		default:
		}

		snapshot := poll.RunPollCycle(context.Background(), opts)
		logSnapshot(snapshot, cfg.MaxBitrateBps)

		// Sleep with interruptibility.
		select {
		case <-sigCh:
			fmt.Println("\nmtx-watcher stopped")
			return nil
		case <-timeAfterMs(cfg.PollIntervalMs):
		}
	}
}

func authLabel(cfg *config.Config) string {
	if cfg.ApiAuth == nil {
		return "auth=none"
	}
	if cfg.ApiAuth.Scheme == "bearer" {
		return "auth=bearer"
	}
	return fmt.Sprintf("auth=basic user=%s", cfg.ApiAuth.Username)
}

func pathFilterLabel(cfg *config.Config) string {
	if cfg.PathIncludeRegex == nil {
		return "paths=all"
	}
	return fmt.Sprintf("paths=/%s/", cfg.PathIncludeRegex.String())
}

func logEnforcedRow(row poll.SessionRow, maxBps int) {
	limitLabel := bitrate.FormatBitrate(maxBps)

	if row.Status == "warming" {
		fmt.Printf("[%s] warming up (%s)\n", row.Name, row.SourceType)
		return
	}

	bitrateLabel := bitrate.FormatBitrate(func() int {
		if row.BitrateBps != nil {
			return *row.BitrateBps
		}
		return 0
	}())

	switch row.Status {
	case "kicked":
		fmt.Fprintf(os.Stderr, "[%s] %s exceeds limit %s, kicking %s %s\n",
			row.Name, bitrateLabel, limitLabel, row.SourceType, row.SourceID)
		fmt.Fprintf(os.Stderr, "[%s] kicked\n", row.Name)
	case "kick_failed":
		fmt.Fprintf(os.Stderr, "[%s] %s exceeds limit %s, kicking %s %s\n",
			row.Name, bitrateLabel, limitLabel, row.SourceType, row.SourceID)
		fmt.Fprintf(os.Stderr, "[%s] kick failed: %s\n", row.Name, row.StatusDetail)
	default:
		fmt.Printf("[%s] %s / %s\n", row.Name, bitrateLabel, limitLabel)
	}
}

func logSnapshot(snapshot poll.PollSnapshot, maxBps int) {
	if snapshot.PollError != "" {
		fmt.Fprintf(os.Stderr, "poll error: %s\n", snapshot.PollError)
		return
	}

	for _, row := range snapshot.Enforced {
		logEnforcedRow(row, maxBps)
	}

	if len(snapshot.Enforced) == 0 {
		fmt.Println("no active publishing paths")
	}
}

// timeAfterMs is a var for testability.
var timeAfterMs = func(d int) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		select {
		case <-time.After(time.Duration(d) * time.Millisecond):
			close(ch)
		}
	}()
	return ch
}
