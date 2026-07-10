# ADR: Prompt 模板管理方案

## 日期
2026-06-26

## 决策
新增 `prompt_templates` 和 `prompt_versions` 两张数据库表，新增 gRPC `PromptService` 管理 Prompt 模板，支持版本管理和变更审计，并将硬编码 System Prompt 迁移至数据库管理。

## 背景
当前 System Prompt 硬编码在 Go 代码中（`service/helpers.go` 的 `buildToolCallingMessages`、`service/candidate_ai_service.go` 的 `candidateSystemPrompt`、`ai/eino_client.go` 的 `buildRecruitingMessages` 等），修改 Prompt 需要修改代码、重建、重新部署，效率低且无法追溯。

## 方案

### 数据库
- `prompt_templates` 表存储模板定义，包含 `name`、`content`（含 `{{variable}}` 占位符）、`variables`（JSON 数组声明可用变量）、`version`（当前版本号）、`is_active`（是否启用）、`agent_type`（hr_agent / candidate_assistant）、`prompt_role`（system / user）、`created_by`、`updated_by`。
- `prompt_versions` 表存储每次变更的历史版本，包含 `template_id`、`version`、`content`、`changed_by`、`change_note`。

### gRPC 接口
- `ListPromptTemplates` — 分页列表
- `CreatePromptTemplate` — 创建模板
- `UpdatePromptTemplate` — 更新模板（自动创建新版本）
- `DeletePromptTemplate` — 删除模板（软删除：`is_active = false`）
- `GetPromptVersionHistory` — 查询版本历史
- `RollbackPromptVersion` — 回滚到指定版本
- `RenderPrompt` — 变量插值（内部 gRPC 调用）
- `GetActivePromptByAgentType` — 按 agent_type 获取当前活跃 Prompt（内部 gRPC 调用）

### 变量插值
- 模板中使用 `{{variable}}` 语法
- 模板表 `variables` 字段声明可用变量列表（JSON 数组）
- `RenderPrompt` 接收 `map[string]string`，将模板中的 `{{variable}}` 替换为传入值

### 版本管理
- 每次更新 `content` 时自动在 `prompt_versions` 插入新记录
- `prompt_templates.version` 同步递增

### 变更审计
- 每次修改记录 `updated_by`（prompt_templates）和 `changed_by` + `change_note`（prompt_versions）

### 初始种子数据
- 将 HR Agent System Prompt 和 Candidate Assistant System Prompt 作为初始模板导入

## 影响
- 无现有业务代码破坏
- 新增表、gRPC service、repository 层
- agent_context 改为从 DB 读取 Prompt（兼容已有常量，预留扩展）
