#!/usr/bin/env bash
# Stop local dev services started by start-dev.sh.
# Usage:
#   ./stop-dev.sh                         # stop the full dev stack
#   ./stop-dev.sh business                # stop all business services
#   ./stop-dev.sh identity recruitment    # stop selected business services
#   ./stop-dev.sh gateway frontends       # stop gateway and all frontends
#   ./stop-dev.sh logs                    # stop the local dev log viewer

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="${ROOT}/.dev"
PID_DIR="${STATE_DIR}/pids"

info() {
    printf '[dev] %s\n' "$*"
}

die() {
    printf '[dev] ERROR: %s\n' "$*" >&2
    exit 1
}

usage() {
    cat <<'EOF'
Usage:
  ./stop-dev.sh [target ...]

Targets:
  all                 Full dev stack: business services, gateway, and frontends.
  business            All business services only.
  backend             Business services plus gateway.
  gateway             HTTP gateway only.
  frontends           Staff workspace, candidate portal, and platform console.
  logs | log-viewer   Local dev log viewer only.

Business service aliases:
  identity            identity-service
  recruitment         recruitment-service
  interview           interview-service
  offer               offer-service
  notification        notification-service
  ai-agent | ai       ai-agent-service
  analytics           analytics-service
  billing             billing-service
  worker              worker-service

Frontend aliases:
  hr                  hr-frontend
  user                user-frontend
  platform            platform-frontend

Examples:
  ./stop-dev.sh business
  ./stop-dev.sh identity recruitment notification
  ./stop-dev.sh backend hr
  ./stop-dev.sh logs
EOF
}

BUSINESS_SERVICES=(
    identity-service
    recruitment-service
    interview-service
    offer-service
    notification-service
    ai-agent-service
    analytics-service
    billing-service
    worker-service
)

FRONTEND_SERVICES=(
    hr-frontend
    user-frontend
    platform-frontend
)

ALL_SERVICES=(
    "${BUSINESS_SERVICES[@]}"
    smart-recruit-gateway
    "${FRONTEND_SERVICES[@]}"
)

TARGETS=()

add_target() {
    local target="$1"
    local existing
    for existing in "${TARGETS[@]-}"; do
        if [ "${existing}" = "${target}" ]; then
            return 0
        fi
    done
    TARGETS+=("${target}")
}

add_many() {
    local target
    for target in "$@"; do
        add_target "${target}"
    done
}

expand_target() {
    local raw="$1"
    case "${raw}" in
        -h|--help|help)
            usage
            exit 0
            ;;
        all)
            add_many "${ALL_SERVICES[@]}"
            ;;
        business|business-services|services)
            add_many "${BUSINESS_SERVICES[@]}"
            ;;
        backend|backends)
            add_many "${BUSINESS_SERVICES[@]}" smart-recruit-gateway
            ;;
        gateway|smart-recruit-gateway)
            add_target smart-recruit-gateway
            ;;
        frontends|frontend)
            add_many "${FRONTEND_SERVICES[@]}"
            ;;
        logs|log-viewer|dev-log-viewer)
            add_target dev-log-viewer
            ;;
        identity|identity-service)
            add_target identity-service
            ;;
        recruitment|recruitment-service)
            add_target recruitment-service
            ;;
        interview|interview-service)
            add_target interview-service
            ;;
        offer|offer-service)
            add_target offer-service
            ;;
        notification|notification-service)
            add_target notification-service
            ;;
        ai|ai-agent|ai-agent-service)
            add_target ai-agent-service
            ;;
        analytics|analytics-service)
            add_target analytics-service
            ;;
        billing|billing-service)
            add_target billing-service
            ;;
        worker|worker-service)
            add_target worker-service
            ;;
        hr|hr-frontend)
            add_target hr-frontend
            ;;
        user|user-frontend)
            add_target user-frontend
            ;;
        platform|platform-frontend)
            add_target platform-frontend
            ;;
        *)
            die "Unknown target: ${raw}. Run ./stop-dev.sh --help for supported targets."
            ;;
    esac
}

parse_targets() {
    if [ "$#" -eq 0 ]; then
        add_many "${ALL_SERVICES[@]}"
        return 0
    fi

    local arg
    for arg in "$@"; do
        expand_target "${arg}"
    done
}

target_selected() {
    local target="$1"
    local selected
    for selected in "${TARGETS[@]-}"; do
        if [ "${selected}" = "${target}" ]; then
            return 0
        fi
    done
    return 1
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
    if [ -z "${pids}" ] && command -v ss >/dev/null 2>&1; then
        pids="$(
            ss -H -ltnp sport = :"${port}" 2>/dev/null \
                | sed -n 's/.*pid=\([0-9]\+\).*/\1/p' \
                | sort -u \
                | tr '\n' ' '
        )"
    fi
    if [ -z "${pids}" ]; then
        info "${name} (port ${port}) is not running."
        return 0
    fi

    info "Stopping ${name} fallback listener(s) on port ${port}..."
    for pid in ${pids}; do
        kill "${pid}" >/dev/null 2>&1 && info "Killed PID ${pid}." || true
    done
}

parse_targets "$@"

info "Stopping selected dev services: ${TARGETS[*]}"

target_selected user-frontend && stop_pid_file "user-frontend"
target_selected hr-frontend && stop_pid_file "hr-frontend"
target_selected platform-frontend && stop_pid_file "platform-frontend"
target_selected dev-log-viewer && stop_pid_file "dev-log-viewer"
target_selected smart-recruit-gateway && stop_pid_file "smart-recruit-gateway"
target_selected worker-service && stop_pid_file "worker-service"
target_selected analytics-service && stop_pid_file "analytics-service"
target_selected billing-service && stop_pid_file "billing-service"
target_selected ai-agent-service && stop_pid_file "ai-agent-service"
target_selected notification-service && stop_pid_file "notification-service"
target_selected offer-service && stop_pid_file "offer-service"
target_selected interview-service && stop_pid_file "interview-service"
target_selected recruitment-service && stop_pid_file "recruitment-service"
target_selected identity-service && stop_pid_file "identity-service"

target_selected user-frontend && kill_port 5174 "User Frontend"
target_selected hr-frontend && kill_port 5173 "HR Frontend"
target_selected platform-frontend && kill_port 5175 "Platform Frontend"
target_selected dev-log-viewer && kill_port 8090 "Dev Log Viewer"
target_selected smart-recruit-gateway && kill_port 8080 "Smart Recruit Gateway"
target_selected worker-service && kill_port 50068 "Worker Service"
target_selected analytics-service && kill_port 50067 "Analytics Service"
target_selected billing-service && kill_port 50069 "Billing Service"
target_selected ai-agent-service && kill_port 50066 "AI Agent Service"
target_selected notification-service && kill_port 50065 "Notification Service"
target_selected offer-service && kill_port 50064 "Offer Service"
target_selected interview-service && kill_port 50063 "Interview Service"
target_selected recruitment-service && kill_port 50062 "Recruitment Service"
target_selected identity-service && kill_port 50061 "Identity Service"

info "All done."
