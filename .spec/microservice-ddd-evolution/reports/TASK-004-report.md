# TASK Report - TASK-004

## 1. TASK ID

TASK-004 - Offer domain/application 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-offer-service/internal/domain/model/status.go`
- `smart-recruit-offer-service/internal/domain/model/offer.go`
- `smart-recruit-offer-service/internal/domain/model/offer_test.go`
- `smart-recruit-offer-service/internal/domain/event/offer_event.go`
- `smart-recruit-offer-service/internal/domain/repository/offer_repository.go`
- `smart-recruit-offer-service/internal/domain/service/offer_policy.go`
- `smart-recruit-offer-service/internal/application/command/commands.go`
- `smart-recruit-offer-service/internal/application/query/queries.go`
- `smart-recruit-offer-service/internal/application/port/offer_ports.go`
- `smart-recruit-offer-service/internal/application/service/offer_service.go`
- `smart-recruit-offer-service/internal/application/service/offer_service_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-004-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-004-evidence.json`

## 3. Change Summary by File

- `smart-recruit-offer-service/internal/domain/model/status.go`: 本地化 Offer 相关 application status key、label 与状态迁移校验，供 Offer 用例使用。
- `smart-recruit-offer-service/internal/domain/model/offer.go`: 新增本地 Offer domain model，承载草稿创建、草稿更新、发送快照、撤回、接受、拒绝、过期和候选人匹配规则。
- `smart-recruit-offer-service/internal/domain/model/offer_test.go`: 覆盖 Offer domain 生命周期、发送快照、撤回规则、接受/过期规则和 application transition。
- `smart-recruit-offer-service/internal/domain/event/offer_event.go`: 新增本地 Offer domain event type 与事件构造。
- `smart-recruit-offer-service/internal/domain/repository/offer_repository.go`: 新增本地 Offer repository port、transaction writer、详情与候选人列表 DTO。
- `smart-recruit-offer-service/internal/domain/service/offer_policy.go`: 新增 Offer application status transition policy，封装 create/send/withdraw/accept/reject 所需迁移。
- `smart-recruit-offer-service/internal/application/command/commands.go`: 新增 Offer command DTO。
- `smart-recruit-offer-service/internal/application/query/queries.go`: 新增 Offer query DTO。
- `smart-recruit-offer-service/internal/application/port/offer_ports.go`: 新增 Authorizer、ApplicationSnapshotReader、ApplicationLifecycle、OutboxPublisher、Clock 等 application port。
- `smart-recruit-offer-service/internal/application/service/offer_service.go`: 新增本地 Offer application service，保持 create/update/send/withdraw/accept/reject/get/list/list-events 旧语义的用例编排、事务意图、事件写入和 outbox 发布时机。
- `smart-recruit-offer-service/internal/application/service/offer_service_test.go`: 覆盖 create、update、send、withdraw、accept、reject、get self-service、list events 等关键 application 规则。
- `.spec/microservice-ddd-evolution/reports/TASK-004-report.md`: 新增 TASK 报告。
- `.spec/microservice-ddd-evolution/reports/TASK-004-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-004` 通过。

变更均在 TASK-004 allowed files 内：`smart-recruit-offer-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-002：Offer `domain` 层承载本地实体、状态机、领域事件、领域服务和 repository port，未依赖 GORM/proto/gRPC。
- FR-003：Offer `application` 层承载 command/query 编排、事务边界、权限端口、跨上下文 application lifecycle port 和 outbox 发布时机。
- FR-008：Offer 作为第一个服务自治 DDD 试点，已迁出 domain/application 层的 lifecycle、event 和 repository interface。
- NFR-002/NFR-003：`smart-recruit-offer-service` 独立 `go test ./...` 通过，依赖方向保持 domain -> 无外层、application -> domain/port。
- CR-001 至 CR-003：未修改 HTTP API、protobuf、Gateway route mode 或 gRPC contract。

未修改 shared module、protobuf、schema、deployment、package/lockfile 或 public API 行为。

## 6. SDD Comparison Result

符合 SDD：

- 3.4 Per-Service Migration Pattern：完成 Offer domain model/state machine/domain event/repository interface 与 application command/query handler 的本地迁移。
- 6. Algorithm or Workflow Changes：保持旧 Offer create/update/send/withdraw/accept/reject/list 语义，将状态机迁入 domain/application 测试覆盖。
- 9. Error Handling and Fallback Design：domain 返回 typed/domain error，application 保留错误上下文并通过 port 暴露给后续 interfaces 层映射。

## 7. Acceptance Comparison Result

- Offer domain model/policy/event/repository port 已本地化。
- Application command/query 保持现有 Offer 语义：鉴权、scope、application status transition、Offer event、candidate/staff notification 和 email outbox 时机均按 legacy 行为迁移为 port 编排。
- Domain 不依赖 GORM/proto/gRPC；`node scripts/check-backend-boundaries.mjs` 通过。
- Runtime 切换、删除 shared 旧实现、proto/schema 修改均未发生，符合 Out-of-Scope。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | 无输出；当前 TASK 新增文件仍未跟踪，真实文件列表由本报告和 evidence 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-004` | 0 | passed | `Scope check passed for TASK-004. Changed files: 13` |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；在 `smart-recruit-offer-service` 内运行 `go test ./...` 并通过。 |
| `go test ./...` in `smart-recruit-offer-service` | 0 | passed | Offer service 所有 package 编译/测试通过；domain/application 新测试通过。 |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree edc2507a0546807e426cfb65331ad3481431e9ea` exit 1。
- 失败原因仍是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 已不存在；TASK-001 已创建 `.knowledge/inbox/microservice-ddd-evolution-boundaries.md` 作为 candidate。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-offer-service/internal/domain/model/**
    - smart-recruit-offer-service/internal/domain/event/**
    - smart-recruit-offer-service/internal/domain/repository/**
    - smart-recruit-offer-service/internal/domain/service/**
    - smart-recruit-offer-service/internal/application/**
  reviewed_documents:
    - .knowledge/architecture/system-overview.md
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/pitfalls/status-notification-drift.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-boundaries.md
  coverage_gap: false
  reason: 当前 active knowledge 尚未正式更新为 smart-recruit-* 微服务根，继续沿用 TASK-001 candidate 记录。
```

## 10. Risks

- 当前 TASK 尚未切换 runtime，新增本地 domain/application 暂不改变生产路径；真正 wiring 与 compatibility 风险留到 TASK-005 验证。
- Application ports 尚无 infrastructure adapter，实现将在 TASK-005 内完成。
- 知识库 active 文档仍存在 pre-existing source_ref 债务，本 TASK 仅记录 candidate_required，不扩大 scope 修改正式知识。

## 11. Follow-up Items

- TASK-005 迁移 Offer infrastructure、gRPC interfaces、runtime wiring 和集成测试。
- 后续 TASK 完成 runtime 切换后，再清理或标记 shared Offer legacy 实现。

## 12. Whether the Next TASK Can Start

可以开始 TASK-005。独立只读 self-review verdict: 通过。
