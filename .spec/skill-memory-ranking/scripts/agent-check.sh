#!/usr/bin/env bash
# Run project-appropriate validation checks for skill-memory-ranking.
# Includes Go tests, frontend typecheck, and a few sanity greps.
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
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

# Backend checks (most tasks touch logic-grpc-service).
if [ -d "logic-grpc-service" ]; then
  run_cmd "logic-grpc-service gofmt" bash -lc "cd logic-grpc-service && gofmt -l service/skill_memory_ranking.go service/skill_memory_ranking_test.go service/agent_skill_selector.go service/agent_skill_selector_test.go service/agent_context.go service/agent_context_memory_test.go service/agent_skill_service.go"
  run_cmd "logic-grpc-service go build" bash -lc "cd logic-grpc-service && go build ./..."
  run_cmd "logic-grpc-service go test" bash -lc "cd logic-grpc-service && go test ./..."
fi

# Web gin proto regeneration sanity (only run if protoc is available).
if [ -d "web-gin-service" ]; then
  run_cmd "web-gin-service go build" bash -lc "cd web-gin-service && go build ./..."
fi

# Frontend checks (TASK-007 may touch types).
if [ -f "hr-frontend/package.json" ]; then
  run_cmd "hr-frontend typecheck" bash -lc "pnpm --filter hr-frontend typecheck"
fi

# Sanity: ensure no TODO/FIXME blocks are added without context (best effort).
TODO_HITS=$(grep -RIn --include="*.go" -E "TODO|FIXME" logic-grpc-service/service/skill_memory_ranking*.go 2>/dev/null || true)
if [ -n "${TODO_HITS}" ]; then
  echo "INFO: TODO/FIXME markers in skill_memory_ranking*.go:"
  echo "${TODO_HITS}"
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi

echo "agent-check.sh completed"
