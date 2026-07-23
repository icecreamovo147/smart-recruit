# TASK Report - TASK-023

## 1. TASK ID

TASK-023 - AI Agent chat/agent-run/prompt domain/application 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-ai-agent-service/internal/application/command/agent.go`
- `smart-recruit-ai-agent-service/internal/application/dto/agent.go`
- `smart-recruit-ai-agent-service/internal/application/port/agent.go`
- `smart-recruit-ai-agent-service/internal/application/service/agent_service.go`
- `smart-recruit-ai-agent-service/internal/application/service/agent_service_test.go`
- `smart-recruit-ai-agent-service/internal/docs/ai_agent_dependency_inventory.md`
- `smart-recruit-ai-agent-service/internal/domain/model/agent.go`
- `smart-recruit-ai-agent-service/internal/domain/policy/agent.go`
- `smart-recruit-ai-agent-service/internal/domain/policy/agent_test.go`
- `smart-recruit-ai-agent-service/internal/domain/repository/agent.go`
- `.spec/microservice-ddd-evolution/reports/TASK-023-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-023-evidence.json`

## 3. Change Summary by File

- `internal/domain/model/agent.go`: 新增本地 ChatSession、ChatMessage、AgentRun、AgentRunEvent、PromptTemplate、PromptVersion、ModelConfig、ProviderCandidate 和 AuditEvent domain model，并封装 terminal/active 状态 helper。
- `internal/domain/policy/agent.go`: 新增 agent run 状态机、事件顺序校验、创建请求校验、prompt 版本策略和 provider fallback 选择策略。
- `internal/domain/repository/agent.go`: 新增 chat session、agent run、prompt 和 audit sink repository ports。
- `internal/application/command/agent.go`: 新增创建、取消、确认 agent run 以及创建/更新 prompt 的 command DTO。
- `internal/application/dto/agent.go`: 新增 AgentRun、ChatSession、PromptTemplate、PromptVersion、ModelConfig 等 application DTO。
- `internal/application/port/agent.go`: 新增 DurableRunDispatcher 与 ProviderSelector application ports。
- `internal/application/service/agent_service.go`: 新增 AgentRunService 和 PromptService，覆盖 idempotent create、cancel、confirm、audit、durable dispatch、prompt version bump 和 provider fallback 入口。
- `internal/domain/policy/agent_test.go`: 新增 agent run state、stale/duplicate event、prompt version 和 provider fallback 策略测试。
- `internal/application/service/agent_service_test.go`: 新增 durable run create/cancel/confirm 与 prompt versioning 应用服务测试。
- `internal/docs/ai_agent_dependency_inventory.md`: 补充本地 chat、agent run、prompt application boundary 与 TASK-023 已迁移能力说明。
- `.spec/.../TASK-023-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-023-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-023` 通过。

变更均在 TASK-023 allowed files 内：`smart-recruit-ai-agent-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-002 / FR-003：AI Agent chat、agent run、prompt/config 业务语义进入本地 domain/application 分层。
- FR-014：保留现有 agent run durable、cancel、confirm、audit、provider fallback 兼容边界。
- Compatibility：未修改 protobuf、schema、provider 行为、stream API 或 runtime wiring。
- SSR-005：provider fallback、audit、安全相关能力以本地 port/policy 表达，不绕过现有安全边界。

## 6. SDD Comparison Result

符合 SDD：

- 6. Algorithm or Workflow Changes：agent run 状态流转、confirm/cancel、durable dispatch 和 prompt versioning 已本地化为 domain/application 逻辑。
- 9. Error Handling and Fallback Design：新增 provider fallback selector policy，并保留 disabled/provider 不可用时的显式错误。
- 11. Testing Strategy：覆盖 agent run state/durable/prompt 的本地单元测试。
- Implementation boundaries：本 TASK 仅迁移 AI Agent domain/application 层，不修改 infrastructure wiring、protobuf、schema 或 shared module。

## 7. Acceptance Comparison Result

- Chat/session/agent run/prompt/model config 本地化：已完成，新增本地 domain model、repository ports、application commands/DTO/ports/service。
- stream、cancel、confirm、audit、provider fallback 兼容：已完成，cancel/confirm/audit/fallback 进入本地服务和策略；stream 行为不改变，后续 runtime adapter 仍可接入该 application boundary。
- 覆盖 agent run state/durable/prompt 测试：已完成，新增 domain policy 与 application service 测试。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./internal/domain/policy ./internal/application/service` in `smart-recruit-ai-agent-service` | 0 | passed | Agent run state/fallback/prompt policy 与 durable application service targeted tests 通过。 |
| `go test ./...` in `smart-recruit-ai-agent-service` | 0 | passed | AI Agent 全 package 测试通过。 |
| `go run ./cmd/ai-agent-service --check` in `smart-recruit-ai-agent-service` | 0 | passed | `ai-agent-service runtime check passed`。 |
| `git diff --name-only` | 0 | passed | 已记录 tracked diff；新增 Go 文件通过 `git status --short` 与 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-023` | 0 | passed | Scope check passed，变更文件均在 TASK-023 allowed files 内。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 AI Agent module 变更并运行 `go test ./...` 通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-ddd-evolution/reports/TASK-023-evidence.json` | 0 | passed | `evidence_result: PASS` |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree c1cf72200dd5f06b8719834c13236660741f97c2` | 1 | non-blocking | 既有 active knowledge 仍引用已迁移/缺失的旧 `logic-grpc-service` / `web-gin-service` source_refs；本 TASK scope 不允许改 `.knowledge/**`，记录为 candidate_required。 |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-ai-agent-service/internal/domain/**
    - smart-recruit-ai-agent-service/internal/application/**
    - smart-recruit-ai-agent-service/internal/docs/ai_agent_dependency_inventory.md
  reviewed_documents:
    - .knowledge/architecture/agent-runtime.md
    - .knowledge/domains/ai-configuration-governance.md
    - .knowledge/domains/mcp-tool-governance.md
    - .knowledge/architecture/semantic-retrieval.md
    - .knowledge/runbooks/service-binary-convention.md
  coverage_gap: false
  reason: TASK-023 localizes AI Agent chat/run/prompt domain/application behavior, but active knowledge validation is blocked by pre-existing legacy source_ref debt outside this TASK scope.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

自审期间发现并修复 1 个实现质量问题：application service 不再用 `_ =` 忽略 audit sink 错误；agent run 和 prompt audit 写入失败会显式返回错误，Prompt audit 同时补充 `OccurredAt`。

Reviewer 核对结果：

- 新增/修改文件均在 `smart-recruit-ai-agent-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或全局配置。
- Agent run state machine 覆盖 create、planning/running、waiting confirmation、cancel request、terminal 状态和 invalid transition。
- Application service 覆盖 idempotent create、durable dispatch、cancel、confirm、audit 和 prompt version bump。
- Provider fallback 只选择 enabled candidate，不新增 provider 行为。

## 11. Risks

- Active runtime adapter 仍未接入本地 application service；后续 TASK 需要在 infrastructure/interfaces 层完成 wiring。
- Stream 行为保持不变但尚未由本地 application service 直接驱动，后续迁移需避免破坏现有 gRPC stream 兼容。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-024 可开始 AI Agent runtime/infrastructure/interface 收敛。
- 后续 `.knowledge/**` scope 可用时，应更新 AI Agent runtime、MCP、embedding、provider fallback 知识 source_refs。

## 13. Whether the Next TASK Can Start

TASK-023 通过；TASK-024 可以开始。
