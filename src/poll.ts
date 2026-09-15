import type { BitrateTracker } from "./bitrate.ts";
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

export type PollSnapshot = {
  enforced: SessionRow[];
  other: SessionRow[];
  pollError?: string;
};

export type RunPollCycleOptions = {
  client: MediaMtxClient;
  tracker: BitrateTracker;
  maxBitrateBps: number;
  pathIncludeRegex: RegExp | null;
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

export async function runPollCycle(
  options: RunPollCycleOptions,
): Promise<PollSnapshot> {
  const { client, tracker, maxBitrateBps, pathIncludeRegex } = options;

  try {
    const paths = await client.listPaths();
    const { enforced, other } = splitPublishingPaths(paths, pathIncludeRegex);
    const allPublishing = [...enforced, ...other];
    const activeNames = new Set(allPublishing.map((path) => path.name));

    tracker.forgetMissing(activeNames);

    const nowMs = Date.now();

    const enforcedRows: SessionRow[] = [];
    for (const path of enforced) {
      const bitrateBps = tracker.update(path.name, path.inboundBytes, nowMs);
      const row = buildSessionRow(path, bitrateBps, maxBitrateBps);
      enforcedRows.push(await processEnforcedPath(path, row, client, tracker));
    }

    const otherRows: SessionRow[] = other.map((path) => {
      const bitrateBps = tracker.update(path.name, path.inboundBytes, nowMs);
      return buildSessionRow(path, bitrateBps, maxBitrateBps);
    });

    return { enforced: enforcedRows, other: otherRows };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return { enforced: [], other: [], pollError: message };
  }
}
