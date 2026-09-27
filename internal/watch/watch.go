package watch

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// BuildWatchURL URL-encodes each path segment and joins with the RTSP base URL.
func BuildWatchURL(rtspBaseURL, pathName string) string {
	parts := strings.Split(pathName, "/")
	encoded := make([]string, len(parts))
	for i, p := range parts {
		encoded[i] = url.PathEscape(p)
	}
	return fmt.Sprintf("%s/%s", rtspBaseURL, strings.Join(encoded, "/"))
}

// WatchStream spawns ffplay (or configured player) to watch a stream. Returns the URL.
func WatchStream(pathName, rtspBaseURL, player string) (string, error) {
	streamURL := BuildWatchURL(rtspBaseURL, pathName)

	cmd := exec.Command(
		player,
		"-autoexit",
		"-window_title", pathName,
		"-rtsp_transport", "tcp",
		"-loglevel", "quiet",
		streamURL,
	)

	cmd.Stdin = devNull()
	cmd.Stdout = devNull()
	cmd.Stderr = devNull()
	cmd.SysProcAttr = detachedProcAttr()

	if err := cmd.Start(); err != nil {
		return streamURL, fmt.Errorf("watch stream: %w", err)
	}

	// Detach — don't Wait, let it live independently.
	go cmd.Process.Release()

	return streamURL, nil
}

func devNull() *os.File { f, _ := os.Open("/dev/null"); return f }

func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
}
