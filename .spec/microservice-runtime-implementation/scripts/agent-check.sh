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
run_cmd "diff whitespace check" git diff --check -- "${FEATURE_DIR}" smart-recruit-proto smart-recruit-platform-go smart-recruit-domain-go smart-recruit-gateway smart-recruit-identity-service smart-recruit-recruitment-service smart-recruit-interview-service smart-recruit-offer-service smart-recruit-notification-service smart-recruit-ai-agent-service smart-recruit-analytics-service smart-recruit-worker-service smart-recruit-deploy docker deploy scripts docs go.work go.work.sum
run_cmd "task scope json parse" node -e 'JSON.parse(require("fs").readFileSync(process.argv[1], "utf8"))' "${FEATURE_DIR}/task-scope.json"
run_cmd "mysql table ownership check" node scripts/check-mysql-table-ownership.mjs
run_cmd "redis prefix check" node scripts/check-redis-prefixes.mjs
run_cmd "microservice image build target check" bash scripts/build-microservice-images.sh --check
run_cmd "compose microservice smoke check" bash scripts/compose-microservice-smoke.sh --check
run_cmd "microservice repo export dry-run" node scripts/export-microservice-repos.mjs --check
run_cmd "backend boundary check" node scripts/check-backend-boundaries.mjs

CHANGED="$(git diff --name-only; git diff --cached --name-only; git ls-files --others --exclude-standard)"

for module in smart-recruit-platform-go smart-recruit-domain-go smart-recruit-gateway smart-recruit-identity-service smart-recruit-recruitment-service smart-recruit-interview-service smart-recruit-offer-service smart-recruit-notification-service smart-recruit-ai-agent-service smart-recruit-analytics-service smart-recruit-worker-service; do
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
