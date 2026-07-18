# TASK Report - TASK-004

## 1. TASK ID

TASK-004 - Restore Candidate AI Runtime and Compatibility Route

## 2. Status

Completed.

The first independent self-review returned `不通过` because resume summary text could be persisted into `ai_tool_traces.result_summary`. The fix keeps the candidate-owned resume summary available to the provider prompt but sanitizes the persisted resume tool trace. The second independent self-review returned `通过`.

## 3. Modified File List

- `.spec/ai-agent-runtime-recovery/pipeline-state.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-004-evidence.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-004-report.md`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_candidate_runtime_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_ai_chat_test.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-gateway/handler/candidate/ai.go`
- `smart-recruit-gateway/handler/candidate/ai_test.go`
- `smart-recruit-gateway/router/contract_baseline_test.go`
- `smart-recruit-gateway/router/router.go`

## 4. Change Summary by File

- `native_servers.go`: candidate chat now loads candidate-scoped runtime context, renders it into the provider prompt, records candidate-context tool traces, strips suggested-question marker JSON before persisting assistant content, emits fallback suggested questions, writes candidate usage/auth audit, and uses a conservative context fallback when provider generation fails after context is available. Resume summary is sanitized before trace persistence.
- `native_store.go`: adds NativeStore candidate runtime context loading for candidate-owned applications, resume metadata/summary, active jobs with applied marks, candidate-visible interviews, and candidate offers; adds usage log plus AI auth-context audit writes.
- `native_candidate_runtime_test.go`: covers candidate data isolation across applications, resumes, interviews, offers, jobs, and candidate usage/auth audit persistence.
- `native_ai_chat_test.go`: covers candidate context prompt/traces, resume trace sanitization, clean assistant persistence, suggested-question extraction/fallback, usage audit writes, and provider-error fallback behavior.
- `candidate/ai.go`: adds non-streaming `POST /api/v1/candidate/ai/chat` handler that aggregates `CandidateChatStream` into the frontend response shape.
- `candidate/ai_test.go`: verifies non-streaming aggregation, user/session forwarding, reply concatenation, timestamp, session id, and suggested questions.
- `router.go`: registers the candidate non-streaming AI chat route with candidate AI quota/risk/body/auth middleware.
- `contract_baseline_test.go`: adds the compatibility route to the route baseline.
- `pipeline-state.json`: records TASK-004 runtime status.

## 5. Scope Check Result

Passed.

Command:

```bash
bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-004
```

Result:

```text
scope_result: PASS (TASK-004)
```

## 6. SPEC Comparison Result

Passed.

The implementation restores candidate AI runtime behavior within the current microservice boundary without proto, schema, auth/RBAC, dependency, lockfile, global config, K8s, or repository-root docs changes.

## 7. SDD Comparison Result

Passed.

The SDD candidate runtime path is restored through the native AI Agent service provider path: candidate-owned context is loaded, persisted assistant content is clean, audit records are written, suggested questions are surfaced, and the gateway compatibility route wraps the streaming RPC without introducing a new proto method.

## 8. Acceptance Comparison Result

Passed.

- Candidate ChatStream uses candidate-scoped context for candidate-owned applications, resumes, jobs, interviews, and offers: implemented and tested.
- Candidate responses persist clean assistant content and restore usage audit/auth context: implemented and tested.
- Suggested questions are extracted from marker JSON or generated through a safe fallback: implemented and tested.
- `POST /api/v1/candidate/ai/chat` is registered and returns the user-frontend response shape: implemented and tested.
- Candidate data isolation is preserved: implemented and tested.
- Resume sensitive text is not persisted into tool traces: fixed after review and tested.

## 9. Test Commands and Results

| Command | Result |
|---|---|
| `GOWORK=off go test ./internal/interfaces/grpc -run 'TestCandidateChatStream' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./internal/infrastructure/persistence -run 'TestNativeStoreLoadCandidateRuntimeContextScopesCandidateData|TestNativeStoreRecordCandidateUsageAuditWritesUsageAndAuthContext' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./handler/candidate ./router -run 'TestCandidateChatAggregatesStreamResponse|TestCoreHTTPRouteGroupsRemainRegistered' -count=1 -v` from `smart-recruit-gateway` | Passed |
| `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./...` from `smart-recruit-gateway` | Passed |
| `git diff --name-only` | Passed; output recorded TASK-004 modified tracked files and untracked TASK-004 test/report files separately |
| `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-004` | Passed |
| `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh` | Passed |
| `TASK_BASE_TREE=bda1084740e8177e3cc4ce16e2df799a15482f9f node .knowledge/scripts/detect-impact.mjs` | Passed; result `update_required` |

## 10. Knowledge Impact

Result: `update_required`.

No `.knowledge/**` files were edited because they are outside TASK-004 scope. Candidate knowledge debt was recorded for AI runtime, gateway/API contracts, sensitive resume data, persistence, service boundaries, and related routed documents.

Reviewed documents: `agent-runtime`, `agent-skill`, `ai-configuration-governance`, `api-contracts-and-gateway`, `auth-permission-alignment`, `auth-rbac-security`, `debug-agent-retrieval`, `debug-ai-configuration`, `debug-auth-permissions`, `embedding-fallback`, `local-development`, `mcp-policy-audit`, `mcp-tool-governance`, `memory-and-context`, `migration-model-drift`, `persistence-and-migrations`, `protobuf-and-migration-change`, `protobuf-synchronization`, `resume-intelligence`, `resume-sensitive-data`, `semantic-retrieval`, `service-boundaries`, and `system-overview`.

## 11. Review Rounds

- Round 1: `不通过`; reported candidate resume summary could be persisted into tool traces.
- Round 2: `通过`; confirmed resume trace sanitization, candidate isolation, suggested questions, usage/auth audit, and gateway compatibility route.

## 12. Risks

- No live provider/browser smoke was run; behavior is covered through fake-provider, in-memory persistence, route, and service tests.
- Candidate runtime context is implemented through existing database-backed NativeStore read paths to stay within TASK scope and avoid proto/startup wiring changes.

## 13. Follow-up Items

- Update relevant `.knowledge/**` documents in a future knowledge-maintenance scope for candidate runtime restoration and gateway compatibility route behavior.

## 14. Whether the Next TASK Can Start

Yes. TASK-004 checks passed and independent self-review returned `通过`; TASK-005 can start.
