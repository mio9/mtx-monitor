import { createNodeApp } from "@rezi-ui/node";
import { BitrateTracker } from "../bitrate.ts";
import type { Config } from "../config.ts";
import { MediaMtxClient } from "../mediamtx/client.ts";
import { runPollCycle } from "../poll.ts";
import { createInitialState } from "./state.ts";
import { renderDashboard } from "./view.ts";

export async function runTui(config: Config): Promise<void> {
  let running = true;

  const app = createNodeApp({
    initialState: createInitialState(config),
    config: { fpsCap: 10 },
  });

  app.view(renderDashboard);

  app.keys({
    q: () => {
      running = false;
      void app.stop();
    },
  });

  const client = new MediaMtxClient(config.apiUrl, config.apiAuth);
  const tracker = new BitrateTracker();

  const stop = () => {
    running = false;
    void app.stop();
  };

  process.on("SIGINT", stop);
  process.on("SIGTERM", stop);

  const pollLoop = async () => {
    await app.ready();

    while (running) {
      const snapshot = await runPollCycle({
        client,
        tracker,
        maxBitrateBps: config.maxBitrateBps,
        pathIncludeRegex: config.pathIncludeRegex,
      });

      if (running) {
        app.update((previous) => ({
          ...previous,
          snapshot,
          lastUpdatedMs: Date.now(),
        }));
      }

      await Bun.sleep(config.pollIntervalMs);
    }
  };

  void pollLoop();
  await app.run();
}
