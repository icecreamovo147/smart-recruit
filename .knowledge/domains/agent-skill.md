---
schema_version: 1
id: agent-skill
title: Agent Skill domain
kind: domain
status: active
owners:
  - agent-platform
tags:
  - agent
  - skill
  - retrieval
applies_to:
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - hr-frontend/src/views/hr/admin/AgentSkillManageView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/domain/model/capability.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go
  - hr-frontend/src/views/hr/admin/AgentSkillManageView.vue
last_verified: 2026-07-16
review_after: 2026-10-14
---

# Agent Skill Domain

Agent Skills are database-backed instructions, versions, and retrieval metadata used by AI Agent runtime. They are distinct from the separate Skill registry for versioned manifests/tool wrappers. Selection changes must preserve eligibility, status/version semantics, fallback, governance metadata, and deterministic scoring.

Native Agent Skill admin methods now support detail, create, update, version create/list/activate, status update, preview, and list operations against existing Agent Skill tables. The separate `SkillService` registry uses `ai_skills`, `ai_skill_versions`, and `ai_skill_tools`; runtime tool execution remains outside the registry path and must be handled by an explicit runner before returning success.

HR runtime selection admits only enabled, eligible Agent Skills whose positive `current_version_id` resolves to a version owned by that same Skill with non-empty `skill_md`. Missing, stale, mismatched, or unreadable current versions are skipped with governance error evidence; runtime never substitutes an arbitrary previous version or the Skill description. Prompt instructions receive the exact published body, while persisted chat process evidence and durable Run result evidence record only selection metadata plus `skill_id` and `version_id`, not the body. Existing Run evidence therefore retains the exact selected version even after a later version switch. Skill instructions cannot grant Tool authority: model Tool execution is independently intersected with the active Agent's per-request allowlist before reaching Recruitment clients.

## Verification

Verified against cumulative Agent Skill selection, exact-version enrichment, durable audit, Tool-boundary, and HR runtime tests on 2026-07-16.
