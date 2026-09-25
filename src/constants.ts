import type { PathSourceType } from "./mediamtx/types.ts";

/** Source types that publish into MediaMTX and can be kicked. */
export const PUBLISHER_SOURCE_TYPES = new Set<PathSourceType>([
  "rtspSession",
  "rtspsSession",
  "rtmpConn",
  "rtmpsConn",
  "srtConn",
  "webRTCSession",
]);

/** Kick API path prefix per publisher source type (v1.21.1 openapi). */
export const KICK_ENDPOINTS: Partial<Record<PathSourceType, string>> = {
  rtspSession: "/v3/rtsp/sessions/kick",
  rtspsSession: "/v3/rtsps/sessions/kick",
  rtmpConn: "/v3/rtmp/conns/kick",
  rtmpsConn: "/v3/rtmps/conns/kick",
  srtConn: "/v3/srt/conns/kick",
  webRTCSession: "/v3/webrtc/sessions/kick",
};

/**
 * Deprecated aliases kept by MediaMTX for compatibility.
 * @see https://github.com/bluenviron/mediamtx/blob/v1.21.1/internal/api/api.go
 */
export const DEPRECATED_API_PATHS: Record<string, string> = {
  "/v3/rtsp/sessions/list": "/v3/rtspsessions/list",
  "/v3/rtsps/sessions/list": "/v3/rtspssessions/list",
  "/v3/rtmp/conns/list": "/v3/rtmpconns/list",
  "/v3/rtmps/conns/list": "/v3/rtmpsconns/list",
  "/v3/rtsp/sessions/kick": "/v3/rtspsessions/kick",
  "/v3/rtsps/sessions/kick": "/v3/rtspssessions/kick",
  "/v3/rtmp/conns/kick": "/v3/rtmpconns/kick",
  "/v3/rtmps/conns/kick": "/v3/rtmpsconns/kick",
  "/v3/srt/conns/kick": "/v3/srtconns/kick",
  "/v3/webrtc/sessions/kick": "/v3/webrtcsessions/kick",
};

export const DEFAULT_API_URL = "http://127.0.0.1:9997";
export const DEFAULT_RTSP_PORT = 8554;
export const DEFAULT_POLL_INTERVAL_SEC = 3;
export const DEFAULT_MAX_BITRATE_KBPS = 5000;
export const DEFAULT_WATCH_PLAYER = "ffplay";
export const PATHS_PAGE_SIZE = 100;

/** MediaMTX internal reader entries that are not real stream viewers. */
export const HIDDEN_READER_TYPE = "hidden";

/** Playback direction for RTSP sessions and RTMP connections. */
export const PLAYBACK_SESSION_STATE = "read";

export const PATHS_LIST = "/v3/paths/list";

/** v1.21.1 openapi list endpoints for playback sessions. */
export const RTSP_SESSIONS_LIST = "/v3/rtsp/sessions/list";
export const RTSPS_SESSIONS_LIST = "/v3/rtsps/sessions/list";
export const RTMP_CONNS_LIST = "/v3/rtmp/conns/list";
export const RTMPS_CONNS_LIST = "/v3/rtmps/conns/list";
