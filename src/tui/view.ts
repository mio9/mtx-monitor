import { ui } from "@rezi-ui/core";
import { formatBitrate, formatBytes } from "../bitrate.ts";
import { VIEWER_SESSION_LIST_MAX_HEIGHT } from "../constants.ts";
import type { SessionRow, ViewerGroup, ViewerRow } from "../poll.ts";
import {
  flattenViewers,
  isViewerPathCollapsed,
  isViewerPathPinned,
  prepareViewerGroups,
  type DashboardSection,
  type DashboardState,
} from "./state.ts";

export type DashboardHandlers = {
  onSectionChange: (section: DashboardSection) => void;
  onSelectRow: (section: DashboardSection, rowKey: string | null) => void;
  onViewerFilterChange: (query: string) => void;
  onViewerSelectPath: (pathName: string) => void;
  onViewerToggleCollapse: (pathName: string) => void;
  onViewerTogglePin: (pathName: string) => void;
  onCancelKick: () => void;
  onConfirmKick: () => void;
};

function formatTimestamp(timeMs: number | null): string {
  if (timeMs === null) {
    return "never";
  }

  return new Date(timeMs).toLocaleTimeString();
}

function statusBadge(row: SessionRow, enforced: boolean) {
  switch (row.status) {
    case "warming":
      return ui.badge("warming", { variant: "info" });
    case "ok":
      return ui.badge("ok", { variant: "success" });
    case "over":
      return enforced
        ? ui.badge("over", { variant: "error" })
        : ui.badge("over (not enforced)", { variant: "warning" });
    case "kicked":
      return ui.badge("kicked", { variant: "error" });
    case "kick_failed":
      return ui.badge("kick failed", { variant: "error" });
  }
}

function sectionTab(
  section: DashboardSection,
  label: string,
  count: number,
  activeSection: DashboardSection,
) {
  const active = section === activeSection;

  return ui.box(
    {
      border: active ? "heavy" : "single",
      p: 1,
      style: active ? { bold: true } : undefined,
    },
    [ui.text(`${active ? "▶ " : "  "}${label} (${count})`)],
  );
}

function sessionTable(
  tableId: string,
  section: DashboardSection,
  rows: readonly SessionRow[],
  maxBitrateBps: number,
  enforced: boolean,
  selectedKey: string | null,
  activeSection: DashboardSection,
  handlers: DashboardHandlers,
) {
  const limitLabel = formatBitrate(maxBitrateBps);
  const isActive = section === activeSection;

  if (rows.length === 0) {
    return ui.text("No active publishing paths");
  }

  return ui.table({
    id: tableId,
    focusable: isActive,
    selectionMode: "single",
    selection: selectedKey ? [selectedKey] : [],
    onSelectionChange: (keys) => {
      handlers.onSelectRow(section, keys[0] ?? null);
    },
    columns: [
      { key: "name", header: "Path", flex: 2 },
      { key: "sourceType", header: "Protocol", width: 14 },
      {
        key: "bitrateBps",
        header: "Bitrate",
        width: 14,
        render: (value) =>
          ui.text(value === null ? "—" : formatBitrate(value as number)),
      },
      {
        key: "limit",
        header: "Limit",
        width: 14,
        render: () => ui.text(limitLabel),
      },
      {
        key: "status",
        header: "Status",
        width: 22,
        render: (_value, row) => statusBadge(row, enforced),
      },
    ],
    data: rows,
    getRowKey: (row) => row.name,
    virtualized: rows.length > 20,
    border: "single",
  });
}

function encodePathId(pathName: string): string {
  return pathName.replace(/[^a-zA-Z0-9_-]/g, "_");
}

function viewerSessionListHeight(sessionCount: number): number {
  if (sessionCount === 0) {
    return 0;
  }

  return Math.min(sessionCount, VIEWER_SESSION_LIST_MAX_HEIGHT);
}

function estimateViewerGroupHeight(
  group: ViewerGroup,
  collapsed: boolean,
): number {
  if (collapsed) {
    return 2;
  }

  if (group.readers.length === 0) {
    return 4;
  }

  // Path header, session count, column headings, list viewport, box padding.
  return 5 + viewerSessionListHeight(group.readers.length);
}

function formatViewerReaderLine(reader: ViewerRow): string {
  const outboundLabel =
    reader.outboundBytes === null ? "—" : formatBytes(reader.outboundBytes);

  return [
    reader.readerType.padEnd(14),
    reader.sessionIdPrefix,
    reader.remoteAddr,
    outboundLabel,
  ].join("  ");
}

function viewerGroupHeader(
  group: ViewerGroup,
  collapsed: boolean,
  pinned: boolean,
  focused: boolean,
) {
  const markers = [
    pinned ? "*" : " ",
    collapsed ? "[+]" : "[-]",
    focused ? ">" : " ",
  ].join("");
  const outboundLabel =
    group.estimatedOutboundBps === null
      ? "est —"
      : `est ${formatBitrate(group.estimatedOutboundBps)}`;
  const publisherLabel =
    group.publisherBitrateBps === null
      ? "src —"
      : `src ${formatBitrate(group.publisherBitrateBps)}`;

  return ui.text(
    `${markers} ${group.pathName} (${group.readers.length} viewer${group.readers.length === 1 ? "" : "s"}, ${publisherLabel}, ${outboundLabel})`,
    { style: focused || pinned ? { bold: true } : undefined },
  );
}

function viewerSessionList(group: ViewerGroup) {
  const sessionCount = group.readers.length;
  if (sessionCount === 0) {
    return ui.text("  no sessions");
  }

  const needsScroll = sessionCount > VIEWER_SESSION_LIST_MAX_HEIGHT;
  const sessionCountLabel = needsScroll
    ? `  ${sessionCount} sessions (scroll for more)`
    : `  ${sessionCount} session${sessionCount === 1 ? "" : "s"}`;

  const sessionLines = group.readers.map((reader) =>
    ui.text(formatViewerReaderLine(reader)),
  );

  return ui.column({ gap: 0 }, [
    ui.text(sessionCountLabel, { style: { bold: true } }),
    ui.text("  Protocol        Session  Remote              Outbound", {
      style: { bold: true },
    }),
    needsScroll
      ? ui.box(
          {
            id: `viewer-sessions-${encodePathId(group.pathName)}`,
            overflow: "scroll",
            maxHeight: VIEWER_SESSION_LIST_MAX_HEIGHT,
          },
          sessionLines,
        )
      : ui.column({ gap: 0 }, sessionLines),
  ]);
}

function viewersPanel(state: DashboardState, handlers: DashboardHandlers) {
  const { snapshot, viewerUi } = state;
  const preparedGroups = prepareViewerGroups(snapshot.viewers, viewerUi);
  const selectedIndex = preparedGroups.findIndex(
    (group) => group.pathName === viewerUi.selectedPathName,
  );
  const totalPathCount = snapshot.viewers.length;
  const totalSessionCount = flattenViewers(snapshot).length;
  const visibleSessionCount = preparedGroups.reduce(
    (sum, group) => sum + group.readers.length,
    0,
  );
  const sessionCountLabel =
    visibleSessionCount === totalSessionCount
      ? `${totalSessionCount} sessions`
      : `${visibleSessionCount}/${totalSessionCount} sessions`;
  const serverOutboundLabel =
    snapshot.estimatedServerOutboundBps === null
      ? "est server outbound —"
      : `est server outbound ${formatBitrate(snapshot.estimatedServerOutboundBps)}`;

  if (totalPathCount === 0) {
    return ui.text("No connected viewers");
  }

  return ui.column({ gap: 1, flex: 1 }, [
    ui.input({
      id: "viewer-filter-input",
      value: viewerUi.filterQuery,
      placeholder: "Filter paths...",
      onInput: (value) => {
        handlers.onViewerFilterChange(value);
      },
    }),
    ui.text(
      `${sessionCountLabel} | ${preparedGroups.length}/${totalPathCount} paths | ${serverOutboundLabel} | * pin | c collapse | p pin`,
    ),
    preparedGroups.length === 0
      ? ui.text("No paths match filter")
      : ui.virtualList({
          id: "viewer-path-list",
          flex: 1,
          items: preparedGroups,
          estimateItemHeight: (group) =>
            estimateViewerGroupHeight(
              group,
              isViewerPathCollapsed(viewerUi, group.pathName),
            ),
          ensureVisibleIndex: selectedIndex >= 0 ? selectedIndex : undefined,
          renderItem: (group, _index, focused) => {
            const collapsed = isViewerPathCollapsed(viewerUi, group.pathName);
            const pinned = isViewerPathPinned(viewerUi, group.pathName);

            return ui.box(
              {
                border: focused ? "heavy" : "single",
                p: 1,
              },
              collapsed
                ? [viewerGroupHeader(group, collapsed, pinned, focused)]
                : [
                    viewerGroupHeader(group, collapsed, pinned, focused),
                    viewerSessionList(group),
                  ],
            );
          },
          onSelect: (group) => {
            handlers.onViewerSelectPath(group.pathName);
          },
        }),
  ]);
}

function kickConfirmDialog(
  row: SessionRow,
  handlers: DashboardHandlers,
) {
  return ui.dialog({
    id: "kick-confirm-dialog",
    title: "Kick publisher?",
    message: `Kick "${row.name}" (${row.sourceType})?`,
    onClose: handlers.onCancelKick,
    actions: [
      {
        id: "kick-cancel",
        label: "Cancel",
        onPress: handlers.onCancelKick,
      },
      {
        id: "kick-confirm",
        label: "Kick",
        intent: "danger",
        onPress: handlers.onConfirmKick,
      },
    ],
  });
}

export function renderDashboard(
  state: DashboardState,
  handlers: DashboardHandlers,
) {
  const { config, snapshot, lastUpdatedMs, activeSection, selectedKeys } =
    state;
  const regexLabel = config.pathIncludeRegex
    ? `/${config.pathIncludeRegex.source}/`
    : "all";
  const limitLabel = formatBitrate(config.maxBitrateBps);
  const pollLabel = `${config.pollIntervalMs / 1000}s`;
  const errorText = snapshot.pollError ?? "";
  const viewerRows = flattenViewers(snapshot);
  const selectedPublisher = snapshot.enforced
    .concat(snapshot.other)
    .find((row) => row.name === selectedKeys[activeSection]);
  const selectedLabel =
    activeSection === "viewers"
      ? (state.viewerUi.selectedPathName ?? "none")
      : (selectedPublisher?.name ?? "none");
  const viewerCount = viewerRows.length;
  const statusHints =
    activeSection === "viewers"
      ? [
          ui.text("↑/↓ path"),
          ui.text("c collapse"),
          ui.text("p pin"),
          ui.text("Tab filter"),
          ui.text("q quit"),
        ]
      : [
          ui.text("←/→ section"),
          ui.text("w watch"),
          ui.text("k kick"),
          ui.text("q quit"),
        ];

  const mainContent = ui.column({ gap: 1, p: 1, flex: 1 }, [
    ui.text("mtx-watcher", { style: { bold: true } }),
    ui.statusBar({
      id: "status-bar",
      left: [
        ui.text(`api ${config.apiUrl}`),
        ui.text(`poll ${pollLabel}`),
        ui.text(`limit ${limitLabel}`),
        ui.text(`regex ${regexLabel}`),
      ],
      right: [
        ui.text(`updated ${formatTimestamp(lastUpdatedMs)}`),
        ...statusHints,
      ],
    }),
    ...(state.actionMessage
      ? [ui.callout(state.actionMessage, { variant: "info", title: "Action" })]
      : []),
    ...(errorText
      ? [ui.callout(errorText, { variant: "error", title: "Poll error" })]
      : []),
    ui.row({ gap: 1 }, [
      sectionTab(
        "enforced",
        "Enforced",
        snapshot.enforced.length,
        activeSection,
      ),
      sectionTab(
        "other",
        "Other publishers",
        snapshot.other.length,
        activeSection,
      ),
      sectionTab(
        "viewers",
        "Viewers",
        viewerCount,
        activeSection,
      ),
    ]),
    ui.text(`selected: ${selectedLabel}`),
    ui.box({ flex: 1 }, [
      ui.focusZone({ id: "section-focus-zone" }, [
        activeSection === "enforced"
          ? ui.card({ title: "Enforced" }, [
              sessionTable(
                "enforced-table",
                "enforced",
                snapshot.enforced,
                config.maxBitrateBps,
                true,
                selectedKeys.enforced,
                activeSection,
                handlers,
              ),
            ])
          : activeSection === "other"
            ? ui.card({ title: "Other publishers" }, [
                sessionTable(
                  "other-table",
                  "other",
                  snapshot.other,
                  config.maxBitrateBps,
                  false,
                  selectedKeys.other,
                  activeSection,
                  handlers,
                ),
              ])
            : ui.card({ title: "Viewers" }, [viewersPanel(state, handlers)]),
      ]),
    ]),
  ]);

  if (!state.confirmKick) {
    return mainContent;
  }

  return ui.layers([
    mainContent,
    kickConfirmDialog(state.confirmKick, handlers),
  ]);
}
