# AGENT RULES - pr-review-fixes

## Feature Scope

This feature fixes production-readiness issues found during PR review:

- CI trigger coverage for PRs to `dev`.
- Global LLM default model consistency.
- Agent SKILL embedding lifecycle consistency.
- Provider extra header validation and masking.
- Management API pagination limits.

## General Rules

- Execute exactly one TASK at a time.
- Do not modify files outside the current TASK scope.
- Do not refactor unrelated code.
- Do not modify package manifests or lockfiles.
- Do not modify protobuf files unless a TASK explicitly allows it.
- Do not change auth, RBAC, data scopes, or route permissions.
- Do not change public route paths.
- Do not physically delete historical `ai_embeddings` rows for disabled SKILLs.
- Do not return raw provider secrets or raw extra header values.

## Task Order

TASK-001 must be completed first. Later TASKs may be implemented in order unless the user explicitly reprioritizes.

## Testing Requirements

After each implementation task, run:

```bash
git diff --name-only
bash .spec/pr-review-fixes/scripts/check-task-scope.sh <TASK-ID>
bash .spec/pr-review-fixes/scripts/agent-check.sh
```

Run all TASK-specific checks listed in the acceptance file.

## Compatibility Rules

- Existing protobuf request and response shapes must remain compatible.
- Existing frontend calls must continue working.
- Oversized `page_size` must be clamped, not rejected.
- Default model fallback to first enabled model is allowed only when no enabled default exists.
- Provider extra headers may be displayed masked but must never be displayed raw.

## Security Rules

- Never log raw API keys or header values.
- Invalid legacy header data must not be echoed in responses.
- Header validators must reject malformed or unsafe header names.

## Hard Stop Conditions

Stop and request confirmation if a TASK requires:

- Modifying package manifests or lockfiles.
- Adding dependencies.
- Changing protobuf definitions.
- Changing authentication or authorization behavior.
- Modifying files outside TASK scope.
- Physically deleting historical embedding rows.
- Changing the product decisions recorded in the SPEC.

## Report Requirements

Each implemented TASK must create or update:

```text
.spec/pr-review-fixes/reports/<TASK-ID>-report.md
```

The report must include modified files, scope status, SPEC/SDD comparison, acceptance comparison, test results, risks, and whether the next TASK can start.
