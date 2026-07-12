#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "${ROOT}"

protoc --go_out=. --go-grpc_out=. proto/recruitment.proto
node ../scripts/check-proto-sync.mjs --sync
