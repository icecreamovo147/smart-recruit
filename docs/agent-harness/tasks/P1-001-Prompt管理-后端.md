# 任务编号：P1-001

## 1. 任务目标

新增 Prompt 模板管理后端能力：新建 `prompt_templates` 表、gRPC CRUD 方法、版本管理、变量插值支持。目标是让 System Prompt 不再硬编码在 Go 代码中。

## 2. 对应缺陷

- **缺陷 3**：System Prompt 硬编码，无 Prompt 管理/版本（后端侧）

## 3. 前置条件

- M2 阶段（建议 P0-005/P0-006 完成后开始）

## 4. 允许修改的文件

- `logic-grpc-service/proto/recruitment.proto`
- `logic-grpc-service/recruitment/pb/`
- `logic-grpc-service/model/model.go`
- `logic-grpc-service/repository/`（新增 prompt_template_repo.go）
- `logic-grpc-service/service/`（新增 prompt_service.go）
- `logic-grpc-service/migrations/`（新增 migration）
- `logic-grpc-service/service/agent_context.go`（改为从 Prompt 服务读取）
- `web-gin-service/proto/recruitment.proto`（同步）
- `web-gin-service/recruitment/pb/`（同步）
- `web-gin-service/router/router.go`
- `web-gin-service/handler/`（新增 handler）

## 5. 禁止修改的文件

- `ai/adk_agent.go`（不改变 Agent 核心创建，仅改变 Instruction 注入来源）
- `ai/tools.go`、`ai/hr_adk_tools.go`
- `ai/eino_client.go`
- 已有业务服务

## 6. 实现步骤

### 6.1 数据库

新增表 `prompt_templates`：
- `id`, `name`, `content`（TEXT，含 `{{variable}}` 占位符）, `variables`（JSON 数组，如 `["job_title","department"]`）, `version`（整数）, `is_active`, `agent_type`（hr_agent / candidate_assistant）, `prompt_role`（system / user）, `created_by`, `updated_by`, `created_at`, `updated_at`
新增表 `prompt_versions`：
- `id`, `template_id`, `version`, `content`, `changed_by`, `change_note`, `created_at`

### 6.2 后端

1. 新增 proto：`ListPromptTemplates` / `CreatePromptTemplate` / `UpdatePromptTemplate` / `DeletePromptTemplate` / `GetPromptVersionHistory` / `RollbackPromptVersion`
2. 新增 `PromptService`
3. 变量插值：`RenderPrompt(templateID, vars map[string]string) (string, error)`
4. 修改 `agent_context.go`：System Prompt 从 Prompt 服务读取（按 `agent_type` + `is_active=true`）
5. 版本管理：每次更新 content 自动创建新版本记录
6. 变更审计：每次修改记录操作人和变更内容

## 7. 数据库变更

- 新增表：`prompt_templates`、`prompt_versions`
- Migration：`NNNNNN_add_prompt_templates.sql` + down

## 8. 接口变更

- 新增 gRPC service：`PromptService`（或追加到 AdminService）
- method：`ListPromptTemplates` / `CreatePromptTemplate` / `UpdatePromptTemplate` / `DeletePromptTemplate` / `GetPromptVersionHistory` / `RollbackPromptVersion`

## 9. 前端变更

不涉及（P1-002 任务）

## 10. 权限与安全要求

- 管理接口需 `SYSTEM_CONFIG_MANAGE` 权限
- Prompt 内容不对外部用户暴露
- `render` 接口仅内部 gRPC 调用（不暴露 HTTP）
- Prompt 变更记录完整审计（操作人、时间、变更内容）

## 11. 验收标准

- [ ] `prompt_templates` 和 `prompt_versions` 表创建成功
- [ ] CRUD 接口正常
- [ ] 版本历史可查询
- [ ] 回滚功能正常（恢复到历史版本）
- [ ] Agent 上下文构建改为从 DB 读取 Prompt
- [ ] 已有硬编码 Prompt 可作为初始模板导入
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

## 完成记录

- **完成日期**: 2026-06-26
- **完成分支**: `integration/agent-platform`（当前分支）
- **验证结果**:
  - [x] `go vet ./...` — logic-grpc-service: PASS; web-gin-service: PASS（仅有预存 vet 警告）
  - [x] `go test ./...` — logic-grpc-service: ALL PASS; web-gin-service: ALL PASS
  - [x] `go build ./...` — logic-grpc-service: PASS; web-gin-service: PASS
  - [x] Migration 版本号连续
  - [x] 新增权限校验（SYSTEM_CONFIG_MANAGE）
  - [x] 无 TODO/FIXME 残留
  - [x] 无调试代码

## 14. 回滚方案

- 执行 migration down
- 恢复 `agent_context.go` 硬编码 Prompt 逻辑
