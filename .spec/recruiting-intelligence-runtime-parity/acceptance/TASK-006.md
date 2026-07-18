# Acceptance - TASK-006

## Outcome

Per-requirement results are aggregated deterministically into dev-compatible dimensions, knockout/risk-adjusted total, recommendation, Chinese summary, and a versioned transactional evaluation.

## Criteria

- Dimension weights, knockout cap, risk penalty, recommendation thresholds, and Chinese summary rules match the SDD; model output never directly controls final total.
- Total/dimensions/risks/recommendation and real evidence source IDs are reproducible for fixed input.
- `agent_run_id`, version increment, latest switch, evidence truncation, and transaction rollback are preserved.
- A failed main chain with fallback disabled persists no new evaluation; historical evaluations are unchanged.

## Tests and checks

- Focused knockout/score/recommendation/risk/dimension/source-ID/version/latest/rollback tests.
- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service`.
- All global checks in `AGENT_RULES.md`.

## Out of scope

Historical backfill, public API changes, Proto, and schema.
