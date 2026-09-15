import { ui } from "@rezi-ui/core";
import { formatBitrate } from "../bitrate.ts";
import type { SessionRow } from "../poll.ts";
import type { DashboardSection, DashboardState } from "./state.ts";

export type DashboardHandlers = {
  onSectionChange: (section: DashboardSection) => void;
  onSelectRow: (section: DashboardSection, rowKey: string | null) => void;
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
  const selectedRow = snapshot.enforced
    .concat(snapshot.other)
    .find((row) => row.name === selectedKeys[activeSection]);
  const selectedLabel = selectedRow?.name ?? "none";

  const mainContent = ui.column({ gap: 1, p: 1 }, [
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
        ui.text("←/→ section"),
        ui.text("w watch"),
        ui.text("k kick"),
        ui.text("q quit"),
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
    ]),
    ui.text(`selected: ${selectedLabel}`),
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
        : ui.card({ title: "Other publishers" }, [
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
