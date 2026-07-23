# Legacydomain Migration Finalization SDD

## 1. Existing Architecture Summary

`smart-recruit-recruitment-service/internal/runtime/runtime.go` already exposes explicit runtime dependency interfaces for jobs, taxonomy, admin, usage, candidate, application, application-owner contract, and collaboration. The current binary wires all of those interfaces from `internal/infrastructure/persistence.NewNativeBundle`, which returns one `nativeAdapter` value implementing many unrelated APIs.

`smart-recruit-ai-agent-service/internal/runtime/runtime.go` registers nine gRPC services: AI, LLM config, Prompt, AgentConfig, MCP, Skill, AgentSkill, RecruitingIntelligence, and EmbeddingConfig. The current binary wires these with `internal/interfaces/grpc.NewNativeRuntimeDeps`, which builds native gRPC servers directly over `AIStore`. Some list methods are database-backed, while many methods are inherited from `Unimplemented...Server` or return success with empty payloads.

Gateway routes already expose the relevant AI Agent management APIs under existing HR admin routes. `smart-recruit-proto/proto/recruitment.proto` is the wire contract and must remain compatible.

## 2. Problem Analysis

The previous migration removed `legacydomain` physically, but AI Agent and Recruitment still carry migration residue:

- AI Agent native gRPC services became the active business surface, so partial stubs now affect real frontend workflows.
- AI Agent lacks an application-service boundary for configuration/governance features, making missing behavior easy to hide behind empty success responses.
- Recruitment's monolithic native adapter is implemented, but it concentrates multiple bounded contexts in a single persistence object. This is not the desired final post-legacy shape because it makes ownership, testing, and future changes harder.
- Knowledge documents now describe the intermediate native adapter shape and must be updated as the final architecture lands.

## 3. Proposed Design

### AI Agent

Introduce explicit application services and ports for the AI Agent contexts currently hidden behind native gRPC stubs:

- `LlmConfigService` application service for provider/model CRUD and connection tests.
- `PromptService` application service for templates, versions, rollback, rendering, and active prompt lookup.
- `AgentConfigService` application service for agent configs, capabilities, and internal config lookup.
- `MCPService` application service for servers, policies, logs, connection tests, tool listing, and calls.
- `SkillService` application service for the separate Skill registry.
- `AgentSkillService` application service for database-backed agent skills, versions, preview, status, embedding regeneration, and semantic debugging.
- `EmbeddingConfigService` application service for embedding providers/models, default model selection, test, and backfill trigger.
- `RecruitingIntelligenceService` application service for resume profile and candidate-match evaluation flows.

Keep GORM records private under `internal/infrastructure/persistence`. Keep provider/MCP/embedding runtime integrations under `internal/infrastructure/provider`, `internal/infrastructure/mcp`, or focused infrastructure packages. The gRPC layer should map proto requests/responses and delegate to application services.

### Recruitment

Replace the catch-all `nativeAdapter` bundle with focused adapters matching runtime dependencies:

- job adapter/service
- taxonomy and taxonomy-admin adapter/service
- candidate/resume adapter/service
- application lifecycle adapter/service
- application-owner contract adapter
- collaboration adapter/service
- invite/admin adapter/service
- usage stats/audit adapter/service
- outbox publisher/record adapter where needed

This can be done incrementally while preserving `runtime.Deps`. `native_adapters.go` should be split or retired once each focused adapter exists.

### Guardrails

Extend scripts and harness checks to detect:

- reintroduced `internal/legacydomain`;
- non-test `legacydomain` imports;
- active runtime files returning successful empty responses for unsupported or unimplemented behavior;
- AI Agent native gRPC service registration that bypasses application services after final TASKs.

## 4. Data Structure Changes

No database schema or protobuf data structure changes are planned. New Go structs may be introduced as private DTOs, repository records, ports, or application-level command/result types.

## 5. API and Interface Changes

No public Gateway HTTP route or frontend API shape changes are planned.

Internal Go interfaces may be introduced or refined under service-local `internal/application/**`, `internal/domain/**`, and `internal/infrastructure/**` packages.

Existing proto RPCs should be implemented or explicitly fail with non-success application codes. Adding new proto RPCs is out of scope unless a TASK hits a confirmed hard stop.

## 6. Algorithm or Workflow Changes

- AI Agent provider connection tests should load provider/model configuration, resolve redacted credentials safely, and perform a lightweight validation path using existing provider helpers.
- AI Agent management mutations should validate input, persist through service-local adapters, and return current records with secrets redacted.
- MCP governance operations should preserve policy evaluation, log redaction, and audit semantics.
- Recruitment adapter extraction should keep existing SQL behavior and lifecycle status transitions while moving code into focused files/types.

## 7. Configuration Design

No new environment variables are planned. Runtime startup should fail clearly when required AI Agent store/provider/worker components are missing for enabled features.

## 8. Compatibility Strategy

- Keep proto and Gateway route compatibility.
- Use current database tables and ownership manifest.
- Preserve existing response codes for successful behavior.
- Replace silent empty success with explicit non-success errors only where the prior response represented missing implementation or misconfiguration.

## 9. Error Handling and Fallback Design

AI Agent application services should return typed or clearly wrapped errors that the gRPC layer maps to safe application responses. Runtime dependency absence should be treated as configuration failure, not a successful empty state.

Provider and MCP errors should be visible enough for admin debugging but must redact secrets and sensitive candidate/resume content.

## 10. Observability and Debug Output Design

- Tests should cover connection-test error mapping and empty-stub prevention.
- Logs should identify provider/model/MCP identifiers and runtime component names without secrets.
- TASK reports and evidence must include `knowledge_impact`.

## 11. Testing Strategy

For each implementation TASK, run:

- `go test ./...` in each touched Go service module.
- `node scripts/check-backend-boundaries.mjs`.
- `node scripts/check-mysql-table-ownership.mjs`.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`.
- `node .knowledge/scripts/check-references.mjs --root .`.
- `bash .spec/legacydomain-migration-finalization/scripts/check-task-scope.sh <TASK-ID>`.
- `bash .spec/legacydomain-migration-finalization/scripts/agent-check.sh`.

TASK-specific tests should be added for AI Agent gRPC methods, provider test behavior, Recruitment adapter parity, and guardrails.

## 12. Migration Risks

- AI Agent legacy behavior was broad; implementing it in focused services may expose hidden schema or provider assumptions.
- Recruitment native adapter extraction can accidentally change SQL queries or lifecycle side effects.
- Boundary checks for empty success stubs may need careful allowlists for legitimate empty list responses.
- Existing admin pages may rely on permissive success responses and need explicit error handling verification.

## 13. Implementation Boundaries

- Keep feature-owned docs under `.spec/legacydomain-migration-finalization/**`.
- Allow `.knowledge/**` updates only when routed and within the current TASK scope.
- Do not modify frontend, Gateway public route definitions, protobuf files, migrations, package manifests, or shared modules unless a TASK explicitly allows it and records the compatibility reason.
- Do not change services outside Recruitment and AI Agent except shared guardrail scripts and routed knowledge documents.

## 14. Alternatives Considered

- Keep native adapters and only patch missing methods: rejected because it preserves the interim architecture and makes future gaps likely.
- Reintroduce shared domain packages: rejected because it violates the legacydomain retirement goal.
- Rewrite public APIs around new AI Agent contracts: rejected because public compatibility is required.

## 15. Assumptions Requiring Confirmation

- Existing AI Agent tables support the currently exposed admin operations.
- Existing provider/MCP helper packages can support connectivity checks without new dependencies.
- Recruitment's current native adapter behavior is acceptable as the parity source during extraction.

## 16. Open Questions

- Which non-success application code should standardize unsupported AI Agent operations during incremental rollout?
- Should semantic retrieval debugging be completed in this feature or deferred if it needs broader embedding/runtime work?
