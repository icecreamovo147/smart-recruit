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
  - smart-recruit-gateway/middleware/jwt.go
  - smart-recruit-gateway/middleware/role.go
  - smart-recruit-gateway/router/router.go
  - smart-recruit-identity-service/**
  - hr-frontend/src/router/index.ts
source_refs:
  - smart-recruit-gateway/middleware/jwt.go
  - smart-recruit-gateway/middleware/role.go
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/handler/auth.go
  - smart-recruit-identity-service/internal/application/service/auth_service.go
  - smart-recruit-identity-service/internal/application/service/admin_service.go
  - smart-recruit-identity-service/internal/infrastructure/persistence/identity_repository.go
  - hr-frontend/src/router/index.ts
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Debug Auth and Permission Failures

Check cookie/JWT issuance, gateway JWT parsing, current-principal refresh from Identity, route role/permission declarations, seeded permission catalog, user roles/data scopes, token-version cache invalidation, frontend route metadata, and audit records.

## Verification

Verified against current repository files on 2026-07-14.
