#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR=".spec/recruiting-intelligence-runtime-parity"
FAIL=0

run_cmd() {
  local label="$1"
  shift
  echo "==> ${label}"
  "$@"
  local code=$?
  if [ ${code} -ne 0 ]; then
    echo "FAIL: ${label} exited with ${code}"
    FAIL=1
  else
    echo "OK: ${label}"
  fi
}

cd "${ROOT}" || exit 1
run_cmd "feature validation" node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature "${FEATURE_DIR}" --require-pipeline
run_cmd "feature JSON parse" node -e 'JSON.parse(require("fs").readFileSync(process.argv[1], "utf8")); JSON.parse(require("fs").readFileSync(process.argv[2], "utf8"))' "${FEATURE_DIR}/task-scope.json" "${FEATURE_DIR}/pipeline-state.json"
run_cmd "whitespace check" git diff --check -- "${FEATURE_DIR}" smart-recruit-ai-agent-service smart-recruit-commons/ai .knowledge

CHANGED="$(git diff --name-only; git diff --cached --name-only; git ls-files --others --exclude-standard)"
if printf "%s\n" "${CHANGED}" | grep -q '^smart-recruit-ai-agent-service/'; then
  run_cmd "AI Agent gofmt" bash -lc "cd smart-recruit-ai-agent-service && files=\$(gofmt -l \$(find . -name '*.go')); if [ -n \"\$files\" ]; then echo \"\$files\"; exit 1; fi"
  run_cmd "AI Agent tests" bash -lc "cd smart-recruit-ai-agent-service && GOWORK=off go test ./..."
fi
if printf "%s\n" "${CHANGED}" | grep -q '^smart-recruit-commons/ai/'; then
  run_cmd "Commons AI gofmt" bash -lc "cd smart-recruit-commons && files=\$(gofmt -l \$(find ai -name '*.go')); if [ -n \"\$files\" ]; then echo \"\$files\"; exit 1; fi"
  run_cmd "Commons AI tests" bash -lc "cd smart-recruit-commons && GOWORK=off go test ./ai/..."
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi
echo "agent-check.sh completed"
