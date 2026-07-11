#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR=".spec/backend-ddd-microservices-evolution"
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

run_cmd "feature validation" node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature "${FEATURE_DIR}"
run_cmd "git diff whitespace check" git diff --check -- .spec/backend-ddd-microservices-evolution logic-grpc-service web-gin-service deploy docker docs .knowledge scripts db.sql

CHANGED="$(git diff --name-only; git diff --cached --name-only; git ls-files --others --exclude-standard)"

if printf "%s\n" "${CHANGED}" | grep -q "^logic-grpc-service/"; then
  run_cmd "logic-grpc-service gofmt check" bash -lc 'cd logic-grpc-service && files=$(gofmt -l $(find . -name "*.go" -not -path "./recruitment/pb/*")); if [ -n "$files" ]; then echo "$files"; exit 1; fi'
  run_cmd "logic-grpc-service go test" bash -lc 'cd logic-grpc-service && go test ./...'
fi

if printf "%s\n" "${CHANGED}" | grep -q "^web-gin-service/"; then
  run_cmd "web-gin-service gofmt check" bash -lc 'cd web-gin-service && files=$(gofmt -l $(find . -name "*.go" -not -path "./recruitment/pb/*")); if [ -n "$files" ]; then echo "$files"; exit 1; fi'
  run_cmd "web-gin-service go test" bash -lc 'cd web-gin-service && go test ./...'
fi

if printf "%s\n" "${CHANGED}" | grep -q "^.knowledge/"; then
  if [ -f .knowledge/scripts/validate-knowledge.mjs ]; then
    run_cmd "knowledge validation" node .knowledge/scripts/validate-knowledge.mjs
  else
    echo "WARN: .knowledge changed but validator not found"
  fi
fi

SCRIPT_FILES="$(printf "%s\n" "${CHANGED}" | grep "^scripts/.*\\.sh$" || true)"
if [ -n "${SCRIPT_FILES}" ]; then
  while IFS= read -r file; do
    [ -z "${file}" ] && continue
    run_cmd "shell syntax ${file}" bash -n "${file}"
  done <<< "${SCRIPT_FILES}"
fi

if [ ${FAIL} -ne 0 ]; then
  echo "agent-check.sh completed with failures"
  exit 1
fi

echo "agent-check.sh completed"
