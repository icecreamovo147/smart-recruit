#!/usr/bin/env bash
# Start all local dev services without depending on iTerm2, Terminal, or AppleScript.
# Usage: ./start-dev.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="${ROOT}/.dev"
PID_DIR="${STATE_DIR}/pids"
LOG_DIR="${STATE_DIR}/logs"
BIN_DIR="${STATE_DIR}/bin"
PNPM_VERSION="10.19.0"

mkdir -p "${PID_DIR}" "${LOG_DIR}" "${BIN_DIR}"

info() {
    printf '[dev] %s\n' "$*"
}

warn() {
    printf '[dev] WARN: %s\n' "$*" >&2
}

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

print_missing_runtime_services() {
    local missing="$1"

    cat >&2 <<EOF
[dev] ERROR: Required local runtime service(s) are not running or not installed:
${missing}

Please install and start the missing service(s) yourself, then rerun ./start-dev.sh.

Expected local endpoints:
  MySQL:    127.0.0.1:3306
  Redis:    127.0.0.1:6379
  RabbitMQ: 127.0.0.1:5672

You can also use Docker Compose for the full containerized stack:
  cd docker
  docker-compose up -d --build
EOF
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
    elif command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update
        sudo apt-get install -y npm
        npm install -g "pnpm@${PNPM_VERSION}"
    else
        die "Cannot install pnpm because neither corepack nor npm is available."
    fi

    command -v pnpm >/dev/null 2>&1 || die "pnpm installation did not make the command available."
}

ensure_system_dependencies() {
    install_with_package_manager go go golang-go
    install_with_package_manager node node nodejs
    ensure_pnpm
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
        print_missing_runtime_services "${missing}"
        exit 1
    fi

    info "Required local runtime services are available."
}

install_go_dependencies() {
    local dir="$1"
    local name="$2"

    info "Downloading ${name} Go dependencies..."
    (cd "${dir}" && go mod download)
}

build_go_service() {
    local dir="$1"
    local name="$2"
    local output="${BIN_DIR}/${name}"

    info "Building ${name}..."
    (cd "${dir}" && go build -o "${output}" .)
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

logic_jwt_secret() {
    local config_file="${ROOT}/logic-grpc-service/config/config.yaml"

    [ -f "${config_file}" ] || die "Missing ${config_file}; create it from config.example.yaml first."

    awk '
        /^[[:space:]]*jwt:[[:space:]]*$/ { in_jwt = 1; next }
        /^[^[:space:]][^:]*:[[:space:]]*$/ { in_jwt = 0 }
        in_jwt && /^[[:space:]]*secret:[[:space:]]*/ {
            line = $0
            sub(/^[[:space:]]*secret:[[:space:]]*/, "", line)
            gsub(/^[\"\047]|[\"\047]$/, "", line)
            print line
            exit
        }
    ' "${config_file}"
}

port_in_use() {
    local port="$1"
    lsof -tiTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
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

install_with_package_manager lsof lsof lsof
ensure_runtime_services
ensure_system_dependencies
install_go_dependencies "${ROOT}/logic-grpc-service" "Logic gRPC"
install_go_dependencies "${ROOT}/web-gin-service" "Web Gin"
build_go_service "${ROOT}/logic-grpc-service" "logic-grpc-service"
build_go_service "${ROOT}/web-gin-service" "web-gin-service"
install_frontend_dependencies "${ROOT}/hr-frontend" "HR frontend"
install_frontend_dependencies "${ROOT}/user-frontend" "User frontend"

JWT_SECRET="${JWT_SECRET:-$(logic_jwt_secret)}"
[ -n "${JWT_SECRET}" ] || die "JWT secret is empty. Set jwt.secret in logic-grpc-service/config/config.yaml or export JWT_SECRET."
export JWT_SECRET

GRPC_INTERNAL_TOKEN="${GRPC_INTERNAL_TOKEN:-local-dev-internal-token}"
export GRPC_INTERNAL_TOKEN

start_service "logic-grpc-service" "${ROOT}/logic-grpc-service" 50051 "${BIN_DIR}/logic-grpc-service"
sleep 3
start_service "web-gin-service" "${ROOT}/web-gin-service" 8080 env JWT_SECRET="${JWT_SECRET}" GRPC_INTERNAL_TOKEN="${GRPC_INTERNAL_TOKEN}" "${BIN_DIR}/web-gin-service"
start_service "hr-frontend" "${ROOT}/hr-frontend" 5173 pnpm run dev
start_service "user-frontend" "${ROOT}/user-frontend" 5174 pnpm run dev

cat <<EOF

Done. Dev services are starting in the background.
  gRPC: http://localhost:50051
  API:  http://localhost:8080
  HR:   http://localhost:5173
  User: http://localhost:5174

Logs:
  ${LOG_DIR}

Stop:
  ./stop-dev.sh
EOF
