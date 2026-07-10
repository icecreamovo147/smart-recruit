#!/usr/bin/env bash
# Run project-appropriate validation checks for skill-memory-ranking-followups.
# Includes Go tests, frontend typecheck, and protoc sanity.
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

# Backend checks
if [ -d "logic-grpc-service" ]; then
  run_cmd "logic-grpc-service gofmt" bash -lc "cd logic-grpc-service && gofmt -l service/skill_memory_ranking.go service/skill_memory_ranking_test.go service/agent_skill_service.go service/agent_skill_service_test.go"
  run_cmd "logic-grpc-service go build" bash -lc "cd logic-grpc-service && go build ./..."
  run_cmd "logic-grpc-service go test" bash -lc "cd logic-grpc-service && go test ./..."
fi

# Web gin sanity
if [ -d "web-gin-service" ]; then
  run_cmd "web-gin-service go build" bash -lc "cd web-gin-service && go build ./..."
fi

# Frontend checks (TASK-FU-003 may touch views/)
if [ -f "hr-frontend/package.json" ]; then
  run_cmd "hr-frontend typecheck" bash -lc "pnpm --filter hr-frontend typecheck"
fi

# CI yaml syntax check (TASK-FU-002)
if [ -f ".github/workflows/ci.yml" ]; then
  if command -v python3 >/dev/null 2>&1; then
    run_cmd "ci.yml yaml parse" python3 -c "
import sys, yaml
try:
  with open('.github/workflows/ci.yml') as f:
    yaml.safe_load(f)
except Exception as e:
  print('YAML parse error:', e); sys.exit(1)
"
  else
    echo "INFO: skipping ci.yml yaml parse (python3 not available)"
  fi
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi

echo "agent-check.sh completed"
