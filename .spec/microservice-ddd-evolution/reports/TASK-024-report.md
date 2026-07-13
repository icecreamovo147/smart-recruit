# TASK Report - TASK-024

## 1. TASK ID

TASK-024 - AI Agent MCP/skill/embedding/intelligence 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-ai-agent-service/internal/application/command/capability.go`
- `smart-recruit-ai-agent-service/internal/application/dto/capability.go`
- `smart-recruit-ai-agent-service/internal/application/service/capability_service.go`
- `smart-recruit-ai-agent-service/internal/application/service/capability_service_test.go`
- `smart-recruit-ai-agent-service/internal/docs/ai_agent_dependency_inventory.md`
- `smart-recruit-ai-agent-service/internal/domain/model/capability.go`
- `smart-recruit-ai-agent-service/internal/domain/policy/capability.go`
- `smart-recruit-ai-agent-service/internal/domain/policy/capability_test.go`
- `smart-recruit-ai-agent-service/internal/domain/repository/capability.go`
- `.spec/microservice-ddd-evolution/reports/TASK-024-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-024-evidence.json`

## 3. Change Summary by File

- `internal/domain/model/capability.go`: 新增 MCP server/policy、Skill manifest/version、Agent Skill selection、embedding provider/model/runtime state、candidate match aggregation 等本地模型。
- `internal/domain/policy/capability.go`: 新增 MCP 私网限制、stdio command allowlist、MCP tool policy evaluation、Skill manifest/version、Agent Skill selection、embedding fallback、candidate match scoring/knockout 聚合策略。
- `internal/domain/repository/capability.go`: 新增 MCP policy、Skill/AgentSkill、Embedding config、CandidateMatch persistence ports。
- `internal/application/command/capability.go`: 新增 MCP policy evaluation、Skill selection/version、Embedding runtime resolve、CandidateMatch aggregate commands。
- `internal/application/dto/capability.go`: 新增 capability application result DTO。
- `internal/application/service/capability_service.go`: 新增 MCPPolicyService、SkillService、EmbeddingRuntimeService、CandidateMatchService，本地编排 policy 和 ports。
- `internal/domain/policy/capability_test.go`: 覆盖 MCP policy 决策顺序、私网/allowlist、Skill version/selection、Embedding fallback、CandidateMatch knockout。
- `internal/application/service/capability_service_test.go`: 覆盖 MCP repository policy evaluation、Skill version activation/audit、Embedding runtime resolve、CandidateMatch persistence。
- `internal/docs/ai_agent_dependency_inventory.md`: 记录 TASK-024 本地 capability boundary 与 TASK-025 runtime cutover 剩余债务。
- `.spec/.../TASK-024-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-024-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-024` 通过。

变更均在 TASK-024 allowed files 内：`smart-recruit-ai-agent-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-014：MCP、Skill、AgentSkill、Embedding、RecruitingIntelligence/CandidateMatch 业务语义进入 AI Agent 本地 domain/application 边界。
- SSR-005：MCP 私网限制、command allowlist、policy decision、credential encryption requirement、embedding fallback 和 sensitive match evidence 风险均以本地策略表达。
- ARC-015：未修改 public API、protobuf、schema、安全策略或 active runtime wiring。

## 6. SDD Comparison Result

符合 SDD：

- 3.4 Per-Service Migration Pattern：新增 domain model/policy、repository ports、application services；未把 business state machine 留在 transport/infrastructure。
- 9. Error Handling and Fallback Design：embedding unavailable 显式返回 fallback state；MCP policy rate/confirm/deny 路径可测试。
- 11. Testing Strategy：覆盖 MCP policy、Skill version、embedding config、candidate match 测试。
- Implementation boundaries：本 TASK 仅迁移 AI Agent domain/application capability 语义，不修改 shared module、schema、proto、provider SDK 或 runtime cutover。

## 7. Acceptance Comparison Result

- MCP/Skill/Embedding/RecruitingIntelligence/CandidateMatch 本地化：已完成，新增本地 model、policy、repository ports、application commands/DTO/services。
- 私网限制、allowlist、credential encryption、fallback、semantic retrieval 语义兼容：已完成，MCP private-network/allowlist、credential encrypted marker、embedding unavailable fallback、Agent Skill selection gate 和 candidate match scoring 均有本地策略与测试。
- 覆盖 MCP policy、skill version、embedding config、candidate match 测试：已完成，新增 domain policy 与 application service 测试。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./internal/domain/policy ./internal/application/service` in `smart-recruit-ai-agent-service` | 0 | passed | MCP policy、Skill version/selection、Embedding runtime、CandidateMatch targeted tests 通过。 |
| `go test ./...` in `smart-recruit-ai-agent-service` | 0 | passed | AI Agent 全 package 测试通过。 |
| `go run ./cmd/ai-agent-service --check` in `smart-recruit-ai-agent-service` | 0 | passed | `ai-agent-service runtime check passed`。 |
| `git diff --name-only` | 0 | passed | 已记录 tracked diff；新增 Go 文件通过 `git status --short` 与 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-024` | 0 | passed | Scope check passed，变更文件均在 TASK-024 allowed files 内。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 AI Agent module 变更并运行 `go test ./...` 通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-ddd-evolution/reports/TASK-024-evidence.json` | 0 | passed | `evidence_result: PASS` |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 02407c2bb2622234d0c5b442d82215adb4d9d28c` | 1 | non-blocking | 既有 active knowledge 仍引用已迁移/缺失的旧 `logic-grpc-service` / `web-gin-service` source_refs；本 TASK scope 不允许改 `.knowledge/**`，记录为 candidate_required。 |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-ai-agent-service/internal/domain/**
    - smart-recruit-ai-agent-service/internal/application/**
    - smart-recruit-ai-agent-service/internal/docs/ai_agent_dependency_inventory.md
  reviewed_documents:
    - .knowledge/domains/ai-configuration-governance.md
    - .knowledge/domains/mcp-tool-governance.md
    - .knowledge/domains/agent-skill.md
    - .knowledge/architecture/semantic-retrieval.md
    - .knowledge/pitfalls/embedding-fallback.md
    - .knowledge/pitfalls/mcp-policy-audit.md
    - .knowledge/domains/resume-intelligence.md
    - .knowledge/runbooks/debug-ai-configuration.md
  coverage_gap: false
  reason: TASK-024 localizes AI Agent capability behavior, but active knowledge validation is blocked by pre-existing legacy source_ref debt outside this TASK scope.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 新增/修改文件均在 `smart-recruit-ai-agent-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或全局配置。
- MCP no-policy allow、deny、role/scope、args、rate limit、confirmation、redaction fields 语义有本地策略。
- Skill versioning、Agent Skill selection gate、embedding unavailable fallback、candidate match knockout cap 有测试覆盖。
- Provider credential 未出现明文处理；仅保留 `HasEncryptedCredential` compatibility marker。

## 11. Risks

- Active gRPC runtime 仍未接入本地 MCP/Skill/Embedding/CandidateMatch application services；TASK-025 需要完成 adapters/wiring。
- 本 TASK 不新增 provider、MCP SDK、embedding backfill 或 schema 行为；runtime cutover 时仍需 fake/env-gated tests。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-025 可开始 AI Agent infrastructure/interfaces/runtime/tests 收敛。
- 后续 `.knowledge/**` scope 可用时，应更新 AI configuration、MCP governance、Agent Skill、semantic retrieval、resume intelligence 知识 source_refs。

## 13. Whether the Next TASK Can Start

TASK-024 通过；TASK-025 可以开始。
