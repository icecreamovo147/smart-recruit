# Acceptance - TASK-007

## Outcome

The three live prompts are demonstrably consumed, safe observability and routed knowledge are current, and integration/smoke evidence supports runtime parity.

## Criteria

- Tests or runtime evidence show active `resume_profile_extractor`, `job_requirement_extractor`, and `candidate_match_evaluator` system prompts are loaded.
- Logs expose only prompt identity/version, model, stage, fallback, outcome, and duration; no full prompt, resume body, raw response, evidence body, secret, or PII.
- Only the six allowed knowledge files are updated, their critical claims are checked against source refs, and impact detection has no unresolved in-scope stale document.
- Resume reparse and candidate match smoke paths pass locally, or live provider execution is explicitly skipped with reason and fake-provider/integration evidence.
- `reports/compatibility-and-risk.md` truthfully documents dev/current parity, residual risks, and operational rollback.

## Tests and checks

- Full `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service`; relevant Commons AI tests.
- Privacy scan of logs/reports/evidence and the local smoke procedure.
- All global checks in `AGENT_RULES.md`, final feature validation, and pipeline-state validation.

## Out of scope

Provider credential provisioning, production rollout, historical backfill, public APIs, Proto, schema, dependencies, and lockfiles.
