# Acceptance - TASK-FU-001

## TASK Summary

把 `agent_skill_service.go` 内 `lastDebugPoolConfidence` / `lastDebugMemoryRankings` 包级变量改为 `context.Context` 私有 key 传递；消除 `DebugSemanticRetrieval` 并发请求下的状态污染。

## SPEC References

- SPEC §5 FR-FU-001：context-based debug 状态
- SPEC §7 兼容性
- SPEC §11 AC-FU-001

## SDD References

- SDD §3.1 context-based debug 状态
- SDD §11 测试策略

## Acceptance Criteria

- [ ] AC-001：包级 `var lastDebugPoolConfidence` / `var lastDebugMemoryRankings` 删除
- [ ] AC-002：新增 `withDebugPoolConfidence` / `getDebugPoolConfidenceFromContext` / `withDebugMemoryRankings` / `getDebugMemoryRankingsFromContext` 4 个 helper
- [ ] AC-003：`DebugSemanticRetrieval` 内部用 ctx helper 传递状态
- [ ] AC-004：`semanticDebugMemories` 内部用 ctx helper 写入
- [ ] AC-005：现有 2 个 TASK-006 既有测试 `TestAgentSkillServiceDebugSemanticRetrievalPopulatesScoreBreakdown` / `...FallbackPopulatesBreakdown` 通过
- [ ] AC-006：`go test ./...` 通过

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过
- [ ] `gofmt -l logic-grpc-service/service/agent_skill_service.go` 无输出
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/service/agent_skill_service.go` / `agent_skill_service_test.go`
- [ ] `bash .spec/skill-memory-ranking-followups/scripts/check-task-scope.sh TASK-FU-001` 提示无越界

## Manual Verification, if needed

- 阅读 `agent_skill_service.go` 中 `withDebugPoolConfidence` / `getDebugPoolConfidenceFromContext` 等 helper。
- 跑 `TestDebugContextStateIsolation` 验证并发隔离。
- 跑 `TestDebugContextStateEmptyDefaults` 验证 nil ctx 默认值。

## Out-of-Scope

- 不得修改 proto / pb / config / 前端。
- 不得修改 TASK-001..008 已落地的算法公式。
- 不得修改 `DebugSemanticRetrieval` / `semanticDebugMemories` / `semanticDebugSkillsToPB` 函数签名。
