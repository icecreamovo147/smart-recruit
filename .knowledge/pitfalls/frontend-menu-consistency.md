---
schema_version: 1
id: frontend-menu-consistency
title: Operational admin menu consistency pitfall
kind: pitfall
status: active
owners:
  - frontend-platform
tags:
  - frontend
  - hr
  - platform
  - admin
  - navigation
applies_to:
  - AGENTS.md
  - hr-frontend/src/router/**
  - platform-frontend/src/router/**
  - platform-frontend/src/views/ai/**
  - platform-frontend/src/components/admin-console/**
source_refs:
  - AGENTS.md
  - hr-frontend/src/router/index.ts
  - platform-frontend/src/views/ai/AgentManageView.vue
  - platform-frontend/src/views/ai/PromptManageView.vue
  - platform-frontend/src/components/admin-console/PageHeader.vue
  - platform-frontend/src/components/admin-console/FilterToolbar.vue
  - platform-frontend/src/components/admin-console/DataTableCard.vue
last_verified: 2026-07-20
review_after: 2026-10-08
---

# Operational Admin Menu Consistency Pitfall

Operational admin pages can drift visually or navigationally when a page is moved or added without the established admin-console layout conventions. Technical AI administration belongs to the platform console; tenant business administration remains in HR.

## Trigger Conditions

- Adding a new HR left-menu page.
- Moving routes between `/hr/admin` and `/platform/ai`.
- Changing admin-console header, toolbar, table, empty-state, or status components.
- Adding pages for Agent, Skill, MCP, Embedding, LLM, Prompt, or other system administration surfaces.

## Risk

Users see inconsistent spacing, title/description treatment, action placement, or permission visibility. Reviewers may miss it if only route behavior is tested.

## Prevention

- Reuse `admin-console` components and existing page-header layout before introducing page-specific variants.
- Check route metadata, menu grouping, permissions, gateway route permission, and actual page component together.
- For AI/admin pages, compare platform router meta with `smart-recruit-gateway/router/router.go` permissions and frontend API helper paths.
- Verify responsive text fit and avoid introducing marketing-style hero/card layouts into operational admin pages.
- Keep visual consistency work scoped to the page family being changed.
- Run the frontend validation runbook for the touched app before reporting completion.

## Verification

This pitfall was verified from `AGENTS.md`, HR/platform router configuration, migrated Agent Management and Prompt Management pages, and platform admin-console components on 2026-07-20.
