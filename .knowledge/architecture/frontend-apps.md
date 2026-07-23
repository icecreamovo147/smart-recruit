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
  - homepage/src/**
  - hr-frontend/src/**
  - user-frontend/src/**
  - platform-frontend/src/**
  - packages/shared/src/**
source_refs:
  - pnpm-workspace.yaml
  - homepage/package.json
  - homepage/src/App.vue
  - homepage/vite.config.ts
  - .github/workflows/deploy-homepage.yml
  - interviewer-frontend/README.md
  - hr-frontend/tsconfig.json
  - user-frontend/tsconfig.json
  - platform-frontend/tsconfig.json
  - hr-frontend/src/router/index.ts
  - user-frontend/src/router/index.ts
  - platform-frontend/src/router/index.ts
  - hr-frontend/src/stores/auth.ts
  - user-frontend/src/stores/auth.ts
  - platform-frontend/src/stores/auth.ts
  - hr-frontend/src/api/request.ts
  - user-frontend/src/api/request.ts
  - platform-frontend/src/api/http.ts
  - hr-frontend/src/components/NotificationBell.vue
  - user-frontend/src/components/NotificationBell.vue
  - platform-frontend/src/components/admin-console/PageHeader.vue
  - platform-frontend/src/components/admin-console/DataTableCard.vue
  - packages/shared/src/components/EmailSetupDialog.vue
  - packages/shared/src/types/domain.ts
  - packages/shared/src/utils/token.ts
  - packages/shared/src/utils/format.ts
  - start-dev.sh
last_verified: 2026-07-23
review_after: 2026-10-21
---

# Frontend App Architecture

The repository has three active authenticated Vue 3 + Vite product apps with shared patterns but different user contracts:

- `hr-frontend` (port `5173`, `X-Client-App: hr`) serves staff, recruiting, admin, AI, analytics, collaboration, and interviewer workflows under `/hr/my-interviews`.
- `user-frontend` (port `5174`, `X-Client-App: candidate`) serves candidates, public job browsing, profile, resume upload, applications, interviews, offers, and candidate AI.
- `platform-frontend` (port `5175`, `X-Client-App: platform`) serves platform administrators, operators, and auditors with tenant governance, platform-account RBAC, audit, plan/version, subscription, entitlement, usage, quota-alert, and global AI configuration/release workflows.

`homepage/` is a separate public Vue 3 + Vite marketing site for product and open-source discovery. It has no router, auth state, API client, `X-Client-App` header, or backend dependency. It provides typed Chinese/English content, persisted light/dark theme selection, stable in-page anchors, and GitHub/documentation links. It is deployed through GitHub Actions to GitHub Pages at `https://recruit.jkghjk123.site`.

`interviewer-frontend/` is a legacy source tree retained temporarily for rollback comparison only. It is not in `pnpm-workspace.yaml`, Docker, or local-dev startup targets. Do not add features there; do not use root `pnpm --filter interviewer-frontend` commands as validation evidence.

Each authenticated product app keeps its own `src/router`, `src/stores`, `src/api`, `src/types`, `src/components`, `src/views`, `src/utils`, and app-level styles. The public homepage stays API-free and owns its marketing content and generated imagery locally. Deliberately shared components, types, utilities, and brand assets live under `packages/shared/src/` and are imported through the configured `@shared/*` alias. Do not import source directly from another frontend app; keep behavior- or permission-specific wiring local even when a lower-level primitive is shared.

User-visible instants use `packages/shared/src/utils/format.ts`: formatting is pinned to `Asia/Shanghai`, datetime-local wall clocks serialize with `+08:00`, Unix seconds/milliseconds remain absolute instants, and pure calendar dates are not timezone-shifted. Frontend code must not rely on the browser's local timezone for plan, interview, order, audit, or quota-reset displays.

## Routing and Auth

- HR routing is permission-based. Routes use `meta.requiresAuth` and `meta.requiresPermission`, and the auth store exposes roles and permissions from `/auth/me`. Interviewer workbench routes live in HR and keep compatibility redirects from the former interviewer deep links.
- Candidate routing uses candidate-role checks for authenticated candidate surfaces and keeps public job list/detail routes open.
- Platform routing is permission-based, isolates the `platform` account type and client app, and selects the first allowed control-plane route for restricted operator or auditor roles.
- All three active apps rely on httpOnly auth cookies and local cached user metadata for route decisions. Route guards can restore session before redirecting to login.

## Request Wrappers

Each active app has an Axios wrapper with:

- `withCredentials: true`;
- `X-Client-App` header identifying `hr`, `candidate`, or `platform`;
- business-code unwrapping so API helpers receive `data` directly;
- friendly error mapping and request ID display;
- silent refresh with a queue for concurrent 401 failures.

HR request handling also records richer debug metadata and supports `silentError` for flows that intentionally probe for missing resources.

## API and Types

API helpers live under each app's `src/api`. Types live under `src/types` and should reflect gateway JSON payloads after the response interceptor unwraps `data`. When an API response shape changes, update the relevant helper and type together, then check the consuming views.

## Notifications

HR and candidate notification bells combine REST summary/list calls with SSE streaming, polling fallback, duplicate suppression, and desktop notifications. Candidate adds an initial unread summary.

If notification event shape, link routing, or unread-count semantics changes, review both active staff/candidate surfaces rather than assuming the components are identical.

## Operational Admin Consoles

Platform AI configuration pages use the `admin-console` component family for headers, filter toolbars, table cards, stats, row actions, empty states, and status tags. HR retains tenant business administration pages, but technical AI configuration is not an HR concern. New operational pages should preserve existing page-header spacing, title/description treatment, and primary/secondary action placement before adding page-specific variants.

## Review Triggers

- Route meta, auth restore, role/permission checks, or client app header changes.
- Request wrapper response shape, refresh queue, error mapping, or timeout behavior.
- API helper/type changes that affect gateway contracts.
- Notification polling/SSE/link behavior.
- HR business-admin or platform AI admin menu, route, or admin-console layout changes.
- Platform console navigation, platform permission alignment, tenant governance, plan/subscription, or quota operations changes.
- Shared package changes, which require checking every consuming active app rather than only the file's nearest frontend.
- Any proposal to revive or remove `interviewer-frontend/`.
- Homepage language/theme persistence, public links, stable anchors, responsive behavior, or Pages deployment changes.

## Verification

Verified against the pnpm workspace, homepage content and deployment boundaries, `start-dev.sh` ports, `X-Client-App` wrappers, shared-package aliases and consumers, current routers/auth stores, HR interviewer compatibility routes, and `interviewer-frontend/README.md` on 2026-07-23.
