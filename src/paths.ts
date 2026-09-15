import { PUBLISHER_SOURCE_TYPES } from "./constants.ts";
import type { Path } from "./mediamtx/types.ts";

export type PublishingPath = Path & { source: NonNullable<Path["source"]> };

export function isPublishingPath(path: Path): path is PublishingPath {
  return (
    path.source !== null &&
    path.online &&
    PUBLISHER_SOURCE_TYPES.has(path.source.type)
  );
}

export type SplitPublishingPaths = {
  enforced: PublishingPath[];
  other: PublishingPath[];
};

export function splitPublishingPaths(
  paths: readonly Path[],
  pathIncludeRegex: RegExp | null,
): SplitPublishingPaths {
  const publishingPaths = paths.filter(isPublishingPath);

  if (pathIncludeRegex === null) {
    return { enforced: publishingPaths, other: [] };
  }

  const enforced: PublishingPath[] = [];
  const other: PublishingPath[] = [];

  for (const path of publishingPaths) {
    if (pathIncludeRegex.test(path.name)) {
      enforced.push(path);
    } else {
      other.push(path);
    }
  }

  return { enforced, other };
}
