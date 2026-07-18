# TASK Report - TASK-005

## 1. TASK ID

TASK-005 - Offer infrastructure/interfaces/runtime/tests 收敛。

## 2. Modified File List

实际变更文件：

- `smart-recruit-offer-service/cmd/offer-service/main.go`
- `smart-recruit-offer-service/internal/docs/contract_inventory.md`
- `smart-recruit-offer-service/internal/runtime/runtime.go`
- `smart-recruit-offer-service/internal/runtime/runtime_test.go`
- `smart-recruit-offer-service/internal/infrastructure/persistence/offer_repository.go`
- `smart-recruit-offer-service/internal/infrastructure/client/application_adapter.go`
- `smart-recruit-offer-service/internal/infrastructure/client/authorizer.go`
- `smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher.go`
- `smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher_test.go`
- `smart-recruit-offer-service/internal/interfaces/mapper/offer_mapper.go`
- `smart-recruit-offer-service/internal/interfaces/grpc/offer_server.go`
- `smart-recruit-offer-service/internal/interfaces/grpc/offer_server_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-005-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-005-evidence.json`

## 3. Change Summary by File

- `cmd/offer-service/main.go`: 移除 `service.NewServices`/shared `OfferService` runtime wiring，改为构造 Offer 本地 application service、persistence/client/mq adapters 和 local gRPC server；保留 config、DB、Redis health、metrics、trace、Nacos、internal auth、TLS 和 graceful shutdown 路径。
- `internal/docs/contract_inventory.md`: 更新 Offer runtime 盘点，记录 direct shared `service.OfferService` wiring 已清除，剩余 shared repository/model 仅为 infrastructure 过渡债务。
- `internal/runtime/runtime.go`: runtime 直接接收并注册本地 `pb.OfferServiceServer`。
- `internal/runtime/runtime_test.go`: 调整 fake server 为 `pb.OfferServiceServer` 形态，继续验证 gRPC service registration。
- `internal/infrastructure/persistence/offer_repository.go`: 新增 GORM/shared repository adapter，实现本地 Offer repository port，并通过 context 传递同一个 transaction 给 lifecycle/outbox adapters。
- `internal/infrastructure/client/application_adapter.go`: 新增 application snapshot 与 application lifecycle adapter，保留 application status 更新、legacy status 映射、current round close 与 transition audit 语义。
- `internal/infrastructure/client/authorizer.go`: 新增 authorizer adapter，使用 gRPC metadata、RBAC permission 与 data scope 检查实现 Offer application ports。
- `internal/infrastructure/mq/outbox_publisher.go`: 新增 event_outbox publisher adapter，写入 legacy-compatible envelope payload、idempotency key 和 notification/email consumer 兼容字段。
- `internal/infrastructure/mq/outbox_publisher_test.go`: 覆盖 outbox envelope/idempotency/nested payload 兼容和 tx context 写入分支。
- `internal/interfaces/mapper/offer_mapper.go`: 新增 proto/domain DTO 时间与 Offer/Event 映射。
- `internal/interfaces/grpc/offer_server.go`: 新增 local Offer gRPC server，完成 proto request 到 application command/query 的映射和兼容 response/error mapping。
- `internal/interfaces/grpc/offer_server_test.go`: 覆盖 expires_at 校验、Create success mapping、NotFound bad request、candidate mismatch forbidden 和 event list DTO 映射。
- `.spec/microservice-ddd-evolution/reports/TASK-005-report.md`: 新增 TASK 报告。
- `.spec/microservice-ddd-evolution/reports/TASK-005-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-005` 通过。

变更均在 TASK-005 allowed files 内：`smart-recruit-offer-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-004/FR-005/FR-006：新增 Offer persistence/client/mq adapters、gRPC interface adapter，并由 runtime 装配。
- FR-008/FR-016：Offer runtime 不再直接依赖 shared `service.OfferService`；剩余 shared repository/model 使用限定在 infrastructure 过渡适配层。
- CR-001 至 CR-006：未修改 HTTP API、protobuf、Gateway route mode、schema、部署配置或 Outbox 表结构。
- NFR-002/NFR-003：`smart-recruit-offer-service` 独立 `go test ./...` 通过，domain/application 不反向依赖 infrastructure/interfaces。

## 6. SDD Comparison Result

符合 SDD：

- 3.4 Per-Service Migration Pattern：完成 Offer GORM persistence、outbox adapter、gRPC interface 和 runtime wiring。
- 8. Compatibility Strategy：保留 protobuf response shape、runtime config/health/metrics/trace/Nacos/internal auth/TLS/graceful shutdown。
- 11. Testing Strategy：补充 gRPC adapter 行为测试，继续运行 Offer 服务 `go test ./...` 和 table ownership check。

## 7. Acceptance Comparison Result

- Offer runtime 使用本地 `interfaces/grpc.Server` 注册 `OfferService`。
- gRPC 行为兼容：请求解析、成功响应、bad request/forbidden/gRPC error 分类按 legacy 语义保持。
- 对共享 `service.OfferService` 的 active runtime 直接依赖已清除；剩余 shared repository/model 作为 infrastructure temporary debt 已记录在 `internal/docs/contract_inventory.md`。
- 未删除 shared 旧实现，未修改 protobuf/schema，符合 Out-of-Scope。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | 输出 4 个已跟踪修改文件；另有 7 个未跟踪新增实现/测试文件，报告已完整列出。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-005` | 0 | passed | `Scope check passed for TASK-005. Changed files: 14` |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；在 `smart-recruit-offer-service` 内运行 `go test ./...` 并通过。 |
| `go test ./...` in `smart-recruit-offer-service` | 0 | passed | Offer service 所有 package 编译/测试通过，包含新增 `interfaces/grpc` 与 `infrastructure/mq` 兼容测试。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree d4c3426f8f4a50277f3e58868577745f72d2414f` exit 1。
- 失败原因仍是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 已不存在；TASK-001 已创建 `.knowledge/inbox/microservice-ddd-evolution-boundaries.md` 作为 candidate。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-offer-service/cmd/offer-service/main.go
    - smart-recruit-offer-service/internal/runtime/**
    - smart-recruit-offer-service/internal/infrastructure/**
    - smart-recruit-offer-service/internal/interfaces/**
    - smart-recruit-offer-service/internal/docs/contract_inventory.md
  reviewed_documents:
    - .knowledge/architecture/system-overview.md
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/runbooks/service-binary-convention.md
    - .knowledge/domains/notification-outbox.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-boundaries.md
  coverage_gap: false
  reason: 当前 active knowledge 尚未正式更新为 smart-recruit-* 微服务根，继续沿用 TASK-001 candidate 记录。
```

## 10. Risks

- Offer application lifecycle adapter 仍在单 MySQL 过渡模式下直接更新 `applications`，已记录为 Recruitment owner contract 迁移前的临时债务。
- Outbox payload 保持 consumer 兼容字段，但本地 publisher 暂不启动独立 poller；当前架构仍依赖 existing Worker/Outbox 处理路径。
- Shared repository/model 仍作为 infrastructure adapter 依赖，后续 cleanup/shared-kernel 任务需要继续收敛。

## 11. Follow-up Items

- TASK-006 开始 Interview DDD 骨架与契约盘点。
- 后续 shared cleanup TASK 再处理 shared Offer legacy implementation 删除或 deprecation。

## 12. Whether the Next TASK Can Start

可以开始 TASK-006。独立只读 self-review 第 2 轮 verdict: 通过。

## Repair Summary

### Failed Check

独立只读 self-review 第 1 轮 verdict: 不通过。

### Root Cause

- RF-001：本地 outbox publisher 初版手写了扁平 payload，未保持 legacy standardized envelope、nested payload 和 default idempotency key 语义。
- RF-002：gRPC error mapping 初版过于通用，未按 Accept/Reject/Get/ListOfferEvents/Send 等 endpoint 保留 legacy response message。
- RF-003：测试只覆盖部分 code mapping，未覆盖 outbox envelope 与 legacy endpoint message。

### Files Changed

- `smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher.go`
- `smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher_test.go`
- `smart-recruit-offer-service/internal/interfaces/grpc/offer_server.go`
- `smart-recruit-offer-service/internal/interfaces/grpc/offer_server_test.go`
- `smart-recruit-offer-service/internal/docs/contract_inventory.md`
- `.spec/microservice-ddd-evolution/reports/TASK-005-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-005-evidence.json`

### Fix Summary

- Outbox publisher 改为等价 legacy envelope：DB 字段写入 `schema_version`、`idempotency_key`、metadata；payload 同时包含顶层 legacy consumer 字段和 nested `payload`。
- gRPC adapter 增加 endpoint-aware error mapping：候选人不匹配、未发送状态、Offer 不存在、read scope 失败和 send permission 失败均恢复 legacy 文案。
- 新增 outbox publisher 兼容测试和 gRPC legacy response message 测试。

### Re-run Commands

- `go test ./...` in `smart-recruit-offer-service`
- `git diff --name-only`
- `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-005`
- `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh`
- `node scripts/check-mysql-table-ownership.mjs`
- `node scripts/check-backend-boundaries.mjs`

### Re-run Results

全部通过；scope check 输出 `Scope check passed for TASK-005. Changed files: 14`。

### Remaining Risks

- Recruitment owner contract 仍是过渡债务；本 TASK 未修改 schema/proto 或新增跨服务 API。
