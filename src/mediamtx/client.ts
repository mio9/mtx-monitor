import { KICK_ENDPOINTS, PATHS_PAGE_SIZE } from "../constants.ts";
import type { ApiAuth } from "../config.ts";
import type {
  ErrorResponse,
  OkResponse,
  Path,
  PathListResponse,
  PathSource,
} from "./types.ts";
import type { HeadersInit } from "bun";

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

  async listPaths(): Promise<Path[]> {
    const paths: Path[] = [];
    let page = 0;

    while (true) {
      const url = new URL("/v3/paths/list", this.baseUrl);
      url.searchParams.set("page", String(page));
      url.searchParams.set("itemsPerPage", String(PATHS_PAGE_SIZE));

      const response = await this.request(url);
      if (!response.ok) {
        throw new Error(
          `paths/list failed: ${response.status} ${response.statusText}${authHint(response.status)}`,
        );
      }

      const body = (await response.json()) as PathListResponse;
      paths.push(...body.items);

      page += 1;
      if (page >= body.pageCount) {
        break;
      }
    }

    return paths;
  }

  async kickPublisher(source: PathSource): Promise<void> {
    const endpoint = KICK_ENDPOINTS[source.type];
    if (!endpoint) {
      throw new Error(`no kick endpoint for source type "${source.type}"`);
    }

    const url = new URL(`${endpoint}/${source.id}`, this.baseUrl);
    const response = await this.request(url, { method: "POST" });

    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as
        | ErrorResponse
        | null;
      const detail = body?.error ?? response.statusText;
      throw new Error(
        `kick failed (${response.status}): ${detail}${authHint(response.status)}`,
      );
    }

    const body = (await response.json()) as OkResponse;
    if (body.status !== "ok") {
      throw new Error(`unexpected kick response: ${JSON.stringify(body)}`);
    }
  }
}

function authHint(status: number): string {
  if (status !== 401) {
    return "";
  }

  return " (check MTX_API_USER/MTX_API_PASSWORD or MTX_API_TOKEN)";
}
