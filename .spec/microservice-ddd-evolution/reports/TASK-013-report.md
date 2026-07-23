# TASK Report - TASK-013

## 1. TASK ID

TASK-013 - Identity domain/application 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-identity-service/internal/application/command/identity.go`
- `smart-recruit-identity-service/internal/application/dto/identity.go`
- `smart-recruit-identity-service/internal/application/port/identity.go`
- `smart-recruit-identity-service/internal/application/query/identity.go`
- `smart-recruit-identity-service/internal/application/service/auth_service.go`
- `smart-recruit-identity-service/internal/application/service/admin_service.go`
- `smart-recruit-identity-service/internal/application/service/auth_service_test.go`
- `smart-recruit-identity-service/internal/application/service/admin_service_test.go`
- `smart-recruit-identity-service/internal/application/service/test_fakes_test.go`
- `smart-recruit-identity-service/internal/domain/model/security.go`
- `smart-recruit-identity-service/internal/domain/policy/security.go`
- `smart-recruit-identity-service/internal/domain/policy/security_test.go`
- `smart-recruit-identity-service/internal/domain/repository/identity.go`
- `.spec/microservice-ddd-evolution/reports/TASK-013-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-013-evidence.json`

## 3. Change Summary by File

- `internal/domain/model/security.go`: 新增 Identity 本地 domain model/value objects/constants，覆盖 user、principal、roles、permissions、data scopes、invite code、refresh session 和 auth audit。
- `internal/domain/policy/security.go`: 新增纯领域安全策略，复刻当前密码复杂度、注册角色/邀请码规则、用户名/email 校验和 system_admin revoke guard。
- `internal/domain/repository/identity.go`: 新增本地 repository ports，覆盖 user、refresh token、authz/RBAC/data scope、invite code 和 audit。
- `internal/application/port/identity.go`: 新增 application ports，覆盖 password service、refresh token/family generator、clock、actor verifier、admin authorizer 和 token-version cache。
- `internal/application/command/identity.go`: 新增 Identity write-side command DTO。
- `internal/application/query/identity.go`: 新增 principal、staff、roles、audit read-side query DTO。
- `internal/application/dto/identity.go`: 新增 application result DTO，保持 mapper 与 protobuf 解耦。
- `internal/application/service/auth_service.go`: 新增本地 Auth application service，编排 register/login/refresh/revoke/audit/principal/update-email 语义，但不接入 runtime。
- `internal/application/service/admin_service.go`: 新增本地 Admin application service，编排 role/data-scope/staff/audit 用例和 token-version cache fail-safe 语义，但不接入 runtime。
- `internal/domain/policy/security_test.go`: 覆盖密码复杂度、注册计划和 system_admin revoke guard。
- `internal/application/service/auth_service_test.go`: 覆盖 staff invite 注册默认 recruiter + `own_jobs`、login refresh-token TTL/family 创建、refresh-token reuse 错误映射。
- `internal/application/service/admin_service_test.go`: 覆盖 role revoke token-version cache 双失败回滚、data-scope cache delete fail-safe、admin read permission gate。
- `internal/application/service/test_fakes_test.go`: 新增 application tests 的 in-memory ports。
- `.spec/.../TASK-013-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-013-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-013` 通过。

变更均在 TASK-013 allowed files 内：`smart-recruit-identity-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-002：Identity domain 层新增本地模型、策略和 repository ports，未依赖 GORM、Redis、RabbitMQ、gRPC、HTTP、Nacos、proto 或环境变量。
- FR-003：Identity application 层新增 use-case orchestration、权限/actor/token-version/audit/cache ports 和 command/query/dto。
- FR-011：本 TASK 本地化 auth、refresh token、principal、RBAC、data scope、invite code、auth audit、staff user 的 domain/application 语义。
- SSR-002 / CR-007：未修改 active runtime、protobuf、schema、JWT/Refresh/RBAC/data scope/audit 执行路径或 Gateway 行为。

## 6. SDD Comparison Result

符合 SDD：

- 3.1 Target Service Shape：Identity `domain` 与 `application` 层从 TASK-012 marker 进入可测试实现。
- 3.4 Per-Service Migration Pattern：完成 domain model/policy/repository port 与 application command/query/service/port 迁移；infrastructure/interfaces/runtime 接线留给 TASK-014。
- 9. Error Handling and Fallback Design：保留 refresh-token invalid/reuse 区分、token-version cache SET 失败后 DELETE fail-safe、role revoke cache 双失败回滚等安全失败语义。
- 11. Testing Strategy：新增 domain/application 单元测试并运行 Identity `go test ./...`。

## 7. Acceptance Comparison Result

- Identity domain/application 本地化：已完成，新增本地 domain model/policy/repository ports 与 auth/admin application services。
- JWT/Refresh/RBAC/data scope/audit 语义兼容：已完成，本 TASK 使用端口复刻现有 refresh-token TTL、reuse 错误映射、RBAC/data-scope mutation、token-version cache fail-safe、last-admin/self-revoke guard 和 audit 写入约定；未切换 active runtime。
- Domain 不依赖外层技术：已完成，domain 包仅依赖标准库与 Identity 本地 domain model。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-identity-service` | 0 | passed | Identity 全 package 测试通过，新增 domain/application tests 通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | 当前新增文件为 untracked，tracked diff 为空；完整变更通过 `git status --short`、scope check 和 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-013` | 0 | passed | `Scope check passed for TASK-013. Changed files: 15`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Identity module 变更并运行 `go test ./...` 通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 4504fb9dcf3899f71556904c9b06180c87562c26` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-identity-service/internal/domain/model/security.go
    - smart-recruit-identity-service/internal/domain/policy/security.go
    - smart-recruit-identity-service/internal/domain/repository/identity.go
    - smart-recruit-identity-service/internal/application/**
  reviewed_documents:
    - .knowledge/architecture/auth-rbac-security.md
    - .knowledge/runbooks/debug-auth-permissions.md
    - .knowledge/pitfalls/auth-permission-alignment.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-identity-security.md
  coverage_gap: false
  reason: TASK-013 localizes Identity domain/application security contracts while active knowledge still references legacy logic-grpc-service/web-gin-service source paths. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Human Confirmation

TASK-013 requires human confirmation. User previously confirmed continuing TASK-012 and authorized future `requiresHumanConfirmation` tasks to proceed unless a blocking issue requiring human intervention is encountered. No blocking auth/authz behavior change was needed.

## 11. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 新增文件均在 `smart-recruit-identity-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- Domain 包未 import GORM/proto/gRPC/Redis/RabbitMQ/HTTP/Nacos/platform/shared domain。
- Application service 未接入 runtime，不改变当前认证授权行为。
- Tests 覆盖 password policy、registration invite plan、staff registration RBAC/scope assignment、refresh-token TTL/reuse error mapping、token-version cache fail-safe、role revoke rollback 和 admin read permission gate。
- `go test ./...`、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 12. Risks

- TASK-013 尚未切换 runtime；TASK-014 接线时必须确保 mapper、repository adapter、password/hash/token generator 与现有 shared implementation 完全一致。
- `CreateStaffUser` 与当前 shared service 一样不在 application 方法内强制 admin permission gate；依赖现有 Gateway/AdminService routing 保护。TASK-014 接线时不得意外改变该行为，除非单独确认。
- Invite-code admin RPC 的 runtime 暴露差异仍保留，TASK-014 如调整 runtime facade 必须明确验证路由兼容。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 13. Follow-up Items

- TASK-014 应实现 Identity infrastructure/interfaces/runtime adapter，并把 active runtime 从 shared `AuthService`/`AdminService`/`AnalyticsService` 切到本地 application service，保持 protobuf 与安全语义兼容。
- 后续 `.knowledge/**` scope 可用时，应更新 auth/RBAC/security audit 知识文档的 source_refs 与 Identity DDD 运行路径。

## 14. Whether the Next TASK Can Start

TASK-013 通过；根据用户确认，TASK-014 可继续执行，除非接线阶段发现必须改变 auth/authz 行为、protobuf、schema 或共享模块。
