# Backend DDD Microservices Evolution Harness And Knowledge Baseline

本文档记录 `backend-ddd-microservices-evolution` 的 TASK-BDME-005 Harness 与知识影响基线。

## Harness Validation

`node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/backend-ddd-microservices-evolution --json` 的当前结果：

- `feature_name`: `backend-ddd-microservices-evolution`
- `classification`: `current`
- `issues`: none
- `warnings`: none
- TASK count: 52 (`TASK-BDME-001` through `TASK-BDME-052`)

当前 feature 目录满足 schemaVersion 1 Harness 形态，包含 SPEC、SDD、TASKS、AGENT_RULES、task-scope、acceptance、prompts、scripts、reports 和 pipeline-state。

## Knowledge Routing Baseline

TASK 执行时应先通过 `.knowledge/manifest.yaml` 和 `.knowledge/INDEX.md` 选择相关 active knowledge，再以代码、SPEC、SDD、acceptance、tests、schema/proto 为最终权威。

本功能点的后端演进任务主要路由如下：

- Repository architecture: `.knowledge/architecture/system-overview.md`, `.knowledge/architecture/service-boundaries.md`
- API and gateway contracts: `.knowledge/architecture/api-contracts-and-gateway.md`, `.knowledge/pitfalls/protobuf-synchronization.md`
- Auth, RBAC, and security audit: `.knowledge/architecture/auth-rbac-security.md`, `.knowledge/runbooks/debug-auth-permissions.md`, `.knowledge/pitfalls/auth-permission-alignment.md`
- Persistence and migrations: `.knowledge/architecture/persistence-and-migrations.md`, `.knowledge/runbooks/protobuf-and-migration-change.md`, `.knowledge/pitfalls/migration-model-drift.md`
- Recruitment workflow: `.knowledge/domains/recruitment.md`, `.knowledge/domains/recruitment-lifecycle.md`, `.knowledge/runbooks/debug-recruitment-lifecycle.md`
- Notification and outbox: `.knowledge/domains/notification-outbox.md`, `.knowledge/pitfalls/status-notification-drift.md`
- AI runtime and governance: `.knowledge/architecture/agent-runtime.md`, `.knowledge/domains/ai-configuration-governance.md`, `.knowledge/domains/mcp-tool-governance.md`, `.knowledge/pitfalls/mcp-policy-audit.md`
- Skill, memory, retrieval, and embedding: `.knowledge/architecture/semantic-retrieval.md`, `.knowledge/domains/agent-skill.md`, `.knowledge/domains/memory-and-context.md`, `.knowledge/pitfalls/embedding-fallback.md`
- Operations and local validation: `.knowledge/runbooks/local-development.md`, `.knowledge/runbooks/knowledge-coverage-audit.md`

Excluded by default:

- `.knowledge/inbox/**` unless evaluating candidate knowledge
- `.knowledge/archive/**`
- documents whose frontmatter status is draft, stale, deprecated, or archived

## TASK Report And Evidence Expectations

Every TASK report/evidence must include:

- TASK ID and title
- Base SHA/tree captured before TASK edits
- Modified file list and per-file summary
- Scope result, forbidden/out-of-scope file result, and approved exceptions if any
- SPEC comparison result
- SDD comparison result
- Acceptance comparison result
- Required check commands, exit codes, and summaries
- Knowledge impact result, triggered files, reviewed documents, document verdicts, coverage gaps, and validation result
- Risks and rollback/residual-risk notes
- Whether the next TASK can start
- Human confirmation metadata when required
- Self-review verdict exactly `通过` or `不通过`

When `.knowledge/**` files are not changed, `node .knowledge/scripts/validate-knowledge.mjs` should be recorded as skipped with a reason. When `.knowledge/**` files are changed, that validator must run and be recorded.

## Knowledge Impact Result Meanings

- `none`: no relevant knowledge route was triggered.
- `update_required`: routed active documents were reviewed; no direct document edit is required or any required update was completed in scope.
- `candidate_required`: a potential new knowledge entry belongs in inbox or a later scoped task.
- `stale_detected`: an active document contradicts current source of truth and must not be ignored.
- `conflict_detected`: knowledge conflicts with SPEC/SDD/code/tests/schema/proto.
- `coverage_gap`: no active knowledge route adequately covers the changed area.

Passing self-review cannot contain stale or conflicting knowledge verdicts.
