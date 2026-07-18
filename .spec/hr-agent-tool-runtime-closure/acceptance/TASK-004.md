# TASK-004 Acceptance

- `ACASE-008` / `CHECK-007`: durable Run identity contains the effective Agent ID/type/name; governance evidence contains Prompt ID/version and Skill ID/version without bodies or personal data.
- `ACASE-009` / `CHECK-007`: model text chunks persist as `assistant.delta`; process statuses remain `process.delta`; final state does not append a duplicate full-answer delta.
- Tool Trace remains linked to Run Step/Run and preserves success/error status.
- Scope, Harness, formatting, and knowledge-impact checks pass.
