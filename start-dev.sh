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
BUILD_CACHE_DIR="${STATE_DIR}/build-cache"
BUILD_LOG_DIR="${STATE_DIR}/build-logs"
TOOL_DIR="${STATE_DIR}/tools"
PNPM_VERSION="10.19.0"

mkdir -p "${PID_DIR}" "${LOG_DIR}" "${BIN_DIR}" "${BUILD_CACHE_DIR}" "${BUILD_LOG_DIR}" "${TOOL_DIR}"

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

Build controls:
  DEV_BUILD_JOBS=<n>       Maximum number of binaries checked/built concurrently.
  DEV_FORCE_REBUILD=1      Ignore saved fingerprints and rebuild selected binaries.
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

cpu_count() {
    local count=""
    if command -v getconf >/dev/null 2>&1; then
        count="$(getconf _NPROCESSORS_ONLN 2>/dev/null || true)"
    fi
    if ! [[ "${count}" =~ ^[1-9][0-9]*$ ]] && command -v sysctl >/dev/null 2>&1; then
        count="$(sysctl -n hw.logicalcpu 2>/dev/null || true)"
    fi
    if ! [[ "${count}" =~ ^[1-9][0-9]*$ ]]; then
        count=1
    fi
    printf '%s\n' "${count}"
}

configure_build_parallelism() {
    local selected_count="$1"
    local available
    available="$(cpu_count)"
    if [ -z "${DEV_BUILD_JOBS:-}" ]; then
        DEV_BUILD_JOBS="${available}"
        if [ "${DEV_BUILD_JOBS}" -gt 4 ]; then
            DEV_BUILD_JOBS=4
        fi
    fi
    if ! [[ "${DEV_BUILD_JOBS}" =~ ^[1-9][0-9]*$ ]]; then
        die "DEV_BUILD_JOBS must be a positive integer."
    fi
    if [ "${DEV_BUILD_JOBS}" -gt "${selected_count}" ]; then
        DEV_BUILD_JOBS="${selected_count}"
    fi
    GO_BUILD_PACKAGE_JOBS=$((available / DEV_BUILD_JOBS))
    if [ "${GO_BUILD_PACKAGE_JOBS}" -lt 1 ]; then
        GO_BUILD_PACKAGE_JOBS=1
    fi
    info "Build parallelism: ${DEV_BUILD_JOBS} binaries at a time, ${GO_BUILD_PACKAGE_JOBS} Go package jobs per binary."
}

selected_go_build_count() {
    local count=0
    local target
    if has_log_viewer_target; then
        count=$((count + 1))
    fi
    if has_any_backend_target; then
        count=$((count + 1))
    fi
    for target in "${BUSINESS_SERVICES[@]}" smart-recruit-gateway; do
        if target_selected "${target}"; then
            count=$((count + 1))
        fi
    done
    printf '%s\n' "${count}"
}

ensure_build_fingerprint_tool() {
    local source="${ROOT}/scripts/dev-build-fingerprint.go"
    local output="${TOOL_DIR}/dev-build-fingerprint"
    local temporary="${output}.tmp.$$"
    local checksum_file="${TOOL_DIR}/dev-build-fingerprint.checksum"
    local temporary_checksum="${checksum_file}.tmp.$$"
    local source_checksum
    local previous_checksum=""

    source_checksum="$(cksum <"${source}" | awk '{print $1 "-" $2}')"
    if [ -f "${checksum_file}" ]; then
        previous_checksum="$(cat "${checksum_file}")"
    fi
    if [ -x "${output}" ] && [ "${source_checksum}" = "${previous_checksum}" ]; then
        return 0
    fi
    info "Preparing incremental build fingerprint tool..."
    if ! go build -buildvcs=false -ldflags "-X=main.toolRevision=${source_checksum}" -o "${temporary}" "${source}"; then
        rm -f "${temporary}" "${temporary_checksum}"
        die "Failed to build the incremental build fingerprint tool."
    fi
    mv "${temporary}" "${output}"
    printf '%s\n' "${source_checksum}" >"${temporary_checksum}"
    mv "${temporary_checksum}" "${checksum_file}"
}

compute_go_build_fingerprint() {
    local dir="$1"
    local cmd_path="$2"
    local tags="$3"
    shift 3

    local args=(
        --root "${ROOT}"
        --dir "${dir}"
        --package "${cmd_path}"
        --tags "${tags}"
    )
    local extra
    for extra in "$@"; do
        args+=(--extra "${extra}")
    done
    "${TOOL_DIR}/dev-build-fingerprint" "${args[@]}"
}

build_go_binary() {
    local dir="$1"
    local name="$2"
    local cmd_path="$3"
    local tags="${4:-}"
    shift 4

    local output="${BIN_DIR}/${name}"
    local fingerprint_file="${BUILD_CACHE_DIR}/${name}.fingerprint"
    local fingerprint=""
    local previous_fingerprint=""
    local temporary_output="${output}.tmp.$$"
    local temporary_fingerprint="${fingerprint_file}.tmp.$$"

    if ! fingerprint="$(compute_go_build_fingerprint "${dir}" "${cmd_path}" "${tags}" "$@")"; then
        warn "Could not calculate ${name} build fingerprint; rebuilding without cache reuse."
    fi
    if [ -f "${fingerprint_file}" ]; then
        previous_fingerprint="$(cat "${fingerprint_file}")"
    fi
    if [ "${DEV_FORCE_REBUILD:-0}" != "1" ] && [ -n "${fingerprint}" ] && [ -x "${output}" ] && [ "${fingerprint}" = "${previous_fingerprint}" ]; then
        info "${name} is up to date; reusing ${output}."
        return 0
    fi

    info "Building ${name}..."
    local build_args=(build -buildvcs=false -p "${GO_BUILD_PACKAGE_JOBS}")
    if [ -n "${tags}" ]; then
        build_args+=(-tags "${tags}")
    fi
    build_args+=(-o "${temporary_output}" "${cmd_path}")
    if ! (cd "${dir}" && go "${build_args[@]}"); then
        rm -f "${temporary_output}" "${temporary_fingerprint}"
        return 1
    fi
    mv "${temporary_output}" "${output}"
    if [ -n "${fingerprint}" ]; then
        printf '%s\n' "${fingerprint}" >"${temporary_fingerprint}"
        mv "${temporary_fingerprint}" "${fingerprint_file}"
    else
        rm -f "${fingerprint_file}"
    fi
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
    local web_fingerprint_file="${BUILD_CACHE_DIR}/dev-log-viewer-web.fingerprint"
    local web_fingerprint=""
    local previous_web_fingerprint=""
    local temporary_fingerprint="${web_fingerprint_file}.tmp.$$"
    local web_inputs=(
        "${ROOT}/pnpm-lock.yaml"
        "${ROOT}/pnpm-workspace.yaml"
        "${ROOT}/dev-log-viewer/package.json"
        "${ROOT}/dev-log-viewer/tsconfig.json"
        "${ROOT}/dev-log-viewer/vite.config.ts"
        "${ROOT}/dev-log-viewer/web"
    )

    install_frontend_dependencies "${ROOT}/dev-log-viewer" "Dev log viewer"
    local args=(--root "${ROOT}" --inputs-only)
    local input
    for input in "${web_inputs[@]}"; do
        args+=(--extra "${input}")
    done
    web_fingerprint="$("${TOOL_DIR}/dev-build-fingerprint" "${args[@]}")"
    if [ -f "${web_fingerprint_file}" ]; then
        previous_web_fingerprint="$(cat "${web_fingerprint_file}")"
    fi

    if [ "${DEV_FORCE_REBUILD:-0}" = "1" ] || [ ! -d "${ROOT}/dev-log-viewer/web/dist" ] || [ "${web_fingerprint}" != "${previous_web_fingerprint}" ]; then
        info "Building dev-log-viewer web assets..."
        (cd "${ROOT}" && pnpm --filter dev-log-viewer build)
        printf '%s\n' "${web_fingerprint}" >"${temporary_fingerprint}"
        mv "${temporary_fingerprint}" "${web_fingerprint_file}"
    else
        info "dev-log-viewer web assets are up to date."
    fi

    build_go_binary "${ROOT}/dev-log-viewer" "dev-log-viewer" "./cmd/dev-log-viewer" "prod" "${web_inputs[@]}"
}

BUILD_PIDS=()
BUILD_NAMES=()

wait_build_batch() {
    local failed=0
    local index
    local log_file
    for ((index = 0; index < ${#BUILD_PIDS[@]}; index++)); do
        log_file="${BUILD_LOG_DIR}/${BUILD_NAMES[index]}.log"
        if wait "${BUILD_PIDS[index]}"; then
            cat "${log_file}"
        else
            warn "Build failed for ${BUILD_NAMES[index]}. Build log: ${log_file}"
            cat "${log_file}" >&2
            failed=1
        fi
    done
    BUILD_PIDS=()
    BUILD_NAMES=()
    return "${failed}"
}

queue_build_job() {
    local name="$1"
    shift
    local log_file="${BUILD_LOG_DIR}/${name}.log"

    info "Queueing ${name} build check..."
    ("$@") >"${log_file}" 2>&1 &
    BUILD_PIDS+=("$!")
    BUILD_NAMES+=("${name}")
    if [ "${#BUILD_PIDS[@]}" -ge "${DEV_BUILD_JOBS}" ]; then
        wait_build_batch
    fi
}

build_selected_go_binaries() {
    if ! has_any_backend_target && ! has_log_viewer_target; then
        return 0
    fi

    configure_build_parallelism "$(selected_go_build_count)"
    ensure_build_fingerprint_tool
    if target_selected dev-log-viewer; then
        queue_build_job "dev-log-viewer" build_log_viewer_binary
    fi
    if has_any_backend_target; then
        queue_build_job "smart-recruit-migrate" build_go_binary "${ROOT}/smart-recruit-commons" "smart-recruit-migrate" "./cmd/migrate" ""
    fi
    if target_selected smart-recruit-gateway; then
        queue_build_job "smart-recruit-gateway" build_go_binary "${ROOT}/smart-recruit-gateway" "smart-recruit-gateway" "./cmd/gateway" ""
    fi
    if target_selected identity-service; then
        queue_build_job "identity-service" build_go_binary "${ROOT}/smart-recruit-identity-service" "identity-service" "./cmd/identity-service" ""
    fi
    if target_selected recruitment-service; then
        queue_build_job "recruitment-service" build_go_binary "${ROOT}/smart-recruit-recruitment-service" "recruitment-service" "./cmd/recruitment-service" ""
    fi
    if target_selected interview-service; then
        queue_build_job "interview-service" build_go_binary "${ROOT}/smart-recruit-interview-service" "interview-service" "./cmd/interview-service" ""
    fi
    if target_selected offer-service; then
        queue_build_job "offer-service" build_go_binary "${ROOT}/smart-recruit-offer-service" "offer-service" "./cmd/offer-service" ""
    fi
    if target_selected notification-service; then
        queue_build_job "notification-service" build_go_binary "${ROOT}/smart-recruit-notification-service" "notification-service" "./cmd/notification-service" ""
    fi
    if target_selected ai-agent-service; then
        queue_build_job "ai-agent-service" build_go_binary "${ROOT}/smart-recruit-ai-agent-service" "ai-agent-service" "./cmd/ai-agent-service" ""
    fi
    if target_selected analytics-service; then
        queue_build_job "analytics-service" build_go_binary "${ROOT}/smart-recruit-analytics-service" "analytics-service" "./cmd/analytics-service" ""
    fi
    if target_selected billing-service; then
        queue_build_job "billing-service" build_go_binary "${ROOT}/smart-recruit-billing-service" "billing-service" "./cmd/billing-service" ""
    fi
    if target_selected worker-service; then
        queue_build_job "worker-service" build_go_binary "${ROOT}/smart-recruit-worker-service" "worker-service" "./cmd/worker-service" ""
    fi
    wait_build_batch
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
BILLING_CONFIG_PATH="${BILLING_CONFIG_PATH:-${ROOT}/smart-recruit-billing-service/internal/config/config.yaml}"
if target_selected billing-service && [ ! -f "${BILLING_CONFIG_PATH}" ]; then
    die "Billing config not found at ${BILLING_CONFIG_PATH}. Copy smart-recruit-billing-service/internal/config/config.example.yaml to smart-recruit-billing-service/internal/config/config.yaml and fill in the Alipay sandbox values."
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
export TZ="${TZ:-Asia/Shanghai}"
export APP_LOCALE="${APP_LOCALE:-zh-CN}"
export MYSQL_DSN="${MYSQL_DSN:-root:Aa123456@tcp(127.0.0.1:3306)/recruitment?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai&time_zone=%27%2B08%3A00%27}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"
export RABBITMQ_URL="${RABBITMQ_URL:-amqp://guest:guest@127.0.0.1:5672/}"
export JWT_SECRET="${JWT_SECRET:-dev-jwt-secret-at-least-32-chars-long!!}"
export GRPC_INTERNAL_TOKEN="${GRPC_INTERNAL_TOKEN:-local-dev-internal-token-at-least-32!!}"
export GRPC_INTERNAL_AUTH="${GRPC_INTERNAL_AUTH:-required}"
export GRPC_INTERNAL_TLS="${GRPC_INTERNAL_TLS:-optional}"
export ALLOW_INSECURE_DEV_CONFIG="${ALLOW_INSECURE_DEV_CONFIG:-true}"
export STATIC_FALLBACK="${STATIC_FALLBACK:-false}"
export SERVICE_ENV="${SERVICE_ENV:-local}"
export BILLING_HR_RETURN_URL="${BILLING_HR_RETURN_URL:-http://localhost:5173/hr/billing}"
export BILLING_CANDIDATE_RETURN_URL="${BILLING_CANDIDATE_RETURN_URL:-http://localhost:5174/billing}"

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
  Gateway API: http://localhost:8080
  Identity:    127.0.0.1:50061
  Recruitment: 127.0.0.1:50062
  Interview:   127.0.0.1:50063
  Offer:       127.0.0.1:50064
  Notification:127.0.0.1:50065
  AI Agent:    127.0.0.1:50066
  Analytics:   127.0.0.1:50067
  Worker:      127.0.0.1:50068
  Billing:     127.0.0.1:50069 (Alipay sandbox)
  Staff:       http://localhost:5173
  Candidate:   http://localhost:5174
  Platform:    http://localhost:5175
  Log Viewer:  http://localhost:8090

Logs:
  ${LOG_DIR}

Stop:
  ./stop-dev.sh ${TARGETS[*]}
EOF
