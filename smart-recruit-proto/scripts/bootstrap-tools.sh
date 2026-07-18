#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=tool-versions.env
source "${SCRIPT_DIR}/tool-versions.env"

cache_root="${PROTO_TOOL_CACHE_DIR:-${XDG_CACHE_HOME:-${HOME}/.cache}/smart-recruit/protobuf}"
tool_root="${cache_root}/${PROTOC_VERSION}"
bin_dir="${tool_root}/bin"

log() {
    printf '[proto-tools] %s\n' "$*" >&2
}

require_command() {
    if ! command -v "$1" >/dev/null 2>&1; then
        printf '[proto-tools] ERROR: required command not found: %s\n' "$1" >&2
        exit 1
    fi
}

protoc_asset_suffix() {
    local os arch
    os="$(uname -s)"
    arch="$(uname -m)"

    case "${os}:${arch}" in
        Linux:x86_64) printf 'linux-x86_64\n' ;;
        Linux:aarch64|Linux:arm64) printf 'linux-aarch_64\n' ;;
        Darwin:x86_64) printf 'osx-x86_64\n' ;;
        Darwin:arm64) printf 'osx-aarch_64\n' ;;
        *)
            printf '[proto-tools] ERROR: unsupported platform: %s/%s\n' "${os}" "${arch}" >&2
            exit 1
            ;;
    esac
}

protoc_matches() {
    [ -x "${bin_dir}/protoc" ] && [ "$("${bin_dir}/protoc" --version 2>/dev/null)" = "libprotoc ${PROTOC_VERSION}" ]
}

plugin_matches() {
    local executable expected
    executable="$1"
    expected="$2"
    [ -x "${bin_dir}/${executable}" ] && [ "$("${bin_dir}/${executable}" --version 2>/dev/null)" = "${expected}" ]
}

mkdir -p "${bin_dir}"

if ! protoc_matches; then
    require_command curl
    require_command unzip
    asset_suffix="$(protoc_asset_suffix)"
    archive="protoc-${PROTOC_VERSION}-${asset_suffix}.zip"
    download_url="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${archive}"
    temp_dir="$(mktemp -d)"
    trap 'rm -rf "${temp_dir}"' EXIT

    log "Downloading protoc ${PROTOC_VERSION} for ${asset_suffix}"
    curl --fail --location --silent --show-error "${download_url}" --output "${temp_dir}/${archive}"
    unzip -q -o "${temp_dir}/${archive}" -d "${tool_root}"
fi

if ! plugin_matches protoc-gen-go "protoc-gen-go v${PROTOC_GEN_GO_VERSION}"; then
    require_command go
    log "Installing protoc-gen-go v${PROTOC_GEN_GO_VERSION}"
    GOBIN="${bin_dir}" go install "google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOC_GEN_GO_VERSION}"
fi

if ! plugin_matches protoc-gen-go-grpc "protoc-gen-go-grpc ${PROTOC_GEN_GO_GRPC_VERSION}"; then
    require_command go
    log "Installing protoc-gen-go-grpc v${PROTOC_GEN_GO_GRPC_VERSION}"
    GOBIN="${bin_dir}" go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${PROTOC_GEN_GO_GRPC_VERSION}"
fi

log "Toolchain ready: $("${bin_dir}/protoc" --version), $("${bin_dir}/protoc-gen-go" --version), $("${bin_dir}/protoc-gen-go-grpc" --version)"
printf '%s\n' "${bin_dir}"
