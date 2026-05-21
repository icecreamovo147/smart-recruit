#!/bin/bash
# Stop all 4 dev services (kill by port) and close their iTerm2 tabs (matched by [dev] badge)
# Usage: ./stop-dev.sh

kill_port() {
    local port="$1"
    local name="$2"
    local pids=$(lsof -ti :"$port" 2>/dev/null)
    if [ -n "$pids" ]; then
        echo "Stopping ${name} (port ${port})..."
        for pid in $pids; do
            kill "$pid" 2>/dev/null && echo "  Killed PID ${pid}" || true
        done
    else
        echo "${name} (port ${port}) is not running"
    fi
}

echo "=== Stopping all dev services ==="
echo ""

# 1. Close iTerm2 tabs first (sends SIGHUP to the shell, which kills the process)
#    Match tabs by the "[dev]" badge set by start-dev.sh
#    Iterate backwards so closing tabs doesn't shift indices
if command -v iterm2 &> /dev/null || [ -d "/Applications/iTerm.app" ]; then
    echo "Closing iTerm2 dev tabs..."
    osascript \
        -e "tell application \"iTerm\"" \
        -e "  repeat with w in windows" \
        -e "    repeat with i from (count of tabs of w) to 1 by -1" \
        -e "      set t to tab i of w" \
        -e "      try" \
        -e "        set bdg to badge of current session of t" \
        -e "        if bdg is equal to \"[dev]\" then" \
        -e "          tell t to close" \
        -e "        end if" \
        -e "      end try" \
        -e "    end repeat" \
        -e "  end repeat" \
        -e "end tell"
    echo "Dev tabs closed."
fi

echo ""

# 2. Kill any remaining processes by port (fallback for non-iTerm2 processes)
kill_port 50051 "Logic gRPC"
kill_port 8080  "Web Gin"
kill_port 5173  "HR Frontend"
kill_port 5174  "User Frontend"

echo ""
echo "All done."
