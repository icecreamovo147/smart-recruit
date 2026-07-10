# TASK Report - TASK-003

## 1. TASK ID

TASK-003

## 2. Modified File List

- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/service/agent_skill_service_test.go`

## 3. Change Summary by File

- `agent_skill_service.go`: `semanticDebugSkillScores` 改用 `SearchWithMeta()` 获取 request-local `SearchMeta`；`DebugSemanticRetrieval` 使用 Skill 搜索返回值构建响应。
- `agent_skill_service_test.go`: 保留并验证 Skill 阶段 metadata 不被 Memory 检索覆盖的回归测试。
- `embedding_service.go`: 新增 `SearchWithMeta()` / `searchWithMeta()` / `buildSearchMeta()`，由单次搜索直接构造并返回元数据，不再依赖事后读取 `LastSearchMeta()`。
- `embedding_service_test.go`: 新增 `TestEmbeddingServiceSearchWithMetaIsRequestLocal`。

## 4. Scope Check Result

`bash .spec/agent-skill-review-fixes/scripts/check-task-scope.sh TASK-003` 在仅含本 TASK 文件时可验证通过；合并工作区需按 TASK 分批检查。

## 5. SPEC Comparison Result

满足 FR-003、AC-005：元数据由本次 `SearchWithMeta` 调用直接返回，避免依赖可变 service-level `lastSearchMeta`，并发请求间不会互相污染。

## 6. SDD Comparison Result

与 SDD「SearchWithMeta 返回 {Results, Meta}」一致；`Search()` 内部委托 `searchWithMeta()` 并保持 `LastSearchMeta()` 兼容。

## 7. Acceptance Comparison Result

- AC-005: 通过（request-local meta + 集成/单元测试）。
- AC-006: 通过。

## 8. Test Commands and Results

```bash
cd logic-grpc-service && go test ./service -run 'TestEmbeddingServiceSearchWithMetaIsRequestLocal|TestAgentSkillServiceDebugSemanticRetrievalUsesSkillSearchMetadata' -count=1
bash .spec/agent-skill-review-fixes/scripts/check-task-scope.sh TASK-003
```

## Repair Summary

### Failed Check

- self-review: TASK-003 仍通过 `embeddingMetaSnapshot(LastSearchMeta)` 间接读取共享状态。
- self-review: `check-task-scope.sh` Node heredoc 参数传递错误导致脚本无法运行。

### Root Cause

- Skill metadata 在 `Search()` 之后从 service 全局状态读取，存在同请求 Memory 覆盖与同 service 并发覆盖风险。
- `node <<'NODE' "$ARGS"` 会把首个参数当作脚本路径。

### Files Changed

- `.spec/agent-skill-review-fixes/scripts/check-task-scope.sh`
- `logic-grpc-service/service/embedding_service.go`
- `logic-grpc-service/service/embedding_service_test.go`
- `logic-grpc-service/service/agent_skill_service.go`

### Fix Summary

- 脚本改为 `node - "$TASK_ID" ... <<'NODE'`。
- 实现 `SearchWithMeta()` 返回 request-local `SearchMeta`；语义调试路径改用该 API。

### Re-run Commands

见上方 Test Commands。

### Remaining Risks

- `SearchObjects` 仍使用 `LastSearchMeta` 写入模式；仅 Skill debug 路径改为 request-local，符合 TASK 范围。

## 9. Risks

- 响应中的通用 embedding 元数据字段仍只描述 Skill 阶段；若未来需分别展示 Memory 元数据，需单独设计字段。

## 10. Follow-up Items

- 无阻塞项。

## 11. Whether the Next TASK Can Start

无后续 TASK；pipeline 可汇总完成。
