# Dev Log Viewer

Local-only log viewer for the Smart Recruit development stack. It reads the fixed service logs under `.dev/logs`, exposes read-only APIs, and serves the React UI from the same loopback HTTP process.

## Ports And Targets

- Production viewer: `http://127.0.0.1:8090`
- Vite development UI: `http://127.0.0.1:8091`
- Start target: `./start-dev.sh logs` or `./start-dev.sh log-viewer`
- Stop target: `./stop-dev.sh logs` or `./stop-dev.sh log-viewer`

The viewer is not part of the default `./start-dev.sh` or `./start-dev.sh all` expansion.

## Build

Build the frontend first, then build the Go binary with the `prod` tag so `web/dist` is embedded into the binary:

```bash
pnpm --filter dev-log-viewer build
cd dev-log-viewer
go build -tags prod -o ../.dev/bin/dev-log-viewer ./cmd/dev-log-viewer
```

Equivalent helper:

```bash
dev-log-viewer/scripts/build-production.sh
```

## Run

Recommended local run through the root script:

```bash
./start-dev.sh logs
```

Manual run after building:

```bash
cd dev-log-viewer
../.dev/bin/dev-log-viewer -addr 127.0.0.1:8090 -root ..
```

The process writes its PID to `.dev/pids/dev-log-viewer.pid` and logs to `.dev/logs/dev-log-viewer.log` when started by `start-dev.sh`.

## Configuration

- `DEV_LOG_VIEWER_ADDR`: loopback bind address, default `127.0.0.1:8090`.
- `DEV_LOG_VIEWER_ROOT`: repository root. The command-line `-root` flag takes the same value.

The server rejects non-loopback bind addresses.

## Capacity

- Tail snapshot: up to 1,000 lines per requested service.
- Server event buffer: bounded by `stream.DefaultEventBuffer`.
- Client row buffer: 10,000 rows.
- Client export uses the current filtered in-memory rows only.

## Security Notes

- The viewer is for local development only. Do not expose it to a LAN or public address.
- APIs are read-only and restricted to the fixed service catalog.
- Logs may contain personal data, credentials, tokens, or candidate information.
- Log lines are rendered as text in the React UI.
- Static responses include CSP, `X-Content-Type-Options: nosniff`, and `Referrer-Policy: no-referrer`.

## Troubleshooting

- Empty page after manual build: rerun `pnpm --filter dev-log-viewer build`, then rebuild with `go build -tags prod`.
- `repository root is required`: pass `-root ..` from `dev-log-viewer/` or set `DEV_LOG_VIEWER_ROOT`.
- `address must bind to loopback`: use `127.0.0.1:<port>` or `localhost:<port>`.
- Missing services: start the relevant development process with `./start-dev.sh <target>`.
- Stop stale viewer process: run `./stop-dev.sh logs`.
