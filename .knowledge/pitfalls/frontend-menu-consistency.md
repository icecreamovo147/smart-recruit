---
schema_version: 1
id: frontend-menu-consistency
title: HR admin menu consistency pitfall
kind: pitfall
status: active
owners:
  - frontend-platform
tags:
  - frontend
  - hr
  - admin
  - navigation
applies_to:
  - AGENTS.md
  - hr-frontend/src/router/**
  - hr-frontend/src/views/hr/admin/**
  - hr-frontend/src/components/admin-console/**
source_refs:
  - AGENTS.md
  - hr-frontend/src/router/index.ts
  - hr-frontend/src/views/hr/admin/AgentManageView.vue
  - hr-frontend/src/views/hr/PromptManageView.vue
last_verified: 2026-07-10
review_after: 2026-10-08
---

# HR Admin Menu Consistency Pitfall

HR admin pages can drift visually or navigationally when a page is added by copying a nearby route but not the established admin-console layout conventions. The repository rule for new HR left-menu pages explicitly calls out the Agent Management and Prompt Management page-header pattern.

## Trigger Conditions

- Adding a new HR left-menu page.
- Moving routes under `/hr/admin`.
- Changing admin-console header, toolbar, table, empty-state, or status components.
- Adding pages for Agent, Skill, MCP, Embedding, LLM, Prompt, or other system administration surfaces.

## Risk

Users see inconsistent spacing, title/description treatment, action placement, or permission visibility. Reviewers may miss it if only route behavior is tested.

## Prevention

- Reuse `admin-console` components and existing page-header layout before introducing page-specific variants.
- Check route metadata, menu grouping, permissions, and actual page component together.
- Verify responsive text fit and avoid introducing marketing-style hero/card layouts into operational admin pages.
- Keep visual consistency work scoped to the page family being changed.

## Verification

This pitfall was verified from `AGENTS.md`, HR router configuration, Agent Management, Prompt Management, and admin-console components on 2026-07-10.
