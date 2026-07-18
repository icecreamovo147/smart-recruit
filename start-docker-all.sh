#!/usr/bin/env bash
# Start the full Smart Recruit Docker stack:
# infra + backend microservices + observability + frontends.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_NAME="${COMPOSE_PROJECT_NAME:-smart-recruit}"
MICRO_COMPOSE="${ROOT}/smart-recruit-deploy/docker-compose.microservices.yml"
FRONTEND_COMPOSE="${ROOT}/docker/docker-compose.yml"
PROFILES="${COMPOSE_PROFILES:-infra,services,observability}"
COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-1}"
SKIP_BUILD="${SKIP_BUILD:-false}"

info() {
    printf '[docker-all] %s\n' "$*"
}

warn() {
    printf '[docker-all] WARN: %s\n' "$*" >&2
}

die() {
    printf '[docker-all] ERROR: %s\n' "$*" >&2
    exit 1
}

compose_cmd() {
    if command -v docker-compose >/dev/null 2>&1; then
        docker-compose "$@"
        return
    fi

    if docker compose version >/dev/null 2>&1; then
        docker compose "$@"
        return
    fi

    die "Docker Compose is not available. Install docker-compose or Docker Compose v2."
}

check_docker() {
    docker info >/dev/null 2>&1 || die "Docker daemon is not available."
}

check_resources() {
    local info_json
    info_json="$(docker info --format '{{.NCPU}} {{.MemTotal}}' 2>/dev/null || true)"
    [ -n "${info_json}" ] || return 0

    local cpus mem_bytes mem_gib
    cpus="$(printf '%s' "${info_json}" | awk '{print $1}')"
    mem_bytes="$(printf '%s' "${info_json}" | awk '{print $2}')"
    mem_gib="$(awk -v bytes="${mem_bytes}" 'BEGIN { printf "%.1f", bytes / 1024 / 1024 / 1024 }')"

    info "Docker resources: ${cpus} CPU, ${mem_gib} GiB memory."

    if [ "${cpus}" -lt 4 ] || [ "${mem_bytes}" -lt 6442450944 ]; then
        warn "Full stack is heavy. Recommended: colima start --cpu 4 --memory 8"
    fi
}

nacos_put_inline() {
    local data_id="$1"
    local content="$2"

    curl -fsS -X POST "http://127.0.0.1:8848/nacos/v1/cs/configs" \
        --data-urlencode "dataId=${data_id}" \
        --data-urlencode "group=DEFAULT_GROUP" \
        --data-urlencode "type=yaml" \
        --data-urlencode "content=${content}" >/dev/null
}

nacos_put_file() {
    local data_id="$1"
    local file="$2"

    curl -fsS -X POST "http://127.0.0.1:8848/nacos/v1/cs/configs" \
        --data-urlencode "dataId=${data_id}" \
        --data-urlencode "group=DEFAULT_GROUP" \
        --data-urlencode "type=yaml" \
        --data-urlencode "content@${file}" >/dev/null
}

wait_for_nacos() {
    info "Waiting for Nacos to become healthy..."
    for _ in $(seq 1 36); do
        local health
        health="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' smart-recruit-nacos 2>/dev/null || true)"
        if [ "${health}" = "healthy" ]; then
            return 0
        fi
        sleep 5
    done

    docker logs --tail 120 smart-recruit-nacos >&2 || true
    die "Nacos did not become healthy in time."
}

seed_nacos_configs() {
    command -v curl >/dev/null 2>&1 || die "curl is required to seed Nacos configs."

    info "Seeding Nacos configs..."
    nacos_put_inline "identity-service.yaml" $'service:\n  name: identity-service\n'
    nacos_put_inline "interview-service.yaml" $'service:\n  name: interview-service\n'
    nacos_put_inline "offer-service.yaml" $'service:\n  name: offer-service\n'
    nacos_put_inline "notification-service.yaml" $'service:\n  name: notification-service\n'
    nacos_put_inline "ai-agent-service.yaml" $'service:\n  name: ai-agent-service\n'
    nacos_put_inline "analytics-service.yaml" $'service:\n  name: analytics-service\n'
    nacos_put_inline "worker-service.yaml" $'service:\n  name: worker-service\nworker:\n  workloads: all\n'
    nacos_put_file "gateway.yaml" "${ROOT}/smart-recruit-deploy/nacos/seed-config/gateway.yaml"
    nacos_put_file "services.yaml" "${ROOT}/smart-recruit-deploy/nacos/seed-config/services.yaml"
}

start_microservices() {
    export COMPOSE_PARALLEL_LIMIT
    local up_args=(up -d)
    if [ "${SKIP_BUILD}" != "true" ]; then
        up_args+=(--build)
    fi

    info "Starting infra and observability..."
    COMPOSE_PROFILES="infra,observability" compose_cmd \
        -p "${PROJECT_NAME}" \
        -f "${MICRO_COMPOSE}" \
        "${up_args[@]}"

    wait_for_nacos
    seed_nacos_configs

    info "Starting backend services..."
    export COMPOSE_PROFILES="${PROFILES}"
    compose_cmd \
        -p "${PROJECT_NAME}" \
        -f "${MICRO_COMPOSE}" \
        "${up_args[@]}"
}

start_frontends() {
    info "Starting frontend apps on the same Docker Compose network..."
    export COMPOSE_PARALLEL_LIMIT
    local up_args=(up -d)
    if [ "${SKIP_BUILD}" != "true" ]; then
        up_args+=(--build)
    fi
    compose_cmd \
        -p "${PROJECT_NAME}" \
        -f "${FRONTEND_COMPOSE}" \
        "${up_args[@]}"
}

print_status() {
    info "Current containers for project ${PROJECT_NAME}:"
    docker ps \
        --filter "label=com.docker.compose.project=${PROJECT_NAME}" \
        --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
}

check_docker
check_resources
start_microservices
start_frontends
print_status

cat <<EOF

Done.

Entry points:
  Gateway:      http://127.0.0.1:8080
  HR:           http://127.0.0.1:5173
  User:         http://127.0.0.1:5174
  Interviewer:  http://127.0.0.1:5175
  Nacos:        http://127.0.0.1:8848
  RabbitMQ:     http://127.0.0.1:15672
  Prometheus:   http://127.0.0.1:9090
  Grafana:      http://127.0.0.1:3000
  Jaeger:       http://127.0.0.1:16686
  Loki:         http://127.0.0.1:3100

Logs:
  docker-compose -p ${PROJECT_NAME} -f smart-recruit-deploy/docker-compose.microservices.yml logs -f

Stop all:
  docker-compose -p ${PROJECT_NAME} -f smart-recruit-deploy/docker-compose.microservices.yml down
  docker-compose -p ${PROJECT_NAME} -f docker/docker-compose.yml down

Start without rebuilding:
  SKIP_BUILD=true ./start-docker-all.sh
EOF
