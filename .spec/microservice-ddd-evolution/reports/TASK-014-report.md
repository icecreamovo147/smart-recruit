# TASK Report - TASK-014

## 1. TASK ID

TASK-014 - Identity infrastructure/interfaces/runtime/tests 收敛。

## 2. Modified File List

实际变更文件：

- `smart-recruit-identity-service/README.md`
- `smart-recruit-identity-service/cmd/identity-service/main.go`
- `smart-recruit-identity-service/internal/application/service/admin_service.go`
- `smart-recruit-identity-service/internal/docs/security_contract_inventory.md`
- `smart-recruit-identity-service/internal/infrastructure/cache/token_version.go`
- `smart-recruit-identity-service/internal/infrastructure/client/authorizer.go`
- `smart-recruit-identity-service/internal/infrastructure/client/security.go`
- `smart-recruit-identity-service/internal/infrastructure/persistence/identity_repository.go`
- `smart-recruit-identity-service/internal/interfaces/grpc/identity_server.go`
- `.spec/microservice-ddd-evolution/reports/TASK-014-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-014-evidence.json`

## 3. Change Summary by File

- `README.md`: 更新 Identity runtime 说明，记录 active runtime 使用本地 domain/application/infrastructure/interfaces implementation。
- `cmd/identity-service/main.go`: 将 active Identity runtime 装配从 shared `repository.New*` / `service.New*` 切换为本地 persistence、cache、client、application service 和 gRPC adapter。
- `internal/application/service/admin_service.go`: 新增 audit query typed error，确保 audit repository 查询失败映射为内部错误而不是权限错误。
- `internal/docs/security_contract_inventory.md`: 更新 TASK-014 后的运行时事实，记录本地 implementation 已成为 active auth/admin/audit provider。
- `internal/infrastructure/cache/token_version.go`: 新增 Redis token-version cache adapter，保留 SET 失败后 DELETE fail-safe 所需端口行为。
- `internal/infrastructure/client/authorizer.go`: 新增 actor match verifier 与 admin permission authorizer，复刻 gRPC metadata fail-closed 行为和 permission lookup。
- `internal/infrastructure/client/security.go`: 新增 bcrypt password service 与 crypto-random refresh token/family generator。
- `internal/infrastructure/persistence/identity_repository.go`: 新增本地 GORM repositories，覆盖 users、refresh_tokens、invite_codes、RBAC/data-scope、authorization_audit_logs。
- `internal/interfaces/grpc/identity_server.go`: 新增 protobuf-facing Auth/Admin/Audit adapter，映射本地 application services 到现有 pb response/error semantics。
- `.spec/.../TASK-014-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-014-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-014` 通过。

变更均在 TASK-014 allowed files 内：`smart-recruit-identity-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-004：新增 Identity 本地 GORM persistence、Redis cache、security/authorizer adapters。
- FR-005：新增 local protobuf-facing interfaces/grpc adapter。
- FR-006：runtime 继续负责平台装配、Nacos、health、metrics、trace、logging、DB/Redis 和 gRPC server lifecycle。
- FR-011：Identity auth、refresh token、principal、RBAC、data scope、invite code、auth audit、staff user active runtime 已使用本地 implementation。
- FR-016：active runtime 不再直接依赖 shared auth/admin/analytics implementation；剩余 `smart-recruit-commons` 引用仅为 module/config 兼容债务。
- SSR-002 / CR-007：未修改 protobuf、schema、权限表、JWT/Refresh/RBAC/data scope/audit 外部语义。

## 6. SDD Comparison Result

符合 SDD：

- 3.1 Target Service Shape：Identity `infrastructure`、`interfaces` 和 runtime wiring 已填充。
- 3.4 Per-Service Migration Pattern：完成 GORM repository、cache/client adapter、gRPC adapter、runtime 装配和测试收敛。
- 8. Compatibility Strategy：保持 gRPC service registration、internal auth/TLS、health、metrics、trace、logging 和 Nacos lifecycle。
- 11. Testing Strategy：运行 Identity `go test ./...`、table ownership、backend boundary、scope、agent-check。

## 7. Acceptance Comparison Result

- Identity runtime 使用本地 implementation：已完成，`cmd/identity-service/main.go` 构造本地 repositories/application services/interfaces server 并传入 runtime。
- Auth/Admin 安全语义与 audit 行为兼容：已完成，local adapter 保留密码复杂度、invite validation、opaque refresh-token hash/rotation/reuse detection、token-version cache fail-safe、last-admin/self-revoke guard、admin permission gate 和 audit write/query 语义。
- 对共享 auth/admin implementation 的直接依赖清除或记录债务：已完成，`rg "smart-recruit-commons/(repository|service)" smart-recruit-identity-service` 无命中；剩余 `smart-recruit-commons` module/config 引用记录为兼容债务。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-identity-service` | 0 | passed | Identity 全 package 测试通过，本地 infrastructure/interfaces/runtime 装配可编译。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | Tracked diff 列出 4 个已跟踪文件；新增文件通过 `git status --short`、scope check 和 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-014` | 0 | passed | `Scope check passed for TASK-014. Changed files: 11`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Identity module 变更并运行 `go test ./...` 通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree ace3d614c29c54af8a0b6ce2dd72dac37dd97537` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-identity-service/cmd/identity-service/main.go
    - smart-recruit-identity-service/internal/infrastructure/**
    - smart-recruit-identity-service/internal/interfaces/grpc/identity_server.go
    - smart-recruit-identity-service/internal/docs/security_contract_inventory.md
  reviewed_documents:
    - .knowledge/architecture/auth-rbac-security.md
    - .knowledge/runbooks/debug-auth-permissions.md
    - .knowledge/pitfalls/auth-permission-alignment.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-identity-security.md
  coverage_gap: false
  reason: TASK-014 switches active Identity runtime to local implementation while active knowledge still references legacy logic-grpc-service/web-gin-service source paths. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Human Confirmation

TASK-014 requires human confirmation. User previously confirmed continuing TASK-012 and authorized future `requiresHumanConfirmation` tasks to proceed unless a blocking issue requiring human intervention is encountered. No blocking auth/authz behavior change was needed.

## 11. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 新增/修改文件均在 `smart-recruit-identity-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- `cmd/identity-service/main.go` 不再 import shared `repository` 或 `service`。
- Local gRPC adapter returns existing pb code/message patterns for auth, refresh, role/data-scope, staff and audit flows.
- Local persistence writes only Identity-owned tables; no new cross-service writes introduced.
- `go test ./...`、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 12. Risks

- `go.mod` 仍保留 `smart-recruit-commons` 依赖和 `CONFIG_PATH` fallback 仍指向 domain config template；这是 service module/config compatibility debt，后续 shared cleanup/commons rename TASK 处理。
- Invite-code admin RPC 仍未在 Identity runtime facade 暴露；本 TASK 保持既有暴露面，未扩大 public auth/admin 行为。
- 本 TASK 未执行真实 MySQL/Redis integration；GORM/Redis adapters 通过编译与 service tests 覆盖，真实连接仍由现有 runtime config 管理。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 13. Follow-up Items

- TASK-015 可开始 Recruitment DDD 骨架与核心域盘点。
- 后续 `.knowledge/**` scope 可用时，应更新 auth/RBAC/security audit 知识文档的 source_refs 与 Identity DDD 运行路径。

## 14. Whether the Next TASK Can Start

TASK-014 通过；根据用户确认，后续 requiresHumanConfirmation TASK 可继续执行，除非遇到需要人工介入的阻塞性问题。
