export type PathSourceType =
  | "hlsSource"
  | "redirect"
  | "rpiCameraSource"
  | "rtmpConn"
  | "rtmpsConn"
  | "rtmpSource"
  | "rtspSession"
  | "rtspSource"
  | "rtspsSession"
  | "srtConn"
  | "srtSource"
  | "mpegtsSource"
  | "rtpSource"
  | "webRTCSession"
  | "webRTCSource";

export type PathReaderType =
  | "hlsSession"
  | "rtmpConn"
  | "rtmpsConn"
  | "rtspConn"
  | "rtspSession"
  | "rtspsConn"
  | "rtspsSession"
  | "srtConn"
  | "webRTCSession"
  | "moqSession"
  | "hidden";

export type PathSource = {
  type: PathSourceType;
  id: string;
};

export type PathReader = {
  type: PathReaderType;
  id: string;
};

export type Path = {
  name: string;
  source: PathSource | null;
  inboundBytes: number;
  online: boolean;
  readers?: PathReader[];
};

export type PathListResponse = {
  pageCount: number;
  itemCount: number;
  items: Path[];
};

export type PlaybackSessionState = "idle" | "read" | "publish";

export type RtspSession = {
  id: string;
  path: string;
  state: PlaybackSessionState;
  remoteAddr: string;
};

export type RtmpConn = {
  id: string;
  path: string;
  state: PlaybackSessionState;
  remoteAddr: string;
};

export type PaginatedListResponse<T> = {
  pageCount: number;
  itemCount: number;
  items: T[];
};

export type OkResponse = {
  status: "ok";
};

export type ErrorResponse = {
  error: string;
};
