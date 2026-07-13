# TASK Report - TASK-030

## 1. TASK ID

TASK-030 - 最终重命名 `smart-recruit-domain-go` 为 `smart-recruit-commons`。

## 2. Modified File List

完整变更清单见 `.spec/microservice-ddd-evolution/reports/TASK-030-evidence.json` 的 `changed_files`。

主要变更：

- `smart-recruit-domain-go/**` -> `smart-recruit-commons/**`。
- Go module path、imports、`replace`、`go.work` 全部改为 `smart-recruit-commons`。
- 服务启动配置 fallback、Docker build context、scripts、architecture docs、active `.spec/microservice-ddd-evolution/**` 引用同步更新。
- scope checker 增强：TASK-030 允许 migration SQL 纯 rename，但仍禁止 SQL 内容变更。

## 3. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-030` 通过。

说明：migration SQL 文件仅随目录 rename 移动，内容未修改；scope checker 已按 `R100` pure rename 放行。

## 4. SPEC Comparison Result

符合 SPEC：

- FR-027 / FR-028：shared module 已最终重命名为 `smart-recruit-commons`。
- CR-011 / AC-012：业务代码、测试、构建脚本和部署引用已切到新 module/path。

## 5. SDD Comparison Result

符合 SDD：

- 3.6 Final Commons Target：目录、module path、workspace 和 import path 已完成 rename。
- 13. Implementation Boundaries：未修改 protobuf 行为、schema SQL 内容、auth/security 行为、package.json 或 pnpm lockfile。

## 6. Acceptance Comparison Result

- 目录与 Go module path 重命名为 `smart-recruit-commons`：已完成。
- 旧 import/path 在业务代码、测试、构建、部署和文档中清零：active scope 内已清零；剩余旧名引用见批准历史清单。
- `smart-recruit-commons` 只包含 shared kernel 和通用技术能力：继承 TASK-029 收缩结果，保留 ai provider helpers、config、email、events、migration runner、mq、oss、pkg、resumeparser。
- 全仓库验证通过或记录不可运行原因：所有 Go modules `go test ./...` 通过，边界/表归属通过。

## 7. Approved Historical References

`rg --hidden "smart-recruit-domain-go"` 剩余引用仅限：

- `db.sql` 与 `smart-recruit-commons/migrations/000051_standardize_event_outbox.sql`：历史 outbox producer 默认值，schema/db dump 禁改。
- `AGENTS.md`、`.gitignore`、`reasonix.toml`：未纳入 TASK-030 allowedFiles 的 repo/global/tooling 文件，未改动。
- `.spec/microservice-runtime-implementation/scripts/agent-check.sh`：另一个历史 feature harness，不属于本 feature scope。
- `.spec/microservice-ddd-evolution/task-scope.json`：TASK-030 必须保留 old source pattern 以校验 rename source。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | TASK-030 diff listed. |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-030` | 0 | passed | Scope check passed；Changed files: 374. |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness validation passed；changed Go modules tests passed. |
| All Go modules `go test ./...` loop | 0 | passed | commons、platform、proto、gateway、identity、recruitment、interview、offer、notification、ai-agent、analytics、worker 全部通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `rg --hidden -n "smart-recruit-domain-go" -g "!.git" -g "!node_modules"` | 0 | passed | 仅剩批准历史/禁改引用。 |
| `git diff --check` | 0 | passed | 无 whitespace error。 |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-commons/**
    - go.work
    - smart-recruit-*-service/go.mod
    - scripts/**
    - docs/**
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/architecture/persistence-and-migrations.md
    - .knowledge/runbooks/service-binary-convention.md
  coverage_gap: false
  reason: Active knowledge validation remains blocked by pre-existing legacy logic-grpc-service/web-gin-service source_ref debt; rename-relevant active docs/scripts in this feature scope were updated.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- Go code/go.mod/go.work no longer reference `smart-recruit-domain-go`.
- `smart-recruit-domain-go` directory no longer exists; `smart-recruit-commons` exists and tests pass.
- SQL migrations were moved only, not edited.
- Remaining old-name references are either forbidden historical schema/tooling files, another feature harness, or TASK-030 scope source patterns.

## 11. Risks

- `.gitignore` 和 `AGENTS.md` 仍有 old directory text because they are outside TASK-030 scope; future housekeeping should update them with explicit scope.
- Historical outbox producer value remains in schema SQL/db dump by design; runtime producer metadata now uses `smart-recruit-commons.outbox`.
- Knowledge source_ref debt remains broader than this feature and should be handled as a dedicated knowledge refresh.

## 12. Whether the Next TASK Can Start

TASK-030 是最后一个迁移 TASK；本功能点可以进入 pipeline finalization。
