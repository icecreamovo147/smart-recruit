# TASK Report - TASK-012

## 1. TASK ID

TASK-012 - Identity DDD 骨架与安全契约盘点。

## 2. Modified File List

实际变更文件：

- `smart-recruit-identity-service/internal/application/doc.go`
- `smart-recruit-identity-service/internal/application/command/doc.go`
- `smart-recruit-identity-service/internal/application/dto/doc.go`
- `smart-recruit-identity-service/internal/application/port/doc.go`
- `smart-recruit-identity-service/internal/application/query/doc.go`
- `smart-recruit-identity-service/internal/application/service/doc.go`
- `smart-recruit-identity-service/internal/domain/doc.go`
- `smart-recruit-identity-service/internal/domain/event/doc.go`
- `smart-recruit-identity-service/internal/domain/model/doc.go`
- `smart-recruit-identity-service/internal/domain/policy/doc.go`
- `smart-recruit-identity-service/internal/domain/repository/doc.go`
- `smart-recruit-identity-service/internal/domain/service/doc.go`
- `smart-recruit-identity-service/internal/infrastructure/doc.go`
- `smart-recruit-identity-service/internal/infrastructure/cache/doc.go`
- `smart-recruit-identity-service/internal/infrastructure/client/doc.go`
- `smart-recruit-identity-service/internal/infrastructure/persistence/doc.go`
- `smart-recruit-identity-service/internal/interfaces/doc.go`
- `smart-recruit-identity-service/internal/interfaces/grpc/doc.go`
- `smart-recruit-identity-service/internal/interfaces/mapper/doc.go`
- `smart-recruit-identity-service/internal/docs/security_contract_inventory.md`
- `.spec/microservice-ddd-evolution/reports/TASK-012-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-012-evidence.json`

## 3. Change Summary by File

- `internal/domain/**/doc.go`: 新增 Identity domain/package marker，声明模型、仓储 port、领域服务、事件和安全策略的目标职责边界；不引入运行时代码。
- `internal/application/**/doc.go`: 新增 command/query/dto/port/service package marker，声明用例编排、事务边界、token-version invalidation、audit 和 mapper 的后续迁移位置；不接入现有 runtime。
- `internal/infrastructure/**/doc.go`: 新增 persistence/cache/client package marker，声明 GORM、Redis token-version cache 和 outbound client adapter 的目标归属。
- `internal/interfaces/**/doc.go`: 新增 inbound adapter 与 gRPC/mapper package marker，声明 protobuf 到 application DTO 的映射位置。
- `internal/docs/security_contract_inventory.md`: 盘点当前 Identity runtime、Auth/Refresh/Principal/RBAC/Data Scope/Invite/Staff/Audit 契约、表归属、transitional read、测试锚点和后续迁移注意事项。
- `.spec/.../TASK-012-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-012-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-012` 通过。

变更均在 TASK-012 allowed files 内：`smart-recruit-identity-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-001：Identity 服务本地 `domain`、`application`、`infrastructure`、`interfaces`、`runtime` 目标职责边界已建立；本 TASK 只新增 package marker 和文档。
- FR-011：Identity auth、refresh token、principal、RBAC、data scope、invite code、auth audit、staff user 能力已完成契约盘点。
- SSR-002 / CR-007：未修改认证、授权、RBAC、data scope、JWT、Refresh Token、内部 gRPC 鉴权、TLS、protobuf、schema 或 Gateway 行为。
- FR-016：记录当前仍由 `smart-recruit-commons/service`、`repository`、`pkg` 提供 active 行为，作为后续迁移债务。

## 6. SDD Comparison Result

符合 SDD：

- 3.1 Target Service Shape：Identity 已具备目标 DDD package skeleton。
- 3.4 Per-Service Migration Pattern：本 TASK 完成服务级 protobuf/runtime/repository/table/test/security contract 盘点，作为后续 domain/application 与 infrastructure/interfaces/runtime 迁移基线。
- 5. API and Interface Changes：未修改 protobuf 或 gRPC service registration 行为；当前 runtime facade 仍保持 `AuthService` 与 Identity-owned `AdminService` 子集。
- 8. Compatibility Strategy：保持外部行为、运行时配置、Nacos、health/metrics/trace/logging、MySQL/Redis wiring 不变。

## 7. Acceptance Comparison Result

- Identity DDD 骨架存在：已完成，新增 `domain`、`application`、`infrastructure`、`interfaces` 及子 package marker。
- auth/refresh/principal/RBAC/data scope/invite/audit 契约盘点完成：已完成，见 `smart-recruit-identity-service/internal/docs/security_contract_inventory.md`。
- 未改变认证授权行为：已完成，本 TASK 未修改 active runtime、shared auth/admin/analytics service、repository、proto、schema、gateway 或配置。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-identity-service` | 0 | passed | Identity 服务全部 package 编译与测试通过，新增 package marker 均可编译。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | 当前新增文件为 untracked，tracked diff 为空；完整变更通过 `git status --short`、scope check 和 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-012` | 0 | passed | Scope check passed for TASK-012。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Identity module 变更并运行 `go test ./...` 通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree f4d91176d0e390eaf01ac1ca070ac0b6f0bc4179` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-identity-service/internal/domain/**
    - smart-recruit-identity-service/internal/application/**
    - smart-recruit-identity-service/internal/infrastructure/**
    - smart-recruit-identity-service/internal/interfaces/**
    - smart-recruit-identity-service/internal/docs/security_contract_inventory.md
  reviewed_documents:
    - .knowledge/architecture/auth-rbac-security.md
    - .knowledge/runbooks/debug-auth-permissions.md
    - .knowledge/pitfalls/auth-permission-alignment.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-identity-security.md
  coverage_gap: false
  reason: TASK-012 creates the Identity local DDD boundary and documents security contracts while active knowledge still references legacy logic-grpc-service/web-gin-service source paths. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Human Confirmation

TASK-012 requires human confirmation. User confirmed continuing TASK-012 and authorized future `requiresHumanConfirmation` tasks to proceed unless a blocking issue requiring human intervention is encountered.

## 11. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 新增文件均在 `smart-recruit-identity-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- 新增 Go 文件均为 package documentation markers，无 imports、init、global state 或 runtime wiring。
- 契约盘点覆盖 auth、refresh token、principal、RBAC、data scope、invite code、staff user、audit、表归属、transitional read 和现有测试锚点。
- `go test ./...`、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 12. Risks

- Identity 安全语义复杂；后续 TASK-013/TASK-014 迁移 active behavior 时必须逐项保持 password validation、invite validation、refresh-token rotation/reuse detection、token-version invalidation、Redis fail-safe、last-admin guard 和 audit writes。
- 当前 Identity runtime 不暴露 shared AdminService 的 invite-code admin RPC 方法，契约清单已记录该 routing/runtime 差异；后续如改变路由或暴露面必须按安全变更处理。
- `AuthzRepo` scope evaluation helpers 当前仍含 recruitment/interview transitional reads；本 TASK 只盘点，不扩大或修复这些读取。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 13. Follow-up Items

- TASK-013 可基于本 TASK 骨架开始 Identity domain/application 迁移。
- 后续 `.knowledge/**` scope 可用时，应为 Identity DDD 安全边界创建或更新候选知识文档。

## 14. Whether the Next TASK Can Start

TASK-012 通过；根据用户确认，后续 requiresHumanConfirmation TASK 可继续执行，除非遇到需要人工介入的阻塞性问题。
