# TASK-MRI-015 Report

## TASK ID

TASK-MRI-015

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-015-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-015-report.md`
- `smart-recruit-deploy/nacos/seed-config/gateway.yaml`
- `smart-recruit-gateway/internal/runtime/offer_cutover.go`
- `smart-recruit-gateway/internal/runtime/offer_cutover_test.go`
- `smart-recruit-offer-service/README.md`
- `smart-recruit-offer-service/cmd/offer-service/main.go`
- `smart-recruit-offer-service/cmd/offer-service/main_test.go`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-015` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-016`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-015-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-015-report.md`: 记录本 TASK 的人工可读验收报告。
- `smart-recruit-deploy/nacos/seed-config/gateway.yaml`: 增加 Offer cutover/rollback 配置示例，默认 route mode 仍保持 `logic`。
- `smart-recruit-gateway/internal/runtime/offer_cutover.go`: 新增 Offer cutover plan，覆盖 Offer route target、readiness target、rollback 和 lifecycle smoke 方法清单。
- `smart-recruit-gateway/internal/runtime/offer_cutover_test.go`: 覆盖 Offer create/send/accept/reject/withdraw/list smoke plan 与 rollback 到 logic。
- `smart-recruit-offer-service/README.md`: 将 Offer serve 示例端口更新为 `:50064`，对齐部署 seed。
- `smart-recruit-offer-service/cmd/offer-service/main.go`: 将 Offer 服务默认 gRPC 端口调整为 `:50064`，避免与 Interview `:50063` 冲突。
- `smart-recruit-offer-service/cmd/offer-service/main_test.go`: 将 Nacos instance/static fallback 测试端口更新为 `50064`。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=da37cac77e634623ed28bb91ac64593252691a5e bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-015
```

所有变更均匹配 `smart-recruit-gateway/**`、`smart-recruit-offer-service/**`、`smart-recruit-deploy/**` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。Gateway 具备 Offer route cutover/rollback 验证计划，Offer 服务端口与部署发现配置一致；默认 route mode 保持 `logic`，不影响现有回滚。

## SDD 对照结果

通过。实现只强化 gateway route/cutover 计划和服务发现配置，不改变公开 HTTP/protobuf contract，不新增跨服务写表，不拆分 MySQL。

## Acceptance 对照结果

通过。Offer route 可通过 route mode 切到 `offer` target；smoke plan 覆盖 create/send/accept/reject/withdraw/list；rollback plan 验证 Offer 回到 logic 时不加入独立 ready target。

## 测试命令和结果

- `cd smart-recruit-gateway && GOWORK=off go test ./...`: passed
- `cd web-gin-service && GOWORK=off go test ./rpc ./config`: passed
- `cd smart-recruit-offer-service && GOWORK=off go test ./...`: passed
- `cd smart-recruit-offer-service && GOWORK=off go run ./cmd/offer-service --check`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree da37cac77e634623ed28bb91ac64593252691a5e --json`: passed
- `TASK_BASE_TREE=da37cac77e634623ed28bb91ac64593252691a5e bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-015`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-015-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增 Offer cutover runtime plan、测试、Nacos seed 示例和服务端口对齐。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP routes、handlers、protobuf 或生成代码；web-gin route mode 既有逻辑保持。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，本 TASK 未修改数据库实例、schema 或跨服务写表边界。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，目标测试、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/system-overview.md` 和 `.knowledge/runbooks/local-development.md`，结论均为 `UNCHANGED`；本 TASK 增加 Offer cutover 验证和 seed 示例，不改变公开 HTTP API 或默认本地启动路径。

## 风险

- 本 TASK 使用自动化 cutover smoke plan 验证 Offer lifecycle RPC 清单；真实 HTTP 端到端 create/send/accept/reject/withdraw/list 请求仍依赖后续 compose/smoke TASK 启动完整栈。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-016。
