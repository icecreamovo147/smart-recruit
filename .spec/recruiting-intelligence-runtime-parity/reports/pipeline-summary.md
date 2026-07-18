# Pipeline Summary - recruiting-intelligence-runtime-parity

## 执行概况

- 起始 TASK：TASK-001
- 结束 TASK：TASK-007
- 完成：7 / 7
- 失败：无
- 阻塞：无
- 最终状态：`completed_with_exceptions`
- Git 基线：`98165e7cdac06c0aa2a0685ecf1ad7c00a3eb518`

## 各 TASK 结果

| TASK | 状态 | 最终审查轮次 | 最终 Verdict | 报告 |
|---|---:|---:|---|---|
| TASK-001 | ✅ | 3 | 通过 | [TASK-001-report.md](TASK-001-report.md) |
| TASK-002 | ✅ | 2 | 通过 | [TASK-002-report.md](TASK-002-report.md) |
| TASK-003 | ✅ | 3 | 通过 | [TASK-003-report.md](TASK-003-report.md) |
| TASK-004 | ✅ | 3 | 通过 | [TASK-004-report.md](TASK-004-report.md) |
| TASK-005 | ✅ | 3 | 通过 | [TASK-005-report.md](TASK-005-report.md) |
| TASK-006 | ✅ | 4 | 通过 | [TASK-006-report.md](TASK-006-report.md) |
| TASK-007 | ✅ | 5 | 通过 | [TASK-007-report.md](TASK-007-report.md) |

每个 TASK 均有独立 `base_sha`、`base_tree`、通过的 scope/check 状态、machine-readable evidence 和 fresh independent Reviewer 最终 `verdict: 通过`。

## 恢复结果

- `resume_profile_extractor`：每请求读取数据库 active `system` Prompt，使用真正 System/User 消息；严格结构校验、策略化 heuristic fallback、版本/current 切换与事务持久化已恢复。
- `job_requirement_extractor`：数据库 Prompt、严格 requirement schema、确定性权重归一化和策略化 fallback 已恢复，并作为匹配内部组件使用。
- `candidate_match_evaluator`：真实 source ID 证据索引、确定性优先、缺失证据逐要求模型评估、严格 provenance 校验、确定性聚合、knockout/风险/推荐和事务持久化已恢复。
- Runtime policy：消费既有 feature flags 与 timeout；未修改共享配置定义。
- Observability：Prompt/model/version/stage/fallback/count/duration 仅记录安全元数据；外部 request ID 使用进程随机密钥的域分离 HMAC；两个 native 方法所有返回路径恰好一个 terminal。

## 验证与 Smoke

- AI Agent：`GOWORK=off go test ./...` 通过。
- Commons structured completion：`GOWORK=off go test ./ai -run TestGenerateStructured -count=10` 通过。
- focused schema/fallback/evidence/aggregation/persistence/observability 矩阵通过，关键 privacy/terminal 路径以 `-count=10` 重复验证。
- 真实 `NativeStore` 版本、latest/current、child mapping、evidence、Agent-run 与回滚测试通过。
- 本地 MySQL metadata-only 查询确认三个 agent type 均存在 active version 2、role `system`；未读取 Prompt 内容。
- 本地 Identity、Recruitment、AI Agent、Gateway 未监听，live provider/API smoke 如实记录为 `SKIP`；fake-provider native orchestration 与真实 persistence integration 通过，不把 skip 描述为 live PASS。
- 知识 validator、reference check、impact detection、feature validator、evidence validator、pipeline-state validator、privacy scan 与 `git diff --check` 通过。

## Scope 审计说明

每个 TASK 的 scope 结果在其 TASK 基线和 evidence 中为 `passed`。历史 TASK 的 scope 脚本不能对最终累计工作树重新解释：例如从 TASK-001 的 `base_tree` 计算最终 diff 会包含 TASK-002～007 的合法后续文件，因而预期报告越界。最终完成判断使用保存的 TASK-local base tree、当轮 scope 输出和已通过的 canonical evidence validator；TASK-007 对最终累计状态的当前 scope 检查通过。

## 已批准例外

- TASK-001：仅为 `smart-recruit-commons/go.sum` 增加 12 条已有依赖 checksum；未修改 `go.mod` 或依赖版本。
- TASK-003：仅扩展 `native_recruiting_generation_test.go` 测试范围；未改生产 persistence、schema、Proto、依赖或公开 API。
- TASK-006：用户批准 `native_store.go`、相关 persistence test 和一个 gRPC stale test 的精确范围扩展。
- TASK-006：用户批准一次 Reviewer R4 扩展。
- TASK-007：用户分别批准 Reviewer R4 和 Reviewer R5 的两次精确扩展；文件 scope 未扩大。

所有例外均在 `pipeline-state.json` 中记录 `task_id`、批准对象、批准人、时间、范围和原因，因此最终状态使用 `completed_with_exceptions`。

## 总修改范围

- `smart-recruit-commons/ai/`：结构化 System/User provider 入口及测试。
- `smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/`：Runtime policy、三个专用组件、证据、matcher、legacy/enhanced aggregator 与 observability。
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/`：native orchestration、authorization/source/persistence 边界、terminal observability 与集成测试。
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/`：Prompt runtime、真实 source mapping、事务 persistence 测试。
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/`：既有配置到 immutable runtime policy 的注入。
- 六份 `.knowledge/` 文档与完整 `.spec/recruiting-intelligence-runtime-parity/` 控制面、报告和 evidence。

未修改 Proto、数据库 schema/migration、权限模型、外部 HTTP/gRPC API、依赖版本、`go.mod`、前端、Gateway 或部署配置。

## 残余风险

- live external provider/API smoke 未执行，原因和替代证据已明确记录。
- HMAC correlation 是进程内稳定，不支持跨重启关联。
- 并发版本分配、保守 CJK 校验和 heuristic fidelity 等既有残余风险见 [compatibility-and-risk.md](compatibility-and-risk.md)。
