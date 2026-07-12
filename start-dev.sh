#!/usr/bin/env bash
# Start the local Smart Recruit microservice dev stack without Docker.
# Usage: ./start-dev.sh

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
    install_with_package_manager go go golang-go
    install_with_package_manager node node nodejs
    ensure_pnpm
}

port_in_use() {
    local port="$1"
    lsof -tiTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
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

ensure_system_dependencies
ensure_runtime_services

build_go_binary "${ROOT}/smart-recruit-gateway" "smart-recruit-gateway" "./cmd/gateway"
build_go_binary "${ROOT}/smart-recruit-identity-service" "identity-service" "./cmd/identity-service"
build_go_binary "${ROOT}/smart-recruit-recruitment-service" "recruitment-service" "./cmd/recruitment-service"
build_go_binary "${ROOT}/smart-recruit-interview-service" "interview-service" "./cmd/interview-service"
build_go_binary "${ROOT}/smart-recruit-offer-service" "offer-service" "./cmd/offer-service"
build_go_binary "${ROOT}/smart-recruit-notification-service" "notification-service" "./cmd/notification-service"
build_go_binary "${ROOT}/smart-recruit-ai-agent-service" "ai-agent-service" "./cmd/ai-agent-service"
build_go_binary "${ROOT}/smart-recruit-analytics-service" "analytics-service" "./cmd/analytics-service"
build_go_binary "${ROOT}/smart-recruit-worker-service" "worker-service" "./cmd/worker-service"

install_frontend_dependencies "${ROOT}/hr-frontend" "HR frontend"
install_frontend_dependencies "${ROOT}/user-frontend" "User frontend"
install_frontend_dependencies "${ROOT}/interviewer-frontend" "Interviewer frontend"

export CONFIG_PATH="${CONFIG_PATH:-${ROOT}/smart-recruit-domain-go/config/config.yaml}"
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

start_service "identity-service" "${ROOT}/smart-recruit-identity-service" 50061 "${BIN_DIR}/identity-service" --serve --addr :50061
start_service "recruitment-service" "${ROOT}/smart-recruit-recruitment-service" 50062 "${BIN_DIR}/recruitment-service" --serve --addr :50062
start_service "interview-service" "${ROOT}/smart-recruit-interview-service" 50063 "${BIN_DIR}/interview-service" --serve --addr :50063
start_service "offer-service" "${ROOT}/smart-recruit-offer-service" 50064 "${BIN_DIR}/offer-service" --serve --addr :50064
start_service "notification-service" "${ROOT}/smart-recruit-notification-service" 50065 "${BIN_DIR}/notification-service" --serve --addr :50065
start_service "ai-agent-service" "${ROOT}/smart-recruit-ai-agent-service" 50066 "${BIN_DIR}/ai-agent-service" --serve --addr :50066
start_service "analytics-service" "${ROOT}/smart-recruit-analytics-service" 50067 "${BIN_DIR}/analytics-service" --serve --addr :50067
start_service "worker-service" "${ROOT}/smart-recruit-worker-service" 50068 "${BIN_DIR}/worker-service" --serve --health-addr :50068

start_service "smart-recruit-gateway" "${ROOT}/smart-recruit-gateway" 8080 \
    env HTTP_PORT=8080 \
    GRPC_ADDR=127.0.0.1:50062 \
    IDENTITY_ROUTE_MODE=identity IDENTITY_GRPC_ADDR=127.0.0.1:50061 \
    RECRUITMENT_ROUTE_MODE=recruitment RECRUITMENT_GRPC_ADDR=127.0.0.1:50062 \
    INTERVIEW_ROUTE_MODE=interview INTERVIEW_GRPC_ADDR=127.0.0.1:50063 \
    OFFER_ROUTE_MODE=offer OFFER_GRPC_ADDR=127.0.0.1:50064 \
    NOTIFICATION_ROUTE_MODE=notification NOTIFICATION_GRPC_ADDR=127.0.0.1:50065 \
    AI_AGENT_ROUTE_MODE=ai-agent AI_AGENT_GRPC_ADDR=127.0.0.1:50066 \
    ANALYTICS_ROUTE_MODE=analytics ANALYTICS_GRPC_ADDR=127.0.0.1:50067 \
    "${BIN_DIR}/smart-recruit-gateway"

start_service "hr-frontend" "${ROOT}/hr-frontend" 5173 pnpm run dev
start_service "user-frontend" "${ROOT}/user-frontend" 5174 pnpm run dev
start_service "interviewer-frontend" "${ROOT}/interviewer-frontend" 5175 pnpm run dev

cat <<EOF

Done. Dev services are starting in the background.
  Gateway API: http://localhost:8080
  Identity:    localhost:50061
  Recruitment: localhost:50062
  Interview:   localhost:50063
  Offer:       localhost:50064
  Notification:localhost:50065
  AI Agent:    localhost:50066
  Analytics:   localhost:50067
  Worker:      localhost:50068
  HR:          http://localhost:5173
  User:        http://localhost:5174
  Interviewer: http://localhost:5175

Logs:
  ${LOG_DIR}

Stop:
  ./stop-dev.sh
EOF
