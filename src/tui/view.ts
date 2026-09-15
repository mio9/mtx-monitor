import { ui } from "@rezi-ui/core";
import { formatBitrate } from "../bitrate.ts";
import type { SessionRow } from "../poll.ts";
import type { DashboardState } from "./state.ts";

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

function sessionTable(
  tableId: string,
  rows: readonly SessionRow[],
  maxBitrateBps: number,
  enforced: boolean,
) {
  const limitLabel = formatBitrate(maxBitrateBps);

  if (rows.length === 0) {
    return ui.text("No active publishing paths");
  }

  return ui.table({
    id: tableId,
    columns: [
      { key: "name", header: "Path", flex: 2 },
      { key: "sourceType", header: "Protocol", width: 14 },
      {
        key: "bitrateBps",
        header: "Bitrate",
        width: 14,
        render: (value) =>
          ui.text(
            value === null ? "—" : formatBitrate(value as number),
          ),
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

export function renderDashboard(state: DashboardState) {
  const { config, snapshot, lastUpdatedMs } = state;
  const regexLabel = config.pathIncludeRegex
    ? `/${config.pathIncludeRegex.source}/`
    : "all";
  const limitLabel = formatBitrate(config.maxBitrateBps);
  const pollLabel = `${config.pollIntervalMs / 1000}s`;
  const errorText = snapshot.pollError ?? "";

  return ui.column({ gap: 1, p: 1 }, [
    ui.text("mtx-kicker", { style: { bold: true } }),
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
        ui.text("q quit"),
      ],
    }),
    ...(errorText
      ? [ui.callout(errorText, { variant: "error", title: "Poll error" })]
      : []),
    ui.card("Enforced", [
      sessionTable(
        "enforced-table",
        snapshot.enforced,
        config.maxBitrateBps,
        true,
      ),
    ]),
    ui.card("Other publishers", [
      sessionTable(
        "other-table",
        snapshot.other,
        config.maxBitrateBps,
        false,
      ),
    ]),
  ]);
}
