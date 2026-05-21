#!/bin/bash
# Start all 4 dev services, each in its own iTerm2 tab
# Each tab gets a "[dev]" badge so stop-dev.sh can find and close them
# Usage: ./start-dev.sh

set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"

LAUNCH_LOGIC="cd ${ROOT}/logic-grpc-service && echo '=== Logic gRPC (50051) ===' && go run main.go"
LAUNCH_GIN="cd ${ROOT}/web-gin-service && sleep 3 && echo '=== Web Gin (8080) ===' && go run main.go"
LAUNCH_HR="cd ${ROOT}/hr-frontend && echo '=== HR Frontend (5173) ===' && pnpm run dev"
LAUNCH_USER="cd ${ROOT}/user-frontend && echo '=== User Frontend (5174) ===' && pnpm run dev"

if command -v iterm2 &> /dev/null || [ -d "/Applications/iTerm.app" ]; then
    osascript \
        -e "tell application \"iTerm\"" \
        -e "  activate" \
        -e "  tell current window to create tab with default profile" \
        -e "  tell current session of current tab of current window" \
        -e "    set badge to \"[dev]\"" \
        -e "    write text \"${LAUNCH_LOGIC}\"" \
        -e "  end tell" \
        -e "  tell current window to create tab with default profile" \
        -e "  tell current session of current tab of current window" \
        -e "    set badge to \"[dev]\"" \
        -e "    write text \"${LAUNCH_GIN}\"" \
        -e "  end tell" \
        -e "  tell current window to create tab with default profile" \
        -e "  tell current session of current tab of current window" \
        -e "    set badge to \"[dev]\"" \
        -e "    write text \"${LAUNCH_HR}\"" \
        -e "  end tell" \
        -e "  tell current window to create tab with default profile" \
        -e "  tell current session of current tab of current window" \
        -e "    set badge to \"[dev]\"" \
        -e "    write text \"${LAUNCH_USER}\"" \
        -e "  end tell" \
        -e "end tell"
else
    osascript \
        -e "tell application \"Terminal\"" \
        -e "  activate" \
        -e "  do script \"${LAUNCH_LOGIC}\"" \
        -e "  do script \"${LAUNCH_GIN}\"" \
        -e "  do script \"${LAUNCH_HR}\"" \
        -e "  do script \"${LAUNCH_USER}\"" \
        -e "end tell"
fi

echo "Done! 4 iTerm2 tabs opened (badge: [dev])."
echo "  gRPC=50051  Gin=8080  HR=5173  User=5174"
