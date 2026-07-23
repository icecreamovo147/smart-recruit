# TASK-MRI-011 Report

## TASK ID

TASK-MRI-011

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-011-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-011-report.md`
- `smart-recruit-deploy/nacos/seed-config/gateway.yaml`
- `smart-recruit-gateway/internal/runtime/identity_cutover.go`
- `smart-recruit-gateway/internal/runtime/identity_cutover_test.go`
- `web-gin-service/rpc/identity_admin_client_test.go`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-011` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-012`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-011-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-011-report.md`: 记录本 TASK 的人工可读验收报告。
- `smart-recruit-deploy/nacos/seed-config/gateway.yaml`: 增加 identity cutover/rollback 配置示例。
- `smart-recruit-gateway/internal/runtime/identity_cutover.go`: 新增 identity cutover smoke plan，覆盖 login、refresh、principal、RBAC 和 audit 方法，并校验 cutover/rollback ready targets。
- `smart-recruit-gateway/internal/runtime/identity_cutover_test.go`: 覆盖 identity 切流目标、ready target、smoke 方法清单和 logic rollback。
- `web-gin-service/rpc/identity_admin_client_test.go`: 验证 RBAC/audit AdminService 子集在 identity mode 下委派到 identity client 而非 logic client。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=2e697dc1664987d3ff4539d08554d70e9de75496 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-011
```

所有变更均匹配 `smart-recruit-gateway/**`、`web-gin-service/**`、`smart-recruit-deploy/**` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。Gateway identity route mode 具备明确 cutover plan，auth/RBAC/principal/audit smoke 方法被纳入验证，rollback 到 logic 保持可用。

## SDD 对照结果

通过。实现未改变公开 HTTP/protobuf contract；只强化 gateway route/client 验证和 identity-owned AdminService 子集委派测试。

## Acceptance 对照结果

通过。测试覆盖 login、refresh、principal、RBAC、audit smoke plan；web-gin RPC 测试验证 RBAC/audit 委派到 identity；rollback plan 验证 identity 回 logic 时不加入独立 ready target。

## 测试命令和结果

- `cd smart-recruit-gateway && GOWORK=off go test ./...`: passed
- `cd web-gin-service && GOWORK=off go test ./rpc ./config`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 2e697dc1664987d3ff4539d08554d70e9de75496 --json`: passed
- `TASK_BASE_TREE=2e697dc1664987d3ff4539d08554d70e9de75496 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-011`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `rm -f go.work.sum smart-recruit-gateway/gateway`: passed, removed local workspace/build artifacts if generated
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-011-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增 cutover/rollback runtime plan、gateway tests、web-gin Admin 委派测试和 Nacos seed 示例。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未改变 HTTP routes 或 protobuf；默认 route mode 仍可回滚 logic。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，本 TASK 未修改数据库实例、schema 或跨服务写表边界。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，目标测试、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/system-overview.md`、`.knowledge/runbooks/local-development.md`、`.knowledge/architecture/service-boundaries.md` 和 `.knowledge/architecture/api-contracts-and-gateway.md`，结论均为 `UNCHANGED`；本 TASK 增加 cutover 验证，不改变公开 HTTP API 或默认本地启动路径。

## 风险

- 本 TASK 使用自动化 smoke plan 和 RPC 委派测试验证 identity cutover；真实登录/刷新/权限端到端请求仍需要后续 compose/smoke 环境。
- Identity cutover 默认仍未启用；生产切流需要显式 route mode 和 rollback 操作。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-012。
