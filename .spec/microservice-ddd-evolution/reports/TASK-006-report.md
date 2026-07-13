# TASK Report - TASK-006

## 1. TASK ID

TASK-006 - Interview DDD 骨架与契约盘点。

## 2. Modified File List

实际变更文件：

- `smart-recruit-interview-service/internal/domain/doc.go`
- `smart-recruit-interview-service/internal/domain/model/doc.go`
- `smart-recruit-interview-service/internal/domain/repository/doc.go`
- `smart-recruit-interview-service/internal/domain/service/doc.go`
- `smart-recruit-interview-service/internal/domain/event/doc.go`
- `smart-recruit-interview-service/internal/application/doc.go`
- `smart-recruit-interview-service/internal/application/command/doc.go`
- `smart-recruit-interview-service/internal/application/query/doc.go`
- `smart-recruit-interview-service/internal/application/port/doc.go`
- `smart-recruit-interview-service/internal/application/service/doc.go`
- `smart-recruit-interview-service/internal/infrastructure/doc.go`
- `smart-recruit-interview-service/internal/infrastructure/persistence/doc.go`
- `smart-recruit-interview-service/internal/infrastructure/client/doc.go`
- `smart-recruit-interview-service/internal/infrastructure/mq/doc.go`
- `smart-recruit-interview-service/internal/interfaces/doc.go`
- `smart-recruit-interview-service/internal/interfaces/grpc/doc.go`
- `smart-recruit-interview-service/internal/interfaces/mapper/doc.go`
- `smart-recruit-interview-service/internal/docs/contract_inventory.md`
- `.spec/microservice-ddd-evolution/reports/TASK-006-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-006-evidence.json`

## 3. Change Summary by File

- `internal/domain/**/doc.go`: 创建 Interview domain、model、repository、service、event 包边界，声明后续迁移面试聚合、状态规则、领域事件和 repository port 的目标位置。
- `internal/application/**/doc.go`: 创建 Interview application、command、query、port、service 包边界，声明后续迁移 use case 编排、事务边界、跨上下文 port 和 command/query DTO 的目标位置。
- `internal/infrastructure/**/doc.go`: 创建 Interview infrastructure、persistence、client、mq 包边界，声明后续迁移 GORM adapter、跨上下文 client adapter 和事件/outbox adapter 的目标位置。
- `internal/interfaces/**/doc.go`: 创建 Interview interfaces、grpc、mapper 包边界，声明后续迁移 gRPC server 和 proto mapper 的目标位置。
- `internal/docs/contract_inventory.md`: 新增 Interview 合约盘点，记录当前 protobuf/runtime contract、shared `service.InterviewService` wiring、schedule/update/cancel/batch cancel/feedback/listing 依赖、表归属、事件副作用和测试覆盖缺口。
- `.spec/microservice-ddd-evolution/reports/TASK-006-report.md`: 新增 TASK 报告。
- `.spec/microservice-ddd-evolution/reports/TASK-006-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-006` 通过。

变更均在 TASK-006 allowed files 内：`smart-recruit-interview-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、schema/migration、deployment、package/lockfile 或全局配置。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-001/FR-007/FR-009：按 Offer 之后的顺序为 Interview 创建本地 `domain/application/infrastructure/interfaces/runtime` 目标分层，并记录 Interview schedule、feedback、batch cancellation、listing 与 lifecycle 依赖。
- FR-016/FR-017：记录当前对 shared `service.InterviewService`、shared repositories 和 shared model 的依赖基线，为后续减少/消除直接依赖提供证据。
- CR-001/CR-002/CR-005/CR-006：未修改 HTTP API、protobuf、Gateway route mode、schema、table ownership 或 Outbox/Inbox 结构。
- NFR-002/NFR-003：`smart-recruit-interview-service` 独立 `go test ./...` 通过；新增包均为 doc-only package boundary，没有改变依赖方向。

## 6. SDD Comparison Result

符合 SDD：

- 3.3 Required Migration Order：TASK-006 在 Offer TASK-003 至 TASK-005 完成后开始 Interview 阶段。
- 3.4 Per-Service Migration Pattern：完成步骤 1 和步骤 2，即盘点 protobuf/runtime/repository/table/event/test，并创建服务本地 DDD 目录骨架。
- 8. Compatibility Strategy：不改 protobuf，不切 runtime，不改变 config/health/metrics/trace/Nacos/internal auth/TLS/graceful shutdown。

## 7. Acceptance Comparison Result

- Interview DDD 骨架已存在：`domain`、`application`、`infrastructure`、`interfaces` 及关键子包均已创建。
- schedule/update/cancel/batch cancel/feedback/listing 依赖已记录在 `internal/docs/contract_inventory.md`。
- protobuf、schema、shared module、runtime wiring 和用户可见行为均未改变。
- Offer 迁移成果未回退；本 TASK 未触碰 `smart-recruit-offer-service/**`。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | 对纯未跟踪新增文件输出为空；实际新增文件由 `git status --short` 和本报告完整记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-006` | 0 | passed | `Scope check passed for TASK-006. Changed files: 20` |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；在 `smart-recruit-interview-service` 内运行 `go test ./...` 并通过。 |
| `go test ./...` in `smart-recruit-interview-service` | 0 | passed | Interview service 所有 package 编译/测试通过，新增 doc-only DDD 包均可编译。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 9e3c3d77e12c73b507718015d7882a9c219e6644` exit 1。
- 失败原因仍是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 已不存在；TASK-001 已创建 `.knowledge/inbox/microservice-ddd-evolution-boundaries.md` 作为 candidate，本 TASK scope 不允许修改 `.knowledge/**`。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-interview-service/internal/domain/**
    - smart-recruit-interview-service/internal/application/**
    - smart-recruit-interview-service/internal/infrastructure/**
    - smart-recruit-interview-service/internal/interfaces/**
    - smart-recruit-interview-service/internal/docs/contract_inventory.md
  reviewed_documents:
    - .knowledge/architecture/system-overview.md
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment.md
    - .knowledge/domains/recruitment-lifecycle.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-boundaries.md
  coverage_gap: false
  reason: 当前 active knowledge 尚未正式更新为 smart-recruit-* 微服务根；本 TASK 只能记录 candidate_required，不能越界修改知识库。
```

## 10. Self-review and Repair

独立只读 self-review 第 1 轮 verdict: 不通过。

发现：

- H-001：缺少 TASK-006 report/evidence。
- H-002：report/evidence 缺失导致 knowledge impact 未记录。

修复：

- 新增 `.spec/microservice-ddd-evolution/reports/TASK-006-report.md`。
- 新增 `.spec/microservice-ddd-evolution/reports/TASK-006-evidence.json`。
- 在 report/evidence 中记录 knowledge impact 非阻塞候选债务。

独立只读 self-review 第 2 轮 verdict: 不通过。

发现：

- H-003：report 记录第 1 轮 review 不通过，但 evidence 将 `round: 1` 写成 `verdict: 通过`，缺少修复后的明确通过轮次。

修复：

- 将 evidence review 修正为 `round: 3`、`verdict: 通过`，并完整记录 H-001、H-002、H-003 均已修复。
- 本报告明确保留第 1 轮和第 2 轮不通过记录，并将最终闭环结论记录为第 3 轮通过。

独立只读 self-review 第 3 轮 verdict: 通过。

## 11. Risks

- Interview runtime 仍使用 shared `service.InterviewService`，这是 TASK-006 有意记录的基线，后续 TASK-008 才会切换 active runtime path。
- Interview 当前对 Recruitment/Identity/Notification/OSS/Outbox 有跨上下文读写和事件副作用；本 TASK 只盘点，不迁移。
- Active knowledge 仍有旧路径 source_refs 债务；本 TASK scope 不允许修复 `.knowledge/**`。

## 12. Follow-up Items

- TASK-007 迁移 Interview domain/application，建立本地领域模型、状态规则、repository ports、application command/query 与测试。
- TASK-008 收敛 Interview infrastructure/interfaces/runtime/tests，消除 active runtime 对 shared `service.InterviewService` 的直接依赖。

## 13. Whether the Next TASK Can Start

可以开始 TASK-007。
