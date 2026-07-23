# Acceptance - TASK-001

## Outcome

Structured completions load the active database system prompt per request and deliver explicit System/User messages through the shared provider controls, without the generic HR Markdown prompt.

## Criteria

- Prompt lookup is keyed by exact `agent_type` and `system`, chooses the active version deterministically, and observes a newly activated version on a later request.
- The provider receives distinct System and User roles and reuses timeout, retry, concurrency, and circuit-breaker behavior.
- Provider/prompt failures are typed or classifiable and logs contain identities, not prompt/message/model-output bodies.
- No external contract, dependency, Proto, configuration schema, or database schema changes.
- Evidence records the exact previously granted shared-module authorization from `AGENT_RULES.md`.

## Tests and checks

- Focused tests for active prompt selection/version refresh, System/User roles, absence of generic Markdown contamination, timeout/retry/failure behavior.
- `GOWORK=off go test ./...` in `smart-recruit-commons` and `smart-recruit-ai-agent-service`.
- All global scope, agent, evidence, knowledge-impact, privacy, and independent-review checks in `AGENT_RULES.md`.

## Out of scope

Extractor schemas, runtime policy wiring, public APIs, shared config definitions, dependencies, Proto, and migrations.
