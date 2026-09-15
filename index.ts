import { BitrateTracker, formatBitrate } from "./src/bitrate.ts";
import { loadConfig } from "./src/config.ts";
import { PUBLISHER_SOURCE_TYPES } from "./src/constants.ts";
import { MediaMtxClient } from "./src/mediamtx/client.ts";
import type { Path } from "./src/mediamtx/types.ts";

function isPublishingPath(path: Path): path is Path & { source: NonNullable<Path["source"]> } {
  return (
    path.source !== null &&
    path.online &&
    PUBLISHER_SOURCE_TYPES.has(path.source.type)
  );
}

async function pollOnce(
  client: MediaMtxClient,
  tracker: BitrateTracker,
  maxBitrateBps: number,
  pathIncludeRegex: RegExp | null,
): Promise<void> {
  const paths = await client.listPaths();
  const publishingPaths = paths
    .filter(isPublishingPath)
    .filter(
      (path) => pathIncludeRegex === null || pathIncludeRegex.test(path.name),
    );
  const activeNames = new Set(publishingPaths.map((path) => path.name));

  tracker.forgetMissing(activeNames);

  const nowMs = Date.now();

  for (const path of publishingPaths) {
    const bitrateBps = tracker.update(path.name, path.inboundBytes, nowMs);

    if (bitrateBps === null) {
      console.log(`[${path.name}] warming up (${path.source.type})`);
      continue;
    }

    const bitrateLabel = formatBitrate(bitrateBps);
    const limitLabel = formatBitrate(maxBitrateBps);

    if (bitrateBps > maxBitrateBps) {
      console.warn(
        `[${path.name}] ${bitrateLabel} exceeds limit ${limitLabel}, kicking ${path.source.type} ${path.source.id}`,
      );

      try {
        await client.kickPublisher(path.source);
        tracker.forget(path.name);
        console.warn(`[${path.name}] kicked`);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`[${path.name}] kick failed: ${message}`);
      }

      continue;
    }

    console.log(`[${path.name}] ${bitrateLabel} / ${limitLabel}`);
  }

  if (publishingPaths.length === 0) {
    console.log("no active publishing paths");
  }
}

async function main(): Promise<void> {
  const config = loadConfig();
  const client = new MediaMtxClient(config.apiUrl, config.apiAuth);
  const tracker = new BitrateTracker();

  const authLabel = config.apiAuth
    ? config.apiAuth.scheme === "bearer"
      ? "auth=bearer"
      : `auth=basic user=${config.apiAuth.username}`
    : "auth=none";

  const pathFilterLabel = config.pathIncludeRegex
    ? `paths=/${config.pathIncludeRegex.source}/`
    : "paths=all";

  console.log(
    `mtx-kicker started: api=${config.apiUrl} ${authLabel} ${pathFilterLabel} poll=${config.pollIntervalMs}ms limit=${formatBitrate(config.maxBitrateBps)}`,
  );

  let running = true;

  const stop = () => {
    running = false;
  };

  process.on("SIGINT", stop);
  process.on("SIGTERM", stop);

  while (running) {
    try {
      await pollOnce(
        client,
        tracker,
        config.maxBitrateBps,
        config.pathIncludeRegex,
      );
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`poll error: ${message}`);
    }

    await Bun.sleep(config.pollIntervalMs);
  }

  console.log("mtx-kicker stopped");
}

await main();
