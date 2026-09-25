import type { BitrateTracker } from "./bitrate.ts";
import {
  HIDDEN_READER_TYPE,
  PLAYBACK_SESSION_STATE,
} from "./constants.ts";
import type { Path, PathReaderType, RtmpConn, RtspSession } from "./mediamtx/types.ts";
import { splitPublishingPaths, type PublishingPath } from "./paths.ts";
import type { MediaMtxClient } from "./mediamtx/client.ts";

export type SessionStatus =
  | "warming"
  | "ok"
  | "over"
  | "kicked"
  | "kick_failed";

export type SessionRow = {
  name: string;
  sourceType: string;
  sourceId: string;
  bitrateBps: number | null;
  overLimit: boolean;
  status: SessionStatus;
  statusDetail?: string;
};

export type ViewerRow = {
  rowKey: string;
  pathName: string;
  readerType: string;
  readerId: string;
  remoteAddr?: string;
};

export type ViewerGroup = {
  pathName: string;
  readers: ViewerRow[];
  publisherBitrateBps: number | null;
  estimatedOutboundBps: number | null;
};

export type PollSnapshot = {
  enforced: SessionRow[];
  other: SessionRow[];
  viewers: ViewerGroup[];
  estimatedServerOutboundBps: number | null;
  pollError?: string;
};

export type RunPollCycleOptions = {
  client: MediaMtxClient;
  tracker: BitrateTracker;
  maxBitrateBps: number;
  pathIncludeRegex: RegExp | null;
};

type PlaybackSessionSource = {
  sessions: readonly RtspSession[] | readonly RtmpConn[];
  readerType: PathReaderType;
};

function buildSessionRow(
  path: PublishingPath,
  bitrateBps: number | null,
  maxBitrateBps: number,
): SessionRow {
  const overLimit = bitrateBps !== null && bitrateBps > maxBitrateBps;

  let status: SessionStatus = "ok";
  if (bitrateBps === null) {
    status = "warming";
  } else if (overLimit) {
    status = "over";
  }

  return {
    name: path.name,
    sourceType: path.source.type,
    sourceId: path.source.id,
    bitrateBps,
    overLimit,
    status,
  };
}

async function processEnforcedPath(
  path: PublishingPath,
  row: SessionRow,
  client: MediaMtxClient,
  tracker: BitrateTracker,
): Promise<SessionRow> {
  if (!row.overLimit) {
    return row;
  }

  try {
    await client.kickPublisher(path.source);
    tracker.forget(path.name);
    return {
      ...row,
      status: "kicked",
      statusDetail: "kicked for exceeding limit",
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      ...row,
      status: "kick_failed",
      statusDetail: message,
    };
  }
}

function estimateOutboundBitrate(
  publisherBitrateBps: number | null,
  playbackSessionCount: number,
): number | null {
  if (publisherBitrateBps === null || playbackSessionCount === 0) {
    return null;
  }

  return publisherBitrateBps * playbackSessionCount;
}

function sumEstimatedOutbound(
  groups: readonly ViewerGroup[],
): number | null {
  let total = 0;
  let hasEstimate = false;

  for (const group of groups) {
    if (group.estimatedOutboundBps !== null) {
      total += group.estimatedOutboundBps;
      hasEstimate = true;
    }
  }

  return hasEstimate ? total : null;
}

function playbackSessionsToViewerRows(
  source: PlaybackSessionSource,
): ViewerRow[] {
  return source.sessions
    .filter(
      (session) =>
        session.state === PLAYBACK_SESSION_STATE && session.path.length > 0,
    )
    .map((session) => ({
      rowKey: `${session.path}:${source.readerType}:${session.id}`,
      pathName: session.path,
      readerType: source.readerType,
      readerId: session.id,
      remoteAddr: session.remoteAddr,
    }));
}

function collectViewerGroups(
  paths: readonly Path[],
  playbackSources: readonly PlaybackSessionSource[],
  pathBitrateBps: ReadonlyMap<string, number | null>,
): ViewerGroup[] {
  const readersByPath = new Map<string, Map<string, ViewerRow>>();

  const addReader = (reader: ViewerRow) => {
    let pathReaders = readersByPath.get(reader.pathName);
    if (!pathReaders) {
      pathReaders = new Map();
      readersByPath.set(reader.pathName, pathReaders);
    }

    pathReaders.set(reader.rowKey, reader);
  };

  for (const path of paths) {
    for (const reader of path.readers ?? []) {
      if (reader.type === HIDDEN_READER_TYPE) {
        continue;
      }

      addReader({
        rowKey: `${path.name}:${reader.type}:${reader.id}`,
        pathName: path.name,
        readerType: reader.type,
        readerId: reader.id,
      });
    }
  }

  for (const source of playbackSources) {
    for (const reader of playbackSessionsToViewerRows(source)) {
      addReader(reader);
    }
  }

  const groups: ViewerGroup[] = [];

  for (const [pathName, pathReaders] of readersByPath) {
    const readers = [...pathReaders.values()];
    readers.sort((left, right) => {
      const typeOrder = left.readerType.localeCompare(right.readerType);
      if (typeOrder !== 0) {
        return typeOrder;
      }

      return left.readerId.localeCompare(right.readerId);
    });

    const publisherBitrateBps = pathBitrateBps.get(pathName) ?? null;
    groups.push({
      pathName,
      readers,
      publisherBitrateBps,
      estimatedOutboundBps: estimateOutboundBitrate(
        publisherBitrateBps,
        readers.length,
      ),
    });
  }

  groups.sort((left, right) => left.pathName.localeCompare(right.pathName));
  return groups;
}

export async function runPollCycle(
  options: RunPollCycleOptions,
): Promise<PollSnapshot> {
  const { client, tracker, maxBitrateBps, pathIncludeRegex } = options;

  try {
    const [
      paths,
      rtspSessions,
      rtspsSessions,
      rtmpConns,
      rtmpsConns,
    ] = await Promise.all([
      client.listPaths(),
      client.listRtspSessions(),
      client.listRtspsSessions(),
      client.listRtmpConns(),
      client.listRtmpsConns(),
    ]);

    const { enforced, other } = splitPublishingPaths(paths, pathIncludeRegex);
    const allPublishing = [...enforced, ...other];
    const activePathNames = new Set(paths.map((path) => path.name));

    tracker.forgetMissing(activePathNames);

    const nowMs = Date.now();
    const pathBitrateBps = new Map<string, number | null>();
    for (const path of paths) {
      pathBitrateBps.set(
        path.name,
        tracker.update(path.name, path.inboundBytes, nowMs),
      );
    }

    const enforcedRows: SessionRow[] = [];
    for (const path of enforced) {
      const bitrateBps = pathBitrateBps.get(path.name) ?? null;
      const row = buildSessionRow(path, bitrateBps, maxBitrateBps);
      enforcedRows.push(await processEnforcedPath(path, row, client, tracker));
    }

    const otherRows: SessionRow[] = other.map((path) => {
      const bitrateBps = pathBitrateBps.get(path.name) ?? null;
      return buildSessionRow(path, bitrateBps, maxBitrateBps);
    });

    const viewers = collectViewerGroups(
      paths,
      [
        { sessions: rtspSessions, readerType: "rtspSession" },
        { sessions: rtspsSessions, readerType: "rtspsSession" },
        { sessions: rtmpConns, readerType: "rtmpConn" },
        { sessions: rtmpsConns, readerType: "rtmpsConn" },
      ],
      pathBitrateBps,
    );

    return {
      enforced: enforcedRows,
      other: otherRows,
      viewers,
      estimatedServerOutboundBps: sumEstimatedOutbound(viewers),
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      enforced: [],
      other: [],
      viewers: [],
      estimatedServerOutboundBps: null,
      pollError: message,
    };
  }
}
