import { BitrateTracker, formatBitrate } from "./bitrate.ts";
import type { Config } from "./config.ts";
import { MediaMtxClient } from "./mediamtx/client.ts";
import { runPollCycle, type PollSnapshot, type SessionRow } from "./poll.ts";

function authLabel(config: Config): string {
  if (!config.apiAuth) {
    return "auth=none";
  }

  if (config.apiAuth.scheme === "bearer") {
    return "auth=bearer";
  }

  return `auth=basic user=${config.apiAuth.username}`;
}

function pathFilterLabel(config: Config): string {
  if (!config.pathIncludeRegex) {
    return "paths=all";
  }

  return `paths=/${config.pathIncludeRegex.source}/`;
}

function logEnforcedRow(row: SessionRow, maxBitrateBps: number): void {
  const limitLabel = formatBitrate(maxBitrateBps);

  if (row.status === "warming") {
    console.log(`[${row.name}] warming up (${row.sourceType})`);
    return;
  }

  const bitrateLabel = formatBitrate(row.bitrateBps ?? 0);

  if (row.status === "kicked") {
    console.warn(
      `[${row.name}] ${bitrateLabel} exceeds limit ${limitLabel}, kicking ${row.sourceType} ${row.sourceId}`,
    );
    console.warn(`[${row.name}] kicked`);
    return;
  }

  if (row.status === "kick_failed") {
    console.warn(
      `[${row.name}] ${bitrateLabel} exceeds limit ${limitLabel}, kicking ${row.sourceType} ${row.sourceId}`,
    );
    console.error(`[${row.name}] kick failed: ${row.statusDetail}`);
    return;
  }

  console.log(`[${row.name}] ${bitrateLabel} / ${limitLabel}`);
}

function logSnapshot(snapshot: PollSnapshot, maxBitrateBps: number): void {
  if (snapshot.pollError) {
    console.error(`poll error: ${snapshot.pollError}`);
    return;
  }

  for (const row of snapshot.enforced) {
    logEnforcedRow(row, maxBitrateBps);
  }

  if (snapshot.enforced.length === 0) {
    console.log("no active publishing paths");
  }
}

export async function runCli(config: Config): Promise<void> {
  const client = new MediaMtxClient(config.apiUrl, config.apiAuth);
  const tracker = new BitrateTracker();

  console.log(
    `mtx-watcher started: api=${config.apiUrl} ${authLabel(config)} ${pathFilterLabel(config)} poll=${config.pollIntervalMs}ms limit=${formatBitrate(config.maxBitrateBps)}`,
  );

  let running = true;

  const stop = () => {
    running = false;
  };

  process.on("SIGINT", stop);
  process.on("SIGTERM", stop);

  while (running) {
    const snapshot = await runPollCycle({
      client,
      tracker,
      maxBitrateBps: config.maxBitrateBps,
      pathIncludeRegex: config.pathIncludeRegex,
    });

    logSnapshot(snapshot, config.maxBitrateBps);
    await Bun.sleep(config.pollIntervalMs);
  }

  console.log("mtx-watcher stopped");
}
