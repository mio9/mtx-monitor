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

export type PathSource = {
  type: PathSourceType;
  id: string;
};

export type Path = {
  name: string;
  source: PathSource | null;
  inboundBytes: number;
  online: boolean;
};

export type PathListResponse = {
  pageCount: number;
  itemCount: number;
  items: Path[];
};

export type OkResponse = {
  status: "ok";
};

export type ErrorResponse = {
  error: string;
};
