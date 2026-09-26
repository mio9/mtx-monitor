type ByteSample = {
  bytes: number;
  timeMs: number;
};

export class BitrateTracker {
  private readonly samples = new Map<string, ByteSample>();

  /** Returns instantaneous bitrate in bits per second, or null on first sample. */
  update(pathName: string, bytes: number, timeMs: number): number | null {
    const previous = this.samples.get(pathName);
    this.samples.set(pathName, { bytes, timeMs });

    if (!previous) {
      return null;
    }

    const deltaBytes = bytes - previous.bytes;
    const deltaMs = timeMs - previous.timeMs;

    if (deltaBytes < 0 || deltaMs <= 0) {
      return null;
    }

    return (deltaBytes * 8 * 1000) / deltaMs;
  }

  forget(pathName: string): void {
    this.samples.delete(pathName);
  }

  forgetMissing(activePathNames: ReadonlySet<string>): void {
    for (const pathName of this.samples.keys()) {
      if (!activePathNames.has(pathName)) {
        this.samples.delete(pathName);
      }
    }
  }
}

export function formatBitrate(bps: number): string {
  if (bps >= 1_000_000) {
    return `${(bps / 1_000_000).toFixed(2)} Mbps`;
  }

  if (bps >= 1_000) {
    return `${(bps / 1_000).toFixed(1)} kbps`;
  }

  return `${Math.round(bps)} bps`;
}

export function formatBytes(bytes: number): string {
  if (bytes >= 1_000_000_000) {
    return `${(bytes / 1_000_000_000).toFixed(2)} GB`;
  }

  if (bytes >= 1_000_000) {
    return `${(bytes / 1_000_000).toFixed(2)} MB`;
  }

  if (bytes >= 1_000) {
    return `${(bytes / 1_000).toFixed(1)} KB`;
  }

  return `${bytes} B`;
}
