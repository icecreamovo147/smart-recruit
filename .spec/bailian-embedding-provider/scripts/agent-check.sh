#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FAIL=0

mark_fail() {
  FAIL=1
}

has_pnpm_script() {
  local pkg="$1"
  local script="$2"
  node -e "const p=require('./${pkg}/package.json'); process.exit(p.scripts && p.scripts['${script}'] ? 0 : 1)" >/dev/null 2>&1
}

run_cmd() {
  local label="$1"
  shift
  echo "==> ${label}"
  "$@"
  local code=$?
  if [ $code -ne 0 ]; then
    echo "FAIL: ${label} exited with ${code}"
    mark_fail
  else
    echo "OK: ${label}"
  fi
}

run_pnpm_script() {
  local pkg="$1"
  local script="$2"
  if [ ! -f "${ROOT}/${pkg}/package.json" ]; then
    echo "SKIP: ${pkg} has no package.json"
    return 0
  fi
  if has_pnpm_script "${pkg}" "${script}"; then
    run_cmd "pnpm --filter ${pkg} ${script}" pnpm --filter "${pkg}" "${script}"
  else
    echo "SKIP: ${pkg} 暂未配置 ${script} 脚本"
  fi
}

cd "${ROOT}" || exit 1

# Required frontend checks. Missing scripts are reported as SKIP, not treated as success.
run_pnpm_script "hr-frontend" "lint"
run_pnpm_script "hr-frontend" "typecheck"
run_pnpm_script "hr-frontend" "test"
run_pnpm_script "hr-frontend" "build"

# Backend checks are relevant for most tasks in this feature.
if [ -d "logic-grpc-service" ]; then
  run_cmd "logic-grpc-service go test ./..." bash -lc "cd logic-grpc-service && go test ./..."
fi

if [ -d "web-gin-service" ]; then
  run_cmd "web-gin-service go test ./..." bash -lc "cd web-gin-service && go test ./..."
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check completed with failures"
  exit 1
fi

echo "agent-check completed"
