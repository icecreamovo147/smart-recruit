# TASK Report - TASK-016

## 1. TASK ID

TASK-016 - Recruitment job/candidate/resume domain/application 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-recruitment-service/internal/application/command/recruitment.go`
- `smart-recruit-recruitment-service/internal/application/dto/recruitment.go`
- `smart-recruit-recruitment-service/internal/application/port/recruitment.go`
- `smart-recruit-recruitment-service/internal/application/service/job_candidate_resume_service.go`
- `smart-recruit-recruitment-service/internal/application/service/job_candidate_resume_service_test.go`
- `smart-recruit-recruitment-service/internal/domain/model/recruitment.go`
- `smart-recruit-recruitment-service/internal/domain/policy/recruitment.go`
- `smart-recruit-recruitment-service/internal/domain/policy/recruitment_test.go`
- `smart-recruit-recruitment-service/internal/domain/repository/recruitment.go`
- `.spec/microservice-ddd-evolution/reports/TASK-016-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-016-evidence.json`

## 3. Change Summary by File

- `internal/domain/model/recruitment.go`: 新增 Recruitment 本地 Job、CandidateProfile、Resume、PresignSession、UsageLogEntry、OutboxEvent、ResumeParsePayload 模型和状态常量。
- `internal/domain/policy/recruitment.go`: 新增 job 构建校验、candidate profile 完整度、resume 文件类型、content-type、文件名清洗、strict/legacy resume confirm 校验策略。
- `internal/domain/repository/recruitment.go`: 新增 Job、scope checker、department-location validator、candidate profile、resume、OSS、outbox、usage log repository/application ports。
- `internal/application/command/recruitment.go`: 新增 job/candidate/resume command DTO。
- `internal/application/dto/recruitment.go`: 新增 application result DTO。
- `internal/application/port/recruitment.go`: 新增 clock 和 upload-id generator ports。
- `internal/application/service/job_candidate_resume_service.go`: 新增本地 JobService 与 CandidateResumeService，保留 job create/online/offline、profile completeness、OSS presign、strict/legacy confirm、usage log、resume.parse outbox 语义。
- `internal/application/service/job_candidate_resume_service_test.go`: 覆盖 job lifecycle、scope dispatch、candidate profile completeness、resume presign/confirm、usage log 和 outbox payload。
- `internal/domain/policy/recruitment_test.go`: 覆盖 profile completeness、resume file type、strict session 和 legacy key policy。
- `.spec/.../TASK-016-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-016-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-016` 通过。

变更均在 TASK-016 allowed files 内：`smart-recruit-recruitment-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-002 / FR-003：Job、Candidate Profile、Resume 本地 domain/application 层已建立，外部依赖通过 repository/application ports 表达。
- FR-012：实现遵循 TASK-015 盘点出的 job、candidate、resume、OSS、usage log、outbox 边界。
- Compatibility：未修改 protobuf、schema、gateway、runtime wiring 或 active shared implementation，因此不改变招聘 API 行为。

## 6. SDD Comparison Result

符合 SDD：

- 6. Algorithm or Workflow Changes：job lifecycle、resume upload confirm、outbox payload 和 usage log 语义在本地 application service 中以端口化方式保留。
- 11. Testing Strategy：新增 targeted unit tests 覆盖 job lifecycle、candidate profile 和 resume upload confirm。
- 3.4 Per-Service Migration Pattern：本 TASK 完成 domain/application 迁移准备；infrastructure/interface/runtime 切换留给后续 TASK。

## 7. Acceptance Comparison Result

- Job/Candidate/Resume domain/application 本地化：已完成，本地模型、policy、repository ports、command/dto/ports 与 application services 已新增。
- OSS presign、resume confirm、usage log、outbox 语义兼容：已完成，presign 使用 tmp key/session/content-type，confirm 严格校验 upload session、OSS object/size、copy/delete、写 `resume.parse` outbox 和 `oss_confirm` usage log。
- 覆盖 job lifecycle 和 resume upload confirm 测试：已完成，`internal/application/service` 与 `internal/domain/policy` targeted tests 覆盖。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./internal/application/service ./internal/domain/policy` in `smart-recruit-recruitment-service` | 0 | passed | Targeted job lifecycle、profile、resume presign/confirm、policy 测试通过。 |
| `go test ./...` in `smart-recruit-recruitment-service` | 0 | passed | Recruitment 全 package 测试通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | 当前 TASK 新增文件为 untracked；通过 `git status --short`、scope check 和 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-016` | 0 | passed | `Scope check passed for TASK-016. Changed files: 11`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Recruitment module 变更并运行 `go test ./...` 通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree bbe015be86e0be3d84c8925683fa5c4d918b189a` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-recruitment-service/internal/domain/model/recruitment.go
    - smart-recruit-recruitment-service/internal/domain/policy/recruitment.go
    - smart-recruit-recruitment-service/internal/domain/repository/recruitment.go
    - smart-recruit-recruitment-service/internal/application/service/job_candidate_resume_service.go
  reviewed_documents:
    - .knowledge/domains/recruitment.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/domains/resume-intelligence.md
    - .knowledge/runbooks/debug-resume-intelligence.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-recruitment-job-resume.md
  coverage_gap: false
  reason: TASK-016 creates local Recruitment job/candidate/resume domain/application anchors while active knowledge still references legacy logic-grpc-service/web-gin-service source paths. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 新增文件均在 `smart-recruit-recruitment-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- 未切换 active runtime，符合 TASK-016 out-of-scope 的 infrastructure/interface/runtime 收敛边界。
- Resume confirm 测试覆盖 upload session、OSS copy/delete、resume.parse outbox、usage log 和 payload 字段。
- `go test ./...`、targeted tests、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 11. Risks

- Active Recruitment runtime 仍使用 shared implementation；后续 TASK-018 才会接入本地 infrastructure/interfaces/runtime。
- 本地 ports 当前覆盖 job/candidate/resume 迁移所需的核心行为，application/collaboration/taxonomy 不在本 TASK 范围内。
- Usage log 写入在本地 service 中通过端口同步表达，真实异步/超时行为需在 infrastructure adapter 接入时保持兼容。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-017 可开始 Recruitment application/collaboration/taxonomy 迁移。
- TASK-018 需要实现 persistence/OSS/outbox adapters、gRPC mapper 和 runtime wiring，确保 active runtime 使用本地 services。

## 13. Whether the Next TASK Can Start

TASK-016 通过；TASK-017 可以开始。
