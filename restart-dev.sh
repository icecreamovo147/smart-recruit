#!/usr/bin/env bash
# Restart local dev services managed by start-dev.sh and stop-dev.sh.
# Usage:
#   ./restart-dev.sh                         # restart the full dev stack
#   ./restart-dev.sh business                # restart all business services
#   ./restart-dev.sh identity recruitment    # restart selected business services
#   ./restart-dev.sh gateway frontends       # restart gateway and all frontends

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
    cat <<'EOF'
Usage:
  ./restart-dev.sh [target ...]

Targets are the same as start-dev.sh and stop-dev.sh:
  all, business, backend, gateway, frontends
  identity, recruitment, interview, offer, notification
  ai-agent | ai, analytics, worker
  hr, user, platform

Examples:
  ./restart-dev.sh business
  ./restart-dev.sh identity recruitment notification
  ./restart-dev.sh backend hr
EOF
}

case "${1:-}" in
    -h|--help|help)
        usage
        exit 0
        ;;
esac

printf '[dev] Restarting selected dev services: %s\n' "${*:-all}"
"${ROOT}/stop-dev.sh" "$@"
"${ROOT}/start-dev.sh" "$@"
