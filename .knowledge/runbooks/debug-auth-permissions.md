---
schema_version: 1
id: debug-auth-permissions
title: Debug auth and permission failures
kind: runbook
status: active
owners:
  - engineering-platform
tags:
  - auth
  - rbac
  - debug
  - permissions
applies_to:
  - web-gin-service/middleware/jwt.go
  - web-gin-service/middleware/role.go
  - web-gin-service/router/router.go
  - logic-grpc-service/service/auth_service.go
  - logic-grpc-service/repository/authz_repo.go
source_refs:
  - web-gin-service/middleware/jwt.go
  - web-gin-service/middleware/role.go
  - web-gin-service/router/router.go
  - web-gin-service/handler/auth.go
  - logic-grpc-service/service/auth_service.go
  - logic-grpc-service/repository/authz_repo.go
  - hr-frontend/src/router/index.ts
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Debug Auth and Permission Failures

Use this runbook when users can log in but receive `401`, `403`, stale-permission, missing-menu, or wrong-frontend-cookie behavior.

## Diagnostic Path

1. Identify the client app from `X-Client-App` and expected cookie namespace.
2. Check whether the failure is authentication (`401`) or authorization (`403`).
3. For `401`, inspect access-cookie selection, JWT validity, `token_version`, Redis availability, and `ValidateCurrentPrincipal` behavior.
4. For `403`, inspect the route's `RequirePermission`, `RequireAnyPermission`, or `RequireAnyRole` declaration.
5. Compare the route permission key with seeded RBAC permissions and frontend route metadata.
6. If permissions recently changed, check that `token_version` was incremented and synchronized to Redis or safely invalidated.
7. For admin role/data-scope changes, inspect `AdminService` mutation paths and `AuthzRepo` role/data-scope methods.
8. For audit questions, check denied-decision emission in gateway middleware and authorization audit query paths.

## Common Evidence Anchors

- Gateway cookie/auth handling: `web-gin-service/handler/auth.go`
- JWT principal and token-version checks: `web-gin-service/middleware/jwt.go`
- Permission middleware: `web-gin-service/middleware/role.go`
- Route declarations: `web-gin-service/router/router.go`
- Login, refresh, and principal loading: `logic-grpc-service/service/auth_service.go`
- Roles, permissions, data scopes, and audit logs: `logic-grpc-service/repository/authz_repo.go`
- HR route guards: `hr-frontend/src/router/index.ts`

## Expected Failure Clues

- `401` with stale permission wording usually points to token-version mismatch.
- `403` with `required_permission` means authentication succeeded but route permission failed.
- Missing HR menu entries can be a frontend permission metadata issue even when backend access works.
- A Redis cache miss for token version is fail-closed; Redis transport errors degrade revocation enforcement but still verify JWT signatures.

## Safety

Do not paste live tokens, cookies, user passwords, or raw production audit logs into knowledge reports. Use user IDs, role keys, and permission keys only when they are sanitized and task-relevant.
