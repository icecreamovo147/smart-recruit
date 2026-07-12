# Backend Core Workflow Regression Baseline

本文档记录 `backend-ddd-microservices-evolution` 的 TASK-BDME-004 核心工作流回归基线。目标是保护现有行为，不引入未来微服务拆分后的新行为。

## Coverage Matrix

| Workflow | Current Behavior Anchor | Automated Coverage | Gap / Residual Risk |
| --- | --- | --- | --- |
| Auth, RBAC, token invalidation | Gateway JWT/current-principal middleware plus logic RBAC repositories/services | `web-gin-service/middleware/route_auth_test.go`, `web-gin-service/middleware/jwt_principal_test.go`, `logic-grpc-service/service/rbac_test.go`, `logic-grpc-service/repository/refresh_token_repo_test.go`, `logic-grpc-service/server/interceptor_test.go` | Full browser cookie refresh/logout flow is not end-to-end tested in this TASK. |
| Recruitment jobs and applications | Public/staff job routes, application create/list/status update, status state machine | `web-gin-service/handler/cursor_test.go`, `web-gin-service/handler/candidate/apply_cursor_test.go`, `web-gin-service/handler/hr/job_cursor_test.go`, `logic-grpc-service/service/transition_validator_test.go`, `logic-grpc-service/model/core_workflow_status_test.go`, `logic-grpc-service/repository/application_repo_test.go` | Public JSON snapshots for every job/application endpoint are not exhaustive. |
| Interview scheduling and feedback | Interview scheduling/update/cancel/list plus interviewer task and feedback state | `logic-grpc-service/service/interview_service_test.go`, `logic-grpc-service/repository/interview_repo_test.go`, `logic-grpc-service/email/email_flow_test.go` | Gateway handler-level interview route mapping is not exhaustively tested. |
| Offer lifecycle | Offer create/update/send/withdraw/candidate decision/event history | `logic-grpc-service/service/offer_service_test.go`, `logic-grpc-service/repository/offer_repo.go`, `web-gin-service/handler/hr/offer.go` under full `go test ./...` compilation | Offer HTTP handler behavior has compile coverage but limited focused handler tests. |
| Notification delivery and unread state | Notification list/unread/summary/mark-read/SSE stream and business identifiers | `logic-grpc-service/service/notification_service_test.go`, `logic-grpc-service/repository/notification_repo_test.go`, `web-gin-service/handler/cursor_test.go`, `web-gin-service/middleware/route_auth_test.go` | SSE streaming behavior is not fully end-to-end tested in this TASK. |
| AI and agent workflows | Candidate/HR chat, agent runs, skill selection, prompt/runtime config, MCP governance, embedding fallback | `logic-grpc-service/service/ai_service_test.go`, `logic-grpc-service/service/agent_run_durable_test.go`, `logic-grpc-service/service/agent_skill_service_test.go`, `logic-grpc-service/ai/*_test.go`, `web-gin-service/handler/hr/ai_agent_run_test.go`, `web-gin-service/handler/hr/mcp_test.go` | Live provider calls are intentionally not part of this baseline; provider integration remains mocked or skipped. |

## New Regression Guard

`logic-grpc-service/model/core_workflow_status_test.go` locks current recruitment lifecycle invariants:

- New applications default to `applied`.
- Only configured terminal/closeout statuses allow reapplication.
- Active statuses cannot be used for reapplication.
- Candidate-facing and HR-facing closeout labels remain audience-specific.

These assertions protect the current candidate-visible and HR-visible status semantics before later extraction tasks move recruitment, interview, offer, notification, or AI boundaries.

## Compatibility Rule

Later TASKs must not silently change any workflow state, role/permission gate, notification business identifier, AI agent run status, or offer/interview lifecycle transition. If a later TASK intentionally changes behavior, it must carry explicit scope, tests, rollback notes, and human confirmation.
