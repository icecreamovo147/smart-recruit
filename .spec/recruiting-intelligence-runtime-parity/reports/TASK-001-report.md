# TASK Report - TASK-001

## 1. TASK ID

TASK-001 — Structured Provider and Prompt Runtime (`fix-check-failures`, repair round 2 after independent review round 2).

Harness classification: `current`. Reliable baseline: `base_sha=98165e7cdac06c0aa2a0685ecf1ad7c00a3eb518`, `base_tree=fb76dfce94d5068494f7be7c5b6519df582cae2e`.

The required user confirmation was present before editing:

> 用户明确选择“授权修改”，仅新增内部结构化 System/User 消息调用能力，不修改依赖、Proto 或公共 HTTP API。

On 2026-07-15 the user additionally authorized exactly the 12 previously verified existing-dependency `go.mod` checksum lines in `smart-recruit-commons/go.sum`, with no `go.mod` or dependency-version change.

## 2. Modified File List

- `smart-recruit-commons/ai/structured_completion.go`
- `smart-recruit-commons/ai/structured_completion_test.go`
- `smart-recruit-commons/ai/eino_client.go`
- `smart-recruit-commons/go.sum`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go`
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime_prompt_test.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_structured_runtime_test.go`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-001-report.md`
- `.spec/recruiting-intelligence-runtime-parity/reports/TASK-001-evidence.json`
- `.spec/recruiting-intelligence-runtime-parity/task-scope.json`
- `.spec/recruiting-intelligence-runtime-parity/pipeline-state.json` (root-owned baseline/confirmation update; not edited by the Developer)

`pipeline-state.json` also differs from the synthetic TASK base tree because the root agent recorded the TASK run and confirmation before Developer execution. The Developer did not modify it.

## 3. Change Summary by File

- `structured_completion.go`: adds an exact two-message System/User completion path on the shared AI client. It calls the privacy-safe guarded call path, preserving timeout, retry, semaphore, and circuit-breaker behavior while omitting raw provider errors from logs.
- `structured_completion_test.go`: verifies unchanged roles/content, absence of generic HR Markdown contamination, retry, timeout, circuit-breaker rejection, classified provider failures, and log redaction using a sensitive provider-error marker.
- `eino_client.go`: retains the existing generic-call behavior and adds a structured-only privacy-safe retry/circuit logging mode that emits stable error classifications instead of raw provider error strings.
- `go.sum`: adds exactly 12 previously verified `go.mod` checksums for existing dependency versions; no line was removed and no module version or `go.mod` entry changed.
- `structured_runtime.go`: adds the internal recruiting Prompt store/provider ports, prompt descriptor metadata, per-request loader, supported agent types, and sanitized/classifiable runtime errors.
- `structured_runtime_test.go`: verifies exact `agent_type + system` lookup on every request, immediate prompt-version refresh, strict prompt validation, and body-safe error strings.
- `llm_runtime.go`: adapts `NativeStore` to the structured provider and prompt-store ports; active prompt selection uses exact agent type/role/active predicates and deterministic `updated_at DESC, id DESC` ordering. Every request still reads DB configuration, while clients are shared by complete configuration fingerprint rather than written through a mutable current-client slot. A late old-config request therefore cannot replace the already-observed new-config client's semaphore or breaker state.
- `llm_runtime_prompt_test.go`: verifies wrong-role/wrong-agent exclusion, later active prompt observation, cross-request concurrency limiting, shared circuit opening, ordinary model refresh, and a controlled old-read/new-read interleaving in which the new configuration's open breaker remains shared after the old request finishes late.
- `native_servers.go`: wires the internal structured runtime only when both narrow adapters are available; existing public gRPC behavior remains unchanged.
- `native_structured_runtime_test.go`: verifies fail-closed optional wiring and successful adapter binding.
- `task-scope.json`: records the approved checksum exception with the same `approved_at`, scope, and reason as root-owned pipeline state.
- TASK report/evidence: preserves all three review rounds and both repair histories, records all 19 mechanically routed knowledge verdicts, and reconciles changed files and the approved exception.

## 4. Scope Check Result

`bash .spec/recruiting-intelligence-runtime-parity/scripts/check-task-scope.sh TASK-001` passed. All code, test, report, and root-owned state differences match TASK-001 `allowedFiles`; forbidden and out-of-scope lists are empty. The authorized `go.sum` diff is exactly `+12/-0`; no dependency version, `go.mod`, Proto, API, schema, migration, config, permission, frontend, or root-doc file changed.

## 5. SPEC Comparison Result

- FR-001 satisfied for this TASK: structured calls carry exactly one System and one User message and reuse shared provider controls without the generic HR Markdown prompt.
- FR-002 foundation satisfied: the three supported recruiting agent types use exact active `system` prompt lookup per request, with ID/name/version retained.
- Compatibility and privacy constraints satisfied: legacy generic completion is unchanged; no external contract or sensitive body logging was added.
- Extractor schemas, policy wiring, matching, aggregation, and persistence behavior remain intentionally deferred to TASK-002 through TASK-006.

## 6. SDD Comparison Result

The implementation matches SDD sections 3, 5, 6.1, 7, 8, 9, 10, and 13: narrow internal ports, current DB prompt semantics, exact message roles, shared resilience controls across requests, safe configuration refresh, classified privacy-safe logs, and no external API/dependency-version change. The client cache is scoped to `NativeStore` and a complete runtime-configuration fingerprint; prompt content remains uncached and is still loaded per request.

## 7. Acceptance Comparison Result

- Exact prompt key and deterministic active selection: passed by application and persistence tests.
- Newly activated version observed on a later request: passed.
- Exact System/User roles and generic Markdown absence: passed.
- Timeout, retry, circuit-breaker, and provider classification: passed.
- Scope and compatibility restrictions: passed.
- Required `GOWORK=off` module checks: AI Agent passed; Commons passed after the exactly authorized checksum-only repair.
- Independent review round 1: `verdict: 不通过`.
- Repair round 1 fixed R1-001, R1-002, and R1-003.
- Independent review round 2: `verdict: 不通过`, with R2-001 (late old-config cache overwrite), R2-002 (incomplete mechanical knowledge-route verdicts), and R2-003 (report/evidence/state reconciliation).
- Repair round 2 fixed R2-001, R2-002, and R2-003; a fresh independent review is still required.

TASK-001 cannot be declared complete until a fresh independent Reviewer returns `verdict: 通过`.

## 8. Test Commands and Results

| Command | Result |
|---|---|
| `go test ./smart-recruit-commons/ai ./smart-recruit-ai-agent-service/internal/application/recruiting_intelligence ./smart-recruit-ai-agent-service/internal/infrastructure/persistence ./smart-recruit-ai-agent-service/internal/interfaces/grpc` | PASS |
| `cd smart-recruit-ai-agent-service && GOWORK=off go test ./...` | PASS |
| `cd smart-recruit-commons && GOWORK=off go test ./...` | PASS |
| `cd smart-recruit-ai-agent-service && GOWORK=off go test -race ./internal/infrastructure/persistence -run 'TestNativeStoreCompleteStructured\|TestLoadActiveRecruitingPrompt' -count=1` | PASS, including controlled old/new configuration interleaving |
| `cd smart-recruit-commons && GOWORK=off go test ./ai -run 'TestGenerateStructured' -count=1 -v` | PASS, including sensitive-marker log capture |
| `git diff --check -- smart-recruit-commons/ai smart-recruit-ai-agent-service/internal` | PASS |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree fb76dfce94d5068494f7be7c5b6519df582cae2e` | PASS; mechanical result `update_required`, manually classified `candidate_required` under TASK-001 knowledge rules |
| `node .knowledge/scripts/knowledge-validator.test.mjs` | PASS (17 tests) |
| `node .knowledge/scripts/validate-knowledge.mjs --root .` | PASS (37 formal documents) |
| `node .knowledge/scripts/check-references.mjs --root .` | PASS |
| `bash .spec/recruiting-intelligence-runtime-parity/scripts/check-task-scope.sh TASK-001` | PASS |
| `bash .spec/recruiting-intelligence-runtime-parity/scripts/agent-check.sh` | PASS |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .../TASK-001-evidence.json --require-knowledge-impact --allow-pending-review` | PASS |
| `git diff --name-only` plus untracked-file inventory | PASS; results reconcile with the base-tree scope output above |
| privacy scan for structured log body fields and raw recruiting payload markers in reports | PASS |

`git diff --numstat -- smart-recruit-commons/go.sum` reports exactly `12 0`; the standalone Commons module now verifies without workspace checksum assistance.

## Knowledge Impact

Result: `candidate_required`; coverage gap: false. Impact detection mechanically routed 19 documents. Each receives exactly one verdict below. TASK-001 through TASK-006 may not edit `.knowledge`, so relevant runtime/governance documentation changes are deferred to TASK-007.

| Routed document | Verdict | Verified evidence |
|---|---|---|
| `.knowledge/architecture/agent-runtime.md` | CANDIDATE | New structured runtime and shared client path are supported by `structured_runtime.go`, `structured_completion.go`, `native_servers.go`, and existing runtime source refs. |
| `.knowledge/domains/agent-skill.md` | UNCHANGED | TASK-001 does not change Agent Skill eligibility, versions, retrieval metadata, administration, or runtime tool execution. |
| `.knowledge/domains/ai-configuration-governance.md` | CANDIDATE | Active prompt selection now drives the internal structured runtime per request and retains ID/name/version metadata; fingerprint-keyed clients preserve resilience across DB config refreshes. |
| `.knowledge/architecture/api-contracts-and-gateway.md` | UNCHANGED | No HTTP route, gateway handler/client, protobuf, or public gRPC shape changed. |
| `.knowledge/runbooks/debug-agent-retrieval.md` | UNCHANGED | Skill, memory, embedding, context-budget, and semantic-retrieval diagnostics are untouched. |
| `.knowledge/pitfalls/embedding-fallback.md` | UNCHANGED | No embedding provider, dimension, retrieval fallback, or visible retrieval status changed. |
| `.knowledge/runbooks/local-development.md` | UNCHANGED | Startup order, local infrastructure, service commands, and endpoint conventions are unchanged. |
| `.knowledge/pitfalls/mcp-policy-audit.md` | UNCHANGED | No MCP policy, confirmation, network/command restriction, redaction, or audit path changed. |
| `.knowledge/domains/mcp-tool-governance.md` | UNCHANGED | No MCP server, policy, log, discovery, or tool-execution behavior changed. |
| `.knowledge/domains/memory-and-context.md` | UNCHANGED | No memory, summary, context assembly, budget, or persistence behavior changed. |
| `.knowledge/pitfalls/migration-model-drift.md` | UNCHANGED | No migration, schema, persistence model, table ownership, or `db.sql` change occurred. |
| `.knowledge/architecture/persistence-and-migrations.md` | UNCHANGED | The DB work is read-only runtime configuration/prompt selection; schema and repository ownership remain unchanged. |
| `.knowledge/runbooks/protobuf-and-migration-change.md` | UNCHANGED | No protobuf generation or migration workflow is implicated by this internal-only change. |
| `.knowledge/pitfalls/protobuf-synchronization.md` | UNCHANGED | Canonical protobuf and generated contracts are unchanged. |
| `.knowledge/architecture/service-boundaries.md` | UNCHANGED | Changes remain internal to AI Agent/Commons and preserve the existing gRPC/service ownership boundary. |
| `.knowledge/domains/resume-intelligence.md` | CANDIDATE | TASK-001 establishes the database-prompt/structured-provider foundation; extractor behavior itself is unchanged until TASK-003. |
| `.knowledge/pitfalls/resume-sensitive-data.md` | UNCHANGED | No resume data is processed by TASK-001; new logs/errors contain counts and identities only, never message bodies or model output. |
| `.knowledge/architecture/semantic-retrieval.md` | UNCHANGED | Agent Skill/memory semantic retrieval and embedding fallback behavior are untouched. |
| `.knowledge/architecture/system-overview.md` | UNCHANGED | No top-level module, service responsibility, protocol location, or shared-platform boundary changed. |

## 9. Risks

- The structured runtime is wired but not consumed by resume/match workflows until later TASKs, by design.
- Knowledge candidates must be reconciled in TASK-007 after end-to-end behavior stabilizes.
- The process-lived `NativeStore` cache retains one structured client per distinct observed configuration fingerprint. This prevents stale write-back and preserves per-configuration resilience, at the cost of retaining prior configuration clients until the store is released; configuration changes are expected to be infrequent.

## 10. Follow-up Items

- TASK-001 review round 3 returned `verdict: 通过`; TASK-002 may start after its independent baseline is recorded.

## 11. Whether the Next TASK Can Start

Yes. All required checks pass and independent review round 3 returned `verdict: 通过`.

## Repair Summary

### Failed Check

Independent review round 1 returned `不通过` with R1-001 (Commons standalone checksum failure), R1-002 (per-request structured client reset resilience state), and R1-003 (raw provider error exposure through retry logs).

### Root Cause

- The Commons module lacked 12 `go.mod` checksum lines already verified in a temporary `-mod=mod` run.
- `NativeStore.CompleteStructured` constructed a new `ai.Client` on every request, resetting its semaphore and breaker.
- Structured completion reused generic retry logging that attached `zap.Error(err)`, allowing an untrusted provider error body to enter logs.

### Files Changed

- `smart-recruit-commons/go.sum`
- `smart-recruit-commons/ai/eino_client.go`
- `smart-recruit-commons/ai/structured_completion.go`
- `smart-recruit-commons/ai/structured_completion_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime_prompt_test.go`
- TASK-001 report and evidence files

### Fix Summary

- Added the exact authorized checksum lines (`+12/-0`) without changing `go.mod` or a dependency version.
- Reused structured clients for identical store/config fingerprints, sharing concurrency and circuit state across calls; configuration changes produce a new client for subsequent requests while in-flight calls remain safe.
- Added a structured-only privacy-safe retry path and a captured-log regression containing a sensitive marker.

### Re-run Commands

The commands and results are recorded in section 8 and the machine-readable evidence. They include both full standalone module suites, focused race testing, scope, agent-check, knowledge impact, checksum diff verification, and pending-review evidence validation.

### Re-run Results

All repair, Harness, scope, standalone-module, focused race, knowledge-impact, privacy, and evidence checks pass.

### Remaining Risks

Independent re-review is pending; no unresolved implementation or check failure remains in repair round 1.

## Repair Summary — Round 2

### Failed Check

Independent review round 2 returned `不通过` with R2-001 (a late old configuration could replace the newer cached client), R2-002 (only 5 of 19 mechanically routed knowledge documents had per-document verdicts), and R2-003 (changed files and approved-exception metadata disagreed across Markdown, evidence, task scope, and pipeline state).

### Root Cause

- The round-1 cache used a single mutable current-client slot; selection occurred before acquiring its lock, so an old DB read could resume late and write the old client after a newer configuration.
- Knowledge reporting used only the TASK-declared review subset instead of the complete mechanical route output.
- The authorized checksum exception was described in prose but omitted from evidence `exceptions`, and `task-scope.json` was missing from the changed-file inventory.

### Files Changed

- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime_prompt_test.go`
- `.spec/recruiting-intelligence-runtime-parity/task-scope.json`
- TASK-001 report and evidence files

### Fix Summary

- Keyed shared clients by complete configuration fingerprint, eliminating stale write-back while retaining per-request DB reads and shared semaphore/breaker state.
- Added a controlled old/new selection interlock test: the new client opens its breaker, the old request completes late, and a later new request must still be rejected by the same breaker without another provider call.
- Re-ran impact detection at the fixed base tree and recorded one verdict for each of all 19 routed documents without editing knowledge content.
- Reconciled Markdown/evidence changed files and copied the root-approved checksum exception exactly, including `approved_at=2026-07-15T02:37:25Z`, scope, and reason.

### Re-run Commands

The commands and truthful results are recorded in section 8 and machine-readable evidence, including both standalone module suites, focused race, impact detection, scope, agent-check, pending-review evidence validation, and diff checks.

### Re-run Results

All repair-round-2 implementation, race, full-module, knowledge-impact, scope, Harness, and evidence checks pass.

### Remaining Risks

Independent review round 3 passed. Distinct historical configuration fingerprints remain cached for the lifetime of the process-lived store; this is intentionally preferred to losing shared resilience state or permitting stale cache replacement.

## Final Independent Review

Round 3 Reviewer `/root/task001_reviewer_r3` independently reproduced the standalone module, race, scope, agent-check, knowledge, evidence, privacy, checksum, and compatibility results. It reported no blocking findings and ended with `verdict: 通过`.
