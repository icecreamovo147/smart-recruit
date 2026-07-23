#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

mkdir -p "${ROOT}/.dev/bin"
(cd "${ROOT}" && pnpm --filter dev-log-viewer build)
(cd "${ROOT}/dev-log-viewer" && go build -tags prod -o "${ROOT}/.dev/bin/dev-log-viewer" ./cmd/dev-log-viewer)
