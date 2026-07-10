#!/usr/bin/env bash
set -euo pipefail

echo "Running pr-review-fixes baseline checks..."

go_test_logic=false
go_test_web=false
hr_frontend=false

while IFS= read -r file; do
  case "$file" in
    logic-grpc-service/*) go_test_logic=true ;;
    web-gin-service/*) go_test_web=true ;;
    hr-frontend/*) hr_frontend=true ;;
  esac
done < <(git diff --name-only)

if [[ "$go_test_logic" == true ]]; then
  (cd logic-grpc-service && go test ./...)
else
  echo "Skipping logic-grpc-service go test: no logic files changed."
fi

if [[ "$go_test_web" == true ]]; then
  (cd web-gin-service && go test ./...)
else
  echo "Skipping web-gin-service go test: no web files changed."
fi

if [[ "$hr_frontend" == true ]]; then
  pnpm --filter hr-frontend typecheck
else
  echo "Skipping hr-frontend typecheck: no HR frontend files changed."
fi

echo "agent-check completed."
