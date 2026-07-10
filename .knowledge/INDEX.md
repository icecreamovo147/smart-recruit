# Knowledge Index

This index provides stable task routing. It intentionally does not enumerate every document; generate a complete local catalog from document frontmatter when needed.

Read this file after `AGENTS.md`, the active feature contract, and `.knowledge/README.md`. Use it only to route relevant active knowledge; it is not a second source of implementation rules.

## Core Navigation

| Task area | Recommended knowledge |
|---|---|
| Repository architecture | `architecture/system-overview.md`, `architecture/service-boundaries.md` |
| Agent runtime | `architecture/agent-runtime.md` |
| Skill, Memory, or Embedding | `architecture/semantic-retrieval.md`, `domains/agent-skill.md`, `domains/memory-and-context.md` |
| Recruitment workflows | `domains/recruitment.md` |
| Local development | `runbooks/local-development.md` |
| Retrieval diagnostics | `runbooks/debug-agent-retrieval.md` |
| Proto changes | `pitfalls/protobuf-synchronization.md` |
| Embedding fallback | `pitfalls/embedding-fallback.md` |
| HR admin pages | `pitfalls/frontend-menu-consistency.md` |
| Agent workflow and knowledge protocol | `architecture/system-overview.md`, `runbooks/local-development.md` |

Some routed documents are created by later TASKs. Until a path exists, use the current code, active `.spec` contract, and tests as the authority.

## Excluded by Default

- `inbox/`: unapproved candidates
- `archive/`: historical material
- documents whose frontmatter status is `draft`, `stale`, `deprecated`, or `archived`
