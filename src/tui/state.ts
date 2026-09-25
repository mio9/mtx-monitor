import type { Config } from "../config.ts";
import type { PollSnapshot, SessionRow, ViewerGroup, ViewerRow } from "../poll.ts";

export type ViewerUiState = {
  filterQuery: string;
  collapsedPaths: readonly string[];
  pinnedPaths: readonly string[];
  selectedPathName: string | null;
};

export const DASHBOARD_SECTIONS = ["enforced", "other", "viewers"] as const;

export type DashboardSection = (typeof DASHBOARD_SECTIONS)[number];

export type PublisherSection = Exclude<DashboardSection, "viewers">;

export type SectionSelections = Record<DashboardSection, string | null>;

export type DashboardState = {
  config: Config;
  snapshot: PollSnapshot;
  lastUpdatedMs: number | null;
  activeSection: DashboardSection;
  selectedKeys: SectionSelections;
  confirmKick: SessionRow | null;
  actionMessage: string | null;
  viewerUi: ViewerUiState;
};

export function createInitialState(config: Config): DashboardState {
  return {
    config,
    snapshot: {
      enforced: [],
      other: [],
      viewers: [],
      estimatedServerOutboundBps: null,
    },
    lastUpdatedMs: null,
    activeSection: "enforced",
    selectedKeys: { enforced: null, other: null, viewers: null },
    confirmKick: null,
    actionMessage: null,
    viewerUi: createInitialViewerUi(),
  };
}

export function createInitialViewerUi(): ViewerUiState {
  return {
    filterQuery: "",
    collapsedPaths: [],
    pinnedPaths: [],
    selectedPathName: null,
  };
}

export function prepareViewerGroups(
  groups: readonly ViewerGroup[],
  viewerUi: ViewerUiState,
): ViewerGroup[] {
  const query = viewerUi.filterQuery.trim().toLowerCase();
  const filtered = query
    ? groups.filter((group) =>
        group.pathName.toLowerCase().includes(query),
      )
    : [...groups];

  const pinnedSet = new Set(viewerUi.pinnedPaths);
  const pinnedOrder = new Map(
    viewerUi.pinnedPaths.map((pathName, index) => [pathName, index]),
  );

  filtered.sort((left, right) => {
    const leftPinned = pinnedSet.has(left.pathName);
    const rightPinned = pinnedSet.has(right.pathName);
    if (leftPinned !== rightPinned) {
      return leftPinned ? -1 : 1;
    }

    if (leftPinned && rightPinned) {
      return (
        (pinnedOrder.get(left.pathName) ?? 0) -
        (pinnedOrder.get(right.pathName) ?? 0)
      );
    }

    return left.pathName.localeCompare(right.pathName);
  });

  return filtered;
}

export function isViewerPathCollapsed(
  viewerUi: ViewerUiState,
  pathName: string,
): boolean {
  return viewerUi.collapsedPaths.includes(pathName);
}

export function isViewerPathPinned(
  viewerUi: ViewerUiState,
  pathName: string,
): boolean {
  return viewerUi.pinnedPaths.includes(pathName);
}

export function toggleViewerPathCollapsed(
  viewerUi: ViewerUiState,
  pathName: string,
): ViewerUiState {
  const collapsed = viewerUi.collapsedPaths.includes(pathName);
  return {
    ...viewerUi,
    collapsedPaths: collapsed
      ? viewerUi.collapsedPaths.filter((entry) => entry !== pathName)
      : [...viewerUi.collapsedPaths, pathName],
  };
}

export function toggleViewerPathPinned(
  viewerUi: ViewerUiState,
  pathName: string,
): ViewerUiState {
  const pinned = viewerUi.pinnedPaths.includes(pathName);
  return {
    ...viewerUi,
    pinnedPaths: pinned
      ? viewerUi.pinnedPaths.filter((entry) => entry !== pathName)
      : [...viewerUi.pinnedPaths, pathName],
  };
}

export function pruneViewerUi(
  viewerUi: ViewerUiState,
  groups: readonly ViewerGroup[],
): ViewerUiState {
  const pathNames = new Set(groups.map((group) => group.pathName));
  const visibleGroups = prepareViewerGroups(groups, viewerUi);
  const selectedPathName =
    viewerUi.selectedPathName && pathNames.has(viewerUi.selectedPathName)
      ? viewerUi.selectedPathName
      : (visibleGroups[0]?.pathName ?? null);

  return {
    ...viewerUi,
    collapsedPaths: viewerUi.collapsedPaths.filter((pathName) =>
      pathNames.has(pathName),
    ),
    pinnedPaths: viewerUi.pinnedPaths.filter((pathName) =>
      pathNames.has(pathName),
    ),
    selectedPathName,
  };
}

export function isPublisherSection(
  section: DashboardSection,
): section is PublisherSection {
  return section !== "viewers";
}

export function cycleDashboardSection(
  current: DashboardSection,
  step: number,
): DashboardSection {
  const index = DASHBOARD_SECTIONS.indexOf(current);
  const nextIndex =
    (index + step + DASHBOARD_SECTIONS.length) % DASHBOARD_SECTIONS.length;
  return DASHBOARD_SECTIONS[nextIndex] ?? current;
}

export function getSectionRows(
  state: DashboardState,
  section: DashboardSection = state.activeSection,
): readonly SessionRow[] {
  if (section === "enforced") {
    return state.snapshot.enforced;
  }

  if (section === "other") {
    return state.snapshot.other;
  }

  return [];
}

export function flattenViewers(
  snapshot: PollSnapshot,
): readonly ViewerRow[] {
  return snapshot.viewers.flatMap((group) => group.readers);
}

export function getSelectedRow(state: DashboardState): SessionRow | null {
  if (!isPublisherSection(state.activeSection)) {
    return null;
  }

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
  const prunePublisher = (
    section: PublisherSection,
    rows: readonly SessionRow[],
  ): string | null => {
    const current = selectedKeys[section];
    if (current && rows.some((row) => row.name === current)) {
      return current;
    }

    return rows[0]?.name ?? null;
  };

  return {
    enforced: prunePublisher("enforced", snapshot.enforced),
    other: prunePublisher("other", snapshot.other),
    viewers: selectedKeys.viewers,
  };
}
