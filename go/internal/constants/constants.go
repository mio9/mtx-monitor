package constants

// Publisher source types that publish into MediaMTX and can be kicked.
var PublisherSourceTypes = map[string]struct{}{
	"rtspSession":   {},
	"rtspsSession":  {},
	"rtmpConn":      {},
	"rtmpsConn":     {},
	"srtConn":       {},
	"webRTCSession": {},
}

// Kick API endpoint paths per publisher source type (v1.21.1 openapi).
var KickEndpoints = map[string]string{
	"rtspSession":   "/v3/rtsp/sessions/kick",
	"rtspsSession":  "/v3/rtsps/sessions/kick",
	"rtmpConn":      "/v3/rtmp/conns/kick",
	"rtmpsConn":     "/v3/rtmps/conns/kick",
	"srtConn":       "/v3/srt/conns/kick",
	"webRTCSession": "/v3/webrtc/sessions/kick",
}

// Deprecated aliases kept by MediaMTX for compatibility.
// See: https://github.com/bluenviron/mediamtx/blob/v1.21.1/internal/api/api.go
var DeprecatedAPIPaths = map[string]string{
	"/v3/rtsp/sessions/list":   "/v3/rtspsessions/list",
	"/v3/rtsps/sessions/list":  "/v3/rtspssessions/list",
	"/v3/rtmp/conns/list":      "/v3/rtmpconns/list",
	"/v3/rtmps/conns/list":     "/v3/rtmpsconns/list",
	"/v3/rtsp/sessions/kick":   "/v3/rtspsessions/kick",
	"/v3/rtsps/sessions/kick":  "/v3/rtspssessions/kick",
	"/v3/rtmp/conns/kick":      "/v3/rtmpconns/kick",
	"/v3/rtmps/conns/kick":     "/v3/rtmpsconns/kick",
	"/v3/srt/conns/kick":       "/v3/srtconns/kick",
	"/v3/webrtc/sessions/kick": "/v3/webrtcsessions/kick",
}

// Default configuration values.
const (
	DefaultAPIURL          = "http://127.0.0.1:9997"
	DefaultRTSPPort        = 8554
	DefaultPollIntervalSec = 3
	DefaultMaxBitrateKbps  = 5000
	DefaultWatchPlayer     = "ffplay"
	PathsPageSize          = 100
)

// MediaMTX internal reader entries that are not real stream viewers.
const HiddenReaderType = "hidden"

// Playback direction for RTSP sessions and RTMP connections.
const PlaybackSessionState = "read"

// API endpoint paths.
const (
	PathsListEndpoint = "/v3/paths/list"
	RtspSessionsList  = "/v3/rtsp/sessions/list"
	RtspsSessionsList = "/v3/rtsps/sessions/list"
	RtmpConnsList     = "/v3/rtmp/conns/list"
	RtmpsConnsList    = "/v3/rtmps/conns/list"
)

// Viewer UI constants.
const (
	ViewerSessionIDPrefixLength = 8
	ViewerSessionListMaxHeight  = 8
)
