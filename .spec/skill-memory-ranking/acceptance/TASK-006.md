# Acceptance - TASK-006

## TASK Summary

在 `agent_skill_service.go` 的 `DebugSemanticRetrieval` 路径下，将新 score breakdown 字段与 `pool_confidence` 填充到 gRPC response。保持旧 `score` 字段值 = `final_rank_score`，兼容现有前端消费方。

## SPEC References

- SPEC §5 FR-007：Debug 字段扩展。
- SPEC §5 FR-008：日志可观测。
- SPEC §5 FR-009：接口兼容。
- SPEC §7 兼容性需求。
- SPEC §11 AC-007。

## SDD References

- SDD §3.5 Skill 排序实现。
- SDD §3.6 Memory 排序实现。
- SDD §3.8 Debug 字段。
- SDD §8 兼容性策略。
- SDD §10 可观测性设计。

## Acceptance Criteria

- [ ] AC-001：`DebugSemanticRetrieval` 响应中每个 Skill item 携带新字段（`vector_score / lexical_score / metadata_score / relevance_score / business_boost / final_rank_score / relevance_mode / pool_rank`）。
- [ ] AC-002：每个 Memory item 携带对应字段。
- [ ] AC-003：response 顶层携带 `skill_pool_confidence` / `memory_pool_confidence`。
- [ ] AC-004：现有 debug 字段（`embedding_available / fallback_reason`）保持。
- [ ] AC-005：`score` 字段值 = `final_rank_score`，保持兼容。
- [ ] AC-006：降级路径下 `relevance_mode = "lexical_metadata"`、`vector_score = 0`。
- [ ] AC-007：`go test ./...` 通过。
- [ ] AC-008：未修改 `DebugSemanticRetrieval` 函数签名、`SemanticSkillDebugItem` / `SemanticMemoryDebugItem` 现有字段。

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过。
- [ ] `gofmt -l logic-grpc-service/service/agent_skill_service.go` 无输出。
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/service/agent_skill_service.go` / `agent_skill_selector.go` / `agent_context.go` / `agent_skill_service_test.go`。
- [ ] `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-006` 提示无越界。

## Manual Verification, if needed

- 阅读 `semanticDebugSkillsToPB` / `semanticDebugMemories`，确认新字段被填充。
- 阅读 `DebugSemanticRetrieval`，确认 `skill_pool_confidence` / `memory_pool_confidence` 从 internal struct 写入 proto 字段。
- 跑 `TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry`，确认 Memory 排序与新模型一致。
- 跑新增的 trace 字段断言测试，确认所有新字段非 0 / 非空。

## Out-of-Scope

- 不得修改 `DebugSemanticRetrieval` 函数签名。
- 不得修改 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` 现有字段。
- 不得修改 proto / pb（由 TASK-005 处理）。
- 不得修改前端类型（由 TASK-007 处理）。
- 不得在 debug 视图引入新 UI。
