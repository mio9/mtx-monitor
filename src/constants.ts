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

/** Kick API path prefix per publisher source type. */
export const KICK_ENDPOINTS: Partial<Record<PathSourceType, string>> = {
  rtspSession: "/v3/rtspsessions/kick",
  rtspsSession: "/v3/rtspssessions/kick",
  rtmpConn: "/v3/rtmpconns/kick",
  rtmpsConn: "/v3/rtmpsconns/kick",
  srtConn: "/v3/srtconns/kick",
  webRTCSession: "/v3/webrtcsessions/kick",
};

export const DEFAULT_API_URL = "http://127.0.0.1:9997";
export const DEFAULT_POLL_INTERVAL_SEC = 3;
export const DEFAULT_MAX_BITRATE_KBPS = 5000;
export const PATHS_PAGE_SIZE = 100;
