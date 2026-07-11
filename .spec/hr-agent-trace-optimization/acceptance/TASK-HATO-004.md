# Acceptance - TASK-HATO-004

## TASK Summary

Add a local raw data viewer for durable step input/output and legacy trace args/results.

## SPEC References

- 4.6 JSON and Raw Data Display
- FR-008
- SSR-002
- SSR-003
- AC-007

## SDD References

- 3.6 JSON Viewer
- 6.4 Copy Workflow
- 9 Error Handling and Fallback Design
- 11 Testing Strategy

## Acceptance Criteria

- Valid JSON is pretty-formatted.
- Invalid JSON and non-JSON content remain visible as raw text.
- Long content is collapsed by default and can be expanded/collapsed.
- Copy action copies the full desensitized content available to the UI.
- Clipboard failures are handled without crashing.
- No third-party JSON viewer dependency is added.
- Existing backend-desensitized content is not modified into unmasked data.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh TASK-HATO-004`
- `bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh TASK-HATO-004`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

## Manual Verification, if needed

- Inspect a trace with long JSON and a trace with plain text output to verify formatting and fallback display.

## Out-of-Scope

- Adding JSON viewer dependencies.
- Backend desensitization changes.
- Public API changes.
