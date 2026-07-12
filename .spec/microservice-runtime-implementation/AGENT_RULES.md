# Agent 规则 - microservice-runtime-implementation

## 核心原则

- 这是 implementation feature，不是规划文档 feature。
- 每个 TASK 必须真实落地代码、配置、部署、脚本、测试或运行证据。
- 禁止把“创建下游 spec”作为 TASK 完成条件。
- 数据库保持单 MySQL 实例。
- 服务间通信使用 gRPC 或 RabbitMQ，不允许新增跨服务直接写表。
- 每个 TASK 结束后必须生成 report/evidence，并在通过 review 后再进入下一 TASK。

## 允许的真实落地范围

根据具体 TASK，可以修改：

- `smart-recruit-*/**`
- `web-gin-service/**`
- `logic-grpc-service/**`
- `docker/**`
- `deploy/**`
- `scripts/**`
- `docs/**`
- `.knowledge/**`
- `.spec/microservice-runtime-implementation/**`
- `go.work`
- `**/go.mod`
- `**/go.sum`

## 禁止事项

- 不得提交 secrets 或真实 `.env`。
- 不得拆 MySQL 实例/schema。
- 不得删除旧 monolith fallback，直到 retirement TASK 验证通过。
- 不得跳过失败的 scope check、agent-check、go test 或 evidence validation。

## 每 TASK 必跑

```bash
git diff --name-only
TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh <TASK-ID>
bash .spec/microservice-runtime-implementation/scripts/agent-check.sh
node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/<TASK-ID>-evidence.json
```

## Hard Stop

- 公开 HTTP API 行为变更未在 TASK 中声明。
- 数据库拆分。
- secrets 泄漏。
- 无法构建或测试关键服务。
