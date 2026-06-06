#!/usr/bin/env bash
# Stop local dev services started by start-dev.sh.
# Usage: ./stop-dev.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="${ROOT}/.dev"
PID_DIR="${STATE_DIR}/pids"

info() {
    printf '[dev] %s\n' "$*"
}

stop_pid_file() {
    local name="$1"
    local pid_file="${PID_DIR}/${name}.pid"

    if [ ! -f "${pid_file}" ]; then
        return 0
    fi

    local pid
    pid="$(cat "${pid_file}")"
    rm -f "${pid_file}"

    if [ -z "${pid}" ] || ! kill -0 "${pid}" >/dev/null 2>&1; then
        info "${name} is not running."
        return 0
    fi

    info "Stopping ${name} (PID ${pid})..."
    kill "${pid}" >/dev/null 2>&1 || true

    for _ in $(seq 1 20); do
        if ! kill -0 "${pid}" >/dev/null 2>&1; then
            info "Stopped ${name}."
            return 0
        fi
        sleep 0.2
    done

    info "${name} did not exit cleanly; force stopping PID ${pid}."
    kill -9 "${pid}" >/dev/null 2>&1 || true
}

kill_port() {
    local port="$1"
    local name="$2"
    local pids

    pids="$(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)"
    if [ -z "${pids}" ]; then
        info "${name} (port ${port}) is not running."
        return 0
    fi

    info "Stopping ${name} fallback listener(s) on port ${port}..."
    for pid in ${pids}; do
        kill "${pid}" >/dev/null 2>&1 && info "Killed PID ${pid}." || true
    done
}

info "Stopping dev services..."

stop_pid_file "user-frontend"
stop_pid_file "hr-frontend"
stop_pid_file "interviewer-frontend"
stop_pid_file "web-gin-service"
stop_pid_file "logic-grpc-service"

kill_port 5174 "User Frontend"
kill_port 5173 "HR Frontend"
kill_port 5175 "Interviewer Frontend"
kill_port 8080 "Web Gin"
kill_port 50051 "Logic gRPC"

info "All done."
