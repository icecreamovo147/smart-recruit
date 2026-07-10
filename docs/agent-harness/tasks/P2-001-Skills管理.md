# 任务编号：P2-001

## 1. 任务目标

引入 Agent Skills 框架（选型需 ADR），支持 Skills 安装/列表/启用禁用/配置/绑定到 Agent。目标是将可复用的 Agent 能力封装为 Skills 包。

> 注意：Skills 是对 MCP 工具的进一步封装和编排。如 M4 阶段 MCP 工具未就绪，本任务可能延后或与 MCP 工具合并设计。

## 2. 对应缺陷

- **缺陷 5**：无 Skills 能力

## 3. 前置条件

- P1-006 完成（MCP 工具中心就绪，Skills 可基于 MCP 工具编排）
- P1-004 完成（Agent 配置管理就绪，Skills 可绑定到 Agent）
- 需评估：如果业务对 Skills 需求不急迫，本任务可推迟或取消

## 4. 允许修改的文件

- `logic-grpc-service/go.mod`
- `logic-grpc-service/proto/recruitment.proto`
- `logic-grpc-service/recruitment/pb/`
- `logic-grpc-service/model/model.go`
- `logic-grpc-service/repository/`（新增 skill_repo.go）
- `logic-grpc-service/service/`（新增 skill_service.go）
- `logic-grpc-service/migrations/`（新增 migration）
- `logic-grpc-service/main.go`
- `hr-frontend/` 新增 Skills 管理页面
- 同步到 `web-gin-service/`

## 5. 禁止修改的文件

- `ai/adk_agent.go`
- `ai/hr_adk_tools.go`
- 已有业务服务

## 6. 实现步骤

### 6.1 数据库

新增表 `skills`：
- `id`、`name`、`display_name`、`description`、`version`
- `manifest`（JSON，Skills 元信息/依赖工具/配置项）
- `is_enabled`、`created_at`、`updated_at`

新增表 `agent_skill_bindings`：
- `id`、`agent_id`、`skill_id`、`is_enabled`、`created_at`

### 6.2 后端

1. 新增 proto rpc：
   - `ListSkills` / `InstallSkill` / `UninstallSkill` / `EnableSkill` / `DisableSkill` / `ConfigureSkill`
2. Skills 执行：通过编排 MCP 工具调用序列 + Prompt 注入实现
3. Skills 执行日志记录（复用 MCP tool_logs 或新增表）

### 6.3 前端

- 新增「Skills 管理」页面（列表 + 安装/卸载 + 启用/禁用 + 配置）

## 7. 数据库变更

- 新增表：`skills`、`agent_skill_bindings`
- Migration：`NNNNNN_add_skills.sql` + down

## 8. 接口变更

- 新增 gRPC service：`SkillService`
- method：`ListSkills` / `InstallSkill` / `UninstallSkill` / `EnableSkill` / `DisableSkill` / `ConfigureSkill`

## 9. 前端变更

- 新增 `SkillManageView.vue`
- 新增 `api/skill.ts`、`types/skill.ts`
- 修改 `router/index.ts` + `App.vue`

## 10. 权限与安全要求

- Skills 管理接口需 `SYSTEM_CONFIG_MANAGE` 权限
- Skills 安装需审批或管理员权限
- 外部 Skills 默认禁用

## 11. 验收标准

- [ ] Skills CRUD 可用
- [ ] Skills 可启用/禁用
- [ ] Skills 可绑定到 Agent
- [ ] Skills 执行有日志记录
- [ ] `go test ./...` 通过
- [ ] `pnpm typecheck` + `pnpm build` 通过

## 12. 必须运行的测试命令

```bash
cd logic-grpc-service && go vet ./... && go test ./... && go build ./...
cd web-gin-service && go vet ./... && go test ./... && go build ./...
cd hr-frontend && pnpm install && pnpm typecheck && pnpm build
```

## 13. 完成后必须输出的内容

- ADR（Skills 框架选型）
- 变更摘要
- Migration SQL 内容
- 测试结果

## 14. 回滚方案

- 执行 migration down
- 移除 Skills 相关代码
