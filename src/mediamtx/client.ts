import {
  DEPRECATED_API_PATHS,
  KICK_ENDPOINTS,
  PATHS_LIST,
  PATHS_PAGE_SIZE,
  RTMP_CONNS_LIST,
  RTMPS_CONNS_LIST,
  RTSP_SESSIONS_LIST,
  RTSPS_SESSIONS_LIST,
} from "../constants.ts";
import type { ApiAuth } from "../config.ts";
import type {
  ErrorResponse,
  OkResponse,
  PaginatedListResponse,
  Path,
  PathSource,
  RtmpConn,
  RtspSession,
} from "./types.ts";
import type { HeadersInit } from "bun";

export function buildApiUrl(baseUrl: string, endpoint: string): URL {
  const normalizedBase = baseUrl.replace(/\/$/, "");
  const normalizedEndpoint = endpoint.startsWith("/")
    ? endpoint
    : `/${endpoint}`;

  if (normalizedBase.endsWith("/v3") && normalizedEndpoint.startsWith("/v3/")) {
    return new URL(`${normalizedBase}${normalizedEndpoint.slice(3)}`);
  }

  return new URL(`${normalizedBase}${normalizedEndpoint}`);
}

function endpointCandidates(endpoint: string): string[] {
  const deprecated = DEPRECATED_API_PATHS[endpoint];
  return deprecated ? [endpoint, deprecated] : [endpoint];
}

export class MediaMtxClient {
  constructor(
    private readonly baseUrl: string,
    private readonly apiAuth: ApiAuth | null = null,
  ) {}

  private authHeaders(): HeadersInit {
    if (!this.apiAuth) {
      return {};
    }

    if (this.apiAuth.scheme === "bearer") {
      return { Authorization: `Bearer ${this.apiAuth.token}` };
    }

    const credentials = btoa(`${this.apiAuth.username}:${this.apiAuth.password}`);
    return { Authorization: `Basic ${credentials}` };
  }

  private async request(url: URL, init?: RequestInit): Promise<Response> {
    return fetch(url, {
      ...init,
      headers: {
        ...this.authHeaders(),
        ...init?.headers,
      },
    });
  }

  private async listPaginatedOnce<T>(
    endpoint: string,
    label: string,
  ): Promise<T[]> {
    const items: T[] = [];
    let page = 0;

    while (true) {
      const url = buildApiUrl(this.baseUrl, endpoint);
      url.searchParams.set("page", String(page));
      url.searchParams.set("itemsPerPage", String(PATHS_PAGE_SIZE));

      const response = await this.request(url);
      if (!response.ok) {
        throw new Error(
          `${label} failed: ${response.status} ${response.statusText}${authHint(response.status)}`,
        );
      }

      const body = (await response.json()) as PaginatedListResponse<T>;
      items.push(...body.items);

      page += 1;
      if (page >= body.pageCount) {
        break;
      }
    }

    return items;
  }

  private async listPaginated<T>(
    endpoint: string,
    label: string,
    optional = false,
  ): Promise<T[]> {
    const candidates = endpointCandidates(endpoint);

    for (const [index, candidate] of candidates.entries()) {
      const isLastCandidate = index === candidates.length - 1;

      try {
        return await this.listPaginatedOnce<T>(candidate, label);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        const is404 = message.includes("404");

        if (is404 && !isLastCandidate) {
          continue;
        }

        if (is404 && optional) {
          return [];
        }

        throw error;
      }
    }

    return [];
  }

  async listPaths(): Promise<Path[]> {
    return this.listPaginated<Path>(PATHS_LIST, "paths/list");
  }

  async listRtspSessions(): Promise<RtspSession[]> {
    return this.listPaginated<RtspSession>(
      RTSP_SESSIONS_LIST,
      "rtsp/sessions/list",
      true,
    );
  }

  async listRtspsSessions(): Promise<RtspSession[]> {
    return this.listPaginated<RtspSession>(
      RTSPS_SESSIONS_LIST,
      "rtsps/sessions/list",
      true,
    );
  }

  async listRtmpConns(): Promise<RtmpConn[]> {
    return this.listPaginated<RtmpConn>(
      RTMP_CONNS_LIST,
      "rtmp/conns/list",
      true,
    );
  }

  async listRtmpsConns(): Promise<RtmpConn[]> {
    return this.listPaginated<RtmpConn>(
      RTMPS_CONNS_LIST,
      "rtmps/conns/list",
      true,
    );
  }

  async kickPublisher(source: PathSource): Promise<void> {
    const endpoint = KICK_ENDPOINTS[source.type];
    if (!endpoint) {
      throw new Error(`no kick endpoint for source type "${source.type}"`);
    }

    const candidates = endpointCandidates(endpoint);
    let lastError: Error | null = null;

    for (const [index, candidate] of candidates.entries()) {
      const isLastCandidate = index === candidates.length - 1;
      const url = buildApiUrl(this.baseUrl, `${candidate}/${source.id}`);
      const response = await this.request(url, { method: "POST" });

      if (response.ok) {
        const body = (await response.json()) as OkResponse;
        if (body.status !== "ok") {
          throw new Error(`unexpected kick response: ${JSON.stringify(body)}`);
        }
        return;
      }

      const body = (await response.json().catch(() => null)) as
        | ErrorResponse
        | null;
      const detail = body?.error ?? response.statusText;
      lastError = new Error(
        `kick failed (${response.status}): ${detail}${authHint(response.status)}`,
      );

      if (response.status === 404 && !isLastCandidate) {
        continue;
      }

      throw lastError;
    }

    if (lastError) {
      throw lastError;
    }
  }
}

function authHint(status: number): string {
  if (status !== 401) {
    return "";
  }

  return " (check MTX_API_USER/MTX_API_PASSWORD or MTX_API_TOKEN)";
}
