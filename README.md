# mtxmon

Polls the MediaMTX Control API, tracks publisher bitrate from byte counters, and kicks sessions over the limit.

Go module: `mio9/mtx-monitor` (`go/go.mod`, Go 1.27).

## Requirements

MediaMTX with the Control API enabled:

```yaml
api: yes
apiAddress: :9997
```

`ffplay` (from ffmpeg) is used when you watch a path. Override the binary with `WATCH_PLAYER`.

## Config

Copy `.env.example` to `.env` in the directory you run `mtxmon` from. The process loads that file from the working directory.

| Variable | Default | Description |
|----------|---------|-------------|
| `MTX_API_URL` | `http://127.0.0.1:9997` | MediaMTX Control API base URL |
| `MTX_API_USER` | — | Basic auth username (`action: api` user in MediaMTX) |
| `MTX_API_PASSWORD` | — | Basic auth password |
| `MTX_API_TOKEN` | — | Bearer token for JWT auth (overrides basic if set) |
| `PATH_INCLUDE_REGEX` | — | Only enforce paths whose name matches this regex |
| `MTX_RTSP_URL` | `rtsp://<api-host>:8554` | RTSP base URL for stream watch |
| `WATCH_PLAYER` | `ffplay` | Player binary (`ffplay` from ffmpeg) |
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
cd go
go build -o ../mtxmon ./cmd/mtxmon
cd ..
./mtxmon
```

That opens the Bubble Tea TUI. Headless log output:

```bash
./mtxmon --noui
```

Version:

```bash
./mtxmon version
```

| Key | Action |
|-----|--------|
| `←` / `→` | Switch dashboard section |
| `w` | Watch the selected publisher with `WATCH_PLAYER` |
| `k` | Kick the selected publisher |
| `y` | Confirm kick |
| `n` / `esc` | Cancel kick |
| `c` | Collapse or expand the selected viewer path |
| `p` | Pin or unpin the selected viewer path |
| `q` | Quit |

### Dashboard sections

- **Enforced** — paths matched by `PATH_INCLUDE_REGEX` (or all publishers when unset). Over-limit sessions are flagged and kicked.
- **Other publishers** — active publisher paths not matched by the regex. Shown for visibility only; over-limit sessions are flagged but not kicked.
- **Viewers** — readers grouped by path. `c` collapses a path, `p` pins it.
- **Instance** — connected MediaMTX server from `/v3/info`: API URL, auth, version, start time, poll interval, bitrate limit, and path filter.

Bitrate is the delta of `inboundBytes` between polls. Kickable publisher types: RTSP, RTSPS, RTMP, RTMPS, SRT, WebRTC.
