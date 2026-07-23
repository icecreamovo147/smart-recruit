---
schema_version: 1
id: frontend-validation
title: Frontend validation runbook
kind: runbook
status: active
owners:
  - frontend-platform
tags:
  - frontend
  - validation
  - vue
  - typecheck
applies_to:
  - hr-frontend/src/**
  - user-frontend/src/**
  - platform-frontend/src/**
  - packages/shared/src/**
source_refs:
  - AGENTS.md
  - pnpm-workspace.yaml
  - interviewer-frontend/README.md
  - hr-frontend/tsconfig.json
  - user-frontend/tsconfig.json
  - platform-frontend/tsconfig.json
  - hr-frontend/src/router/index.ts
  - user-frontend/src/router/index.ts
  - platform-frontend/src/router/index.ts
  - hr-frontend/src/api/request.ts
  - user-frontend/src/api/request.ts
  - platform-frontend/src/api/http.ts
  - hr-frontend/src/stores/auth.ts
  - user-frontend/src/stores/auth.ts
  - platform-frontend/src/stores/auth.ts
  - packages/shared/src/components/EmailSetupDialog.vue
  - packages/shared/src/types/domain.ts
  - packages/shared/src/utils/token.ts
last_verified: 2026-07-23
review_after: 2026-10-21
---

# Frontend Validation Runbook

Use this runbook after frontend route, API, type, store, component, view, notification, or admin-console changes.

## 1. Identify Touched Apps

Map changed files to active apps:

- `hr-frontend` for staff, admin, AI, analytics, recruitment operations, collaboration, and interviewer workflows under `/hr/my-interviews`.
- `user-frontend` for candidate job search, profile, resume, applications, interviews, offers, and candidate AI.
- `platform-frontend` for platform console, tenant governance, plan/subscription, and global AI configuration.
- `packages/shared` for deliberately cross-app components, types, utilities, and assets. Treat a shared-package change as touching every active app that imports the changed surface.

`interviewer-frontend/` is a legacy rollback tree outside the pnpm workspace. Do not treat `pnpm --filter interviewer-frontend ...` as a successful validation command; it reports `No projects matched` with exit code 0 and runs no checks.

Validate only touched apps unless a shared contract changes. For `packages/shared`, use import search to identify consumers and run typecheck/tests for all affected active apps; when a shared type, utility, component, or asset is consumed by all three active apps, validate all three.

## 2. Route and Permission Checks

- Compare route meta with auth store capability checks.
- For HR pages, compare frontend `requiresPermission` with gateway route permission in `smart-recruit-gateway/router/router.go`.
- For candidate pages, confirm public vs `requiresCandidate` route intent.
- For interviewer workbench pages in HR, confirm the `/hr/my-interviews` routes and any compatibility redirects remain aligned with notification deep links.
- For platform pages, confirm `platform` account type / `X-Client-App: platform` isolation and permission-based first-route selection.
- Confirm redirects preserve intended query or deep-link behavior.

## 3. API and Type Checks

- Confirm API helper path, method, payload, and return type match the gateway handler response.
- Remember request interceptors unwrap `response.data.data`.
- Confirm business-code errors, request IDs, quota reset metadata, and `silentError` behavior where applicable.
- If protobuf JSON or gateway response fields change, update TypeScript types and consuming views together.

## 4. UI Checks

- For platform operational pages, reuse `admin-console` layout components where possible.
- For HR admin pages, keep page-header spacing and action placement consistent with existing admin pages.
- Check text fit in compact controls, toolbars, table actions, and responsive headers.
- For notification changes, check REST summary/list, SSE stream, polling fallback, duplicate suppression, read state, and links on the touched active apps.
- For upload or AI flows, check loading states, disabled states, timeout messages, and quota errors.

## 5. Commands

Run the relevant commands from the repository root:

```sh
pnpm --filter hr-frontend typecheck
pnpm --filter user-frontend typecheck
pnpm --filter platform-frontend typecheck
```

For frontend tests in a touched app:

```sh
pnpm --filter hr-frontend test
pnpm --filter user-frontend test
pnpm --filter platform-frontend test
```

Use the specific app command that matches the touched files. If a TASK only changes documentation or knowledge, record that app typechecks were not required.

## Evidence to Record

Record touched app, route paths, permission keys or role guard, API helper names, type names, validation commands, and result. For UI work, include screenshots only when the TASK changes visual behavior.

## Verification

Verified against the current pnpm workspace, `@shared/*` aliases and consumers, active frontend routers, request wrappers, auth stores, and `interviewer-frontend/README.md` on 2026-07-23.
