package mediamtx

// PathSourceType represents the type of a publishing source.
type PathSourceType string

const (
	PathSourceHLSSource       PathSourceType = "hlsSource"
	PathSourceRedirect        PathSourceType = "redirect"
	PathSourceRPICameraSource PathSourceType = "rpiCameraSource"
	PathSourceRTMPConn        PathSourceType = "rtmpConn"
	PathSourceRTMPSConn       PathSourceType = "rtmpsConn"
	PathSourceRTMPSource      PathSourceType = "rtmpSource"
	PathSourceRTSPSession     PathSourceType = "rtspSession"
	PathSourceRTSPSource      PathSourceType = "rtspSource"
	PathSourceRTSPSSession    PathSourceType = "rtspsSession"
	PathSourceSRTConn         PathSourceType = "srtConn"
	PathSourceSRTSource       PathSourceType = "srtSource"
	PathSourceMPEGTSSource    PathSourceType = "mpegtsSource"
	PathSourceRTPSource       PathSourceType = "rtpSource"
	PathSourceWebRTCSession   PathSourceType = "webRTCSession"
	PathSourceWebRTCSource    PathSourceType = "webRTCSource"
)

// PathReaderType represents the type of a path reader.
type PathReaderType string

const (
	PathReaderHLSSession    PathReaderType = "hlsSession"
	PathReaderRTMPConn      PathReaderType = "rtmpConn"
	PathReaderRTMPSConn     PathReaderType = "rtmpsConn"
	PathReaderRTSPConn      PathReaderType = "rtspConn"
	PathReaderRTSPSession   PathReaderType = "rtspSession"
	PathReaderRTSPSConn     PathReaderType = "rtspsConn"
	PathReaderRTSPSSession  PathReaderType = "rtspsSession"
	PathReaderSRTConn       PathReaderType = "srtConn"
	PathReaderWebRTCSession PathReaderType = "webRTCSession"
	PathReaderMOQSession    PathReaderType = "moqSession"
	PathReaderHidden        PathReaderType = "hidden"
)

// PathSource represents a publishing source on a MediaMTX path.
type PathSource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// PathReader represents a reader connected to a MediaMTX path.
type PathReader struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Path represents a MediaMTX path.
type Path struct {
	Name         string       `json:"name"`
	Source       *PathSource  `json:"source"`
	InboundBytes int64        `json:"inboundBytes"`
	Online       bool         `json:"online"`
	Readers      []PathReader `json:"readers,omitempty"`
}

// PathListResponse is the response from the paths/list endpoint.
type PathListResponse struct {
	PageCount int    `json:"pageCount"`
	ItemCount int    `json:"itemCount"`
	Items     []Path `json:"items"`
}

// PlaybackSessionState represents the state of a playback session.
type PlaybackSessionState string

const (
	SessionIdle    PlaybackSessionState = "idle"
	SessionRead    PlaybackSessionState = "read"
	SessionPublish PlaybackSessionState = "publish"
)

// RtspSession represents an RTSP session on MediaMTX.
type RtspSession struct {
	ID            string               `json:"id"`
	Path          string               `json:"path"`
	State         PlaybackSessionState `json:"state"`
	RemoteAddr    string               `json:"remoteAddr"`
	OutboundBytes int64                `json:"outboundBytes"`
}

// RtmpConn represents an RTMP connection on MediaMTX.
type RtmpConn struct {
	ID            string               `json:"id"`
	Path          string               `json:"path"`
	State         PlaybackSessionState `json:"state"`
	RemoteAddr    string               `json:"remoteAddr"`
	OutboundBytes int64                `json:"outboundBytes"`
}

// Info is the MediaMTX /v3/info response.
type Info struct {
	Version string `json:"version"`
	Started string `json:"started"`
}

// OkResponse is a successful API response.
type OkResponse struct {
	Status string `json:"status"`
}

// ErrorResponse is an error API response.
type ErrorResponse struct {
	Error string `json:"error"`
}
