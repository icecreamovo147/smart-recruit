# TASKS - github-ci-current-state-refresh

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-GCI-001 | Align GitHub CI with current repository layout | completed | `.github/workflows/**`, feature evidence | `.spec/github-ci-current-state-refresh/acceptance/TASK-GCI-001.md` |

## TASK-GCI-001 - Align GitHub CI with current repository layout

### Goal

Update GitHub CI workflows so they match current Go modules, frontend workspace, proto contracts, and knowledge feature set.

### Scope

- `.github/workflows/ci.yml`
- `.github/workflows/knowledge-validation.yml`
- `.spec/github-ci-current-state-refresh/**`

### Required Tests

- `rg -n "logic-grpc-service|web-gin-service" .github/workflows`
- Local Go module test loop.
- Proto sync and generation check.
- Frontend filtered typecheck, build, and test commands.
- `bash .spec/github-ci-current-state-refresh/scripts/check-task-scope.sh TASK-GCI-001`
- `bash .spec/github-ci-current-state-refresh/scripts/agent-check.sh`
- `git diff --check`
