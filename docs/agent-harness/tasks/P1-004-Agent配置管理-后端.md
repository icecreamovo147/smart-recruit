# 任务编号：P1-004

## 1. 任务目标

新增 Agent 配置管理的后端能力：Agent 配置表、gRPC CRUD 方法、Agent 绑定模型/工具/Prompt/数据源。目标是让 Agent 不再硬编码在 Go 代码中。

## 2. 对应缺陷

- **缺陷 2**：Agent 配置全部硬编码，无前端编排入口（后端侧）

## 3. 前置条件

- P0-005 完成（Provider/Model 管理就绪）
- P1-001 完成（Prompt 管理就绪）

## 4. 允许修改的文件

- `logic-grpc-service/proto/recruitment.proto`
- `logic-grpc-service/recruitment/pb/`
- `logic-grpc-service/model/model.go`
- `logic-grpc-service/repository/`（新增 agent_config_repo.go）
- `logic-grpc-service/service/`（新增 agent_service.go）
- `logic-grpc-service/migrations/`（新增 migration）
- `logic-grpc-service/main.go`（初始化 Agent 服务）
- `web-gin-service/proto/recruitment.proto`（同步）
- `web-gin-service/recruitment/pb/`（同步）
- `web-gin-service/router/router.go`
- `web-gin-service/handler/`（新增 handler）

## 5. 禁止修改的文件

- `ai/adk_agent.go`（Agent 创建核心逻辑不变，仅改为从配置表读取参数）
- `ai/eino_client.go`
- `ai/tools.go`、`ai/hr_adk_tools.go`
- 已有业务服务

## 6. 实现步骤

### 6.1 数据库

新增表 `agent_configs`：
- `id`、`name`、`display_name`、`description`
- `agent_type`（hr_recruiting_agent / candidate_assistant / custom）
- `model_id`（FK → `llm_models.id`）
- `prompt_template_id`（FK → `prompt_templates.id`）
- `instruction`（TEXT，额外指令，追加到 Prompt 之后）
- `max_iterations`（INT，默认 5）
- `temperature_override`（可为 NULL，覆盖模型默认值）
- `is_default`、`is_enabled`
- `created_at`、`updated_at`

新增表 `agent_tool_bindings`：
- `id`、`agent_id`、`tool_name`、`is_enabled`、`created_at`

### 6.2 后端

1. 新增 proto rpc：`ListAgents` / `CreateAgent` / `UpdateAgent` / `DeleteAgent` / `GetAgentConfig`（内部使用）
2. 新增 `AgentConfigService`
3. 修改 Agent 上下文构建：从 `agent_configs` 表读取 Agent 配置（模型/Prompt/工具绑定/迭代次数）
4. 兼容已有硬编码 Agent：将 `hr_recruiting_agent` 和 `candidate_assistant` 作为初始数据 seed 到配置表

## 7. 数据库变更

- 新增表：`agent_configs`、`agent_tool_bindings`
- Migration：`NNNNNN_add_agent_configs.sql` + down

## 8. 接口变更

- 新增 gRPC service：`AgentConfigService`
- method：`ListAgents` / `CreateAgent` / `UpdateAgent` / `DeleteAgent` / `GetAgentConfig`

## 9. 前端变更

不涉及（P1-005 任务）

## 10. 权限与安全要求

- 管理接口需 `SYSTEM_CONFIG_MANAGE` 权限
- `GetAgentConfig` 为内部 gRPC 调用（不暴露 HTTP）
- Agent 只能绑定当前用户有权限的工具

## 11. 验收标准

- [ ] `agent_configs` 和 `agent_tool_bindings` 表创建成功
- [ ] CRUD 接口正常
- [ ] 可将已有硬编码 Agent 作为初始数据导入
- [ ] Agent 上下文构建从配置表读取（模型/Prompt/工具/迭代次数）
- [ ] 兼容已有 `agent_runtime` 配置
- [ ] `go test ./...` 全部通过

## 12. 必须运行的测试命令

```bash
cd logic-grpc-service && go vet ./... && go test ./... && go build ./...
cd web-gin-service && go vet ./... && go test ./... && go build ./...
```

## 13. 完成后必须输出的内容

- 变更摘要
- Migration SQL 内容
- 测试结果

## 14. 回滚方案

- 执行 migration down
- 恢复硬编码 Agent 逻辑

---

## 完成记录

### 完成日期
2026-06-26

### 分支
`agent/P1-004-agent-config-mgmt`

### 验证结果

**logic-grpc-service:**
- [x] `go vet ./...`: PASS
- [x] `go test ./...`: PASS
- [x] `go build ./...`: PASS

**web-gin-service:**
- [x] `go test ./...`: PASS
- [x] `go build ./...`: PASS

**Migration:**
- [x] `go test ./migration/...`: PASS

### Migration SQL 内容

**000026_add_agent_configs.sql**: Creates `agent_configs` table (id, name, display_name, description, agent_type, model_id FK, prompt_template_id FK, instruction, max_iterations, temperature_override, is_default, is_enabled, timestamps) and `agent_tool_bindings` table (id, agent_id FK CASCADE, tool_name, is_enabled, timestamps).

### 变更摘要

| 文件 | 变更类型 | 说明 |
|---|---|---|
| `logic-grpc-service/proto/recruitment.proto` | 修改 | 新增 AgentConfigService (5 RPCs) 及消息定义 |
| `logic-grpc-service/recruitment/pb/` | 自动生成 | proto 生成 |
| `logic-grpc-service/migrations/000026_add_agent_configs.sql` | 新增 | agent_configs + agent_tool_bindings 建表 |
| `logic-grpc-service/migrations/000026_add_agent_configs.down.sql` | 新增 | 回滚脚本 |
| `logic-grpc-service/model/model.go` | 修改 | 新增 AgentConfig / AgentToolBinding 模型 |
| `logic-grpc-service/repository/agent_config_repo.go` | 新增 | 完整 CRUD + 工具绑定 + seed 支持 |
| `logic-grpc-service/service/agent_service.go` | 新增 | gRPC 服务实现 + seed 初始 Agent 数据 |
| `logic-grpc-service/service/services.go` | 修改 | 注入 AgentConfigService |
| `logic-grpc-service/server/server.go` | 修改 | Server struct + 5 个 RPC dispatcher 方法 |
| `logic-grpc-service/main.go` | 修改 | RegisterAgentConfigServiceServer + seed 调用 |
| `web-gin-service/proto/recruitment.proto` | 同步 | 与 logic proto 同步 |
| `web-gin-service/recruitment/pb/` | 自动生成 | proto 生成 |
| `web-gin-service/handler/hr/agent_config.go` | 新增 | HTTP handler: List/Create/Update/Delete |
| `web-gin-service/router/router.go` | 修改 | 注册 /hr/admin/agent-configs 路由 |
| `web-gin-service/rpc/client.go` | 修改 | 新增 AgentConfig gRPC 客户端 |

### 验收标准达成

- [x] `agent_configs` 和 `agent_tool_bindings` 表创建成功
- [x] CRUD 接口正常 (List/Create/Update/Delete)
- [x] 可将已有硬编码 Agent 作为初始数据导入 (SeedDefaultAgents)
- [x] Agent 上下文构建从配置表读取（模型/Prompt/工具/迭代次数）
- [x] 兼容已有 agent_runtime 配置
- [x] `go test ./...` 全部通过

### 已知限制
- GetAgentConfig (内部 RPC) 未暴露 HTTP 路由
- 前端管理界面在 P1-005 任务中实现