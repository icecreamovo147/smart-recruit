# TASK-GCI-001 Report - Align GitHub CI with Current Repository Layout

## Summary

Updated GitHub CI workflows to match the current microservice repository layout, canonical proto module, root frontend workspace, and active feature validation set.

## Test Results

- `rg -n "logic-grpc-service|web-gin-service" .github/workflows`: PASS, no matches.
- `bash .spec/github-ci-current-state-refresh/scripts/check-task-scope.sh TASK-GCI-001`: PASS.
- `bash .spec/github-ci-current-state-refresh/scripts/agent-check.sh`: PASS.
- `git diff --check`: PASS.
- Go module matrix loop over all current Go modules: PASS.
- Proto lint equivalent with temporary `protoc 25.3`: PASS.
- `pnpm install --frozen-lockfile`: PASS.
- Frontend filtered `typecheck`, `build`, and `test` for all three apps: PASS.

## Local Environment Notes

- MySQL migration tag tests were blocked locally by existing MySQL credentials and an unavailable Docker daemon; GitHub Actions provides the configured `mysql:8.0` service.
- `gitleaks` CLI was not installed locally; the GitHub Action secret scan job is unchanged.
