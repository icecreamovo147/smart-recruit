# Acceptance - TASK-002

## Outcome

The recruiting-intelligence runtime consumes existing feature flags, fallback policy, and timeouts through an explicit immutable policy.

## Criteria

- All seven existing settings named in TASKS are mapped and injected; defaults are deterministic.
- Resume and match timeout contexts are distinct and cancellation propagates.
- Disabled capabilities and fallback combinations are testable without changing shared configuration definitions.
- No behavior outside the three recruiting-intelligence capabilities is changed.

## Tests and checks

- Focused policy/default/disabled/timeout/cancellation tests.
- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service`.
- All global checks in `AGENT_RULES.md`.

## Out of scope

Shared config edits, extractor implementation, public APIs, Proto, schema, and dependencies.
