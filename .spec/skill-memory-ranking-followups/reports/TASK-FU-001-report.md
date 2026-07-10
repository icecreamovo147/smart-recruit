# TASK Report - TASK-FU-001

## 1. TASK ID

TASK-FU-001 — context-based debug 状态

## 2. Modified File List

- `logic-grpc-service/service/agent_skill_service.go`（修改 4 处：删除 2 个 `var` + 4 个旧 helper；新增 4 个 ctx helper + 1 个私有 key 类型；`semanticDebugMemories` 签名返回 ctx）
- `logic-grpc-service/service/agent_skill_service_test.go`（追加 2 个 ctx 隔离 / 默认值测试）
- `.spec/skill-memory-ranking-followups/scripts/agent-check.sh`（用 python3 yaml.safe_load 替代不合适的 `bash -n` 检查）

## 3. Change Summary by File

### `agent_skill_service.go`

1. **删除 2 个包级 `var`**：
   - `var lastDebugPoolConfidence debugPoolConfidenceView`
   - `var lastDebugMemoryRankings []RankedMemoryItem`
2. **删除 4 个旧 helper**：
   - `stashDebugPoolConfidence`
   - `lastDebugPoolConfidenceForTest`
   - `stashLastDebugMemoryRankings`
   - `getLastDebugMemoryRankings`
3. **新增私有 key 类型**：
   ```go
   type debugContextKey struct{}
   var (
       debugPoolConfidenceKey  = debugContextKey{}
       debugMemoryRankingsKey = debugContextKey{}
   )
   ```
4. **新增 4 个 ctx helper**：
   - `withDebugPoolConfidence(ctx, v) context.Context` — 写入
   - `getDebugPoolConfidenceFromContext(ctx) debugPoolConfidenceView` — 读取（缺失时返回 zero value）
   - `withDebugMemoryRankings(ctx, items) context.Context` — 写入（nil 自动转 empty slice）
   - `getDebugMemoryRankingsFromContext(ctx) []RankedMemoryItem` — 读取（缺失时返回 nil）
5. **`semanticDebugMemories` 签名变化**：返回值从 `[]*pb.SemanticMemoryDebugItem` 改为 `([]*pb.SemanticMemoryDebugItem, context.Context)`；内部用 `ctx = withDebugMemoryRankings(ctx, rankings)` 把 rankings 写入 ctx。
6. **`DebugSemanticRetrieval` 调整**：
   - `memories, debugCtx := s.semanticDebugMemories(ctx, req, limit, &embeddingAvailable)`
   - `memoryRankings := getDebugMemoryRankingsFromContext(debugCtx)`
   - `poolView := computeDebugPoolConfidence(selected, memoryRankings)`
   - `debugCtx = withDebugPoolConfidence(debugCtx, poolView)`
   - proto response 字段从 `poolView` 直接读取
7. **`computeDebugPoolConfidence` 签名不变**：仍接受 `(selected, memoryRankings)`；只改了调用方传 ctx 读出的 rankings。

### `agent_skill_service_test.go`

- `TestDebugContextStateIsolation`：4 个子用例（concurrent_contexts_isolated / memory_rankings_isolated / simulated_concurrent_debug_requests）
- `TestDebugContextStateEmptyDefaults`：4 个子用例（nil_ctx_returns_zero_pool / nil_ctx_returns_nil_rankings / write_nil_items_stores_empty_slice / child_ctx_inherits_parent）

### `agent-check.sh`

- 把不合适的 `bash -n .github/workflows/ci.yml`（bash 语法检查应用于 YAML 文件会误报）替换为 `python3 -c 'import yaml; yaml.safe_load(...)'`；python3 不可用时跳过。

## 4. Scope Check Result

- 实际 TASK-FU-001 业务改动：`agent_skill_service.go` / `agent_skill_service_test.go` / `agent-check.sh`，均符合 `task-scope.json` 的 `allowedFiles`（`agent_skill_service.go` / `agent_skill_service_test.go` + harness bookkeeping + task-scope.json）。
- `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` → 全部通过（含修正后的 ci.yml yaml parse）。
- 前序 TASK-001..008 的未 commit 改动仍触发 `check-task-scope.sh` 误报；本 TASK 实际未触碰这些文件。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-FU-001 context-based debug 状态 | ✅ 4 个 ctx helper + 私有 key 类型 |
| §7 兼容性 | ✅ `DebugSemanticRetrieval` 公开签名不变；`semanticDebugMemories` 私有签名变化（TASK-FU-001 spec 允许） |
| §11 AC-FU-001 | ✅ 包级 var 删除；4 个 helper 实现；现有 TASK-006 既有测试通过 |

## 6. SDD Comparison Result

- SDD §3.1 context-based debug 状态：完全对齐。
- SDD §8 兼容性策略：ctx helper 替换包级 var；调用方同步替换。
- SDD §11 测试策略：4 类测试（隔离 / 默认值 / 并发 / 子 ctx 继承）全部覆盖。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 包级 `var lastDebugPoolConfidence` / `var lastDebugMemoryRankings` 删除 | ✅ |
| AC-002 新增 4 个 ctx helper | ✅ |
| AC-003 `DebugSemanticRetrieval` 内部用 ctx helper | ✅ |
| AC-004 `semanticDebugMemories` 内部用 ctx helper 写入 | ✅ |
| AC-005 现有 2 个 TASK-006 既有测试通过 | ✅ |
| AC-006 `go test ./...` 通过 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test -run 'TestDebugContextStateIsolation\|TestDebugContextStateEmptyDefaults' ./service/... -v` | 全部 `PASS` |
| `go test ./...` (logic-grpc-service) | 全部 `ok` |
| `gofmt -l` 两个文件 | 无输出（已 gofmt） |
| `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` | 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck / ci.yml yaml parse） |

## 9. Risks

- **`semanticDebugMemories` 签名变化**：从返回 `[]*pb.SemanticMemoryDebugItem` 改为 `([]*pb.SemanticMemoryDebugItem, context.Context)`。由于该函数是包内私有（`s.semanticDebugMemories`），无外部调用方。但若未来有其他文件调用，需要同步更新。
- **ctx 写入需更新返回值**：调用方如果忽略返回的 ctx，写入会丢失。本 TASK 中 `DebugSemanticRetrieval` 正确使用了 `debugCtx` 变量替换原 `ctx`。
- **AgentSkillService 多个 gRPC handler 共享**：每个 handler 独立 ctx，状态不会跨请求污染；但同一 handler 内的串行调用（如 `selectAgentSkillsWithSemantic` → `semanticDebugMemories`）会通过 ctx 共享状态，符合预期。

## 10. Follow-up Items

- TASK-FU-002：CI 流水线 protoc 步骤。
- TASK-FU-003：debug 视图 UI 升级。
- TASK-FU-004：配置化权重。
- 后续 PR：可考虑把 `debugPoolConfidenceView` / ctx helper 抽到独立子包，便于其他 debug endpoint 复用。

## 11. Whether the Next TASK Can Start

✅ TASK-FU-002 可以开始。ctx-based debug 状态已落地，无依赖。
