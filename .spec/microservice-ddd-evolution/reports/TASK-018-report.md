# TASK Report - TASK-018

## 1. TASK ID

TASK-018 - Recruitment infrastructure/interfaces/runtime/tests 收敛。

## 2. Modified File List

实际变更文件：

- `smart-recruit-recruitment-service/README.md`
- `smart-recruit-recruitment-service/cmd/recruitment-service/main.go`
- `smart-recruit-recruitment-service/internal/docs/runtime_convergence.md`
- `smart-recruit-recruitment-service/internal/interfaces/grpc/adapters.go`
- `smart-recruit-recruitment-service/internal/interfaces/grpc/adapters_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-018-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-018-evidence.json`

## 3. Change Summary by File

- `README.md`: 记录 TASK-018 后 runtime 经本地 `internal/interfaces/grpc` adapters 装配，并声明剩余 shared delegate compatibility debt。
- `cmd/recruitment-service/main.go`: 将 active Recruitment runtime deps 从直接传入 shared service 实例改为先构造本地 grpc adapters，再传入 `internal/runtime.New`。
- `internal/interfaces/grpc/adapters.go`: 新增 Job、JobTaxonomy、TaxonomyAdmin、RecruitmentAdmin、UsageStats、Candidate、Application、Collaboration adapters，作为 protobuf-facing 本地边界。
- `internal/interfaces/grpc/adapters_test.go`: 覆盖 adapter delegate 行为和 missing delegate fail-fast。
- `internal/docs/runtime_convergence.md`: 新增 runtime convergence 说明，记录 active boundary、compatibility debt 和后续 adapter cut points。
- `.spec/.../TASK-018-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-018-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-018` 通过。

变更均在 TASK-018 allowed files 内：`smart-recruit-recruitment-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-004 至 FR-007：新增 Recruitment 本地 protobuf-facing interfaces/grpc adapter，并通过 runtime 装配使用该本地边界。
- FR-012：保留 Job/Candidate/Application/Collaboration/Admin 子集兼容面，未改变 API 契约。
- FR-016：shared recruitment 业务 service 直接作为 runtime dependency 的路径已收敛到本地 adapters；剩余 shared repository/service/oss delegate 记录为 compatibility debt。

## 6. SDD Comparison Result

符合 SDD：

- 8. Compatibility Strategy：未修改 protobuf、schema、gateway、Nacos、health、metrics、trace、MySQL、Redis 或 OSS runtime 行为。
- 11. Testing Strategy：运行 Recruitment `go test ./...`、runtime `--check`、table ownership、backend boundary、scope 和 agent-check。
- Per-service migration pattern：本 TASK 完成 interfaces/runtime 装配收敛，并明确后续替换本地 persistence/OSS/outbox adapters 的切点。

## 7. Acceptance Comparison Result

- Recruitment runtime 使用本地 implementation：已完成，`cmd/recruitment-service/main.go` 将 Job/Candidate/Application/Admin/UsageStats/Collaboration 依赖封装进 `internal/interfaces/grpc` adapters 后传入 runtime。
- Job/Candidate/Application/Collaboration/Admin 子集兼容：已完成，adapter 方法逐一覆盖 runtime 需要的 pb-facing API，Collaboration 保持原 pb server 兼容。
- 对共享 recruitment implementation 的依赖清除或记录债务：已完成，runtime 不再直接接收 shared service 实例；main 中仍构造 shared compatibility delegates，已在 README 和 `internal/docs/runtime_convergence.md` 记录为临时债务。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-recruitment-service` | 0 | passed | Recruitment 全 package 测试通过，新增 interfaces/grpc adapter tests 通过。 |
| `go run ./cmd/recruitment-service --check` | 0 | passed | `recruitment-service runtime check passed`。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | Tracked diff listed README and main; new adapter/doc/report files are listed by `git status --short` and evidence `changed_files`。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-018` | 0 | passed | `Scope check passed for TASK-018. Changed files: 7`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Recruitment module 变更并运行 `go test ./...` 通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 8700ea686db5e50c846bc5e40f6217e8f3962c2d` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-recruitment-service/cmd/recruitment-service/main.go
    - smart-recruit-recruitment-service/internal/interfaces/grpc/adapters.go
    - smart-recruit-recruitment-service/internal/docs/runtime_convergence.md
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/runbooks/service-binary-convention.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-recruitment-runtime.md
  coverage_gap: false
  reason: TASK-018 moves active Recruitment runtime wiring behind local grpc adapters while active knowledge still references legacy logic-grpc-service/web-gin-service source paths. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 变更均在 `smart-recruit-recruitment-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- Runtime `Deps` 现在接收本地 adapter 实例，不再直接接收 shared service instances。
- `rg "smart-recruit-domain-go/(repository|service|oss)" smart-recruit-recruitment-service -n` 仍显示 main 中 compatibility delegate construction；已记录为债务。
- `go test ./...`、runtime `--check`、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 11. Risks

- TASK-018 没有完全删除 shared repository/service/oss delegates；这是为了保持 AdminService split、OSS/outbox/cache 和 protobuf behavior 兼容，已记录为后续 adapter cut point。
- 本地 TASK-016/TASK-017 application services 尚未全部接入 active pb methods；后续真正清债需要实现 local persistence/OSS/outbox adapters 并逐步替换 delegate。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-019 可开始 Analytics DDD 骨架与投影策略盘点。
- 后续 Recruitment cleanup 可以按 `internal/docs/runtime_convergence.md` 的 cut points 删除 shared delegates。

## 13. Whether the Next TASK Can Start

TASK-018 通过；TASK-019 可以开始。
