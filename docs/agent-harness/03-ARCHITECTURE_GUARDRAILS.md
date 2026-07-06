# 03-ARCHITECTURE_GUARDRAILS — 架构护栏

> 版本：v1.0
> 创建日期：2026-06-26

---

## 1. 前端路由和菜单新增规则

### 1.1 路由新增规范

- 新路由必须声明 `meta.perm`（权限常量引用 `PERM.SYSTEM_CONFIG_MANAGE` 或新增权限常量）
- 新增权限常量必须在 `web-gin-service/pkg/authz/` 和 `hr-frontend` 权限定义中同步
- 路由路径格式：`/hr/admin/<feature-name>`
- 必须包含 `meta.title` 中文标题

### 1.2 菜单新增规范

- 所有菜单项必须有图标 + 文字说明（与已有规范一致）
- 仅在菜单项路由实际存在后才添加到菜单
- 使用 `<el-icon>` 包装 Element Plus Icons 图标
- 管理类功能放在「基础数据」折叠组内或独立分组

### 1.3 前端组件新增规范

- 使用 Element Plus 组件，保持与现有风格一致
- 使用 Vue 3 Composition API (`<script setup lang="ts">`)
- 使用 TypeScript 严格模式
- API 调用统一通过 `src/api/` 模块
- 类型定义统一在 `src/types/` 模块

---

## 2. gRPC / HTTP 接口新增规则

### 2.1 Proto 修改规则

- Proto 文件位置：`logic-grpc-service/proto/recruitment.proto`（主定义）
- 新增 Service：按功能模块定义独立 service
- 新增 method：命名格式 `VerbNoun`（如 `ListProviders`、`CreateProvider`）
- Message 字段使用 `snake_case`
- 新增字段只能追加，不能修改已有字段编号或语义
- 修改后需同步更新 `web-gin-service/proto/` 的副本
- 重新生成 pb.go 文件（`recruitment/pb/`）

### 2.2 HTTP API 新增规则

- 在 `web-gin-service/router/router.go` 注册路由
- 必须声明 `perm` 权限常量
- Handler 放在 `web-gin-service/handler/` 对应子目录
- 通过 gRPC 客户端调用 logic 服务，不直接在网关写业务逻辑

### 2.3 接口设计规范

- 列表接口必须支持分页（`page` + `page_size`）
- 错误返回统一使用 `errs.New(code, message)` 格式
- SSE 流式接口遵循已有 `StreamPayload` 结构

---

## 3. Migration 新增规则

### 3.1 文件规范

- 位置：`logic-grpc-service/migrations/`
- 命名：`NNNNNN_description.sql`（版本号 6 位，递增）
- 每个版本对应一个 `.sql` 文件
- **仅允许 ADD COLUMN（新增字段）或 CREATE TABLE（新增表）**
- **禁止** DROP COLUMN / DROP TABLE / RENAME COLUMN / MODIFY COLUMN 改变已有字段类型
- 必须同步创建对应的 `.down.sql` 回滚文件

### 3.2 migration 检查

- migration 由 logic 服务启动时自动执行（runner.go）
- 执行前做 MySQL advisory lock 防并发
- migration 文件有 checksum 校验
- 新增 migration 后必须运行 `go test ./migration/...`

---

## 4. 权限校验规则

### 4.1 原则

- 所有新增的管理类接口必须有权限校验
- 鉴权决策通过现有 RBAC 中间件（不新建独立鉴权体系）
- 权限常量定义在 `web-gin-service/pkg/authz/permissions.go`
- 工具级权限：M4 阶段评估是否需要新增 `TOOL_*` 权限系列

### 4.2 数据权限

- Agent 工具查询必须限定当前 HR 的数据范围（已有 `hr_id` 过滤模式）
- 候选人端 AI 只能查询本人数据
- 管理员可见全部数据但需审计日志

---

## 5. API Key 和敏感信息处理规则

### 5.1 存储

- **禁止明文存储 API Key 在数据库**
- 存储方案：AES-256-GCM 加密，密钥通过环境变量 `ENCRYPTION_KEY` 注入
- 前端展示 API Key 时只显示前 4 位 + 后 4 位，中间 `****`

### 5.2 传输

- API Key 只能通过 HTTPS 传输
- **禁止在 URL query string 中传递 API Key**
- **禁止在日志中打印完整 API Key**

### 5.3 前端

- **禁止在前端 localStorage / sessionStorage 存储完整 API Key**
- 加密后的 Key 不返回前端（由后端解密后调用 LLM）

---

## 6. Trace 数据脱敏规则

### 6.1 工具入参/结果脱敏

- 工具调用 Trace 中 **不得记录**：
  - 候选人手机号、身份证号
  - 候选人邮箱
  - 候选人家庭住址
  - 简历文件直接内容
- 记录前必须通过脱敏中间件处理，替换为 `***` 或字段类型标识

### 6.2 Trace 数据保留

- Trace 数据保留期：90 天（可在配置中调整）
- 超过保留期的 Trace 自动归档或删除

---

## 7. Prompt 变更审计规则

- Prompt 每次修改必须记录：操作人、操作时间、变更前内容、变更后内容
- Prompt 版本号自动递增
- 支持 Prompt 回滚到历史版本
- Prompt 变更后生效：立即（下一次调用时使用最新版本）

---

## 8. MCP 工具安全规则

- 新增 MCP Server 时需管理员权限
- 外部 MCP 工具默认禁用，需手动启用
- MCP 工具调用需要记录审计日志（工具名/入参/结果/耗时/用户）
- 高风险操作（写操作、外部 API 调用）需前端二次确认
- MCP 连接测试必须设超时（默认 10s）

---

## 9. RAG / 文件上传安全规则

### 9.1 知识库文件

- 上传文件需校验类型（白名单：PDF、DOCX、TXT、MD）
- 文件大小限制（默认 20MB）
- 上传文件需隔离到私有 Bucket
- 切片后的文本内容不得包含 PII（个人身份信息）

### 9.2 向量检索

- 检索结果需做权限过滤（知识库有访问权限控制）
- 引用来源展示时最多显示文档标题 + 片段摘要

---

## 10. 不破坏现有功能的规则

### 10.1 不允许的操作

- **不允许重写** `ai/adk_agent.go` 的 ADK Agent 核心创建逻辑（除非 M6 任务明确要求）
- **不允许重写** `ai/eino_client.go` 的 Legacy ReAct 循环
- **不允许删除** 已有 gRPC method
- **不允许更改** 已有数据库表结构（只能 ADD COLUMN）
- **不允许删除** 已有前端路由或页面
- **不允许修改** 已有中间件链顺序

### 10.2 允许的操作

- 新增 Go 文件（在已有包中新增或新建包）
- 新增 Vue 组件
- 新增 DB migration
- 新增 proto message 和 method（追加）
- 新增测试文件
- 在已有文件末尾追加函数/方法（不得在已有逻辑中插入代码）

---

## 11. 提交规范

- Commit message 遵循 Conventional Commits：`feat:` / `fix:` / `refactor:` / `test:` / `docs:` / `chore:`
- 每个任务至少一个独立 commit
- Commit message 需包含任务编号：`feat(P0-001): add retry button for HR AI chat failed messages`
