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
  - smart-recruit-gateway/middleware/jwt.go
  - smart-recruit-gateway/middleware/role.go
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/handler/auth.go
  - smart-recruit-identity-service/**
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000012_add_rbac_schema.sql
source_refs:
  - smart-recruit-gateway/middleware/jwt.go
  - smart-recruit-gateway/middleware/role.go
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/handler/auth.go
  - smart-recruit-gateway/rpc/client.go
  - smart-recruit-identity-service/internal/application/service/auth_service.go
  - smart-recruit-identity-service/internal/application/service/admin_service.go
  - smart-recruit-identity-service/internal/infrastructure/persistence/identity_repository.go
  - smart-recruit-identity-service/internal/interfaces/grpc/identity_server.go
  - smart-recruit-identity-service/internal/runtime/runtime.go
  - smart-recruit-platform-go/server/interceptor.go
  - smart-recruit-platform-go/server/transport_security.go
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000011_add_refresh_tokens.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000012_add_rbac_schema.sql
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000013_seed_rbac_catalog.sql
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Auth, RBAC, and Security Audit Architecture

The gateway owns cookies, JWT parsing, route-level role and permission checks, denied-decision audit emission, and actor metadata forwarding. Identity owns registration, login, refresh-token rotation, principal loading, RBAC catalog, role/data-scope management, token-version invalidation, and security audit query/persistence.

Service-to-service authorization should use Identity owner contracts rather than local copies of authz repositories. `AuthService.GetPrincipal` returns the actor principal, permissions, token version, and data scopes. `AuthService.AuthorizeInternal` evaluates an optional permission key and optional required scope/resource tuple, returns the principal plus matched scopes, and records an allow/deny audit decision.

Permission changes must keep gateway route declarations, authz constants, RBAC seed migrations, frontend route metadata, Identity behavior, and audit logging aligned.

## Verification

Verified against current repository files on 2026-07-14.
