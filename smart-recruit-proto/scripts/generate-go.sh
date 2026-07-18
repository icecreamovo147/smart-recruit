#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT_DIR="${ROOT}/scripts"
# shellcheck source=tool-versions.env
source "${SCRIPT_DIR}/tool-versions.env"

cache_root="${PROTO_TOOL_CACHE_DIR:-${XDG_CACHE_HOME:-${HOME}/.cache}/smart-recruit/protobuf}"
pinned_bin="${PROTO_TOOL_BIN:-${cache_root}/${PROTOC_VERSION}/bin}"
if [ -x "${pinned_bin}/protoc" ]; then
    export PATH="${pinned_bin}:${PATH}"
fi

require_version() {
    local command_name expected actual
    command_name="$1"
    expected="$2"

    if ! command -v "${command_name}" >/dev/null 2>&1; then
        printf '[proto-generate] ERROR: %s is not installed. Run ./scripts/bootstrap-tools.sh first.\n' "${command_name}" >&2
        exit 1
    fi

    actual="$("${command_name}" --version 2>/dev/null)"
    if [ "${actual}" != "${expected}" ]; then
        printf '[proto-generate] ERROR: %s version mismatch.\n' "${command_name}" >&2
        printf '  expected: %s\n' "${expected}" >&2
        printf '  actual:   %s\n' "${actual}" >&2
        printf 'Run ./scripts/bootstrap-tools.sh and ensure its printed bin directory is first on PATH.\n' >&2
        exit 1
    fi
}

cd "${ROOT}"

require_version protoc "libprotoc ${PROTOC_VERSION}"
require_version protoc-gen-go "protoc-gen-go v${PROTOC_GEN_GO_VERSION}"
require_version protoc-gen-go-grpc "protoc-gen-go-grpc ${PROTOC_GEN_GO_GRPC_VERSION}"

protoc --go_out=. --go-grpc_out=. proto/recruitment.proto
node ../scripts/check-proto-sync.mjs --check
