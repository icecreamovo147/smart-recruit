# TASK Report - TASK-002

## 1. TASK ID

TASK-002 - 盘点共享依赖、表归属与服务迁移基线。

## 2. Modified File List

实际变更文件：

- `docs/architecture/microservice-ddd-evolution-service-baseline.md`
- `.spec/microservice-ddd-evolution/reports/TASK-002-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-002-evidence.json`

## 3. Change Summary by File

- `docs/architecture/microservice-ddd-evolution-service-baseline.md`: 新增服务迁移基线，记录 8 个服务对 `smart-recruit-commons` 的依赖、表 owner/read/shared-write 边界、潜在违规写风险和后续迁移关注点。
- `.spec/microservice-ddd-evolution/reports/TASK-002-report.md`: 新增 TASK 报告。
- `.spec/microservice-ddd-evolution/reports/TASK-002-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-002` 通过。

变更均在 TASK-002 allowed files 内：`docs/architecture/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-016：记录每个服务对 `smart-recruit-commons/model/repository/service` 及 AI/OSS/email/MQ 等共享包的剩余依赖。
- FR-018 至 FR-020：记录单 MySQL 过渡模式下的 owner 表、过渡只读表、shared write 表和潜在跨 owner 写风险。
- CR-005：执行并记录 `smart-recruit-deploy/mysql-table-ownership.json` 与 `db.sql` 的一致性校验。

未修改业务代码、表归属、schema、protobuf、部署或 package/lockfile。

## 6. SDD Comparison Result

符合 SDD：

- 1. Existing Architecture Summary：记录当前独立服务根仍通过 shared domain-go repository/service 装配业务能力的事实。
- 2. Problem Analysis：记录 `service.NewServices` 大聚合、跨上下文 repository 装配和 Analytics/AI/Worker 风险。
- 12. Migration Risks：按 Offer 到 Worker 的顺序记录后续迁移风险基线。

## 7. Acceptance Comparison Result

- 每个服务对 `smart-recruit-commons` 的依赖已记录。
- 每个服务 owner 表、过渡只读表、授权 shared write 表和潜在违规写风险已记录。
- 输出文档可作为 TASK-003 及后续服务 TASK 的 baseline。
- 未把历史 `.spec/backend-ddd-microservices-evolution` 当作当前执行合同。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | 无输出；本 TASK 仅新增未跟踪文件，真实文件列表见 Modified File List。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-002` | 0 | passed | `Scope check passed for TASK-002. Changed files: 3` |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；无 Go module 变更，跳过 go test。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 4c47f386e5518a6b0eb16069d903cbc84102c697` exit 1。
- 失败原因仍是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 已不存在；TASK-001 已创建 `.knowledge/inbox/microservice-ddd-evolution-boundaries.md` 作为 candidate，TASK-002 继续记录 `candidate_required`。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - docs/architecture/microservice-ddd-evolution-service-baseline.md
  reviewed_documents:
    - .knowledge/architecture/system-overview.md
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/architecture/persistence-and-migrations.md
    - .knowledge/runbooks/local-development.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-boundaries.md
  coverage_gap: false
  reason: 当前 active knowledge 的服务拓扑和 source_refs 仍偏旧，TASK-001 已创建 inbox candidate；TASK-002 继续沿用该 candidate，不直接修改 active knowledge。
```

## 10. Risks

- 本 TASK 是只读盘点，不修复 shared repository/service 依赖。
- 当前 table ownership manifest 校验通过，但脚本只能扫描有限 `.Table(...)` 模式，不能替代后续服务迁移中的代码级 owner write 审查。
- active knowledge 中旧路径 source_refs 仍导致全量知识检测失败；已作为 candidate_required 债务记录。

## 11. Follow-up Items

- TASK-003 从 Offer 开始，以本文 Offer 基线为迁移前证据。
- 后续服务 TASK 应在本地 DDD 骨架和 domain/application 迁移时逐步消除 shared business dependency。

## 12. Whether the Next TASK Can Start

可以开始 TASK-003。独立只读 self-review verdict: 通过。
