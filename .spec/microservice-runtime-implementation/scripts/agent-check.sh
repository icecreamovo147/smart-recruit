#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR=".spec/microservice-runtime-implementation"
FAIL=0

mark_fail() { FAIL=1; }

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

run_cmd "feature validation" node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature "${FEATURE_DIR}"
run_cmd "diff whitespace check" git diff --check -- "${FEATURE_DIR}" smart-recruit-proto smart-recruit-platform-go smart-recruit-gateway smart-recruit-identity-service smart-recruit-recruitment-service smart-recruit-interview-service smart-recruit-offer-service smart-recruit-notification-service smart-recruit-ai-agent-service smart-recruit-analytics-service smart-recruit-worker-service smart-recruit-deploy web-gin-service logic-grpc-service docker deploy scripts docs go.work
run_cmd "task scope json parse" node -e 'JSON.parse(require("fs").readFileSync(process.argv[1], "utf8"))' "${FEATURE_DIR}/task-scope.json"
run_cmd "mysql table ownership check" node scripts/check-mysql-table-ownership.mjs
run_cmd "redis prefix check" node scripts/check-redis-prefixes.mjs

CHANGED="$(git diff --name-only; git diff --cached --name-only; git ls-files --others --exclude-standard)"

if printf "%s\n" "${CHANGED}" | grep -q "^logic-grpc-service/"; then
  run_cmd "logic-grpc-service gofmt check" bash -lc 'cd logic-grpc-service && files=$(gofmt -l $(find . -name "*.go" -not -path "./recruitment/pb/*")); if [ -n "$files" ]; then echo "$files"; exit 1; fi'
  run_cmd "logic-grpc-service go test" bash -lc 'cd logic-grpc-service && GOWORK=off go test ./...'
fi

if printf "%s\n" "${CHANGED}" | grep -q "^web-gin-service/"; then
  run_cmd "web-gin-service gofmt check" bash -lc 'cd web-gin-service && files=$(gofmt -l $(find . -name "*.go" -not -path "./recruitment/pb/*")); if [ -n "$files" ]; then echo "$files"; exit 1; fi'
  run_cmd "web-gin-service go test" bash -lc 'cd web-gin-service && go test ./...'
fi

for module in smart-recruit-platform-go smart-recruit-gateway smart-recruit-identity-service smart-recruit-recruitment-service smart-recruit-interview-service smart-recruit-offer-service smart-recruit-notification-service smart-recruit-ai-agent-service smart-recruit-analytics-service smart-recruit-worker-service; do
  if [ -f "${module}/go.mod" ] && printf "%s\n" "${CHANGED}" | grep -q "^${module}/"; then
    run_cmd "${module} gofmt check" bash -lc "cd ${module} && files=\$(gofmt -l \$(find . -name '*.go')); if [ -n \"\$files\" ]; then echo \"\$files\"; exit 1; fi"
    run_cmd "${module} go test" bash -lc "cd ${module} && go test ./..."
  fi
done

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi

echo "agent-check.sh completed"
