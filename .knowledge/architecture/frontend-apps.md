---
schema_version: 1
id: frontend-apps
title: Frontend app architecture
kind: architecture
status: active
owners:
  - frontend-platform
tags:
  - frontend
  - vue
  - router
  - auth
  - api
applies_to:
  - hr-frontend/src/**
  - user-frontend/src/**
  - interviewer-frontend/src/**
  - platform-frontend/src/**
  - packages/shared/src/**
source_refs:
  - pnpm-workspace.yaml
  - hr-frontend/tsconfig.json
  - user-frontend/tsconfig.json
  - interviewer-frontend/tsconfig.json
  - platform-frontend/tsconfig.json
  - hr-frontend/src/router/index.ts
  - user-frontend/src/router/index.ts
  - interviewer-frontend/src/router/index.ts
  - platform-frontend/src/router/index.ts
  - hr-frontend/src/stores/auth.ts
  - user-frontend/src/stores/auth.ts
  - interviewer-frontend/src/stores/auth.ts
  - platform-frontend/src/stores/auth.ts
  - hr-frontend/src/api/request.ts
  - user-frontend/src/api/request.ts
  - interviewer-frontend/src/api/request.ts
  - platform-frontend/src/api/http.ts
  - hr-frontend/src/components/NotificationBell.vue
  - user-frontend/src/components/NotificationBell.vue
  - interviewer-frontend/src/components/NotificationBell.vue
  - platform-frontend/src/components/admin-console/PageHeader.vue
  - platform-frontend/src/components/admin-console/DataTableCard.vue
  - packages/shared/src/components/EmailSetupDialog.vue
  - packages/shared/src/types/domain.ts
  - packages/shared/src/utils/token.ts
  - packages/shared/src/utils/format.ts
last_verified: 2026-07-21
review_after: 2026-10-08
---

# Frontend App Architecture

The repository has four Vue 3 + Vite apps with shared patterns but different user contracts:

- `hr-frontend` serves staff, recruiting, admin, AI, analytics, and collaboration workflows.
- `user-frontend` serves candidates, public job browsing, profile, resume upload, applications, interviews, offers, and candidate AI.
- `interviewer-frontend` serves staff interviewers with assigned interviews, feedback, notifications, and profile.
- `platform-frontend` serves platform administrators, operators, and auditors with tenant governance, platform-account RBAC, audit, plan/version, subscription, entitlement, usage, quota-alert, and global AI configuration/release workflows.

Each app keeps its own `src/router`, `src/stores`, `src/api`, `src/types`, `src/components`, `src/views`, `src/utils`, and app-level styles. Deliberately shared components, types, utilities, and brand assets live under `packages/shared/src/` and are imported through the configured `@shared/*` alias. Do not import source directly from another frontend app; keep behavior- or permission-specific wiring local even when a lower-level primitive is shared.

User-visible instants use `packages/shared/src/utils/format.ts`: formatting is pinned to `Asia/Shanghai`, datetime-local wall clocks serialize with `+08:00`, Unix seconds/milliseconds remain absolute instants, and pure calendar dates are not timezone-shifted. Frontend code must not rely on the browser's local timezone for plan, interview, order, audit, or quota-reset displays.

## Routing and Auth

- HR routing is permission-based. Routes use `meta.requiresAuth` and `meta.requiresPermission`, and the auth store exposes roles and permissions from `/auth/me`.
- Candidate routing uses candidate-role checks for authenticated candidate surfaces and keeps public job list/detail routes open.
- Interviewer routing restores session, sets document titles, and rejects non-interviewer staff by logging out and redirecting.
- Platform routing is permission-based, isolates the `platform` account type and client app, and selects the first allowed control-plane route for restricted operator or auditor roles.
- All four apps rely on httpOnly auth cookies and local cached user metadata for route decisions. Route guards can restore session before redirecting to login.

## Request Wrappers

Each app has an Axios wrapper with:

- `withCredentials: true`;
- `X-Client-App` header identifying `hr`, `candidate`, or `interviewer`;
- business-code unwrapping so API helpers receive `data` directly;
- friendly error mapping and request ID display;
- silent refresh with a queue for concurrent 401 failures.

HR request handling also records richer debug metadata and supports `silentError` for flows that intentionally probe for missing resources.

## API and Types

API helpers live under each app's `src/api`. Types live under `src/types` and should reflect gateway JSON payloads after the response interceptor unwraps `data`. When an API response shape changes, update the relevant helper and type together, then check the consuming views.

## Notifications

HR and candidate notification bells combine REST summary/list calls with SSE streaming, polling fallback, duplicate suppression, and desktop notifications. Candidate adds an initial unread summary. Interviewer notification bell currently polls unread count and navigates to the notification page.

If notification event shape, link routing, or unread-count semantics changes, review all three app surfaces rather than assuming the components are identical.

## Operational Admin Consoles

Platform AI configuration pages use the `admin-console` component family for headers, filter toolbars, table cards, stats, row actions, empty states, and status tags. HR retains tenant business administration pages, but technical AI configuration is not an HR concern. New operational pages should preserve existing page-header spacing, title/description treatment, and primary/secondary action placement before adding page-specific variants.

## Review Triggers

- Route meta, auth restore, role/permission checks, or client app header changes.
- Request wrapper response shape, refresh queue, error mapping, or timeout behavior.
- API helper/type changes that affect gateway contracts.
- Notification polling/SSE/link behavior.
- HR business-admin or platform AI admin menu, route, or admin-console layout changes.
- Platform console navigation, platform permission alignment, tenant governance, plan/subscription, or quota operations changes.
- Shared package changes, which require checking every consuming app rather than only the file's nearest frontend.

## Verification

Verified against the pnpm workspace, shared-package aliases and consumers, current routers, auth stores, request wrappers, notification bells, migrated platform admin-console components, and representative HR/candidate/interviewer/platform views on 2026-07-20.
