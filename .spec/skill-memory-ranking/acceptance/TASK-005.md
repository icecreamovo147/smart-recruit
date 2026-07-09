# Acceptance - TASK-005

## TASK Summary

在 `recruitment.proto` 末尾追加 score breakdown 与 `pool_confidence` 字段。同步更新 `logic-grpc-service/recruitment/pb/recruitment.pb.go` 与 `web-gin-service/recruitment/pb/recruitment.pb.go`。本 TASK 涉及公共契约变更，**开始编码前必须先与用户确认**。

## SPEC References

- SPEC §5 FR-007：Debug 字段扩展。
- SPEC §5 FR-009：接口兼容。
- SPEC §7 兼容性需求。
- SPEC §11 AC-007。

## SDD References

- SDD §3.8 Debug 字段。
- SDD §4 数据结构变化。
- SDD §5 API 与接口变化。
- SDD §8 兼容性策略。

## Acceptance Criteria

- [ ] AC-001：proto 字段追加成功，旧字段编号、类型、语义全部不变。
- [ ] AC-002：`SemanticSkillDebugItem` 追加 8 个字段（10..17）。
- [ ] AC-003：`SemanticMemoryDebugItem` 追加 8 个字段（12..19）。
- [ ] AC-004：`DebugSemanticRetrievalResponse` 追加 2 个字段（7..8）。
- [ ] AC-005：pb 文件同步更新（logic-grpc-service 与 web-gin-service 两侧字段一致）。
- [ ] AC-006：未修改 `recruitment_grpc.pb.go`、handler、router、rpc。
- [ ] AC-007：`go build ./...` 在 logic-grpc-service 与 web-gin-service 全部通过。

## Required Checks

- [ ] `cd logic-grpc-service && go build ./...` 通过。
- [ ] `cd web-gin-service && go build ./...` 通过。
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/proto/recruitment.proto` 与 `**/recruitment/pb/recruitment.pb.go`。
- [ ] `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-005` 提示无越界。

## Manual Verification, if needed

- 阅读 `recruitment.proto`，确认追加字段命名（snake_case）、编号（10+ / 12+ / 7+）、类型（double / string / int32）。
- 确认两侧 pb 字段顺序与 proto 一致。
- 确认 web-gin handler / router / rpc 不需要额外修改即可透传新字段。

## Out-of-Scope

- 不得修改 `recruitment_grpc.pb.go`。
- 不得修改 web-gin handler / router / rpc。
- 不得修改前端类型（由 TASK-007 处理）。
- 不得修改 `DebugSemanticRetrieval` 业务逻辑（由 TASK-006 处理）。
- 不得在 proto 中删除 / 修改 / 重用现有字段编号。

## Hard Stop

本 TASK 涉及公共契约变更，开始编码前必须与用户确认：

1. 字段命名是否一致（`vector_score` / `lexical_score` / `metadata_score` / `relevance_score` / `business_boost` / `final_rank_score` / `relevance_mode` / `pool_rank` / `skill_pool_confidence` / `memory_pool_confidence`）。
2. 字段编号分配是否与 SDD §3.8 一致。
3. 是否使用 protoc 重生成（推荐）；如不可用，按字段手工同步最小集合。
4. 是否需要同步更新文档（`docs/`）中的 proto 章节。
