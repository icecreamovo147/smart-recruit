# TASK-MRI-023 Report

## TASK ID

TASK-MRI-023

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-023-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-023-report.md`
- `smart-recruit-deploy/nacos/seed-config/gateway.yaml`
- `smart-recruit-gateway/internal/runtime/analytics_cutover.go`
- `smart-recruit-gateway/internal/runtime/analytics_cutover_test.go`
- `web-gin-service/rpc/analytics_admin_client.go`
- `web-gin-service/rpc/analytics_admin_client_test.go`
- `web-gin-service/rpc/client.go`
- `web-gin-service/rpc/client_test.go`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-023` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-024`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-023-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-023-report.md`: 记录本 TASK 的人工可读验收报告。
- `smart-recruit-deploy/nacos/seed-config/gateway.yaml`: 增加 Analytics cutover/rollback 配置示例，默认 route mode 仍保持 `logic`。
- `smart-recruit-gateway/internal/runtime/analytics_cutover.go`: 新增 Analytics cutover plan，覆盖 analytics route target、readiness target、rollback 与 reporting smoke 方法清单。
- `smart-recruit-gateway/internal/runtime/analytics_cutover_test.go`: 覆盖 Dashboard/funnel/time-in-stage/interview-offer metrics smoke plan 与 rollback 到 logic。
- `web-gin-service/rpc/analytics_admin_client.go`: 新增 reporting-only AdminService wrapper，将四个 Analytics 报表 RPC 转发到 Analytics 连接，其余 AdminService 方法沿用 base/identity client。
- `web-gin-service/rpc/analytics_admin_client_test.go`: 覆盖 reporting-only 转发，确认 `QueryAuthAuditLogs` 不转发到 Analytics。
- `web-gin-service/rpc/client.go`: 增加 `ANALYTICS_ROUTE_MODE`/`ANALYTICS_GRPC_ADDR`、独立 Analytics gRPC 连接、health check、target 记录和 Admin client 组合。
- `web-gin-service/rpc/client_test.go`: 覆盖 Analytics 切流连接、缺地址、非法 route mode 和 readiness health failure。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=5bb56677d9267523711b77f0cae9ded177eaf3f6 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-023
```

所有变更均匹配 `smart-recruit-gateway/**`、`web-gin-service/**`、`smart-recruit-deploy/**` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。Gateway/web-gin 具备 Analytics reporting route cutover/rollback，服务发现配置有示例；默认 route mode 保持 `logic`，保留静态 fallback 和回滚能力。

## SDD 对照结果

通过。实现只改变内部 gRPC target selection，不改变公开 HTTP/protobuf contract；reporting-only wrapper 防止 Identity-owned audit/RBAC 方法被误切到 Analytics。

## Acceptance 对照结果

通过。Analytics/reporting route 可通过 route mode 切到 `analytics` target；smoke plan 覆盖 dashboard、funnel、time-in-stage、interview-offer metrics；rollback plan 验证 Analytics 回到 logic 时不加入独立 ready target。

## 测试命令和结果

- `cd smart-recruit-gateway && GOWORK=off go test ./...`: passed
- `cd web-gin-service && GOWORK=off go test ./rpc ./config`: passed
- `cd smart-recruit-analytics-service && GOWORK=off go test ./... && GOWORK=off go run ./cmd/analytics-service --check`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 5bb56677d9267523711b77f0cae9ded177eaf3f6 --json`: passed
- `TASK_BASE_TREE=5bb56677d9267523711b77f0cae9ded177eaf3f6 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-023`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-023-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增 web-gin Analytics route mode、reporting-only Admin wrapper、gateway cutover plan、测试和 Nacos seed 示例。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP routes、handlers、protobuf 或生成代码；仅在显式 route mode 下切换内部 gRPC target。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，本 TASK 未修改数据库实例、schema 或跨服务写表边界。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，目标测试、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 API/gateway、service boundaries、auth、local-development、notification-outbox、service-binary-convention、system-overview 等 active knowledge，结论均为 `UNCHANGED`；本 TASK 增加内部 Analytics reporting cutover，不改变公开 HTTP API 或默认本地启动路径。

## 风险

- 本 TASK 使用自动化 route/wrapper/cutover smoke plan 验证 Analytics 报表清单；真实报表口径端到端一致性仍依赖后续 compose/smoke TASK 启动完整栈。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-024。
