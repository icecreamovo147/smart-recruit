# TASK Report - TASK-005

## 1. TASK ID

TASK-005 — 扩展 proto 新增 score breakdown 字段

## 2. Modified File List

- `logic-grpc-service/proto/recruitment.proto`（追加 18 行新字段）
- `logic-grpc-service/recruitment/pb/recruitment.pb.go`（protoc 重生成）
- `web-gin-service/recruitment/pb/recruitment.pb.go`（protoc 重生成 + 与 logic-grpc 保持一致）

## 3. Change Summary by File

### `proto/recruitment.proto`

- `SemanticSkillDebugItem` 末尾追加 8 个字段：
  - `double vector_score = 10;`
  - `double lexical_score = 11;`
  - `double metadata_score = 12;`
  - `double relevance_score = 13;`
  - `double business_boost = 14;`
  - `double final_rank_score = 15;`
  - `string relevance_mode = 16;`
  - `int32 pool_rank = 17;`
- `SemanticMemoryDebugItem` 末尾追加 8 个字段：
  - `double vector_score = 12;`
  - `double lexical_score = 13;`
  - `double metadata_score = 14;`
  - `double relevance_score = 15;`
  - `double business_boost = 16;`
  - `double final_rank_score = 17;`
  - `string relevance_mode = 18;`
  - `int32 pool_rank = 19;`
- `DebugSemanticRetrievalResponse` 末尾追加 2 个字段：
  - `string skill_pool_confidence = 7;`
  - `string memory_pool_confidence = 8;`
- 现有字段（1..9 / 1..11 / 1..6）保持不变；编号不重用。

### `recruitment/pb/recruitment.pb.go`（logic-grpc + web-gin）

- 使用 `/tmp/protoc/bin/protoc` + `protoc-gen-go`（位于 `$HOME/go/bin`）重生成两份 pb 文件。
- 命令：
  ```bash
  /tmp/protoc/bin/protoc --proto_path=proto \
    --go_out=recruitment/pb --go_opt=paths=source_relative \
    --go_opt=Mrecruitment.proto=recruitment/pb \
    recruitment.proto
  cp recruitment/pb/recruitment.pb.go web-gin-service/recruitment/pb/recruitment.pb.go
  ```
- 生成的 Go 代码包含新的结构体字段、`GetXxx()` getter、proto descriptor。
- 两侧 pb 文件大小完全一致（`grep -c` 结果均为 36，含 VectorScore/LexicalScore/FinalRankScore/RelevanceMode/PoolRank/SkillPoolConfidence/MemoryPoolConfidence）。

## 4. Scope Check Result

- 实际修改文件均在 `task-scope.json` 的 `allowedFiles` 列表内（`proto/recruitment.proto`、`recruitment/pb/recruitment.pb.go` × 2）。
- `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-005` 同样会触发前序 TASK 的误报；本 TASK 未触碰 service 业务代码、web-gin handler / router / rpc、frontend。
- 实际范围检查（人工 `git diff --name-only | grep -v .spec`）：
  - `logic-grpc-service/proto/recruitment.proto`（本 TASK 改动）
  - `logic-grpc-service/recruitment/pb/recruitment.pb.go`（本 TASK 改动）
  - `web-gin-service/recruitment/pb/recruitment.pb.go`（本 TASK 改动）
  - `logic-grpc-service/service/agent_skill_service.go`（TASK-004 遗留）
  - `logic-grpc-service/service/agent_skill_selector.go`（TASK-002 遗留）
  - `logic-grpc-service/service/agent_context.go`（TASK-003 遗留）
- `bash .spec/skill-memory-ranking/scripts/agent-check.sh` → 全部通过。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| §5 FR-007 Debug 字段扩展（double / string / int32） | ✅ 类型一致；编号 10..17 / 12..19 / 7..8 严格按 SDD §3.8 |
| §5 FR-009 接口兼容（旧字段不变） | ✅ 旧字段 1..9 / 1..11 / 1..6 全部保留 |
| §7 兼容性需求 | ✅ proto 仅追加新字段，不删除 / 不重用编号 |
| §11 AC-007 | ✅ `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` / `DebugSemanticRetrievalResponse` 新增字段，旧字段保持 |

## 6. SDD Comparison Result

- SDD §3.8 字段定义：完全对齐。
- SDD §3.10 兼容映射：proto `score` 字段值 = `final_rank_score`，旧字段不变 → 仍成立。
- SDD §8 兼容性策略：proto 字段编号不重用 → 已遵守。
- SDD §13 实现边界：`recruitment_grpc.pb.go` 不变 → 已遵守。
- 两侧 `recruitment.pb.go` 字段一致 → 已遵守。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 proto 字段追加成功，protoc 编译通过 | ✅ |
| AC-002 pb 文件同步更新；web-gin / logic-grpc 两侧 pb 字段一致 | ✅ |
| AC-003 未触碰现有字段编号与类型 | ✅ |
| AC-004 `go build ./...` 在 logic-grpc-service 与 web-gin-service 全部通过 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go build ./...` (logic-grpc-service) | 无输出（成功） |
| `go build ./...` (web-gin-service) | 无输出（成功） |
| `go test ./...` (logic-grpc-service) | 全部 `ok` |
| `bash .spec/skill-memory-ranking/scripts/agent-check.sh` | 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck） |
| `grep -c "VectorScore\|...\|MemoryPoolConfidence" recruitment/pb/recruitment.pb.go` | 36（logic-grpc 与 web-gin 一致） |

## 9. Risks

- **`recruitment.pb.go` 大幅 diff**：protoc 重生成的代码块顺序、字段顺序、空格等可能与项目历史版本略有差异，导致 diff 达 11000+ 行。这是 protoc 输出格式的正常表现，不影响功能；后续如需"最小 diff"，可改用 `protoc-gen-go-vendor` 等增量生成工具。
- **protoc 二进制来源**：`/tmp/protoc/bin/protoc` 是当前环境内提供的版本（`libprotoc 25.3`），不保证与团队 CI 使用的版本完全一致。TASK-005 完成后建议在 CI 中跑一次完整 protoc 流水线验证。
- **`recruitment_grpc.pb.go` 未触碰**：按 SDD §13 边界要求；与 proto 变更无关，service / handler 代码继续使用既有 gRPC 接口。

## 10. Follow-up Items

- TASK-006：在 `agent_skill_service.go` 读取 `lastDebugPoolConfidence` 与 `lastDebugMemoryRankings`，填充新生成的 proto 字段。
- CI 流水线：建议补一次 protoc 重生成 + 两侧 pb diff 检查。
- 文档：在 `docs/` 记录新字段（可选，本期不强制）。

## 11. Whether the Next TASK Can Start

✅ TASK-006 可以开始。`SemanticSkillDebugItem.VectorScore..PoolRank` / `SemanticMemoryDebugItem.VectorScore..PoolRank` / `DebugSemanticRetrievalResponse.SkillPoolConfidence..MemoryPoolConfidence` 字段已就位；TASK-006 可在 `agent_skill_service.go` 中直接使用。
