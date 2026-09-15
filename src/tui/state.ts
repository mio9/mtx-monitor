import type { Config } from "../config.ts";
import type { PollSnapshot, SessionRow } from "../poll.ts";

export type DashboardSection = "enforced" | "other";

export type SectionSelections = Record<DashboardSection, string | null>;

export type DashboardState = {
  config: Config;
  snapshot: PollSnapshot;
  lastUpdatedMs: number | null;
  activeSection: DashboardSection;
  selectedKeys: SectionSelections;
  confirmKick: SessionRow | null;
  actionMessage: string | null;
};

export function createInitialState(config: Config): DashboardState {
  return {
    config,
    snapshot: { enforced: [], other: [] },
    lastUpdatedMs: null,
    activeSection: "enforced",
    selectedKeys: { enforced: null, other: null },
    confirmKick: null,
    actionMessage: null,
  };
}

export function getSectionRows(
  state: DashboardState,
  section: DashboardSection = state.activeSection,
): readonly SessionRow[] {
  return section === "enforced"
    ? state.snapshot.enforced
    : state.snapshot.other;
}

export function getSelectedRow(state: DashboardState): SessionRow | null {
  const selectedKey = state.selectedKeys[state.activeSection];
  if (!selectedKey) {
    return null;
  }

  return (
    getSectionRows(state).find((row) => row.name === selectedKey) ?? null
  );
}

export function pruneSelections(
  selectedKeys: SectionSelections,
  snapshot: PollSnapshot,
): SectionSelections {
  const pruneSection = (
    section: DashboardSection,
    rows: readonly SessionRow[],
  ): string | null => {
    const current = selectedKeys[section];
    if (current && rows.some((row) => row.name === current)) {
      return current;
    }

    return rows[0]?.name ?? null;
  };

  return {
    enforced: pruneSection("enforced", snapshot.enforced),
    other: pruneSection("other", snapshot.other),
  };
}
