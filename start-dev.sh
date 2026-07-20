#!/usr/bin/env bash
# Start the local Smart Recruit dev stack without Docker.
# Usage:
#   ./start-dev.sh                         # start the full dev stack
#   ./start-dev.sh business                # start all business services
#   ./start-dev.sh identity recruitment    # start selected business services
#   ./start-dev.sh gateway frontends       # start gateway and all frontends
#   ./start-dev.sh logs                    # start the local dev log viewer

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="${ROOT}/.dev"
PID_DIR="${STATE_DIR}/pids"
LOG_DIR="${STATE_DIR}/logs"
BIN_DIR="${STATE_DIR}/bin"
PNPM_VERSION="10.19.0"

mkdir -p "${PID_DIR}" "${LOG_DIR}" "${BIN_DIR}"

info() { printf '[dev] %s\n' "$*"; }
warn() { printf '[dev] WARN: %s\n' "$*" >&2; }
die() {
    printf '[dev] ERROR: %s\n' "$*" >&2
    exit 1
}

usage() {
    cat <<'EOF'
Usage:
  ./start-dev.sh [target ...]

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
  ./start-dev.sh business
  ./start-dev.sh identity recruitment notification
  ./start-dev.sh backend hr
  ./start-dev.sh logs
EOF
}

install_with_package_manager() {
    local command_name="$1"
    local mac_package="$2"
    local apt_package="$3"

    if command -v "${command_name}" >/dev/null 2>&1; then
        return 0
    fi

    warn "${command_name} is not installed; trying to install it."
    if command -v brew >/dev/null 2>&1; then
        brew install "${mac_package}"
    elif command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update
        sudo apt-get install -y "${apt_package}"
    else
        die "Cannot install ${command_name} automatically. Please install ${mac_package} and rerun this script."
    fi

    command -v "${command_name}" >/dev/null 2>&1 || die "${command_name} installation did not make the command available."
}

ensure_pnpm() {
    if command -v pnpm >/dev/null 2>&1; then
        return 0
    fi

    warn "pnpm is not installed; trying to install pnpm@${PNPM_VERSION}."
    if command -v corepack >/dev/null 2>&1; then
        corepack enable
        corepack prepare "pnpm@${PNPM_VERSION}" --activate
    elif command -v npm >/dev/null 2>&1; then
        npm install -g "pnpm@${PNPM_VERSION}"
    else
        die "Cannot install pnpm because neither corepack nor npm is available."
    fi
}

ensure_system_dependencies() {
    install_with_package_manager lsof lsof lsof
    if has_any_backend_target || has_log_viewer_target; then
        install_with_package_manager go go golang-go
    fi
    if has_any_frontend_target || has_log_viewer_target; then
        install_with_package_manager node node nodejs
        ensure_pnpm
    fi
}

port_in_use() {
    local port="$1"

    # Linux: lsof -sTCP:LISTEN often misses system service listeners; prefer ss/nc.
    if command -v ss >/dev/null 2>&1; then
        ss -H -ltn sport = :"${port}" 2>/dev/null | grep -q .
        return $?
    fi

    if lsof -tiTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
        return 0
    fi

    if command -v nc >/dev/null 2>&1; then
        nc -z 127.0.0.1 "${port}" >/dev/null 2>&1
        return $?
    fi

    return 1
}

ensure_runtime_services() {
    local missing=""
    if ! port_in_use 3306; then
        missing="${missing}
  - MySQL is not detected on 127.0.0.1:3306"
    fi
    if ! port_in_use 6379; then
        missing="${missing}
  - Redis is not detected on 127.0.0.1:6379"
    fi
    if ! port_in_use 5672; then
        missing="${missing}
  - RabbitMQ is not detected on 127.0.0.1:5672"
    fi

    if [ -n "${missing}" ]; then
        cat >&2 <<EOF
[dev] ERROR: Required local runtime service(s) are not running:
${missing}

Start the missing service(s), then rerun ./start-dev.sh.

Expected local endpoints:
  MySQL:    127.0.0.1:3306
  Redis:    127.0.0.1:6379
  RabbitMQ: 127.0.0.1:5672
EOF
        exit 1
    fi
}

build_go_binary() {
    local dir="$1"
    local name="$2"
    local cmd_path="$3"
    local output="${BIN_DIR}/${name}"

    info "Downloading ${name} Go dependencies..."
    (cd "${dir}" && go mod download)
    info "Building ${name}..."
    (cd "${dir}" && go build -o "${output}" "${cmd_path}")
}

install_frontend_dependencies() {
    local dir="$1"
    local name="$2"

    if [ ! -d "${dir}/node_modules" ]; then
        info "Installing ${name} dependencies..."
        (cd "${dir}" && pnpm install --frozen-lockfile)
    else
        info "${name} dependencies already installed."
    fi
}

ensure_port_free() {
    local port="$1"
    local name="$2"

    if port_in_use "${port}"; then
        die "${name} cannot start because port ${port} is already in use. Run ./stop-dev.sh or free the port first."
    fi
}

is_running() {
    local pid_file="$1"
    [ -f "${pid_file}" ] || return 1
    local pid
    pid="$(cat "${pid_file}")"
    [ -n "${pid}" ] && kill -0 "${pid}" >/dev/null 2>&1
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
            die "Unknown target: ${raw}. Run ./start-dev.sh --help for supported targets."
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

has_any_backend_target() {
    local target
    for target in "${BUSINESS_SERVICES[@]}" smart-recruit-gateway; do
        if target_selected "${target}"; then
            return 0
        fi
    done
    return 1
}

has_any_frontend_target() {
    local target
    for target in "${FRONTEND_SERVICES[@]}"; do
        if target_selected "${target}"; then
            return 0
        fi
    done
    return 1
}

has_log_viewer_target() {
    target_selected dev-log-viewer
}

build_log_viewer_binary() {
    install_frontend_dependencies "${ROOT}/dev-log-viewer" "Dev log viewer"
    info "Building dev-log-viewer web assets..."
    (cd "${ROOT}" && pnpm --filter dev-log-viewer build)
    info "Downloading dev-log-viewer Go dependencies..."
    (cd "${ROOT}/dev-log-viewer" && go mod download)
    info "Building dev-log-viewer..."
    (cd "${ROOT}/dev-log-viewer" && go build -tags prod -o "${BIN_DIR}/dev-log-viewer" ./cmd/dev-log-viewer)
}

build_selected_go_binaries() {
    target_selected dev-log-viewer && build_log_viewer_binary
    has_any_backend_target && build_go_binary "${ROOT}/smart-recruit-commons" "smart-recruit-migrate" "./cmd/migrate"
    target_selected smart-recruit-gateway && build_go_binary "${ROOT}/smart-recruit-gateway" "smart-recruit-gateway" "./cmd/gateway"
    target_selected identity-service && build_go_binary "${ROOT}/smart-recruit-identity-service" "identity-service" "./cmd/identity-service"
    target_selected recruitment-service && build_go_binary "${ROOT}/smart-recruit-recruitment-service" "recruitment-service" "./cmd/recruitment-service"
    target_selected interview-service && build_go_binary "${ROOT}/smart-recruit-interview-service" "interview-service" "./cmd/interview-service"
    target_selected offer-service && build_go_binary "${ROOT}/smart-recruit-offer-service" "offer-service" "./cmd/offer-service"
    target_selected notification-service && build_go_binary "${ROOT}/smart-recruit-notification-service" "notification-service" "./cmd/notification-service"
    target_selected ai-agent-service && build_go_binary "${ROOT}/smart-recruit-ai-agent-service" "ai-agent-service" "./cmd/ai-agent-service"
    target_selected analytics-service && build_go_binary "${ROOT}/smart-recruit-analytics-service" "analytics-service" "./cmd/analytics-service"
    target_selected billing-service && build_go_binary "${ROOT}/smart-recruit-billing-service" "billing-service" "./cmd/billing-service"
    target_selected worker-service && build_go_binary "${ROOT}/smart-recruit-worker-service" "worker-service" "./cmd/worker-service"
    return 0
}

install_selected_frontend_dependencies() {
    target_selected hr-frontend && install_frontend_dependencies "${ROOT}/hr-frontend" "HR frontend"
    target_selected user-frontend && install_frontend_dependencies "${ROOT}/user-frontend" "User frontend"
    target_selected platform-frontend && install_frontend_dependencies "${ROOT}/platform-frontend" "Platform frontend"
    return 0
}

start_service() {
    local name="$1"
    local dir="$2"
    local port="$3"
    shift 3

    local pid_file="${PID_DIR}/${name}.pid"
    local log_file="${LOG_DIR}/${name}.log"

    if is_running "${pid_file}"; then
        info "${name} is already running (PID $(cat "${pid_file}"))."
        return 0
    fi

    ensure_port_free "${port}" "${name}"
    info "Starting ${name} on port ${port}; log: ${log_file}"

    (
        cd "${dir}"
        nohup "$@" >"${log_file}" 2>&1 &
        echo "$!" >"${pid_file}"
    )

    local pid
    pid="$(cat "${pid_file}")"
    sleep 1
    if ! kill -0 "${pid}" >/dev/null 2>&1; then
        rm -f "${pid_file}"
        warn "${name} failed to stay running. Last log lines:"
        tail -n 40 "${log_file}" >&2 || true
        exit 1
    fi
}

parse_targets "$@"
BILLING_CONFIG_PATH="${BILLING_CONFIG_PATH:-${ROOT}/smart-recruit-billing-service/config.yaml}"
if target_selected billing-service && [ ! -f "${BILLING_CONFIG_PATH}" ]; then
    die "Billing config not found at ${BILLING_CONFIG_PATH}. Copy smart-recruit-billing-service/config.example.yaml to smart-recruit-billing-service/config.yaml and fill in the Alipay sandbox values."
fi

ensure_system_dependencies
if has_any_backend_target; then
    ensure_runtime_services
fi

build_selected_go_binaries
if has_any_frontend_target; then
    install_selected_frontend_dependencies
fi

export CONFIG_PATH="${CONFIG_PATH:-${ROOT}/smart-recruit-commons/config/config.yaml}"
export MYSQL_DSN="${MYSQL_DSN:-root:Aa123456@tcp(127.0.0.1:3306)/recruitment?charset=utf8mb4&parseTime=True&loc=Local}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"
export RABBITMQ_URL="${RABBITMQ_URL:-amqp://guest:guest@127.0.0.1:5672/}"
export JWT_SECRET="${JWT_SECRET:-dev-jwt-secret-at-least-32-chars-long!!}"
export GRPC_INTERNAL_TOKEN="${GRPC_INTERNAL_TOKEN:-local-dev-internal-token-at-least-32!!}"
export GRPC_INTERNAL_AUTH="${GRPC_INTERNAL_AUTH:-required}"
export GRPC_INTERNAL_TLS="${GRPC_INTERNAL_TLS:-optional}"
export ALLOW_INSECURE_DEV_CONFIG="${ALLOW_INSECURE_DEV_CONFIG:-true}"
export STATIC_FALLBACK="${STATIC_FALLBACK:-false}"
export SERVICE_ENV="${SERVICE_ENV:-local}"

if has_any_backend_target; then
    info "Applying database migrations..."
    (
        cd "${ROOT}"
        "${BIN_DIR}/smart-recruit-migrate" --migrations-dir "${ROOT}/smart-recruit-commons/migrations"
    )
fi

target_selected identity-service && start_service "identity-service" "${ROOT}/smart-recruit-identity-service" 50061 "${BIN_DIR}/identity-service" --serve --addr :50061
target_selected recruitment-service && start_service "recruitment-service" "${ROOT}/smart-recruit-recruitment-service" 50062 "${BIN_DIR}/recruitment-service" --serve --addr :50062
target_selected interview-service && start_service "interview-service" "${ROOT}/smart-recruit-interview-service" 50063 "${BIN_DIR}/interview-service" --serve --addr :50063
target_selected offer-service && start_service "offer-service" "${ROOT}/smart-recruit-offer-service" 50064 "${BIN_DIR}/offer-service" --serve --addr :50064
target_selected notification-service && start_service "notification-service" "${ROOT}/smart-recruit-notification-service" 50065 "${BIN_DIR}/notification-service" --serve --addr :50065
target_selected ai-agent-service && start_service "ai-agent-service" "${ROOT}/smart-recruit-ai-agent-service" 50066 "${BIN_DIR}/ai-agent-service" --serve --addr :50066
target_selected analytics-service && start_service "analytics-service" "${ROOT}/smart-recruit-analytics-service" 50067 "${BIN_DIR}/analytics-service" --serve --addr :50067
target_selected worker-service && start_service "worker-service" "${ROOT}/smart-recruit-worker-service" 50068 "${BIN_DIR}/worker-service" --serve --health-addr :50068
target_selected billing-service && start_service "billing-service" "${ROOT}/smart-recruit-billing-service" 50069 "${BIN_DIR}/billing-service" --serve --addr :50069 --config "${BILLING_CONFIG_PATH}"

target_selected smart-recruit-gateway && start_service "smart-recruit-gateway" "${ROOT}/smart-recruit-gateway" 8080 \
    env HTTP_PORT=8080 \
    GRPC_ADDR=127.0.0.1:50062 \
    IDENTITY_ROUTE_MODE=identity IDENTITY_GRPC_ADDR=127.0.0.1:50061 \
    RECRUITMENT_ROUTE_MODE=recruitment RECRUITMENT_GRPC_ADDR=127.0.0.1:50062 \
    INTERVIEW_ROUTE_MODE=interview INTERVIEW_GRPC_ADDR=127.0.0.1:50063 \
    OFFER_ROUTE_MODE=offer OFFER_GRPC_ADDR=127.0.0.1:50064 \
    NOTIFICATION_ROUTE_MODE=notification NOTIFICATION_GRPC_ADDR=127.0.0.1:50065 \
    AI_AGENT_ROUTE_MODE=ai-agent AI_AGENT_GRPC_ADDR=127.0.0.1:50066 \
    ANALYTICS_ROUTE_MODE=analytics ANALYTICS_GRPC_ADDR=127.0.0.1:50067 \
    BILLING_GRPC_ADDR=127.0.0.1:50069 \
    "${BIN_DIR}/smart-recruit-gateway"

target_selected hr-frontend && start_service "hr-frontend" "${ROOT}/hr-frontend" 5173 pnpm run dev
target_selected user-frontend && start_service "user-frontend" "${ROOT}/user-frontend" 5174 pnpm run dev
target_selected platform-frontend && start_service "platform-frontend" "${ROOT}/platform-frontend" 5175 pnpm run dev
target_selected dev-log-viewer && start_service "dev-log-viewer" "${ROOT}/dev-log-viewer" 8090 \
    env DEV_LOG_VIEWER_ADDR=127.0.0.1:8090 DEV_LOG_VIEWER_ROOT="${ROOT}" \
    "${BIN_DIR}/dev-log-viewer" -addr 127.0.0.1:8090 -root "${ROOT}"

cat <<EOF

Done. Selected dev services are starting in the background.
  Gateway API: http://127.0.0.1:8080
  Identity:    127.0.0.1:50061
  Recruitment: 127.0.0.1:50062
  Interview:   127.0.0.1:50063
  Offer:       127.0.0.1:50064
  Notification:127.0.0.1:50065
  AI Agent:    127.0.0.1:50066
  Analytics:   127.0.0.1:50067
  Worker:      127.0.0.1:50068
  Billing:     127.0.0.1:50069 (Alipay sandbox)
  Staff:       http://127.0.0.1:5173
  Candidate:   http://127.0.0.1:5174
  Platform:    http://127.0.0.1:5175
  Log Viewer:  http://127.0.0.1:8090

Logs:
  ${LOG_DIR}

Stop:
  ./stop-dev.sh ${TARGETS[*]}
EOF
