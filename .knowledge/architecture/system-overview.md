---
schema_version: 1
id: system-overview
title: Smart Recruit system overview
kind: architecture
status: active
owners:
  - engineering-platform
tags:
  - architecture
  - services
  - frontend
  - backend
applies_to:
  - dev-log-viewer/**
  - homepage/**
  - hr-frontend/**
  - user-frontend/**
  - platform-frontend/**
  - packages/shared/**
  - smart-recruit-gateway/**
  - smart-recruit-*-service/**
  - smart-recruit-commons/**
  - smart-recruit-platform-go/**
  - smart-recruit-proto/**
  - smart-recruit-deploy/**
  - deploy/**
  - docker/**
source_refs:
  - README.md
  - pnpm-workspace.yaml
  - homepage/package.json
  - homepage/src/App.vue
  - .github/workflows/deploy-homepage.yml
  - hr-frontend/tsconfig.json
  - platform-frontend/tsconfig.json
  - packages/shared/src/components/EmailSetupDialog.vue
  - packages/shared/src/types/domain.ts
  - interviewer-frontend/README.md
  - start-dev.sh
  - stop-dev.sh
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/rpc/client.go
  - smart-recruit-gateway/config/config.go
  - smart-recruit-billing-service/cmd/billing-service/main.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - smart-recruit-platform-go/servicebinary/convention.go
  - smart-recruit-platform-go/i18n/i18n.go
  - packages/shared/src/i18n/index.ts
  - smart-recruit-commons/internal/platform/events/envelope.go
  - docker/docker-compose.yml
  - deploy/k8s/README-service-binaries.md
  - smart-recruit-deploy/README.md
  - smart-recruit-deploy/docker-compose.microservices.yml
  - dev-log-viewer/README.md
  - dev-log-viewer/cmd/dev-log-viewer/main.go
  - dev-log-viewer/internal/server/server.go
last_verified: 2026-07-27
review_after: 2026-10-21
---

# Smart Recruit System Overview

Smart Recruit has three active authenticated Vue product frontends (`hr-frontend`, `user-frontend`, `platform-frontend`), one independent public Vue marketing site (`homepage`), an explicit `packages/shared/` frontend package, a Gin gateway, independent Go services, shared protobuf contracts, shared platform utilities, and shared Commons packages. The gateway owns HTTP routing, middleware, auth/RBAC enforcement, request limits, SSE endpoints, and generated gRPC clients. Tenant quota definitions are owned by Identity, while enforcement is shared through Commons and invoked at each owning service's write boundary.

`homepage/` is a static, API-free bilingual project site. It consumes shared brand assets through `@shared/*`, builds independently in the pnpm workspace, and is deployed from `main` to GitHub Pages at `https://recruit.jkghjk123.site`. It is not part of `start-dev.sh`, Docker Compose, product authentication, or the Gateway runtime.

`interviewer-frontend/` is retained temporarily as a read-only rollback comparison tree. It is not part of the pnpm workspace, Docker Compose, or `start-dev.sh`/`stop-dev.sh` targets. Interviewer workflows now live in `hr-frontend` under `/hr/my-interviews`, and port `5175` is assigned to `platform-frontend`.

Backend responsibilities are split across Identity, Recruitment, Interview, Offer, Notification, AI Agent, Analytics, Billing, and Worker services. Billing owns commercial entitlements, grants, reservations, Alipay payment lifecycle, rate cards, usage events, and ledger mutations; the Gateway dials it on `BILLING_GRPC_ADDR` (default `127.0.0.1:50069`), and AI Agent settles actual token usage through the billing settlement outbox. Shared protocol definitions live in `smart-recruit-proto/`; shared config, Nacos, gRPC, health, trace, metrics, metadata, and service binary conventions live in `smart-recruit-platform-go/`; shared migrations, MQ, OSS, email, authz/JWT helpers, resume parser, event envelope, and AI support live in `smart-recruit-commons/`.

`dev-log-viewer/` is an independent local development utility. It is both a Go module and pnpm workspace package, serves a React UI, read-only service metadata API, and SSE log stream from `127.0.0.1:8090`, and is started only through the explicit `logs` or `log-viewer` target in `start-dev.sh`. It is not part of the default or `all` development stack.

Frontend apps keep app-specific behavior under their own roots and import deliberate cross-app components, types, utilities, and brand assets from `packages/shared/src/` through `@shared/*`. Deployment assets are split by purpose: `docker/` contains the local/full-stack Compose setup and frontend images, `deploy/k8s/` contains Kubernetes manifests, and `smart-recruit-deploy/` contains microservice images, composition, observability, Nacos seed configuration, and table ownership.

Backend logs and system-facing response messages share one deployment locale
through `APP_LOCALE`. Canonical catalogs and rendering live in
`smart-recruit-platform-go/i18n`; generated frontend catalogs and runtime
initialization live in `packages/shared/src/i18n`. Gateway exposes the active
locale through a public runtime-config endpoint, so the three product frontends
and Element Plus follow a backend locale switch after refresh without rebuilding
frontend images.

## Verification

Verified against `pnpm-workspace.yaml`, `start-dev.sh`, the platform i18n
package, shared frontend i18n runtime, Gateway runtime-config route, deployment
locale declarations, and microservice Compose on 2026-07-27.
