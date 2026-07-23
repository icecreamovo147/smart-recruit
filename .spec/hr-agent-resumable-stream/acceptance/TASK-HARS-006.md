# Acceptance - TASK-HARS-006

## TASK Summary

Create the unified HR Agent runtime that all HR Agent streaming entry points will use.

## SPEC References

- FR-012 through FR-016
- AC-002
- AC-007
- AC-010

## SDD References

- Section 3: Proposed Design
- Section 5: API and Interface Changes
- Section 9: Error Handling and Fallback Design

## Acceptance Criteria

- Adds typed frontend API helpers for durable run create, get, active lookup, subscribe, cancel, and confirm.
- Adds a framework-independent reducer for HR Agent run events.
- Reducer ignores stale run ids and duplicate sequence numbers.
- Reducer handles delta, snapshot, status, confirmation, result, error, cancel, and completion events.
- Adds a Vue composable that owns create, subscribe, hydrate, cancel, confirm, and subscription cleanup.
- Aborting a subscription does not call cancel unless the caller explicitly requests cancel.
- Adds focused Vitest coverage for reducer and composable behavior.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-006`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

## Manual Verification, if needed

Confirm frontend types tied to new public APIs were explicitly approved before editing.

## Out-of-Scope

- Migrating `AIChatView.vue` entry points except imports needed by tests.
- Backend files.
- Package manifests or lockfiles.
- Global store introduction.
