#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="${ROOT}/smart-recruit-deploy/docker-compose.microservices.yml"
DOCKERFILE="${ROOT}/smart-recruit-deploy/docker/go-service.Dockerfile"
IMAGE_PREFIX="${IMAGE_PREFIX:-smart-recruit}"
IMAGE_TAG="${IMAGE_TAG:-local}"
DRY_RUN=0
CHECK_ONLY=0

SERVICES=(
  "smart-recruit-gateway:./cmd/gateway:smart-recruit-gateway"
  "smart-recruit-identity-service:./cmd/identity-service:identity-service"
  "smart-recruit-recruitment-service:./cmd/recruitment-service:recruitment-service"
  "smart-recruit-interview-service:./cmd/interview-service:interview-service"
  "smart-recruit-offer-service:./cmd/offer-service:offer-service"
  "smart-recruit-notification-service:./cmd/notification-service:notification-service"
  "smart-recruit-ai-agent-service:./cmd/ai-agent-service:ai-agent-service"
  "smart-recruit-analytics-service:./cmd/analytics-service:analytics-service"
  "smart-recruit-billing-service:./cmd/billing-service:billing-service"
  "smart-recruit-worker-service:./cmd/worker-service:worker-service"
)

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      ;;
    --check)
      CHECK_ONLY=1
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

require_file() {
  if [ ! -f "$1" ]; then
    echo "missing required file: $1" >&2
    exit 1
  fi
}

check_targets() {
  require_file "${COMPOSE_FILE}"
  require_file "${DOCKERFILE}"
  if ! grep -Eq '^ENV[[:space:]]+GOWORK=off([[:space:]]|$)' "${DOCKERFILE}"; then
    echo "docker build must set GOWORK=off to isolate service modules from the root workspace" >&2
    exit 1
  fi
  if grep -Eq '^COPY[[:space:]].*go\.work(\.sum)?([[:space:]]|$)' "${DOCKERFILE}"; then
    echo "docker build must not copy root go.work files when service module mode is enabled" >&2
    exit 1
  fi
  for entry in "${SERVICES[@]}"; do
    IFS=: read -r service_dir cmd_path binary_name <<<"${entry}"
    require_file "${ROOT}/${service_dir}/go.mod"
    require_file "${ROOT}/${service_dir}/${cmd_path#./}/main.go"
    if ! grep -q "SERVICE_DIR: ${service_dir}" "${COMPOSE_FILE}"; then
      echo "compose build target missing SERVICE_DIR for ${service_dir}" >&2
      exit 1
    fi
    if ! grep -q "BINARY_NAME: ${binary_name}" "${COMPOSE_FILE}"; then
      echo "compose build target missing BINARY_NAME for ${service_dir}" >&2
      exit 1
    fi
    if ! grep -q "CMD_PATH: ${cmd_path}" "${COMPOSE_FILE}"; then
      echo "compose build target missing CMD_PATH for ${service_dir}" >&2
      exit 1
    fi
    copy_count="$(grep -Ec "^COPY[[:space:]]+${service_dir}[[:space:]]+" "${DOCKERFILE}" || true)"
    if [ "${copy_count}" -ne 1 ]; then
      echo "docker build context must contain exactly one COPY for ${service_dir}; found ${copy_count}" >&2
      exit 1
    fi
  done
  if grep -RInE '(COPY|ADD).*(\.env|secret|credentials)' "${DOCKERFILE}"; then
    echo "potential secret material found in image build definitions" >&2
    exit 1
  fi
}

run_build() {
  local service_dir="$1"
  local cmd_path="$2"
  local binary_name="$3"
  local image="${IMAGE_PREFIX}/${service_dir}:${IMAGE_TAG}"
  local cmd=(
    docker build
    --file "${DOCKERFILE}"
    --build-arg "SERVICE_DIR=${service_dir}"
    --build-arg "CMD_PATH=${cmd_path}"
    --build-arg "BINARY_NAME=${binary_name}"
    --tag "${image}"
    "${ROOT}"
  )
  if [ "${DRY_RUN}" -eq 1 ]; then
    printf '%q ' "${cmd[@]}"
    printf '\n'
    return
  fi
  "${cmd[@]}"
}

check_targets

if [ "${CHECK_ONLY}" -eq 1 ]; then
  echo "microservice image build check: PASS (${#SERVICES[@]} targets)"
  exit 0
fi

for entry in "${SERVICES[@]}"; do
  IFS=: read -r service_dir cmd_path binary_name <<<"${entry}"
  run_build "${service_dir}" "${cmd_path}" "${binary_name}"
done
