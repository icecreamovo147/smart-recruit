#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ADDR="${DEV_LOG_VIEWER_SMOKE_ADDR:-127.0.0.1:18090}"
BIN="${ROOT}/.dev/bin/dev-log-viewer-smoke"
LOG="${ROOT}/.dev/logs/dev-log-viewer-smoke.log"
PID=""

cleanup() {
    if [ -n "${PID}" ] && kill -0 "${PID}" >/dev/null 2>&1; then
        kill "${PID}" >/dev/null 2>&1 || true
        wait "${PID}" 2>/dev/null || true
    fi
    rm -f "${BIN}"
}
trap cleanup EXIT

mkdir -p "${ROOT}/.dev/bin" "${ROOT}/.dev/logs"

(cd "${ROOT}" && pnpm --filter dev-log-viewer build >/tmp/dev-log-viewer-smoke-build.log)
(cd "${ROOT}/dev-log-viewer" && go build -tags prod -o "${BIN}" ./cmd/dev-log-viewer)

(cd "${ROOT}/dev-log-viewer" && "${BIN}" -addr "${ADDR}" -root "${ROOT}" >"${LOG}" 2>&1 & echo "$!" > /tmp/dev-log-viewer-smoke.pid)
PID="$(cat /tmp/dev-log-viewer-smoke.pid)"

for _ in $(seq 1 40); do
    if curl -fsS "http://${ADDR}/healthz" >/tmp/dev-log-viewer-smoke-health.json 2>/dev/null; then
        break
    fi
    sleep 0.1
done

curl -fsS "http://${ADDR}/healthz" | grep -q '"service":"dev-log-viewer"'
curl -fsS "http://${ADDR}/api/v1/services" | grep -q '"services"'
curl -fsS "http://${ADDR}/" >/tmp/dev-log-viewer-smoke-index.html
curl -fsSI "http://${ADDR}/" >/tmp/dev-log-viewer-smoke-headers.txt

grep -qi '^Content-Security-Policy:' /tmp/dev-log-viewer-smoke-headers.txt
grep -qi '^X-Content-Type-Options: nosniff' /tmp/dev-log-viewer-smoke-headers.txt
grep -qi '^Referrer-Policy: no-referrer' /tmp/dev-log-viewer-smoke-headers.txt

if curl -fsS "http://${ADDR}/../../../../etc/passwd" | grep -q 'root:'; then
    echo "unexpected filesystem content leaked through static route" >&2
    exit 1
fi

if grep -Eiq 'https?://|fonts\.googleapis|cdn\.' /tmp/dev-log-viewer-smoke-index.html; then
    echo "unexpected third-party URL in built index" >&2
    exit 1
fi

timeout 2s curl -fsS --no-buffer "http://${ADDR}/api/v1/logs/stream?tail=0" >/tmp/dev-log-viewer-smoke-sse.txt 2>/dev/null || true
grep -q 'event: snapshot_start' /tmp/dev-log-viewer-smoke-sse.txt
grep -q 'event: snapshot_end' /tmp/dev-log-viewer-smoke-sse.txt

echo "dev-log-viewer smoke: PASS (${ADDR})"
