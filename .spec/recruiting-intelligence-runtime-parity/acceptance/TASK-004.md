# Acceptance - TASK-004

## Outcome

An internal `job_requirement_extractor/system` component produces a validated requirement profile with policy-controlled heuristic fallback.

## Criteria

- Output contains a bounded non-empty requirement list with unique IDs and valid category, priority, weight, knockout, and aliases.
- Illegal enums, duplicate IDs, empty requirements, invalid counts/weights, and structural JSON errors are rejected.
- Only permitted floating-point drift is normalized deterministically; material errors fail.
- Prompt/provider/schema failures use heuristic fallback only when enabled.
- The component is callable by matching but adds no endpoint, Proto field, or table.

## Tests and checks

- Focused valid/invalid enum/duplicate/empty/count/weight/normalization/fallback tests.
- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service`.
- All global checks in `AGENT_RULES.md`.

## Out of scope

Evidence matching, aggregate scoring, public APIs, Proto, and schema.
