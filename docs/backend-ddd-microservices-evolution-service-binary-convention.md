# Backend DDD Microservices Evolution Service Binary Convention

Last verified: 2026-07-12

This document establishes the monorepo convention for independently deployable backend service binaries. It does not route traffic, change public APIs, change database ownership, or alter the current Kubernetes deployments.

## Scope

The current production shape remains:

- `web-gin-service` as the HTTP gateway.
- `logic-grpc-service` as the gRPC monolith backend.
- `logic-grpc-service --worker-only` as the transitional worker deployment.

Future extracted services must follow this convention before receiving gateway traffic or worker ownership.

## Service Unit Registry

The compile-checked registry is in `logic-grpc-service/internal/platform/servicebinary/convention.go`. It records service unit names, roles, command paths, image names, config prefixes, health expectations, and cutover notes.

Required backend units:

| Unit | Role | Command convention | Image convention |
|---|---|---|---|
| `api-gateway` | gateway | `web-gin-service` | `recruitment/web-gin-service` |
| `logic-grpc-service` | service | `logic-grpc-service` | `recruitment/logic-grpc-service` |
| `logic-worker` | worker | `logic-grpc-service --worker-only` | `recruitment/logic-grpc-service` |
| `identity-service` | service | `logic-grpc-service/cmd/identity-service` | `recruitment/identity-service` |
| `recruitment-service` | service | `logic-grpc-service/cmd/recruitment-service` | `recruitment/recruitment-service` |
| `interview-service` | service | `logic-grpc-service/cmd/interview-service` | `recruitment/interview-service` |
| `offer-service` | service | `logic-grpc-service/cmd/offer-service` | `recruitment/offer-service` |
| `notification-service` | service | `logic-grpc-service/cmd/notification-service` | `recruitment/notification-service` |
| `ai-agent-service` | service | `logic-grpc-service/cmd/ai-agent-service` | `recruitment/ai-agent-service` |
| `analytics-service` | service | `logic-grpc-service/cmd/analytics-service` | `recruitment/analytics-service` |
| `worker-services` | worker | `logic-grpc-service/cmd/worker-services` | `recruitment/worker-services` |

## Binary Rules

- Extracted service binaries live under `logic-grpc-service/cmd/<unit-name>/main.go` unless a later SPEC revision moves service source roots.
- Shared backend platform contracts stay under `logic-grpc-service/internal/platform/`.
- A service binary must not import another bounded context's repository package unless the TASK documents an approved transitional adapter.
- A binary may start in shadow mode or dual-run mode, but it must not receive production traffic until a scoped cutover TASK records rollback and verification evidence.
- Package manifests, `go.mod`, and `go.sum` remain unchanged unless a scoped TASK explicitly allows dependency changes.

## Runtime Contract

Each deployable unit must define:

- build target and image name;
- command and args;
- required config prefix and secret sources;
- health and readiness behavior;
- structured logging and request or trace correlation;
- metrics and tracing expectations;
- owner table or projection model;
- rollback plan and cutover guard.

Current transitional behavior:

- `logic-grpc-service` serves gRPC health on port `50051`.
- `logic-worker` uses the same image and command with `--worker-only`.
- Existing manifests in `deploy/k8s/logic-deployment.yaml` and `deploy/k8s/worker-deployment.yaml` remain the active deployment examples.

## Deployment Metadata Convention

Future manifests should include stable labels:

```yaml
app.kubernetes.io/part-of: smart-recruit
app.kubernetes.io/component: backend
backend.smart-recruit/unit: <unit-name>
backend.smart-recruit/role: gateway|service|worker
backend.smart-recruit/cutover-mode: shadow|dual-run|dual-read|routed|none
```

Use `cutover-mode: none` for compiled but unrouted service skeletons.

Current compiled skeletons:

- `notification-service`: compile-safe descriptor and command only; unrouted by default.
- `ai-agent-service`: compile-safe descriptor and command only; unrouted by default.
- `recruitment-service`: compile-safe descriptor and command only; unrouted by default.
- `interview-service`: compile-safe descriptor and command only; unrouted by default; gateway routing can be explicitly enabled with `INTERVIEW_ROUTE_MODE=interview` and rolled back with `INTERVIEW_ROUTE_MODE=logic`.
- `offer-service`: compile-safe descriptor and command only; unrouted by default.

## Cutover Guardrails

- Notification and AI Agent binaries start with shadow or dual-run validation.
- Identity cutover must preserve login, refresh, token invalidation, RBAC, scopes, and audit behavior.
- Recruitment, Interview, and Offer cutovers must preserve all lifecycle transitions and user-visible state.
- Analytics service reads must use owned projection read models and must not call transactional service read APIs as a transition path.
- Gateway routing to extracted services is allowed only in scoped cutover TASKs.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./internal/platform/servicebinary ./...
```

The focused package test verifies that the registry is complete, unique, and copy-safe. The broad test ensures the convention does not break existing backend packages.
