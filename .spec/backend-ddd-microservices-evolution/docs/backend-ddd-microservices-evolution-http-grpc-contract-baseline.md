# Backend HTTP And gRPC Contract Baseline

本文档记录 `backend-ddd-microservices-evolution` 的 TASK-BDME-003 契约基线。该任务只新增测试和文档，不改变公开 HTTP 行为、gRPC proto 源文件或生成代码。

## Public HTTP Surface

当前公开 HTTP 入口由 `web-gin-service/router/router.go` 集中注册，统一挂载在 `/api/v1` 下，除健康检查和 Swagger 外。核心路由组如下：

- Health/readiness: `GET /health`, `GET /livez`, `GET /readyz`
- Auth: `/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/auth/logout`, `/api/v1/auth/refresh`, `/api/v1/auth/me`, `/api/v1/auth/email`
- Public jobs: `GET /api/v1/jobs`, `GET /api/v1/jobs/:job_id`
- Candidate workspace: `/api/v1/candidate/profile`, `/resume`, `/applications`, `/interviews`, `/offers`, `/notifications`, `/ai/*`
- Staff/HR workspace: `/api/v1/hr/jobs`, `/applications`, `/interviews`, `/offers`, `/notifications`, `/dashboard`, `/analytics`, `/collaboration`, `/ai/*`
- Admin: `/api/v1/hr/admin/*` for RBAC, taxonomy, prompt, LLM, embedding, MCP, skill, and agent-skill management

`web-gin-service/router/contract_baseline_test.go` asserts representative routes from every core group remain registered. This is a route-table baseline, not an end-to-end handler behavior test.

## Auth And RBAC Baseline

Current access control is layered:

- `JWTAuthByClient` validates client-specific cookies and token version.
- `ValidateCurrentPrincipal` refreshes roles, permissions, account type, and token version from logic gRPC.
- Candidate routes require `candidate` role plus explicit candidate permissions.
- Staff routes require one of `recruiter`, `recruiting_admin`, `system_admin`, or `interviewer`, then explicit route permissions.
- Admin routes require explicit admin/system/AI permissions; staff role alone is not enough.

Existing automated coverage:

- `web-gin-service/middleware/route_auth_test.go` covers candidate/staff/admin role and permission combinations, including deny paths.
- `web-gin-service/middleware/jwt_principal_test.go` covers current principal refresh and token-version rejection.
- Handler tests cover representative HTTP-to-gRPC request mapping for cursor pagination, AI agent runs, MCP, candidate applications, and related surfaces.

Documented gaps:

- Full end-to-end auth behavior for every route is not exhaustively tested because `router.Setup` wires live handler dependencies and gRPC clients.
- Route-table registration is covered for representative core groups rather than every single route.
- Public JSON response compatibility is covered by targeted handler tests and `go test ./...`, not by snapshot tests for every route.

## gRPC Contract Baseline

The current gateway-to-backend gRPC surface uses:

- Proto source: `logic-grpc-service/proto/recruitment.proto`
- Logic generated code: `logic-grpc-service/recruitment/pb/recruitment.pb.go`, `logic-grpc-service/recruitment/pb/recruitment_grpc.pb.go`
- Web generated copy: `web-gin-service/recruitment/pb/recruitment.pb.go`, `web-gin-service/recruitment/pb/recruitment_grpc.pb.go`

`web-gin-service/recruitment/pb/proto_sync_test.go` validates that the generated Go files in the gateway and logic service are byte-for-byte identical. This prevents the gateway from compiling against a different generated gRPC contract than the backend.

Documented gaps:

- Source-to-generated regeneration is not run in CI by this task because it would require protoc/toolchain availability and may modify generated files.
- The generated headers currently record `protoc-gen-go v1.36.11` and `protoc v4.25.3`; future proto changes should regenerate both service trees together and keep the generated copies synchronized.

## Compatibility Rule

No public route path, HTTP method, auth/RBAC requirement, proto message, service method, field number, or generated gRPC code should change during later extraction tasks unless the task explicitly scopes the contract change and records human confirmation.
