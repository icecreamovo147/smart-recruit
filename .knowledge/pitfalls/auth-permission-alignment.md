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
  - permissions
applies_to:
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/middleware/role.go
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000013_seed_rbac_catalog.sql
  - smart-recruit-identity-service/**
  - hr-frontend/src/router/index.ts
source_refs:
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/middleware/role.go
  - smart-recruit-gateway/pkg/authz/permissions.go
  - smart-recruit-commons/migrations/archive/pre-baseline-000089/000013_seed_rbac_catalog.sql
  - smart-recruit-identity-service/internal/application/service/admin_service.go
  - hr-frontend/src/router/index.ts
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Auth Permission Alignment Pitfall

Permission keys must align across gateway route declarations, authz constants, RBAC seed migrations, frontend route metadata, Identity role/data-scope behavior, token-version invalidation, and audit logging. Frontend hiding is never authorization.

## Verification

Verified against current repository files on 2026-07-14.
