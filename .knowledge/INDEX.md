# Knowledge Index

This index provides stable task routing. It intentionally does not enumerate every document; generate a complete local catalog from document frontmatter when needed.

Read this file after `AGENTS.md`, the active feature contract, and `.knowledge/README.md`. Use it only to route relevant active knowledge; it is not a second source of implementation rules.

## Core Navigation

| Task area | Recommended knowledge |
|---|---|
| Repository architecture | `architecture/system-overview.md`, `architecture/service-boundaries.md` |
| Auth, RBAC, and security audit | `architecture/auth-rbac-security.md`, `runbooks/debug-auth-permissions.md`, `pitfalls/auth-permission-alignment.md` |
| Gateway and API contracts | `architecture/api-contracts-and-gateway.md`, `pitfalls/protobuf-synchronization.md` |
| Persistence and migrations | `architecture/persistence-and-migrations.md`, `runbooks/protobuf-and-migration-change.md`, `pitfalls/migration-model-drift.md` |
| Agent runtime | `architecture/agent-runtime.md` |
| AI platform governance | `domains/ai-configuration-governance.md`, `domains/mcp-tool-governance.md`, `runbooks/debug-ai-configuration.md`, `pitfalls/mcp-policy-audit.md` |
| Skill, Memory, or Embedding | `architecture/semantic-retrieval.md`, `domains/agent-skill.md`, `domains/memory-and-context.md`, `domains/ai-configuration-governance.md` |
| Recruitment workflows | `domains/recruitment.md`, `domains/recruitment-lifecycle.md`, `runbooks/debug-recruitment-lifecycle.md` |
| Notification and outbox | `domains/notification-outbox.md`, `pitfalls/status-notification-drift.md` |
| Resume intelligence and matching | `domains/resume-intelligence.md`, `runbooks/debug-resume-intelligence.md`, `pitfalls/resume-sensitive-data.md` |
| Frontend apps and validation | `architecture/frontend-apps.md`, `runbooks/frontend-validation.md`, `pitfalls/frontend-menu-consistency.md` |
| Local development | `runbooks/local-development.md` |
| Retrieval diagnostics | `runbooks/debug-agent-retrieval.md` |
| Proto changes | `pitfalls/protobuf-synchronization.md` |
| Embedding fallback | `pitfalls/embedding-fallback.md` |
| HR admin pages | `pitfalls/frontend-menu-consistency.md` |
| Agent workflow and knowledge protocol | `architecture/system-overview.md`, `runbooks/local-development.md` |
| Knowledge coverage maintenance | `runbooks/knowledge-coverage-audit.md` |

Some routed documents are created by later TASKs. Until a path exists, use the current code, active `.spec` contract, and tests as the authority.

## Excluded by Default

- `inbox/`: unapproved candidates
- `archive/`: historical material
- documents whose frontmatter status is `draft`, `stale`, `deprecated`, or `archived`
