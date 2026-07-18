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
  - interviewer-frontend/src/**
  - packages/shared/src/**
source_refs:
  - AGENTS.md
  - pnpm-workspace.yaml
  - hr-frontend/tsconfig.json
  - user-frontend/tsconfig.json
  - interviewer-frontend/tsconfig.json
  - hr-frontend/src/router/index.ts
  - user-frontend/src/router/index.ts
  - interviewer-frontend/src/router/index.ts
  - hr-frontend/src/api/request.ts
  - user-frontend/src/api/request.ts
  - interviewer-frontend/src/api/request.ts
  - hr-frontend/src/stores/auth.ts
  - user-frontend/src/stores/auth.ts
  - interviewer-frontend/src/stores/auth.ts
  - packages/shared/src/components/EmailSetupDialog.vue
  - packages/shared/src/types/domain.ts
  - packages/shared/src/utils/token.ts
last_verified: 2026-07-19
review_after: 2026-10-08
---

# Frontend Validation Runbook

Use this runbook after frontend route, API, type, store, component, view, notification, or admin-console changes.

## 1. Identify Touched Apps

Map changed files to apps:

- `hr-frontend` for staff, admin, AI, analytics, recruitment operations, and collaboration.
- `user-frontend` for candidate job search, profile, resume, applications, interviews, offers, and candidate AI.
- `interviewer-frontend` for interviewer workbench, interview detail, feedback, notifications, and profile.
- `packages/shared` for deliberately cross-app components, types, utilities, and assets. Treat a shared-package change as touching every app that imports the changed surface.

Validate only touched apps unless a shared contract changes. For `packages/shared`, use import search to identify consumers and run typecheck/tests for all affected apps; when a shared type, utility, component, or asset is consumed by all three apps, validate all three.

## 2. Route and Permission Checks

- Compare route meta with auth store capability checks.
- For HR pages, compare frontend `requiresPermission` with gateway route permission in `smart-recruit-gateway/router/router.go`.
- For candidate pages, confirm public vs `requiresCandidate` route intent.
- For interviewer pages, confirm `isInterviewer` remains the guard for protected routes.
- Confirm redirects preserve intended query or deep-link behavior.

## 3. API and Type Checks

- Confirm API helper path, method, payload, and return type match the gateway handler response.
- Remember request interceptors unwrap `response.data.data`.
- Confirm business-code errors, request IDs, quota reset metadata, and `silentError` behavior where applicable.
- If protobuf JSON or gateway response fields change, update TypeScript types and consuming views together.

## 4. UI Checks

- For HR admin pages, reuse `admin-console` layout components where possible.
- Check text fit in compact controls, toolbars, table actions, and responsive headers.
- For notification changes, check REST summary/list, SSE stream, polling fallback, duplicate suppression, read state, and links.
- For upload or AI flows, check loading states, disabled states, timeout messages, and quota errors.

## 5. Commands

Run the relevant commands from the repository root:

```sh
pnpm --filter hr-frontend typecheck
pnpm --filter user-frontend typecheck
pnpm --filter interviewer-frontend typecheck
```

For frontend tests in a touched app:

```sh
pnpm --filter hr-frontend test
pnpm --filter user-frontend test
pnpm --filter interviewer-frontend test
```

Use the specific app command that matches the touched files. If a TASK only changes documentation or knowledge, record that app typechecks were not required.

## Evidence to Record

Record touched app, route paths, permission keys or role guard, API helper names, type names, validation commands, and result. For UI work, include screenshots only when the TASK changes visual behavior.

## Verification

Verified against the current pnpm workspace, `@shared/*` aliases and consumers, frontend routers, request wrappers, and auth stores on 2026-07-19.
