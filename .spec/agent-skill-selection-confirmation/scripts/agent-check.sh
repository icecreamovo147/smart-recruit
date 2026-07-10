#!/usr/bin/env bash
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

if [ -d "logic-grpc-service" ]; then
  run_cmd "logic-grpc-service targeted go test" bash -lc "cd logic-grpc-service && go test ./service -run 'Test.*AgentSkill.*|Test.*AgentRun.*|Test.*AI.*'"
fi

if [ -d "web-gin-service" ]; then
  run_cmd "web-gin-service go test" bash -lc "cd web-gin-service && go test ./..."
fi

if [ -f "hr-frontend/package.json" ]; then
  run_cmd "hr-frontend typecheck" bash -lc "pnpm --filter hr-frontend typecheck"
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi

echo "agent-check.sh completed"
