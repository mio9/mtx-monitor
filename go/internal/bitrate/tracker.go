package bitrate

import (
	"fmt"
	"time"
)

// ByteSample holds a single byte counter reading.
type ByteSample struct {
	Bytes  int64
	TimeMs int64
}

// Tracker computes instantaneous bitrate from byte counter deltas.
type Tracker struct {
	samples map[string]*ByteSample
}

// NewTracker creates a new BitrateTracker.
func NewTracker() *Tracker {
	return &Tracker{
		samples: make(map[string]*ByteSample),
	}
}

// Update records a byte counter reading and returns the instantaneous bitrate in bits per second,
// or nil on first sample or if delta is invalid.
func (t *Tracker) Update(pathName string, bytes int64, timeMs int64) *int {
	previous, exists := t.samples[pathName]
	t.samples[pathName] = &ByteSample{Bytes: bytes, TimeMs: timeMs}

	if !exists {
		return nil
	}

	deltaBytes := bytes - previous.Bytes
	deltaMs := timeMs - previous.TimeMs

	if deltaBytes < 0 || deltaMs <= 0 {
		return nil
	}

	bps := int(deltaBytes * 8 * 1000 / deltaMs)
	return &bps
}

// Forget removes the sample for a path.
func (t *Tracker) Forget(pathName string) {
	delete(t.samples, pathName)
}

// ForgetMissing removes samples for paths no longer active.
func (t *Tracker) ForgetMissing(activePathNames map[string]struct{}) {
	for name := range t.samples {
		if _, active := activePathNames[name]; !active {
			delete(t.samples, name)
		}
	}
}

// FormatBitrate formats a bitrate value to a human-readable string.
func FormatBitrate(bps int) string {
	const (
		mbps = 1_000_000
		kbps = 1_000
	)

	if bps >= mbps {
		return fmt.Sprintf("%.2f Mbps", float64(bps)/mbps)
	}
	if bps >= kbps {
		return fmt.Sprintf("%.1f kbps", float64(bps)/kbps)
	}
	return fmt.Sprintf("%d bps", bps)
}

// FormatBytes formats a byte count to a human-readable string.
func FormatBytes(bytes int64) string {
	const (
		gb = 1_000_000_000
		mb = 1_000_000
		kb = 1_000
	)

	if bytes >= gb {
		return fmt.Sprintf("%.2f GB", float64(bytes)/gb)
	}
	if bytes >= mb {
		return fmt.Sprintf("%.2f MB", float64(bytes)/mb)
	}
	if bytes >= kb {
		return fmt.Sprintf("%.1f KB", float64(bytes)/kb)
	}
	return fmt.Sprintf("%d B", bytes)
}

// NowMs returns the current time in milliseconds.
func NowMs() int64 {
	return time.Now().UnixMilli()
}
