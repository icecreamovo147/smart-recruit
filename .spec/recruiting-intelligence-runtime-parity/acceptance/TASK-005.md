# Acceptance - TASK-005

## Outcome

Candidate evidence is indexed with real source IDs, deterministic matching is preferred, and only unresolved requirements invoke `candidate_match_evaluator/system`.

## Criteria

- Evidence sources are allowlisted, bounded, redacted, and retain stable real source IDs.
- A deterministic hit makes zero matcher calls; missing evidence invokes the matcher once per unresolved requirement.
- Matcher output validates status, score, confidence, risk, and every evidence reference against the index.
- Fabricated/unknown source IDs, invalid enums/ranges, or leaked content are rejected.
- Individual model failures preserve deterministic judgments and follow fallback policy.

## Tests and checks

- Focused evidence privacy/bounds/source, deterministic-zero-call, unresolved-call, schema/provenance, and fallback tests.
- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service`.
- All global checks in `AGENT_RULES.md`.

## Out of scope

Final score selection by the model, public APIs, Proto, and schema.
