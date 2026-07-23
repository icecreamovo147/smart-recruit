# TASK Report - TASK-HARS-005

## 1. TASK ID

TASK-HARS-005

## 2. Modified File List

- `web-gin-service/handler/hr/ai.go`
- `web-gin-service/handler/hr/ai_agent_run_test.go` (new)
- `web-gin-service/router/router.go`
- `web-gin-service/rpc/client.go`
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-005-report.md`
- `.spec/hr-agent-resumable-stream/reports/TASK-HARS-005-evidence.json`
- `.spec/hr-agent-resumable-stream/pipeline-state.json`

## 3. Change Summary by File

| File | Summary |
| --- | --- |
| `web-gin-service/handler/hr/ai.go` | Adds thin gateway handlers for Create/Get/Active/Subscribe(SSE)/Cancel/Confirm agent runs. Passes `hr_id` from auth context. SSE uses `id: <seq>` + JSON data; replay cursor from `Last-Event-ID` (precedence) or `after_seq` query. HTTP disconnect cancels only the gRPC subscribe stream (request context), never `CancelAgentRun`. Legacy `ChatStream` untouched. |
| `web-gin-service/handler/hr/ai_agent_run_test.go` | Mock `AIServiceClient` + stream client; httptest coverage for validation, PermissionDenied mapping, field forwarding, after_seq/Last-Event-ID precedence, invalid replay params, and disconnect-without-cancel. |
| `web-gin-service/router/router.go` | Registers authenticated HR routes under `staffGroup` with existing `PermAIHRUse` and AI middleware patterns. Keeps legacy chat/stream routes. |
| `web-gin-service/rpc/client.go` | Adds read-only retry policy entries for `GetAgentRun` and `GetActiveAgentRun` (writes remain non-retry). |
| report/evidence/pipeline-state | Harness artifacts; `current_phase=review`. |

## 4. Scope Check Result

**PASS** — all TASK-local changed files match allowed globs.

```bash
TASK_BASE_TREE=55afd38e7d006c36c69cefea1a30e761603e00f7 \
  bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh TASK-HARS-005
# exit_code=0
```

Allowed hits:

- `web-gin-service/handler/hr/ai.go`
- `web-gin-service/handler/hr/ai_agent_run_test.go` → `*ai*test*.go`
- `web-gin-service/router/router.go` → `router/**`
- `web-gin-service/rpc/client.go` → `rpc/**`
- `.spec/.../pipeline-state.json` (+ report/evidence written after check baseline)

No forbidden or out-of-scope business files. Proto/pb not modified (already present from TASK-HARS-003).

## 5. SPEC Comparison Result

Aligned with:

- FR-007: SSE subscribe with replay after sequence then live stream
- FR-008: Subscription disconnect does not cancel the run
- FR-009: Active-run lookup by session (`GET .../sessions/:session_id/active-run`)
- FR-010: Explicit cancel endpoint
- FR-011 (gateway portion): Confirm endpoint forwards confirmation to logic-grpc
- FR-017 / AC-008: Legacy chat stream endpoints retained
- AC-003: Replay via `after_seq` / `Last-Event-ID`
- AC-004/AC-005: Cancel is an explicit command; disconnect is not cancel

Ownership/auth: handlers always inject `middleware.UserID(c)` as `hr_id`; routes use existing JWT + `PermAIHRUse`. Cross-user enforcement remains in logic-grpc (PermissionDenied mapped to 403).

## 6. SDD Comparison Result

Matches SDD §5 / §8 / §9:

| SDD endpoint | Implemented route |
| --- | --- |
| `POST /api/v1/hr/ai/runs` | `staffGroup.POST("/ai/runs", ... CreateAgentRun)` |
| `GET /api/v1/hr/ai/runs/:run_id` | `staffGroup.GET("/ai/runs/:run_id", ... GetAgentRun)` |
| `GET /api/v1/hr/ai/sessions/:session_id/active-run` | `staffGroup.GET("/ai/sessions/:session_id/active-run", ... GetActiveAgentRun)` |
| `GET /api/v1/hr/ai/runs/:run_id/events` | `staffGroup.GET("/ai/runs/:run_id/events", ... SubscribeAgentRunEvents)` SSE |
| `POST /api/v1/hr/ai/runs/:run_id/cancel` | `staffGroup.POST("/ai/runs/:run_id/cancel", ... CancelAgentRun)` |
| `POST /api/v1/hr/ai/runs/:run_id/confirm` | `staffGroup.POST("/ai/runs/:run_id/confirm", ... ConfirmAgentRun)` |

- Thin wrappers only; no gateway-local run lifecycle ownership
- SSE accepts `after_seq` and `Last-Event-ID` (header wins when present)
- Disconnect closes only the subscription
- Legacy chat stream compatibility preserved

## 7. Acceptance Comparison Result

| Criterion | Result |
| --- | --- |
| Authenticated HR endpoints for create/get/active/subscribe/cancel/confirm | Pass |
| Forward to logic-grpc; no gateway-local lifecycle state | Pass |
| SSE supports `after_seq` and `Last-Event-ID` | Pass |
| HTTP disconnect ends only SSE subscription | Pass (test asserts `CancelAgentRun` not called) |
| Ownership/auth via existing HR middleware + `hr_id` | Pass |
| Handler tests: validation, auth failure, replay, disconnect | Pass |
| Keep legacy chat stream endpoints | Pass (untouched) |

## 8. Test Commands and Results

| Command | Result |
| --- | --- |
| `git diff --name-only` | OK (TASK-local + prior feature work in dirty tree) |
| `TASK_BASE_TREE=55afd38e7d006c36c69cefea1a30e761603e00f7 bash .spec/.../check-task-scope.sh TASK-HARS-005` | PASS (exit 0) |
| `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh TASK-HARS-005` | PASS (exit 0) |
| `cd web-gin-service && go test ./handler/hr/ ./router/ ./rpc/ -count=1 -timeout 120s` | PASS |
| `cd web-gin-service && go build ./...` | PASS |

## 9. Knowledge Impact

- **result**: `update_required` (knowledge files out of scope for this TASK; no `.knowledge` edits)
- **review**:
  - `.knowledge/architecture/api-contracts-and-gateway.md` — **UNCHANGED** for boundary model (gateway remains transport/policy). Docs lag new durable run HTTP paths; update later when scope allows.
  - `.knowledge/architecture/service-boundaries.md` — **UNCHANGED** ownership model still holds (logic owns lifecycle; web-gin forwards).
  - `.knowledge/pitfalls/protobuf-synchronization.md` — **UNCHANGED** (this TASK did not change proto/generated code; pb already aligned from TASK-HARS-003).
- **coverageGap**: false
- No STALE/CONFLICT verdicts recorded (debt captured via `update_required` + UNCHANGED lag notes).

## 10. Risks

- Confirm/create middleware applies AI quota/risk/timeout similar to chat; long-running confirm-driven work happens in logic after the HTTP request returns (create returns immediately with run snapshot). If product later needs streaming create, that is out of scope here.
- SSE uses request context for gRPC subscribe; reverse proxies must not buffer (`X-Accel-Buffering: no` set) and should allow long-lived connections.
- Gateway does not re-check ownership beyond auth middleware; relies on logic-grpc PermissionDenied/NotFound for cross-user access (consistent with existing GetAgentRuns pattern).
- Frontend (TASK-006+) must parse `id:` SSE fields and send `Last-Event-ID` or `after_seq` on reconnect.

## 11. Follow-up Items

- TASK-HARS-006: frontend API/reducer/composable against these endpoints
- Optional later knowledge update for api-contracts-and-gateway durable run routes
- Do not start TASK-HARS-006 until review + human confirmation for this TASK

## 12. Whether the Next TASK Can Start

**No.** Implementation complete and ready for **self-review**. Next TASK (TASK-HARS-006) must not start until TASK-HARS-005 review passes and user/pipeline confirmation is recorded.
