# TASK-003 Acceptance

- `ACASE-006` / `CHECK-005`, `CHECK-006`: only active compatible system Prompts can be bound; allowlisted variables render correctly; unknown variables fail closed; HR admin lists only compatible active Prompts.
- `ACASE-007` / `CHECK-005`: only the exact valid current Agent Skill version enters runtime instructions; missing/mismatched current version is skipped with governance evidence; exact version ID is recorded.
- Prompt/Skill content cannot add tools outside the Agent allowlist.
- Scope, Harness, frontend typecheck, formatting, and knowledge-impact checks pass.
