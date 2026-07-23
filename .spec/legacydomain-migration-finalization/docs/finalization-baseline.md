# Legacydomain Migration Finalization Baseline

## Purpose

This baseline records the current post-`legacydomain` finalization gaps before runtime implementation TASKs begin. It is evidence for TASK-001 and a guardrail input for later TASKs.

## Source Snapshot

- `base_sha`: `f0162252ba30bc0a8a1f6e8578d324f1d0e006ad`
- `base_tree`: `da9570e5bbf2fda0ad961e6b02c9a29d41661000`
- Boundary check source: `scripts/check-backend-boundaries.mjs`
- AI Agent runtime source: `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- AI Agent binary fallback source: `smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go`
- Recruitment runtime source: `smart-recruit-recruitment-service/internal/runtime/runtime.go`
- Recruitment active native adapter source: `smart-recruit-recruitment-service/internal/infrastructure/persistence/native_adapters.go`
- Recruitment binary wiring source: `smart-recruit-recruitment-service/cmd/recruitment-service/main.go`

## AI Agent Runtime Stub Inventory

The current AI Agent runtime still wires `internal/interfaces/grpc.NewNativeRuntimeDeps`, which creates native gRPC service structs over `AIStore` and provider dependencies. TASK-001 does not fix runtime behavior; it records the current gaps and prevents additional gaps from being introduced.

### Unimplemented Native Server Embeds

Current native or noop structs embedding `pb.Unimplemented...ServiceServer`:

- `cmd/ai-agent-service/main.go`: `unavailableEmbeddingConfigService`, `unavailableLlmConfigService`
- `cmd/ai-agent-service/main.go`: `noopAIService`, `noopLlmConfigService`, `noopPromptService`, `noopAgentConfigService`, `noopMCPService`, `noopSkillService`, `noopAgentSkillService`, `noopRecruitingIntelligenceService`, `noopEmbeddingConfigService`
- `internal/interfaces/grpc/native_servers.go`: `nativeAIService`, `nativeRecruitingIntelligenceService`, `nativeLlmConfigService`, `nativePromptService`, `nativeAgentConfigService`, `nativeMCPService`, `nativeSkillService`, `nativeAgentSkillService`, `nativeEmbeddingConfigService`

### Empty Success or Store-Nil Success Gaps

Current known empty-success or store-nil-success findings:

- `CompareCandidatesForJob`: unconditional `Code: 0` success with no comparison payload.
- `UpdateSession`, `DeleteSession`: HR session update/delete return success when `store == nil`.
- `GetToolTraces`, `GetAgentRuns`, `GetActiveAgentRun`: return success with no data when `store == nil`.
- `CreateAgentRun`: creates a fallback in-memory success response when `store == nil`.
- `ListProviders`, `ListModels`, `ListPromptTemplates`, `ListAgents`, `ListMCPServers`, `ListAgentSkills`, `ListAvailableAgentSkills`, `ListEmbeddingProviders`, `ListEmbeddingModels`: return empty success when `store == nil`.
- `ListCapabilities`, `ListMCPToolPolicies`, `ListMCPToolLogs`, `ListSkills`: unconditional empty success responses.

The updated backend boundary check treats these as the TASK-001 baseline and fails if a new targeted AI Agent runtime stub appears outside the baseline.

## Recruitment Native Adapter Responsibility Inventory

Recruitment runtime already exposes focused dependency interfaces in `internal/runtime/runtime.go`, but `cmd/recruitment-service/main.go` still wires every dependency from one `persistence.NewNativeBundle`.

Current `NativeBundle` responsibility groups and extraction targets:

| Runtime dependency | Current implementation owner | Extraction target |
|---|---|---|
| `Job` | `nativeAdapter` methods `CreateJob`, `UpdateJob`, `OfflineJob`, `OnlineJob`, `ListHRJobs`, `ListPublicJobs`, `GetJobDetail` | focused job adapter/service |
| `JobTaxonomy` | `nativeAdapter` methods `ListJobOptions`, `ListDepartmentLocations` | focused taxonomy read adapter |
| `TaxonomyAdmin` | department/location CRUD, status, config, map methods | focused taxonomy admin adapter/service |
| `Admin` | invite code lifecycle and usage log query methods | focused admin/invite adapter |
| `UsageStats` | usage stats and trend methods | focused usage stats adapter |
| `Candidate` | profile, resume, upload presign/confirm methods | focused candidate/resume adapter |
| `Application` | apply/list/update/status transition methods | focused application lifecycle adapter |
| `ApplicationOwnerContract` | snapshot and conditional lifecycle transition methods | focused owner-contract adapter |
| `Collaboration` | notes, tags, follow-up tasks, timeline methods | focused collaboration adapter/service |
| Outbox helpers | `writeOutboxTx` and lifecycle side-effect helpers inside `nativeAdapter` | private outbox helper used by focused adapters |

TASK-002 should preserve SQL behavior and lifecycle side effects while splitting this catch-all adapter into explicit bounded adapters.

## Guardrail Behavior

`scripts/check-backend-boundaries.mjs` now enforces:

- no `internal/legacydomain` directories;
- no non-test Go imports of `internal/legacydomain`;
- no AI Agent targeted runtime stubs outside the TASK-001 baseline.

The current baseline check output is:

```text
backend_boundary_result: PASS
legacydomain_retirement_enforcement: PASS (no legacydomain directories or imports)
runtime_stub_guardrail: PASS (52 known targeted AI Agent runtime stub finding(s), 0 new)
```

## Final Verification Snapshot

TASK-006 final verification keeps the TASK-001 baseline for historical audit, but the implemented TASKs reduced active AI Agent runtime findings from 52 to 35 known targeted findings, all still allowlisted by the guardrail. The remaining findings are static guardrail matches for noop/check-mode service structs, embedded `Unimplemented...Server` markers used for forward compatibility, and coarse `store == nil` patterns that now return non-success configuration failures rather than successful empty runtime behavior.

Final TASK-006 guardrail output:

```text
backend_boundary_result: PASS
legacydomain_retirement_enforcement: PASS (no legacydomain directories or imports)
runtime_stub_guardrail: PASS (35 known targeted AI Agent runtime stub finding(s), 0 new)
```

Final code search confirms no `internal/legacydomain` directories and no non-test Go `legacydomain` imports in targeted service code. Historical service `internal/docs/*.md` files still mention previous `legacydomain` packages as inventory/reference material; TASK-006 scope did not include service docs, and guardrails only enforce directories/imports.

## Knowledge Impact

Reviewed active routed knowledge:

- `.knowledge/architecture/system-overview.md`: unchanged; high-level topology remains accurate.
- `.knowledge/runbooks/local-development.md`: unchanged; validation commands remain accurate.
- `.knowledge/architecture/service-boundaries.md`: unchanged for TASK-001; it intentionally still describes AI Agent and Recruitment as native interim runtime shapes.
- `.knowledge/architecture/agent-runtime.md`: unchanged for TASK-001; later AI Agent implementation TASKs should update it when runtime wiring changes.
- `.knowledge/domains/ai-configuration-governance.md`: unchanged for TASK-001; later configuration implementation should update it when behavior changes.
- `.knowledge/domains/mcp-tool-governance.md`: unchanged for TASK-001; later MCP implementation should update it when behavior changes.
- `.knowledge/domains/agent-skill.md`: unchanged for TASK-001; later Skill/Agent Skill implementation should update it when behavior changes.
- `.knowledge/architecture/semantic-retrieval.md`, `.knowledge/domains/memory-and-context.md`, `.knowledge/pitfalls/embedding-fallback.md`, `.knowledge/domains/resume-intelligence.md`: unchanged for TASK-001; later runtime/intelligence TASKs should update them if behavior changes.
- `.knowledge/domains/recruitment.md`, `.knowledge/domains/recruitment-lifecycle.md`: unchanged for TASK-001; TASK-002 should update them after focused adapters land.
- `.knowledge/runbooks/knowledge-coverage-audit.md`: unchanged; no route coverage change was needed.

Result: `update_required` review was triggered by routed feature and script changes, but no active knowledge document needed a TASK-001 content update because this TASK only records the baseline and adds guardrails.
