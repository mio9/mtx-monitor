import type { Config } from "../config.ts";
import type { PollSnapshot } from "../poll.ts";

export type DashboardState = {
  config: Config;
  snapshot: PollSnapshot;
  lastUpdatedMs: number | null;
};

export function createInitialState(config: Config): DashboardState {
  return {
    config,
    snapshot: { enforced: [], other: [] },
    lastUpdatedMs: null,
  };
}
