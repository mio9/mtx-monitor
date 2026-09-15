import {
  DEFAULT_API_URL,
  DEFAULT_MAX_BITRATE_KBPS,
  DEFAULT_POLL_INTERVAL_SEC,
} from "./constants.ts";

export type ApiAuth =
  | { scheme: "basic"; username: string; password: string }
  | { scheme: "bearer"; token: string };

export type Config = {
  apiUrl: string;
  pollIntervalMs: number;
  maxBitrateBps: number;
  apiAuth: ApiAuth | null;
  /** When set, only paths matching this regex are enforced. */
  pathIncludeRegex: RegExp | null;
};

function parsePositiveNumber(
  value: string | undefined,
  name: string,
  fallback: number,
): number {
  if (value === undefined || value === "") {
    return fallback;
  }

  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(`${name} must be a positive number, got "${value}"`);
  }

  return parsed;
}

function loadApiAuth(): ApiAuth | null {
  const token = Bun.env.MTX_API_TOKEN?.trim();
  if (token) {
    return { scheme: "bearer", token };
  }

  const username = Bun.env.MTX_API_USER?.trim();
  const password = Bun.env.MTX_API_PASSWORD ?? "";

  if (!username) {
    if (password) {
      throw new Error("MTX_API_PASSWORD set without MTX_API_USER");
    }
    return null;
  }

  return { scheme: "basic", username, password };
}

function loadPathIncludeRegex(): RegExp | null {
  const pattern = Bun.env.PATH_INCLUDE_REGEX?.trim();
  if (!pattern) {
    return null;
  }

  try {
    return new RegExp(pattern);
  } catch {
    throw new Error(`PATH_INCLUDE_REGEX is not valid: "${pattern}"`);
  }
}

export function loadConfig(): Config {
  const pollIntervalSec = parsePositiveNumber(
    Bun.env.POLL_INTERVAL_SEC,
    "POLL_INTERVAL_SEC",
    DEFAULT_POLL_INTERVAL_SEC,
  );

  const maxBitrateKbps = parsePositiveNumber(
    Bun.env.MAX_BITRATE_KBPS,
    "MAX_BITRATE_KBPS",
    DEFAULT_MAX_BITRATE_KBPS,
  );

  const apiUrl = (Bun.env.MTX_API_URL ?? DEFAULT_API_URL).replace(/\/$/, "");

  return {
    apiUrl,
    pollIntervalMs: pollIntervalSec * 1000,
    maxBitrateBps: maxBitrateKbps * 1000,
    apiAuth: loadApiAuth(),
    pathIncludeRegex: loadPathIncludeRegex(),
  };
}
