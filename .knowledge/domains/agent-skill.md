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
  - admin
  - ranking
applies_to:
  - logic-grpc-service/service/agent_skill_service.go
  - logic-grpc-service/service/agent_skill_selector.go
  - logic-grpc-service/repository/agent_skill_repo.go
  - hr-frontend/src/views/hr/admin/AgentSkillManageView.vue
  - hr-frontend/src/components/agent-skill/**
source_refs:
  - logic-grpc-service/service/agent_skill_service.go
  - logic-grpc-service/service/agent_skill_selector.go
  - logic-grpc-service/repository/agent_skill_repo.go
  - hr-frontend/src/views/hr/admin/AgentSkillManageView.vue
  - .spec/agent-skill-selection-confirmation/agent-skill-selection-confirmation-SPEC.md
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Agent Skill Domain

Agent Skills are managed capabilities that can be invoked manually or selected automatically for the HR recruiting agent. They carry metadata such as agent type, category, scenario, risk level, required capabilities, semantic tags, output schema, enabled state, priority, and content.

Manual selection is explicit request input. Automatic selection filters runtime-compatible candidates and ranks them using rule, semantic, metadata, and business signals. Selection is capped to avoid overloading the prompt.

The project also has a separate SKILL registry managed by `SkillService`. Registry Skills define versioned manifests and tool wrappers that can become agent capabilities. Agent Skills are database-backed SKILL.md-style instructions and flows used by the HR assistant. Keep these two concepts distinct when changing admin UI, capability binding, or retrieval behavior.

## Ownership

- `AgentSkillService` owns CRUD, versions, status, activation, preview, embedding regeneration hooks, and semantic debug APIs.
- `agent_skill_selector.go` owns request-time selection and candidate ranking behavior.
- `AgentSkillRepo` owns persistence access and runtime records.
- `SkillService` owns the generic Skill registry, manifest validation, version activation, and callable tool collection.
- `AgentConfigService` owns capability binding across builtin, MCP, and Skill sources.
- HR admin views own management UI, editing workflow, and debug surface presentation.

## Review Triggers

- Ranking weights, candidate filters, manual selection semantics, required capabilities, output schema handling, or debug fields.
- SKILL registry manifest schema, runtime tool wrapping, or activation behavior.
- Agent capability binding across builtin, MCP, Skill, and Agent Skill surfaces.
- Embedding regeneration behavior for Skill content.
- HR admin page route/menu changes for Agent Skill management.
- Protobuf fields that affect Skill management or chat stream selection.

## Verification

This domain note was verified against current service, selector, repository, HR admin view, and the Agent Skill selection SPEC on 2026-07-10.
