# mtx-kicker

Polls MediaMTX Control API, tracks publisher bitrate from byte counters, kicks sessions over limit.

## Requirements

MediaMTX with Control API enabled:

```yaml
api: yes
apiAddress: :9997
```

## Config

Copy `.env.example` to `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `MTX_API_URL` | `http://127.0.0.1:9997` | MediaMTX Control API base URL |
| `MTX_API_USER` | — | Basic auth username (`action: api` user in MediaMTX) |
| `MTX_API_PASSWORD` | — | Basic auth password |
| `MTX_API_TOKEN` | — | Bearer token for JWT auth (overrides basic if set) |
| `PATH_INCLUDE_REGEX` | — | Only enforce paths whose name matches this regex |
| `POLL_INTERVAL_SEC` | `3` | Poll interval in seconds |
| `MAX_BITRATE_KBPS` | `5000` | Max allowed publisher bitrate in kbps |

MediaMTX internal auth example:

```yaml
authInternalUsers:
  - user: api
    pass: secret
    ips: []
    permissions:
      - action: api
```

## Run

```bash
bun install
bun run index.ts
```

Default launch opens a Rezi TUI dashboard. Press `q` to quit.

Headless console mode (original log output):

```bash
bun run index.ts --noui
```

### Dashboard sections

- **Enforced** — paths matched by `PATH_INCLUDE_REGEX` (or all publishers when unset). Over-limit sessions are flagged and kicked.
- **Other publishers** — active publisher paths not matched by the regex. Shown for visibility only; over-limit sessions are flagged but not kicked.

Bitrate = delta `inboundBytes` between polls. Supported publisher types: RTSP, RTMP, SRT, WebRTC.
