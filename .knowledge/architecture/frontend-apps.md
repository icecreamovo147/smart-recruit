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
source_refs:
  - hr-frontend/src/router/index.ts
  - user-frontend/src/router/index.ts
  - interviewer-frontend/src/router/index.ts
  - hr-frontend/src/stores/auth.ts
  - user-frontend/src/stores/auth.ts
  - interviewer-frontend/src/stores/auth.ts
  - hr-frontend/src/api/request.ts
  - user-frontend/src/api/request.ts
  - interviewer-frontend/src/api/request.ts
  - hr-frontend/src/components/NotificationBell.vue
  - user-frontend/src/components/NotificationBell.vue
  - interviewer-frontend/src/components/NotificationBell.vue
  - hr-frontend/src/components/admin-console/PageHeader.vue
  - hr-frontend/src/components/admin-console/DataTableCard.vue
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Frontend App Architecture

The repository has three Vue 3 + Vite apps with shared patterns but different user contracts:

- `hr-frontend` serves staff, recruiting, admin, AI, analytics, and collaboration workflows.
- `user-frontend` serves candidates, public job browsing, profile, resume upload, applications, interviews, offers, and candidate AI.
- `interviewer-frontend` serves staff interviewers with assigned interviews, feedback, notifications, and profile.

Each app keeps its own `src/router`, `src/stores`, `src/api`, `src/types`, `src/components`, `src/views`, `src/utils`, and app-level styles. Avoid creating cross-app hidden dependencies; duplicate small app-specific wiring when behavior or permissions differ.

## Routing and Auth

- HR routing is permission-based. Routes use `meta.requiresAuth` and `meta.requiresPermission`, and the auth store exposes roles and permissions from `/auth/me`.
- Candidate routing uses candidate-role checks for authenticated candidate surfaces and keeps public job list/detail routes open.
- Interviewer routing restores session, sets document titles, and rejects non-interviewer staff by logging out and redirecting.
- All three apps rely on httpOnly auth cookies and local cached user metadata for route decisions. Route guards can restore session before redirecting to login.

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

## HR Admin Console

HR admin and AI configuration pages use the `admin-console` component family for headers, filter toolbars, table cards, stats, row actions, empty states, and status tags. New HR left-menu admin pages should preserve existing page-header spacing, title/description treatment, and primary/secondary action placement before adding page-specific variants.

## Review Triggers

- Route meta, auth restore, role/permission checks, or client app header changes.
- Request wrapper response shape, refresh queue, error mapping, or timeout behavior.
- API helper/type changes that affect gateway contracts.
- Notification polling/SSE/link behavior.
- HR admin menu, route, or admin-console layout changes.

## Verification

Verified against current routers, auth stores, request wrappers, notification bells, admin-console components, and representative HR/candidate/interviewer views on 2026-07-10.
