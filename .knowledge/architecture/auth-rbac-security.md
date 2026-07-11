---
schema_version: 1
id: auth-rbac-security
title: Auth, RBAC, and security audit architecture
kind: architecture
status: active
owners:
  - engineering-platform
tags:
  - auth
  - rbac
  - security
  - audit
applies_to:
  - web-gin-service/middleware/jwt.go
  - web-gin-service/middleware/role.go
  - web-gin-service/router/router.go
  - web-gin-service/handler/auth.go
  - logic-grpc-service/service/auth_service.go
  - logic-grpc-service/repository/authz_repo.go
  - logic-grpc-service/migrations/000012_add_rbac_schema.sql
source_refs:
  - web-gin-service/middleware/jwt.go
  - web-gin-service/middleware/role.go
  - web-gin-service/router/router.go
  - web-gin-service/handler/auth.go
  - logic-grpc-service/service/auth_service.go
  - logic-grpc-service/cmd/identity-service/main.go
  - logic-grpc-service/internal/identity/runtime/skeleton.go
  - logic-grpc-service/internal/identity/runtime/runtime.go
  - logic-grpc-service/internal/identity/interfaces/auth_server.go
  - logic-grpc-service/internal/identity/interfaces/admin_server.go
  - logic-grpc-service/service/admin_service.go
  - logic-grpc-service/repository/authz_repo.go
  - logic-grpc-service/migrations/000011_add_refresh_tokens.sql
  - logic-grpc-service/migrations/000012_add_rbac_schema.sql
  - logic-grpc-service/migrations/000013_seed_rbac_catalog.sql
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Auth, RBAC, and Security Audit Architecture

Authentication is split between the HTTP gateway and the logic service. The gateway owns cookie handling, access JWT parsing, route-level role and permission middleware, and forwarding the authenticated actor context. The logic service owns password validation, user creation, refresh-token persistence and rotation, RBAC catalogs, role assignments, data scopes, token-version invalidation, and authorization audit persistence.

## Boundary Model

- `web-gin-service/handler/auth.go` writes short-lived access JWT cookies and opaque refresh-token cookies. Cookie names are separated by client app: candidate, HR, and interviewer.
- `web-gin-service/middleware/jwt.go` validates access tokens, optionally checks `token_version` in Redis, and refreshes the principal from database-backed identity when `ValidateCurrentPrincipal` is used.
- `web-gin-service/middleware/role.go` enforces route role and permission requirements and emits authorization audit events for denied decisions.
- `web-gin-service/router/router.go` declares role and permission requirements close to HTTP routes.
- `logic-grpc-service/service/auth_service.go` owns registration, login, refresh-token rotation, revocation, and database-backed principal loading.
- `logic-grpc-service/repository/authz_repo.go` owns roles, permissions, role assignment, data scopes, authorization audit logs, legacy role migration, and token-version increments.
- `logic-grpc-service/service/admin_service.go` changes staff roles/data scopes and synchronizes token-version cache after permission mutations.
- `logic-grpc-service/cmd/identity-service` is currently an unrouted Identity runtime. It registers AuthService plus Identity-owned AdminService methods only when explicitly started with `--serve`; default execution still binds no listener, and gateway traffic remains on the monolith until a scoped cutover TASK.
- Identity-owned extracted AdminService methods are role/permission listing, user role assignment/revocation, data-scope assignment/revocation, staff identity list/create, and security audit log query. Invite-code, usage-log, department, location, and department-location configuration methods remain outside the Identity runtime in this TASK.

## Security-Relevant State

- Access identity is carried in JWT claims and then revalidated against current principal data.
- Refresh tokens are opaque values stored server-side by hash and rotated by the logic service.
- `token_version` invalidates stale access tokens after permission changes.
- RBAC role keys and permission keys are seeded in migrations and surfaced to gateway route middleware and frontend route guards.
- Data scopes are stored separately from permissions and are used by service-side authorization decisions.
- Authorization audit logs record allow/deny context for route or admin decisions.

## Impact Guidance

- Route permission changes must check gateway route declarations, `authz` constants, seeded permission catalogs, frontend route metadata, and logic-side assumptions.
- Staff role or data-scope changes must check token-version invalidation and Redis cache behavior.
- Cookie namespace changes must check all three frontend apps and refresh/logout behavior.
- Audit changes must check both gateway denied-decision logging and logic-side audit/query APIs.

## Verification

Verified against gateway JWT and RBAC middleware, auth handler cookie logic, route declarations, logic auth/admin services, the unrouted Identity service skeleton, authz repository, and RBAC/refresh-token migrations on 2026-07-12.
