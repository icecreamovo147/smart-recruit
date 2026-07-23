# Agent Rules

- Follow root `AGENTS.md`, `spec-harness`, and `harness-pipeline` in that order.
- Execute one ready TASK at a time and preserve unrelated changes.
- Use a reproducible Git tree as each TASK baseline.
- Contract-owned files change only through Amendment modes after initial plan approval.
- Runtime-owned reports/evidence/state may be updated by the orchestrator.
- Every TASK requires independent Review because the feature changes high-risk tool governance and data-truth behavior.
- TASK-002 requires explicit user confirmation before modifying `smart-recruit-commons/**`.
- No schema, Proto/public API, dependency/lockfile, auth/security policy, or global config changes.
- No auto-execution of side-effecting tools. No sensitive Prompt, Skill, MCP arguments, resume text, or candidate data in evidence/logs.
- Run every declared blocking check, scope check, `agent-check.sh`, and knowledge impact detection.
