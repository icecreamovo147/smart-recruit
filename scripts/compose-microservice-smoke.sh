#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/microservice-runtime-implementation"
COMPOSE_FILE="${ROOT}/smart-recruit-deploy/docker-compose.microservices.yml"
PROJECT_NAME="${PROJECT_NAME:-smart-recruit-mri-smoke}"
LOG_DIR="${LOG_DIR:-${FEATURE_DIR}/reports/compose-smoke/$(date -u +%Y%m%dT%H%M%SZ)}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
DRY_RUN=0
CHECK_ONLY=0
KEEP_RUNNING=0

SERVICES=(
  nacos mysql redis rabbitmq prometheus grafana jaeger loki
  smart-recruit-gateway
  smart-recruit-identity-service
  smart-recruit-recruitment-service
  smart-recruit-interview-service
  smart-recruit-offer-service
  smart-recruit-notification-service
  smart-recruit-ai-agent-service
  smart-recruit-analytics-service
  smart-recruit-worker-service
)

NACOS_SERVICES=(identity recruitment interview offer notification ai-agent analytics worker)
GATEWAY_ROUTE_CHECKS=(
  "identity:POST:http://127.0.0.1:8080/api/v1/auth/login"
  "recruitment:GET:http://127.0.0.1:8080/api/v1/jobs"
  "analytics:GET:http://127.0.0.1:8080/api/v1/hr/analytics/dashboard"
)

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      ;;
    --check)
      CHECK_ONLY=1
      ;;
    --keep-running)
      KEEP_RUNNING=1
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

compose_cmd() {
  if command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
    return
  fi
  docker compose "$@"
}

compose_base() {
  COMPOSE_PROFILES=infra,observability,services compose_cmd -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" "$@"
}

check_static() {
  bash "${ROOT}/scripts/build-microservice-images.sh" --check
  local rendered
  rendered="$(mktemp)"
  COMPOSE_PROFILES=infra,observability,services compose_cmd -f "${COMPOSE_FILE}" config >"${rendered}"
  for service in "${SERVICES[@]}"; do
    if ! grep -q "^  ${service}:" "${rendered}"; then
      echo "compose smoke missing service: ${service}" >&2
      rm -f "${rendered}"
      exit 1
    fi
  done
  for service in "${NACOS_SERVICES[@]}"; do
    if ! grep -q "SERVICE_NAME: ${service}" "${rendered}"; then
      echo "compose smoke missing SERVICE_NAME for ${service}" >&2
      rm -f "${rendered}"
      exit 1
    fi
  done
  for route in IDENTITY RECRUITMENT ANALYTICS; do
    if ! grep -q "${route}_ROUTE_MODE:" "${rendered}"; then
      echo "compose smoke missing gateway ${route}_ROUTE_MODE" >&2
      rm -f "${rendered}"
      exit 1
    fi
  done
  rm -f "${rendered}"
}

write_diagnostics() {
  mkdir -p "${LOG_DIR}"
  {
    echo "project=${PROJECT_NAME}"
    echo "compose_file=${COMPOSE_FILE}"
    echo "timestamp_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  } >"${LOG_DIR}/summary.txt"
  compose_base ps >"${LOG_DIR}/compose-ps.txt" 2>&1 || true
  compose_base logs --no-color >"${LOG_DIR}/compose.log" 2>&1 || true
  docker info >"${LOG_DIR}/docker-info.txt" 2>&1 || true
}

wait_http() {
  local name="$1"
  local url="$2"
  local deadline=$((SECONDS + TIMEOUT_SECONDS))
  until curl -fsS "${url}" >"${LOG_DIR}/${name}.body" 2>"${LOG_DIR}/${name}.err"; do
    if [ "${SECONDS}" -ge "${deadline}" ]; then
      echo "${name} did not become ready: ${url}" >&2
      return 1
    fi
    sleep 3
  done
}

check_nacos_registration() {
  local service="$1"
  local url="http://127.0.0.1:8848/nacos/v1/ns/instance/list?serviceName=${service}&groupName=DEFAULT_GROUP"
  curl -fsS "${url}" >"${LOG_DIR}/nacos-${service}.json"
  if ! grep -q '"hosts"' "${LOG_DIR}/nacos-${service}.json"; then
    echo "nacos response for ${service} did not include hosts" >&2
    return 1
  fi
}

check_gateway_route() {
  local spec="$1"
  IFS=: read -r service method url <<<"${spec}"
  local status
  status="$(curl -sS -o "${LOG_DIR}/gateway-${service}.body" -w '%{http_code}' -X "${method}" "${url}" 2>"${LOG_DIR}/gateway-${service}.err" || true)"
  echo "${status}" >"${LOG_DIR}/gateway-${service}.status"
  case "${status}" in
    2*|3*|4*)
      return 0
      ;;
    *)
      echo "gateway route ${service} returned ${status}" >&2
      return 1
      ;;
  esac
}

check_static

if [ "${CHECK_ONLY}" -eq 1 ]; then
  echo "compose microservice smoke check: PASS (${#SERVICES[@]} services)"
  exit 0
fi

if [ "${DRY_RUN}" -eq 1 ]; then
  echo "COMPOSE_PROFILES=infra,observability,services docker-compose -p ${PROJECT_NAME} -f ${COMPOSE_FILE} up -d --build"
  echo "curl -fsS http://127.0.0.1:8848/nacos/v1/console/health/readiness"
  echo "curl -fsS http://127.0.0.1:8080/health"
  printf 'nacos registrations: %s\n' "${NACOS_SERVICES[*]}"
  printf 'gateway route checks: %s\n' "${GATEWAY_ROUTE_CHECKS[*]}"
  echo "logs: ${LOG_DIR}"
  exit 0
fi

mkdir -p "${LOG_DIR}"
if ! docker info >"${LOG_DIR}/docker-info.txt" 2>&1; then
  echo "docker daemon unavailable; diagnostics: ${LOG_DIR}" >&2
  exit 2
fi

cleanup() {
  local code=$?
  write_diagnostics
  if [ "${KEEP_RUNNING}" -ne 1 ]; then
    compose_base down --remove-orphans >>"${LOG_DIR}/cleanup.log" 2>&1 || true
  fi
  exit "${code}"
}
trap cleanup EXIT

compose_base up -d --build
wait_http "nacos-readiness" "http://127.0.0.1:8848/nacos/v1/console/health/readiness"
wait_http "gateway-health" "http://127.0.0.1:8080/health"
wait_http "prometheus-ready" "http://127.0.0.1:9090/-/ready"

for service in "${NACOS_SERVICES[@]}"; do
  check_nacos_registration "${service}"
done
for route in "${GATEWAY_ROUTE_CHECKS[@]}"; do
  check_gateway_route "${route}"
done

echo "compose microservice smoke: PASS logs=${LOG_DIR}"
