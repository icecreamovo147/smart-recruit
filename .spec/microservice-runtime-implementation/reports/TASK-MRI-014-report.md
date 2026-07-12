# TASK-MRI-014 Report

## TASK ID

TASK-MRI-014

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-014-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-014-report.md`
- `smart-recruit-offer-service/README.md`
- `smart-recruit-offer-service/cmd/offer-service/main.go`
- `smart-recruit-offer-service/cmd/offer-service/main_test.go`
- `smart-recruit-offer-service/doc.go`
- `smart-recruit-offer-service/go.mod`
- `smart-recruit-offer-service/go.sum`
- `smart-recruit-offer-service/internal/runtime/runtime.go`
- `smart-recruit-offer-service/internal/runtime/runtime_test.go`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-014` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-015`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-014-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-014-report.md`: 记录本 TASK 的人工可读验收报告。
- `smart-recruit-offer-service/README.md`: 更新 Offer 服务根为可构建、可运行 gRPC runtime，并记录测试/启动命令。
- `smart-recruit-offer-service/cmd/offer-service/main.go`: 新增独立 Offer gRPC runtime 入口，支持 `--check` 和 `--serve`，接入共享 MySQL、可选 Redis、Nacos Config/Discovery、health、metrics、trace 和日志。
- `smart-recruit-offer-service/cmd/offer-service/main_test.go`: 覆盖 Offer Nacos discovery 名称、实例元数据和本地 static fallback。
- `smart-recruit-offer-service/doc.go`: 将模块服务名对齐为 `offer-service`。
- `smart-recruit-offer-service/go.mod`: 增加 Offer runtime 所需 Go 依赖、本地 `logic-grpc-service` 和 `smart-recruit-platform-go` replace。
- `smart-recruit-offer-service/go.sum`: 记录 Offer 模块依赖校验和。
- `smart-recruit-offer-service/internal/runtime/runtime.go`: 新增 Offer runtime bridge，显式注册 `OfferService` gRPC server 并代理所有 Offer RPC。
- `smart-recruit-offer-service/internal/runtime/runtime_test.go`: 覆盖 runtime 依赖校验和 `OfferService` 注册。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=301af09ea3f5d7767fa590154d92d0b3bfb5ac7e bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-014
```

所有变更均匹配 `smart-recruit-offer-service/**` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。Offer 服务已具备独立源码根、独立构建入口、独立 gRPC 启动路径，并通过 Nacos/health/metrics/trace/logging 接入微服务 runtime；未拆分数据库。

## SDD 对照结果

通过。实现沿用现有 `OfferService` 业务逻辑和共享 MySQL schema，通过 gRPC server 注册暴露服务；没有新增跨服务直接写表，也没有改变 HTTP/protobuf contract。

## Acceptance 对照结果

通过。`smart-recruit-offer-service` 可独立 `go test`、`go build` 和 `go run --check`；runtime 注册 `OfferService`；服务协作保持在现有 gRPC/事件边界内，未新增跨服务写表。

## 测试命令和结果

- `cd smart-recruit-offer-service && GOWORK=off go test ./...`: passed
- `cd smart-recruit-offer-service && GOWORK=off go build ./cmd/offer-service`: passed
- `cd smart-recruit-offer-service && GOWORK=off go run ./cmd/offer-service --check`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 301af09ea3f5d7767fa590154d92d0b3bfb5ac7e --json`: passed
- `TASK_BASE_TREE=301af09ea3f5d7767fa590154d92d0b3bfb5ac7e bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-014`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-014-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增独立 Offer service runtime、cmd、测试、模块依赖和运行验证。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP route、protobuf 或生成代码。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，服务复用共享 MySQL 配置和 schema，未拆分数据库。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，目标测试、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已审阅 `.knowledge/architecture/system-overview.md`、`.knowledge/runbooks/local-development.md`、`.knowledge/architecture/service-boundaries.md` 和 `.knowledge/runbooks/service-binary-convention.md`。前两者保持 `UNCHANGED`；后两者记录 `CANDIDATE` 知识债，因为它们仍描述旧的 `logic-grpc-service/cmd/offer-service` skeleton，后续服务约定或部署 TASK 应更新为 `smart-recruit-offer-service` runtime 指引。

## 风险

- `--serve` 需要本地共享 MySQL 和可选 Redis/Nacos 配置；本 TASK 已验证构建和 runtime wiring，完整 compose/live smoke 由后续 Docker Compose/runtime TASK 覆盖。
- gateway 默认仍走 monolith，Offer 切流和 rollback 验证留给 TASK-MRI-015。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-015。
