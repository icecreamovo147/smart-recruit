#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/hr-agent-resumable-stream"
TASK_ID="${1:-${TASK_ID:-}}"
FAIL=0

mark_fail() {
  FAIL=1
}

run_cmd() {
  local label="$1"
  shift
  echo "==> ${label}"
  "$@"
  local code=$?
  if [ ${code} -ne 0 ]; then
    echo "FAIL: ${label} exited with ${code}"
    mark_fail
  else
    echo "OK: ${label}"
  fi
}

cd "${ROOT}" || exit 1

run_cmd "spec-harness validate feature" \
  node .agents/skills/spec-harness/scripts/validate-feature.mjs \
  --feature "${FEATURE_DIR}" \
  --require-pipeline

run_cmd "shell syntax check-task-scope.sh" bash -n "${FEATURE_DIR}/scripts/check-task-scope.sh"
run_cmd "shell syntax agent-check.sh" bash -n "${FEATURE_DIR}/scripts/agent-check.sh"

need_logic=1
need_web=0
need_frontend=0

case "${TASK_ID}" in
  TASK-HARS-001|TASK-HARS-002|TASK-HARS-003|TASK-HARS-004)
    need_logic=1
    ;;
  TASK-HARS-005)
    need_logic=1
    need_web=1
    ;;
  TASK-HARS-006|TASK-HARS-007)
    need_frontend=1
    ;;
  TASK-HARS-008|"")
    need_logic=1
    need_web=1
    need_frontend=1
    ;;
esac

if [ "${need_logic}" = "1" ] && [ -d "logic-grpc-service" ]; then
  run_cmd "logic-grpc-service agent/run tests" \
    bash -lc "cd logic-grpc-service && go test ./service ./repository ./model -count=1 -run 'Test.*Agent.*|Test.*Run.*|Test.*AI.*|Test.*Model.*'"
fi

if [ "${need_web}" = "1" ] && [ -d "web-gin-service" ]; then
  run_cmd "web-gin-service handler/rpc tests" \
    bash -lc "cd web-gin-service && go test ./handler/... ./rpc/... ./router/... -count=1"
fi

if [ "${need_frontend}" = "1" ] && [ -f "hr-frontend/package.json" ]; then
  run_cmd "hr-frontend typecheck" bash -lc "pnpm --filter hr-frontend typecheck"
  run_cmd "hr-frontend tests" bash -lc "pnpm --filter hr-frontend test"
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi

echo "agent-check.sh completed"
