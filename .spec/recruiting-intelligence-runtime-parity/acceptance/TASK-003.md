# Acceptance - TASK-003

## Outcome

Resume parsing uses `resume_profile_extractor/system`, validates its schema strictly, applies policy-controlled heuristic fallback, and preserves current versioned transactional persistence.

## Criteria

- Exactly one JSON object is decoded; required arrays/types, skill objects, dates, and useful non-empty content are validated.
- `skills: ["Go"]` is a schema failure and falls back only when enabled.
- Prompt, provider, JSON, schema, and timeout failures follow the fallback matrix.
- With fallback disabled, a failed parse creates no new profile and does not change current/version state.
- Successful persistence preserves input hash, parser version, increment, current switch, rollback, and privacy constraints.

## Tests and checks

- Focused strict decode/schema/date/non-empty/fallback/persistence/rollback tests.
- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service`.
- All global checks in `AGENT_RULES.md`.

## Out of scope

Job requirements, matching aggregation, historical profile rewrites, external contracts, Proto, and schema.
