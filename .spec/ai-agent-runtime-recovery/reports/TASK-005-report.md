# TASK Report - TASK-005

## 1. TASK ID

TASK-005 - Restore Recruiting Intelligence Generation

## 2. Status

Completed.

Independent read-only self-review returned `通过` with no findings.

## 3. Modified File List

- `.spec/ai-agent-runtime-recovery/pipeline-state.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-005-evidence.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-005-report.md`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_recruiting_generation_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_recruiting_generation_test.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`

## 4. Change Summary by File

- `native_servers.go`: restores native Recruiting Intelligence generation for `ParseResumeProfile` and `EvaluateCandidateMatch`; adds provider JSON parsing, safe prompt rendering, HR AI permission checks through `ai.hr.use`, generated draft mapping, bounded evidence snippets, and `agent_run_id` preservation for match evaluation.
- `native_store.go`: adds source loading and versioned persistence for generated resume profiles and candidate match evaluations using existing tables; old profile/evaluation rows are demoted from current/latest before new snapshots are saved.
- `native_recruiting_generation_test.go` in `interfaces/grpc`: covers successful generation, persistence handoff, `agent_run_id` preservation, and AI permission denial for parse/evaluate paths.
- `native_recruiting_generation_test.go` in `infrastructure/persistence`: covers NativeStore resume profile versioning/current demotion, match evaluation versioning/latest demotion, evidence persistence, and source loading.
- `pipeline-state.json`: records TASK-005 runtime status.

## 5. Scope Check Result

Passed.

Command:

```bash
TASK_BASE_TREE=52668f973b06f3a737fc9d7758a49e8f39e39183 bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-005
```

Result:

```text
scope_result: PASS (TASK-005)
```

## 6. SPEC Comparison Result

Passed.

The implementation restores Recruiting Intelligence generation without proto, schema, dependency, auth/RBAC catalog, global config, K8s, or repository-root docs changes.

## 7. SDD Comparison Result

Passed.

The native AI Agent service now generates structured resume profile drafts and candidate match drafts through the configured provider, persists them to existing read-model tables, and keeps the current microservice boundary instead of depending on the old monolith runtime.

## 8. Acceptance Comparison Result

Passed.

- `ParseResumeProfile` generates and persists fresh resume profile snapshots: implemented and tested.
- `EvaluateCandidateMatch` generates and persists evaluation/evidence: implemented and tested.
- `agent_run_id` is preserved when provided: implemented and tested.
- Existing read APIs remain compatible: read-through behavior and full AI Agent tests passed.
- Authorization and HR AI permission checks remain intact: application/job access remains required and `ai.hr.use` is checked before generation.
- Raw candidate/resume sensitive data is not written into TASK reports: report/evidence include only IDs, hashes, file names, and behavioral summaries.

## 9. Test Commands and Results

| Command | Result |
|---|---|
| `GOWORK=off go test ./internal/interfaces/grpc -run 'TestParseResumeProfileGeneratesAndPersistsFreshSnapshot|TestParseResumeProfileRejectsAIPermissionDenied|TestEvaluateCandidateMatchGeneratesAndPreservesAgentRunID|TestEvaluateCandidateMatchRejectsAIPermissionDenied|TestParseResumeProfileReadThroughPath|TestEvaluateCandidateMatchReadThroughPath' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./internal/infrastructure/persistence -run 'TestNativeStoreSaveRecruitingResumeProfileDraftVersionsAndChildren|TestNativeStoreSaveRecruitingCandidateMatchDraftVersionsLatestAndAgentRun|TestNativeStoreRecruitingReadModels' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service` | Passed |
| `git diff --name-only` | Passed |
| `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-005` | Passed |
| `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh` | Passed |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 52668f973b06f3a737fc9d7758a49e8f39e39183` | Passed; result `update_required` |

Gateway tests were not required by TASK-005 acceptance because no Gateway files changed in TASK-005; `agent-check.sh` still ran Gateway tests due existing staged TASK-004 Gateway changes and they passed.

## 10. Knowledge Impact

Result: `update_required`.

No `.knowledge/**` files were edited because they are outside TASK-005 scope. Candidate knowledge debt was recorded for Recruiting Intelligence generation, persistence/versioning, sensitive resume data handling, service boundaries, and AI runtime behavior.

## 11. Review Rounds

- Round 1: `通过`; independent read-only subagent Jason reported no findings.

## 12. Risks

- Native AI Agent writes existing recruitment intelligence read-model tables directly in this scoped recovery path. This follows TASK-005 acceptance and avoids proto/schema expansion, but table ownership should be documented in a future knowledge or architecture maintenance task.
- No live provider smoke was run; fake-provider and in-memory persistence tests cover generation and persistence semantics without exposing candidate data.

## 13. Follow-up Items

- Update relevant `.knowledge/**` documents in a future knowledge-maintenance scope for Recruiting Intelligence generation ownership, versioning, and sensitive-data handling.

## 14. Whether the Next TASK Can Start

Yes. TASK-005 checks passed and independent self-review returned `通过`; TASK-006 can start.
