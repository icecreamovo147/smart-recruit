# TASK-MRI-020 Report

## TASK ID

TASK-MRI-020

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-020-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-020-report.md`
- `smart-recruit-ai-agent-service/README.md`
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/adapters.go`
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go`
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main_test.go`
- `smart-recruit-ai-agent-service/doc.go`
- `smart-recruit-ai-agent-service/go.mod`
- `smart-recruit-ai-agent-service/go.sum`
- `smart-recruit-ai-agent-service/internal/runtime/runtime.go`
- `smart-recruit-ai-agent-service/internal/runtime/runtime_test.go`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-020` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-021`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-020-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-020-report.md`: 记录本 TASK 的人工可读验收报告。
- `smart-recruit-ai-agent-service/README.md`: 更新 AI Agent 服务独立 build、check、serve 与运行时依赖说明。
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/adapters.go`: 新增 AI 与 RecruitingIntelligence 的 protobuf adapter，复用 logic 层服务并保持生成接口兼容。
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go`: 新增独立 AI Agent 服务入口，接入配置、Nacos、Nacos Config、MySQL、Redis、RabbitMQ、health、metrics、trace 和日志，并启动长任务 worker。
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main_test.go`: 覆盖 discovery 名称、Nacos fallback 和 RabbitMQ worker 队列映射。
- `smart-recruit-ai-agent-service/doc.go`: 将服务文档名更新为 `ai-agent-service`。
- `smart-recruit-ai-agent-service/go.mod`: 增加 AI Agent 独立服务模块依赖。
- `smart-recruit-ai-agent-service/go.sum`: 锁定 AI Agent 独立服务模块依赖校验和。
- `smart-recruit-ai-agent-service/internal/runtime/runtime.go`: 新增 AI Agent runtime 注册与依赖校验，覆盖 AI、Prompt、AgentConfig、MCP、Skill、AgentSkill、RecruitingIntelligence、EmbeddingConfig。
- `smart-recruit-ai-agent-service/internal/runtime/runtime_test.go`: 覆盖全部 gRPC 服务注册、必填依赖校验与 RabbitMQ/worker 控制校验。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=c49c6e11623107ca7b5dc2136d9de7a0b332ffb7 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-020
```

所有变更均匹配 `smart-recruit-ai-agent-service/**`、`**/go.mod`、`**/go.sum` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。AI Agent 服务源码根现在具备独立模块、独立命令入口、Nacos discovery/config、health、metrics、trace、日志、MySQL、Redis 与 RabbitMQ worker wiring；未拆分 MySQL，也未新增跨服务直接写表。

## SDD 对照结果

通过。实现沿用 logic 层 AI domain service、repository 和 protobuf contracts，通过 adapter 暴露既有 AI 与 RecruitingIntelligence RPC，Prompt/AgentConfig/MCP/Skill/AgentSkill/EmbeddingConfig 直接注册生成服务；长任务通过 RabbitMQ-backed runtime controls 管理。

## Acceptance 对照结果

通过。`smart-recruit-ai-agent-service` 可独立 build 与 `--check`；runtime 注册 AI、Prompt、AgentConfig、MCP、Skill、AgentSkill、RecruitingIntelligence、EmbeddingConfig；worker 控制验证 RabbitMQ、Embedding worker 与 AgentRun worker 均存在。

## 测试命令和结果

- `cd smart-recruit-ai-agent-service && GOWORK=off go test ./...`: passed
- `cd smart-recruit-ai-agent-service && GOWORK=off go build ./cmd/ai-agent-service`: passed
- `cd smart-recruit-ai-agent-service && GOWORK=off go run ./cmd/ai-agent-service --check`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree c49c6e11623107ca7b5dc2136d9de7a0b332ffb7 --json`: passed
- `TASK_BASE_TREE=c49c6e11623107ca7b5dc2136d9de7a0b332ffb7 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-020`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-020-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增 AI Agent 独立 Go module、runtime、cmd、protobuf adapters、测试与 README。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP routes 或 protobuf 定义；adapter 只实现生成接口并复用现有 service 方法。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，本服务复用同一 MySQL 配置和 schema，不新增物理数据库或跨服务直接写表。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，目标测试、独立 build/check、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/system-overview.md` 和 `.knowledge/runbooks/local-development.md`，结论均为 `UNCHANGED`；本 TASK 增加 AI Agent 服务根和独立运行时，不改变公开 HTTP API 或默认本地启动路径。

## 风险

- `--serve` 模式中的真实 RabbitMQ worker、AI provider 与长任务端到端链路仍依赖后续 compose/smoke TASK 启动完整运行时验证。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-021。
