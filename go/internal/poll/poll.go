package poll

import (
	"context"
	"fmt"
	"regexp"

	"mio9/mtx-monitor/internal/bitrate"
	"mio9/mtx-monitor/internal/constants"
	"mio9/mtx-monitor/internal/mediamtx"
	"mio9/mtx-monitor/internal/paths"
)

// playbackSession is the unified interface for playback session adapters.
type playbackSession interface {
	ID() string
	Path() string
	State() string
	RemoteAddr() string
	OutboundBytes() int64
}

// RtspSessionAdapter wraps mediamtx.RtspSession to implement playbackSession.
type RtspSessionAdapter struct {
	mediamtx.RtspSession
}

func (a RtspSessionAdapter) ID() string           { return a.RtspSession.ID }
func (a RtspSessionAdapter) Path() string         { return a.RtspSession.Path }
func (a RtspSessionAdapter) State() string        { return string(a.RtspSession.State) }
func (a RtspSessionAdapter) RemoteAddr() string   { return a.RtspSession.RemoteAddr }
func (a RtspSessionAdapter) OutboundBytes() int64 { return a.RtspSession.OutboundBytes }

// RtmpConnAdapter wraps mediamtx.RtmpConn to implement playbackSession.
type RtmpConnAdapter struct {
	mediamtx.RtmpConn
}

func (a RtmpConnAdapter) ID() string           { return a.RtmpConn.ID }
func (a RtmpConnAdapter) Path() string         { return a.RtmpConn.Path }
func (a RtmpConnAdapter) State() string        { return string(a.RtmpConn.State) }
func (a RtmpConnAdapter) RemoteAddr() string   { return a.RtmpConn.RemoteAddr }
func (a RtmpConnAdapter) OutboundBytes() int64 { return a.RtmpConn.OutboundBytes }

// SessionStatus represents the status of a publisher session.
type SessionStatus string

const (
	StatusWarming    SessionStatus = "warming"
	StatusOK         SessionStatus = "ok"
	StatusOver       SessionStatus = "over"
	StatusKicked     SessionStatus = "kicked"
	StatusKickFailed SessionStatus = "kick_failed"
)

// SessionRow represents a publisher session in the dashboard.
type SessionRow struct {
	Name         string
	SourceType   string
	SourceID     string
	BitrateBps   *int
	OverLimit    bool
	Status       SessionStatus
	StatusDetail string
}

// ViewerRow represents a viewer session in the dashboard.
type ViewerRow struct {
	RowKey        string
	PathName      string
	ReaderType    string
	ReaderID      string
	SessionPrefix string
	RemoteAddr    string
	OutboundBytes *int64
}

// ViewerGroup groups viewer rows by path name.
type ViewerGroup struct {
	PathName             string
	Readers              []ViewerRow
	PublisherBitrateBps  *int
	EstimatedOutboundBps *int
}

// ServerInfo is the connected MediaMTX instance from /v3/info.
type ServerInfo struct {
	Version string
	Started string
	Error   string
}

// PollSnapshot is the result of a single poll cycle.
type PollSnapshot struct {
	Enforced                   []SessionRow
	Other                      []SessionRow
	Viewers                    []ViewerGroup
	EstimatedServerOutboundBps *int
	PollError                  string
	Server                     ServerInfo
}

// RunPollCycleOptions holds dependencies for a poll cycle.
type RunPollCycleOptions struct {
	Client           *mediamtx.Client
	Tracker          *bitrate.Tracker
	MaxBitrateBps    int
	PathIncludeRegex *regexp.Regexp
}

// PlaybackSessionSource groups playback sessions with their reader type.
// Only one of RtspSessions or RtmpConns is populated per source.
type PlaybackSessionSource struct {
	RtspSessions []mediamtx.RtspSession
	RtmpConns    []mediamtx.RtmpConn
	ReaderType   string
}

// rangeSessions iterates over whichever session slice is populated.
func (s PlaybackSessionSource) rangeSessions(fn func(session playbackSession) bool) {
	if s.RtspSessions != nil {
		for _, sess := range s.RtspSessions {
			if !fn(RtspSessionAdapter{RtspSession: sess}) {
				return
			}
		}
	} else if s.RtmpConns != nil {
		for _, conn := range s.RtmpConns {
			if !fn(RtmpConnAdapter{RtmpConn: conn}) {
				return
			}
		}
	}
}

// RunPollCycle executes a full poll cycle: fetch paths and sessions, compute bitrates,
// enforce limits, and collect viewer data.
func RunPollCycle(ctx context.Context, opts RunPollCycleOptions) PollSnapshot {
	client := opts.Client
	tracker := opts.Tracker
	maxBitrateBps := opts.MaxBitrateBps
	pathIncludeRegex := opts.PathIncludeRegex

	_ = constants.HiddenReaderType // used in collectViewerGroups

	// Fetch all data in parallel.
	pathsResult, err1 := client.ListPaths(ctx)
	rtspSessions, err2 := client.ListRtspSessions(ctx)
	rtspsSessions, err3 := client.ListRtspsSessions(ctx)
	rtmpConns, err4 := client.ListRtmpConns(ctx)
	rtmpsConns, err5 := client.ListRtmpsConns(ctx)

	// If any error occurred, return poll error.
	if err1 != nil {
		return attachServerInfo(ctx, client, PollSnapshot{PollError: err1.Error()})
	}
	if err2 != nil {
		return attachServerInfo(ctx, client, PollSnapshot{PollError: fmt.Sprintf("rtsp sessions: %v", err2)})
	}
	if err3 != nil {
		return attachServerInfo(ctx, client, PollSnapshot{PollError: fmt.Sprintf("rtsps sessions: %v", err3)})
	}
	if err4 != nil {
		return attachServerInfo(ctx, client, PollSnapshot{PollError: fmt.Sprintf("rtmp conns: %v", err4)})
	}
	if err5 != nil {
		return attachServerInfo(ctx, client, PollSnapshot{PollError: fmt.Sprintf("rtmps conns: %v", err5)})
	}

	// Split publishing paths.
	enforced, other := paths.SplitPublishingPaths(pathsResult, pathIncludeRegex)

	// Build active path names set for tracker cleanup.
	activePathNames := make(map[string]struct{}, len(pathsResult))
	for _, p := range pathsResult {
		activePathNames[p.Name] = struct{}{}
	}
	tracker.ForgetMissing(activePathNames)

	// Compute bitrates for all paths.
	nowMs := bitrate.NowMs()
	pathBitrateBps := make(map[string]*int, len(pathsResult))
	for _, p := range pathsResult {
		bps := tracker.Update(p.Name, p.InboundBytes, nowMs)
		pathBitrateBps[p.Name] = bps
	}

	// Build enforced rows with kick logic.
	enforcedRows := make([]SessionRow, 0, len(enforced))
	for _, path := range enforced {
		bps := pathBitrateBps[path.Name]
		row := buildSessionRow(path, bps, maxBitrateBps)
		if row.OverLimit {
			row = processEnforcedPath(ctx, path, row, client, tracker)
		}
		enforcedRows = append(enforcedRows, row)
	}

	// Build other rows (no kick logic).
	otherRows := make([]SessionRow, 0, len(other))
	for _, path := range other {
		bps := pathBitrateBps[path.Name]
		row := buildSessionRow(path, bps, maxBitrateBps)
		otherRows = append(otherRows, row)
	}

	// Collect viewer groups.
	viewers := collectViewerGroups(
		pathsResult,
		[]PlaybackSessionSource{
			{RtspSessions: rtspSessions, ReaderType: "rtspSession"},
			{RtspSessions: rtspsSessions, ReaderType: "rtspsSession"},
			{RtmpConns: rtmpConns, ReaderType: "rtmpConn"},
			{RtmpConns: rtmpsConns, ReaderType: "rtmpsConn"},
		},
		pathBitrateBps,
	)

	// Compute total estimated outbound.
	totalOutbound := sumEstimatedOutbound(viewers)

	snapshot := PollSnapshot{
		Enforced:                   enforcedRows,
		Other:                      otherRows,
		Viewers:                    viewers,
		EstimatedServerOutboundBps: totalOutbound,
	}
	return attachServerInfo(ctx, client, snapshot)
}

func attachServerInfo(ctx context.Context, client *mediamtx.Client, snapshot PollSnapshot) PollSnapshot {
	info, err := client.Info(ctx)
	if err != nil {
		snapshot.Server.Error = err.Error()
		return snapshot
	}
	snapshot.Server.Version = info.Version
	snapshot.Server.Started = info.Started
	return snapshot
}

// buildSessionRow creates a SessionRow from a publishing path and bitrate.
func buildSessionRow(path paths.PublishingPath, bitrateBps *int, maxBitrateBps int) SessionRow {
	overLimit := bitrateBps != nil && *bitrateBps > maxBitrateBps

	var status SessionStatus
	if bitrateBps == nil {
		status = StatusWarming
	} else if overLimit {
		status = StatusOver
	} else {
		status = StatusOK
	}

	return SessionRow{
		Name:       path.Name,
		SourceType: path.Source.Type,
		SourceID:   path.Source.ID,
		BitrateBps: bitrateBps,
		OverLimit:  overLimit,
		Status:     status,
	}
}

// processEnforcedPath kicks a publisher if over limit.
func processEnforcedPath(ctx context.Context, path paths.PublishingPath, row SessionRow, client *mediamtx.Client, tracker *bitrate.Tracker) SessionRow {
	if !row.OverLimit {
		return row
	}

	err := client.KickPublisher(ctx, path.Source)
	if err != nil {
		return SessionRow{
			Name:         row.Name,
			SourceType:   row.SourceType,
			SourceID:     row.SourceID,
			BitrateBps:   row.BitrateBps,
			OverLimit:    true,
			Status:       StatusKickFailed,
			StatusDetail: err.Error(),
		}
	}

	tracker.Forget(path.Name)
	return SessionRow{
		Name:         row.Name,
		SourceType:   row.SourceType,
		SourceID:     row.SourceID,
		BitrateBps:   row.BitrateBps,
		OverLimit:    true,
		Status:       StatusKicked,
		StatusDetail: "kicked for exceeding limit",
	}
}

// sumEstimatedOutbound sums all estimated outbound bitrates from viewer groups.
func sumEstimatedOutbound(groups []ViewerGroup) *int {
	var total int
	hasEstimate := false
	for _, g := range groups {
		if g.EstimatedOutboundBps != nil {
			total += *g.EstimatedOutboundBps
			hasEstimate = true
		}
	}
	if !hasEstimate {
		return nil
	}
	return &total
}

// collectViewerGroups aggregates reader sessions by path name.
func collectViewerGroups(
	paths []mediamtx.Path,
	playbackSources []PlaybackSessionSource,
	pathBitrateBps map[string]*int,
) []ViewerGroup {
	readersByPath := make(map[string]map[string]*ViewerRow)
	sessionDetails := buildSessionDetailsMap(playbackSources)

	addReader := func(r *ViewerRow) {
		m, ok := readersByPath[r.PathName]
		if !ok {
			m = make(map[string]*ViewerRow)
			readersByPath[r.PathName] = m
		}
		existing, exists := m[r.RowKey]
		if exists {
			m[r.RowKey] = mergeViewerRows(existing, r)
		} else {
			m[r.RowKey] = r
		}
	}

	// Add readers from path listings.
	for _, p := range paths {
		for _, reader := range p.Readers {
			if reader.Type == constants.HiddenReaderType {
				continue
			}
			details := sessionDetails[sessionKey(reader.Type, reader.ID)]
			addReader(&ViewerRow{
				RowKey:        sessionKey(reader.Type, reader.ID),
				PathName:      p.Name,
				ReaderType:    reader.Type,
				ReaderID:      reader.ID,
				SessionPrefix: sessionIDPrefix(reader.ID),
				RemoteAddr:    details.RemoteAddr,
				OutboundBytes: details.OutboundBytes,
			})
		}
	}

	// Add readers from playback sessions.
	for _, source := range playbackSources {
		source.rangeSessions(func(session playbackSession) bool {
			if session.State() != constants.PlaybackSessionState || session.Path() == "" {
				return true
			}
			details := sessionDetails[sessionKey(source.ReaderType, session.ID())]
			addReader(&ViewerRow{
				RowKey:        sessionKey(source.ReaderType, session.ID()),
				PathName:      session.Path(),
				ReaderType:    source.ReaderType,
				ReaderID:      session.ID(),
				SessionPrefix: sessionIDPrefix(session.ID()),
				RemoteAddr:    details.RemoteAddr,
				OutboundBytes: details.OutboundBytes,
			})
			return true
		})
	}

	// Build groups.
	groups := make([]ViewerGroup, 0, len(readersByPath))
	for pathName, readersMap := range readersByPath {
		readers := make([]ViewerRow, 0, len(readersMap))
		for _, r := range readersMap {
			readers = append(readers, *r)
		}
		sortViewerRows(readers)

		pubBps := pathBitrateBps[pathName]
		estOutbound := estimateOutboundBitrate(pubBps, len(readers))

		groups = append(groups, ViewerGroup{
			PathName:             pathName,
			Readers:              readers,
			PublisherBitrateBps:  pubBps,
			EstimatedOutboundBps: estOutbound,
		})
	}

	sortViewerGroups(groups)
	return groups
}

// sessionKey creates a composite key for a reader/session.
func sessionKey(readerType, readerID string) string {
	return readerType + ":" + readerID
}

// sessionIDPrefix returns the first N characters of an ID.
func sessionIDPrefix(readerID string) string {
	n := constants.ViewerSessionIDPrefixLength
	if len(readerID) < n {
		n = len(readerID)
	}
	return readerID[:n]
}

// SessionDetails holds remote address and outbound bytes from playback sessions.
type SessionDetails struct {
	RemoteAddr    string
	OutboundBytes *int64
}

// buildSessionDetailsMap builds a lookup map from playback sessions.
func buildSessionDetailsMap(sources []PlaybackSessionSource) map[string]SessionDetails {
	details := make(map[string]SessionDetails)
	for _, source := range sources {
		source.rangeSessions(func(session playbackSession) bool {
			if session.State() != constants.PlaybackSessionState || session.Path() == "" {
				return true
			}
			ob := session.OutboundBytes()
			details[sessionKey(source.ReaderType, session.ID())] = SessionDetails{
				RemoteAddr:    session.RemoteAddr(),
				OutboundBytes: &ob,
			}
			return true
		})
	}
	return details
}

// estimateOutboundBitrate estimates total outbound bitrate from publisher and viewer count.
func estimateOutboundBitrate(publisherBps *int, count int) *int {
	if publisherBps == nil || count == 0 {
		return nil
	}
	est := *publisherBps * count
	return &est
}

// mergeViewerRows merges an incoming viewer row into an existing one.
func mergeViewerRows(existing, incoming *ViewerRow) *ViewerRow {
	remoteAddr := incoming.RemoteAddr
	if remoteAddr == "—" && existing.RemoteAddr != "—" {
		remoteAddr = existing.RemoteAddr
	}
	outboundBytes := incoming.OutboundBytes
	if outboundBytes == nil {
		outboundBytes = existing.OutboundBytes
	}
	return &ViewerRow{
		RowKey:        existing.RowKey,
		PathName:      existing.PathName,
		ReaderType:    existing.ReaderType,
		ReaderID:      existing.ReaderID,
		SessionPrefix: existing.SessionPrefix,
		RemoteAddr:    remoteAddr,
		OutboundBytes: outboundBytes,
	}
}

// sortViewerRows sorts viewer rows by reader type then reader ID.
func sortViewerRows(rows []ViewerRow) {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].ReaderType < rows[i].ReaderType ||
				(rows[j].ReaderType == rows[i].ReaderType && rows[j].ReaderID < rows[i].ReaderID) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

// sortViewerGroups sorts viewer groups by path name.
func sortViewerGroups(groups []ViewerGroup) {
	for i := 0; i < len(groups); i++ {
		for j := i + 1; j < len(groups); j++ {
			if groups[j].PathName < groups[i].PathName {
				groups[i], groups[j] = groups[j], groups[i]
			}
		}
	}
}
