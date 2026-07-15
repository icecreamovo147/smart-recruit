#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR=".spec/ai-agent-runtime-recovery"
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
run_cmd "feature whitespace check" git diff --check -- "${FEATURE_DIR}" smart-recruit-ai-agent-service smart-recruit-gateway hr-frontend user-frontend smart-recruit-commons/ai
run_cmd "task scope json parse" node -e 'JSON.parse(require("fs").readFileSync(process.argv[1], "utf8"))' "${FEATURE_DIR}/task-scope.json"

CHANGED="$(git diff --name-only; git diff --cached --name-only; git ls-files --others --exclude-standard)"

if printf "%s\n" "${CHANGED}" | grep -q "^smart-recruit-ai-agent-service/"; then
  run_cmd "smart-recruit-ai-agent-service gofmt check" bash -lc "cd smart-recruit-ai-agent-service && files=\$(gofmt -l \$(find . -name '*.go')); if [ -n \"\$files\" ]; then echo \"\$files\"; exit 1; fi"
  run_cmd "smart-recruit-ai-agent-service go test" bash -lc "cd smart-recruit-ai-agent-service && GOWORK=off go test ./..."
fi

if printf "%s\n" "${CHANGED}" | grep -q "^smart-recruit-gateway/"; then
  run_cmd "smart-recruit-gateway gofmt check" bash -lc "cd smart-recruit-gateway && files=\$(gofmt -l \$(find . -name '*.go')); if [ -n \"\$files\" ]; then echo \"\$files\"; exit 1; fi"
  run_cmd "smart-recruit-gateway go test" bash -lc "cd smart-recruit-gateway && GOWORK=off go test ./..."
fi

if printf "%s\n" "${CHANGED}" | grep -q "^smart-recruit-commons/ai/"; then
  run_cmd "smart-recruit-commons gofmt check" bash -lc "cd smart-recruit-commons && files=\$(gofmt -l \$(find ai -name '*.go')); if [ -n \"\$files\" ]; then echo \"\$files\"; exit 1; fi"
  run_cmd "smart-recruit-commons go test ai" bash -lc "cd smart-recruit-commons && GOWORK=off go test ./ai/..."
fi

if printf "%s\n" "${CHANGED}" | grep -q "^hr-frontend/"; then
  run_cmd "hr-frontend typecheck" pnpm --filter hr-frontend typecheck
fi

if printf "%s\n" "${CHANGED}" | grep -q "^user-frontend/"; then
  run_cmd "user-frontend typecheck" pnpm --filter user-frontend typecheck
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi

echo "agent-check.sh completed"
