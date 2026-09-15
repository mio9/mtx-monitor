import { createNodeApp } from "@rezi-ui/node";
import { BitrateTracker } from "../bitrate.ts";
import type { Config } from "../config.ts";
import { MediaMtxClient } from "../mediamtx/client.ts";
import type { PathSourceType } from "../mediamtx/types.ts";
import { runPollCycle, type SessionRow } from "../poll.ts";
import { watchStream } from "../watch.ts";
import {
  createInitialState,
  getSelectedRow,
  pruneSelections,
  type DashboardSection,
  type DashboardState,
} from "./state.ts";
import { renderDashboard, type DashboardHandlers } from "./view.ts";

export async function runTui(config: Config): Promise<void> {
  let running = true;
  const stateRef: { current: DashboardState } = {
    current: createInitialState(config),
  };

  const app = createNodeApp({
    initialState: stateRef.current,
    config: { fpsCap: 10 },
  });

  const client = new MediaMtxClient(config.apiUrl, config.apiAuth);
  const tracker = new BitrateTracker();

  const kickPublisher = async (row: SessionRow) => {
    try {
      await client.kickPublisher({
        type: row.sourceType as PathSourceType,
        id: row.sourceId,
      });
      tracker.forget(row.name);

      app.update((previous) => ({
        ...previous,
        confirmKick: null,
        actionMessage: `Kicked ${row.name}`,
        selectedKeys: {
          ...previous.selectedKeys,
          [previous.activeSection]: null,
        },
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      app.update((previous) => ({
        ...previous,
        confirmKick: null,
        actionMessage: `Kick failed: ${message}`,
      }));
    }
  };

  const handlers: DashboardHandlers = {
    onSectionChange: (section) => {
      app.update((previous) => ({
        ...previous,
        activeSection: section,
        actionMessage: null,
      }));
    },
    onSelectRow: (section, rowKey) => {
      app.update((previous) => ({
        ...previous,
        selectedKeys: {
          ...previous.selectedKeys,
          [section]: rowKey,
        },
        actionMessage: null,
      }));
    },
    onCancelKick: () => {
      app.update((previous) => ({
        ...previous,
        confirmKick: null,
      }));
    },
    onConfirmKick: () => {
      const row = stateRef.current.confirmKick;
      if (!row) {
        return;
      }

      void kickPublisher(row);
    },
  };

  app.view((state) => {
    stateRef.current = state;
    return renderDashboard(state, handlers);
  });

  const switchSection = (section: DashboardSection) => {
    handlers.onSectionChange(section);
  };

  const requestKick = () => {
    if (stateRef.current.confirmKick) {
      return;
    }

    const selected = getSelectedRow(stateRef.current);
    if (!selected) {
      app.update((previous) => ({
        ...previous,
        actionMessage: "Select a path to kick",
      }));
      return;
    }

    app.update((previous) => ({
      ...previous,
      confirmKick: selected,
      actionMessage: null,
    }));
  };

  const watchSelected = () => {
    const selected = getSelectedRow(stateRef.current);
    if (!selected) {
      app.update((previous) => ({
        ...previous,
        actionMessage: "Select a path to watch",
      }));
      return;
    }

    try {
      const url = watchStream(
        selected.name,
        stateRef.current.config.rtspUrl,
        stateRef.current.config.watchPlayer,
      );

      app.update((previous) => ({
        ...previous,
        actionMessage: `Watching ${selected.name} (${url})`,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      app.update((previous) => ({
        ...previous,
        actionMessage: `Watch failed: ${message}`,
      }));
    }
  };

  app.keys({
    q: () => {
      running = false;
      void app.stop();
    },
    left: {
      handler: () => {
        switchSection("enforced");
      },
      when: (ctx) => ctx.state.confirmKick === null,
    },
    right: {
      handler: () => {
        switchSection("other");
      },
      when: (ctx) => ctx.state.confirmKick === null,
    },
    w: {
      handler: () => {
        watchSelected();
      },
      when: (ctx) => ctx.state.confirmKick === null,
    },
    k: {
      handler: () => {
        requestKick();
      },
      when: (ctx) => ctx.state.confirmKick === null,
    },
  });

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
          selectedKeys: pruneSelections(previous.selectedKeys, snapshot),
          confirmKick:
            previous.confirmKick &&
            [...snapshot.enforced, ...snapshot.other].some(
              (row) => row.name === previous.confirmKick?.name,
            )
              ? previous.confirmKick
              : null,
        }));
      }

      await Bun.sleep(config.pollIntervalMs);
    }
  };

  void pollLoop();
  await app.run();
}
