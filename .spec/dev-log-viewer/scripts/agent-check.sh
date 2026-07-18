#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/dev-log-viewer"
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

check_gofmt() {
  local output
  output="$(find dev-log-viewer -type f -name '*.go' -print0 | xargs -0 -r gofmt -l)"
  if [ -n "${output}" ]; then
    echo "${output}"
    return 1
  fi
}

cd "${ROOT}" || exit 1

run_cmd "spec-harness validate feature" \
  node .agents/skills/spec-harness/scripts/validate-feature.mjs \
  --feature "${FEATURE_DIR}" \
  --require-pipeline
run_cmd "shell syntax check-task-scope.sh" bash -n "${FEATURE_DIR}/scripts/check-task-scope.sh"
run_cmd "shell syntax agent-check.sh" bash -n "${FEATURE_DIR}/scripts/agent-check.sh"

need_go=0
need_go_race=0
need_frontend=0
need_scripts=0
need_knowledge=0

case "${TASK_ID}" in
  TASK-DLV-001)
    need_go=1
    need_frontend=1
    ;;
  TASK-DLV-002)
    need_go=1
    ;;
  TASK-DLV-003|TASK-DLV-004)
    need_go=1
    need_go_race=1
    ;;
  TASK-DLV-005|TASK-DLV-006|TASK-DLV-007)
    need_frontend=1
    ;;
  TASK-DLV-008)
    need_go=1
    need_frontend=1
    need_scripts=1
    need_knowledge=1
    ;;
  TASK-DLV-009|"")
    need_go=1
    need_go_race=1
    need_frontend=1
    need_scripts=1
    need_knowledge=1
    ;;
  *)
    echo "FAIL: unknown TASK-ID ${TASK_ID}" >&2
    exit 2
    ;;
esac

if [ "${need_go}" = "1" ] && [ -f "dev-log-viewer/go.mod" ]; then
  run_cmd "dev-log-viewer gofmt" check_gofmt
  run_cmd "dev-log-viewer Go tests" bash -lc "cd dev-log-viewer && go test ./..."
  run_cmd "dev-log-viewer Go vet" bash -lc "cd dev-log-viewer && go vet ./..."
fi

if [ "${need_go_race}" = "1" ] && [ -f "dev-log-viewer/go.mod" ]; then
  run_cmd "dev-log-viewer Go race tests" bash -lc "cd dev-log-viewer && go test ./... -race"
fi

if [ "${need_frontend}" = "1" ] && [ -f "dev-log-viewer/package.json" ]; then
  run_cmd "dev-log-viewer typecheck" pnpm --filter dev-log-viewer typecheck
  run_cmd "dev-log-viewer frontend tests" pnpm --filter dev-log-viewer test
  run_cmd "dev-log-viewer frontend build" pnpm --filter dev-log-viewer build
fi

if [ "${need_scripts}" = "1" ]; then
  run_cmd "start/stop shell syntax" bash -n start-dev.sh stop-dev.sh
fi

if [ "${need_knowledge}" = "1" ]; then
  run_cmd "knowledge validator tests" node .knowledge/scripts/knowledge-validator.test.mjs
  run_cmd "validate knowledge" node .knowledge/scripts/validate-knowledge.mjs --root .
  run_cmd "check knowledge references" node .knowledge/scripts/check-references.mjs --root .

  BASE_TREE="${TASK_BASE_TREE:-}"
  if [ -z "${BASE_TREE}" ] && [ -f "${FEATURE_DIR}/pipeline-state.json" ] && [ -n "${TASK_ID}" ]; then
    BASE_TREE="$(node -e "const s=require(process.argv[1]); process.stdout.write(s.task_runs?.[process.argv[2]]?.base_tree || '')" "${FEATURE_DIR}/pipeline-state.json" "${TASK_ID}")"
  fi
  if [ -n "${BASE_TREE}" ]; then
    run_cmd "knowledge impact detection" node .knowledge/scripts/detect-impact.mjs --root . --base-tree "${BASE_TREE}"
  else
    echo "FAIL: knowledge impact detection requires TASK_BASE_TREE or pipeline-state base_tree"
    mark_fail
  fi
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi

echo "agent-check.sh completed"
