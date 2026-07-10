---
schema_version: 1
id: auth-permission-alignment
title: Auth permission alignment pitfall
kind: pitfall
status: active
owners:
  - engineering-platform
tags:
  - auth
  - rbac
  - frontend
  - audit
applies_to:
  - web-gin-service/router/router.go
  - web-gin-service/pkg/authz/**
  - logic-grpc-service/pkg/authz/**
  - logic-grpc-service/migrations/000013_seed_rbac_catalog.sql
  - hr-frontend/src/router/index.ts
source_refs:
  - web-gin-service/router/router.go
  - web-gin-service/middleware/role.go
  - logic-grpc-service/repository/authz_repo.go
  - logic-grpc-service/migrations/000013_seed_rbac_catalog.sql
  - hr-frontend/src/router/index.ts
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Auth Permission Alignment Pitfall

Permission work spans seed data, backend constants, gateway route declarations, frontend route guards, token-version invalidation, and audit behavior. Updating only one layer can produce false positives in local UI tests or route tests.

## Trigger Conditions

- Adding or renaming a permission key.
- Moving a route to a different permission.
- Adding an HR admin page or menu entry.
- Changing role assignments or data-scope mutation behavior.
- Changing staff-user, role, permission, or security-audit pages.

## Risk

The UI may hide a page that the backend allows, or show a page that the backend rejects. Existing access JWTs may retain stale permissions if token-version invalidation is not synchronized. Audit logs can also become misleading if route permission keys and recorded decision context drift.

## Prevention

- Check seeded permission catalogs and backend authz constants together.
- Check `web-gin-service/router/router.go` route middleware and HR frontend route metadata together.
- Check `AuthzRepo` role/data-scope mutations and token-version increments when permissions change.
- Check denied-decision audit behavior for new protected routes.
- Keep authorization changes out of incidental UI-only TASKs unless explicitly scoped.

## Verification

This pitfall was verified from route middleware, authz repository, RBAC seed migration, and HR router metadata on 2026-07-10.
