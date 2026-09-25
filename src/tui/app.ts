import { createNodeApp } from "@rezi-ui/node";
import { BitrateTracker } from "../bitrate.ts";
import type { Config } from "../config.ts";
import { MediaMtxClient } from "../mediamtx/client.ts";
import type { PathSourceType } from "../mediamtx/types.ts";
import { runPollCycle, type SessionRow } from "../poll.ts";
import { watchStream } from "../watch.ts";
import {
  createInitialState,
  cycleDashboardSection,
  getSelectedRow,
  isPublisherSection,
  pruneSelections,
  prepareViewerGroups,
  toggleViewerPathCollapsed,
  toggleViewerPathPinned,
  pruneViewerUi,
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
    onViewerFilterChange: (query) => {
      app.update((previous) => {
        const viewerUi = {
          ...previous.viewerUi,
          filterQuery: query,
        };
        const visibleGroups = prepareViewerGroups(
          previous.snapshot.viewers,
          viewerUi,
        );
        const selectedPathName =
          viewerUi.selectedPathName &&
          visibleGroups.some(
            (group) => group.pathName === viewerUi.selectedPathName,
          )
            ? viewerUi.selectedPathName
            : (visibleGroups[0]?.pathName ?? null);

        return {
          ...previous,
          viewerUi: {
            ...viewerUi,
            selectedPathName,
          },
          actionMessage: null,
        };
      });
    },
    onViewerSelectPath: (pathName) => {
      app.update((previous) => ({
        ...previous,
        viewerUi: {
          ...previous.viewerUi,
          selectedPathName: pathName,
        },
        actionMessage: null,
      }));
    },
    onViewerToggleCollapse: (pathName) => {
      app.update((previous) => ({
        ...previous,
        viewerUi: toggleViewerPathCollapsed(previous.viewerUi, pathName),
        actionMessage: null,
      }));
    },
    onViewerTogglePin: (pathName) => {
      app.update((previous) => ({
        ...previous,
        viewerUi: toggleViewerPathPinned(previous.viewerUi, pathName),
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

  const switchSection = (step: number) => {
    handlers.onSectionChange(
      cycleDashboardSection(stateRef.current.activeSection, step),
    );
  };

  const requestKick = () => {
    if (stateRef.current.confirmKick) {
      return;
    }

    if (!isPublisherSection(stateRef.current.activeSection)) {
      app.update((previous) => ({
        ...previous,
        actionMessage: "Select a publisher path to kick",
      }));
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
    if (!isPublisherSection(stateRef.current.activeSection)) {
      app.update((previous) => ({
        ...previous,
        actionMessage: "Select a publisher path to watch",
      }));
      return;
    }

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

  const toggleSelectedViewerPathCollapse = () => {
    const pathName = stateRef.current.viewerUi.selectedPathName;
    if (!pathName) {
      app.update((previous) => ({
        ...previous,
        actionMessage: "Select a path to collapse",
      }));
      return;
    }

    handlers.onViewerToggleCollapse(pathName);
  };

  const toggleSelectedViewerPathPin = () => {
    const pathName = stateRef.current.viewerUi.selectedPathName;
    if (!pathName) {
      app.update((previous) => ({
        ...previous,
        actionMessage: "Select a path to pin",
      }));
      return;
    }

    handlers.onViewerTogglePin(pathName);
  };

  app.keys({
    q: () => {
      running = false;
      void app.stop();
    },
    left: {
      handler: () => {
        switchSection(-1);
      },
      when: (ctx) => ctx.state.confirmKick === null,
    },
    right: {
      handler: () => {
        switchSection(1);
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
    c: {
      handler: () => {
        toggleSelectedViewerPathCollapse();
      },
      when: (ctx) =>
        ctx.state.confirmKick === null && ctx.state.activeSection === "viewers",
    },
    p: {
      handler: () => {
        toggleSelectedViewerPathPin();
      },
      when: (ctx) =>
        ctx.state.confirmKick === null && ctx.state.activeSection === "viewers",
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
          viewerUi: pruneViewerUi(previous.viewerUi, snapshot.viewers),
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
