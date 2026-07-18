# GitHub CI Current State Refresh SPEC

## 1. Background

The repository has moved from the former `logic-grpc-service` and `web-gin-service` layout to current independent Go modules, a canonical proto module, and root-level pnpm frontend workspace execution. The existing GitHub CI workflow still described legacy paths and needed to be aligned before opening a PR to `dev`.

## 2. Goals

- Update GitHub CI to run tests against the current Go module layout.
- Update frontend CI to install dependencies from the repository root and run filtered package scripts.
- Update proto lint to regenerate canonical contracts from `smart-recruit-proto`.
- Keep knowledge validation aware of current feature contracts.

## 3. Non-Goals

- Do not change product code, public APIs, database schema, generated protobuf contracts, package manifests, or lockfiles.
- Do not change CI triggers or secret scanning policy.

## 4. Functional Requirements

- CI must not reference deleted `logic-grpc-service` or `web-gin-service` paths.
- Go tests must cover current Go modules.
- Frontend jobs must use the root pnpm workspace and package filters.
- Proto lint must run `smart-recruit-proto/scripts/generate-go.sh` and require no generated diff.
- Knowledge validation must include active feature contracts added by this work.

## 5. Acceptance Criteria

- Workflow stale path scan passes.
- Go module test matrix passes locally with equivalent commands.
- Frontend typecheck, build, and test commands pass locally.
- Proto sync and generation checks pass locally.
- Feature scope and agent checks pass.
