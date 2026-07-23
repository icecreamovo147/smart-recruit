# TASKS - dev-log-viewer

按顺序执行 TASK。每个 TASK 必须完成实现、Harness 检查、独立只读复审、修复闭环、报告与 evidence 后，才能开始下一项。`task-scope.json` 中的状态只是初始元数据，运行状态以 `pipeline-state.json` 为准。

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-DLV-001 | 建立独立模块与依赖基线 | pending | workspace、Go module、React/Vite/Tailwind 脚手架 | acceptance/TASK-DLV-001.md |
| TASK-DLV-002 | 服务目录、进程状态与基础 HTTP | pending | catalog、配置、healthz、services API | acceptance/TASK-DLV-002.md |
| TASK-DLV-003 | 有界文件 Tail 与混合日志解析 | pending | snapshot、follow、reset、parser | acceptance/TASK-DLV-003.md |
| TASK-DLV-004 | EventHub、SSE 与断线恢复 | pending | 事件环、订阅、补发、dropped、SSE | acceptance/TASK-DLV-004.md |
| TASK-DLV-005 | 前端日志状态模型与流连接 | pending | types、API、reducer、hooks | acceptance/TASK-DLV-005.md |
| TASK-DLV-006 | Canonical UI 与虚拟日志列表 | pending | AppShell、侧栏、筛选栏、列表、详情、响应式 | acceptance/TASK-DLV-006.md |
| TASK-DLV-007 | 完成交互状态、导出与布局持久化 | pending | 暂停/跟随、重连、空状态、导出、localStorage | acceptance/TASK-DLV-007.md |
| TASK-DLV-008 | 静态嵌入、启停脚本与运行文档 | pending | go:embed、构建链、start/stop、README、知识更新 | acceptance/TASK-DLV-008.md |
| TASK-DLV-009 | 集成、安全与视觉验收 | pending | 跨层验证、安全回归、设计基线、知识影响复核 | acceptance/TASK-DLV-009.md |

## TASK-DLV-001 - 建立独立模块与依赖基线

### Goal

在仓库根目录建立可独立构建和测试的 `dev-log-viewer/` Go + React 工程骨架，并纳入 pnpm workspace。

### Scope

- 新增 Go module、React + TypeScript + Vite + Tailwind CSS package 和基础入口。
- 引入 SPEC 已批准的 Vitest、Testing Library 与 `@tanstack/react-virtual`。
- 更新 `pnpm-workspace.yaml` 与根 `pnpm-lock.yaml`。
- 只提供可编译的占位应用，不实现日志功能或正式 UI。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-001 为准。

### Forbidden Files

业务服务、Gateway、三个现有 Vue 应用、启动脚本和 `.knowledge/**`。

### Dependencies

无。

### Acceptance Criteria

见 `acceptance/TASK-DLV-001.md`。

### Required Tests

- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`
- `pnpm --filter dev-log-viewer build`
- `cd dev-log-viewer && go test ./...`

### Risks

workspace 和锁文件属于共享配置；本次用户已批准列明依赖，但不得顺带升级其他 package。

### Notes

完成前必须记录锁文件实际变化和依赖版本。

## TASK-DLV-002 - 服务目录、进程状态与基础 HTTP

### Goal

实现固定 12 服务目录、仓库根定位、PID/日志状态，以及只监听 loopback 的基础 HTTP 服务。

### Scope

- 实现 catalog/config/server/cmd 基础模块。
- 提供 `GET /healthz` 和 `GET /api/v1/services`。
- 验证 PID 缺失、非法、陈旧及日志 missing/unreadable 状态。
- 禁止通过请求参数读取任意路径。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-002 为准。

### Forbidden Files

前端业务代码、tail/parser/SSE、workspace、锁文件、启停脚本。

### Dependencies

TASK-DLV-001。

### Acceptance Criteria

见 `acceptance/TASK-DLV-002.md`。

### Required Tests

- `cd dev-log-viewer && go test ./internal/catalog ./internal/config ./internal/server ./cmd/dev-log-viewer`
- `cd dev-log-viewer && go vet ./...`

### Risks

PID 只表示进程存活，API 和 UI 文案不得表达为健康状态。

### Notes

仓库根必须可显式传入，测试不得依赖开发者绝对路径。

## TASK-DLV-003 - 有界文件 Tail 与混合日志解析

### Goal

可靠读取已有日志末尾和新增内容，并把混合日志安全转换为有界记录。

### Scope

- 实现末尾 300 行快照、最大 1000 行校验和增量读取。
- 处理 UTF-8 分片、半行、截断、删除重建、文件替换和单文件错误隔离。
- 清理 ANSI，解析 Zap/Gin/Vite/JSON 尾部、多行堆栈与关联 ID。
- 对超长记录、行数和解析失败做有界降级。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-003 为准。

### Forbidden Files

HTTP/SSE、React、workspace、启停脚本和知识库。

### Dependencies

TASK-DLV-002。

### Acceptance Criteria

见 `acceptance/TASK-DLV-003.md`。

### Required Tests

- `cd dev-log-viewer && go test ./internal/tailer ./internal/parser -race`
- `cd dev-log-viewer && go vet ./...`

### Risks

多行组装是启发式；不得因无法分类而丢失原始文本。

### Notes

测试必须使用临时目录和仓库实际日志格式的脱敏 fixture，不复制真实日志数据。

## TASK-DLV-004 - EventHub、SSE 与断线恢复

### Goal

以单条 SSE 连接提供多服务实时流、事件补发和有界背压隔离。

### Scope

- 实现 20,000 条事件环、单调事件 ID、订阅过滤和 2,048 条客户端队列。
- 支持 `Last-Event-ID` 补发、不可恢复 reset/snapshot、heartbeat 和 dropped 通知。
- 将 catalog、tailer、parser、stream 接入 HTTP 生命周期与优雅退出。
- 慢客户端、断开连接和单服务异常不得阻塞其他消费者。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-004 为准。

### Forbidden Files

React、workspace、锁文件、启停脚本和知识库。

### Dependencies

TASK-DLV-003。

### Acceptance Criteria

见 `acceptance/TASK-DLV-004.md`。

### Required Tests

- `cd dev-log-viewer && go test ./internal/stream ./internal/server ./internal/tailer -race`
- `cd dev-log-viewer && go test ./...`
- `cd dev-log-viewer && go vet ./...`

### Risks

SSE 写阻塞和重连重复事件容易造成资源泄漏或日志丢失，必须以并发测试覆盖。

### Notes

本 TASK 不提供静态 UI 嵌入。

## TASK-DLV-005 - 前端日志状态模型与流连接

### Goal

建立框架内可测试的服务目录、SSE、日志环、筛选、连接和暂停/跟随状态基础。

### Scope

- 定义与 API 对齐的 TypeScript 类型和 envelope 解码。
- 实现服务目录请求、唯一 EventSource 生命周期与重连状态。
- 实现 10,000 条环形缓冲、EventID 去重、reset/dropped、组合筛选 reducer/selectors。
- 实现暂停渲染、自动跟随和 unseen 的基础 hooks，不实现正式页面布局。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-005 为准。

### Forbidden Files

Go 后端、正式组件样式、workspace、锁文件和知识库。

### Dependencies

TASK-DLV-004。

### Acceptance Criteria

见 `acceptance/TASK-DLV-005.md`。

### Required Tests

- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`

### Risks

浏览器 EventSource 对自定义 header 有限制，重连 ID 语义必须按 SDD API 契约实现并测试。

### Notes

状态逻辑优先保持纯函数，避免把所有行为耦合进组件。

## TASK-DLV-006 - Canonical UI 与虚拟日志列表

### Goal

依据设计基线实现 1440×900 和 1366×768 下可用的高密度主界面。

### Scope

- 实现 AppShell、TopToolbar、ServiceSidebar、FilterBar、虚拟 LogTable/LogRow、DetailPanel 和 StatusBar。
- 映射 DESIGN.md Token 到 Tailwind/CSS variables，使用系统字体回退且不加载外部资源。
- 接入 TASK-DLV-005 状态和 hooks，完成默认、紧凑和选中详情状态。
- 添加组件和可访问性测试。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-006 为准。

### Forbidden Files

Go 后端、workspace、锁文件、启动脚本和知识库。

### Dependencies

TASK-DLV-005。

### Acceptance Criteria

见 `acceptance/TASK-DLV-006.md`。

### Required Tests

- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`
- `pnpm --filter dev-log-viewer build`

### Risks

设计 HTML 仅作结构参考，禁止复制 CDN 依赖、虚构服务或过时端口。

### Notes

日志列表必须通过 `@tanstack/react-virtual` 保持 DOM 有界。

## TASK-DLV-007 - 完成交互状态、导出与布局持久化

### Goal

补齐设计稿覆盖的运行状态和已确认客户端交互。

### Scope

- 完成暂停、恢复、跳到最新、手动重连、清空视图和关联 ID 筛选。
- 完成 reconnecting、paused、high-volume、no-results、no-services 状态。
- 客户端导出当前筛选结果为 UTF-8 `.log`。
- 只持久化侧栏折叠与详情宽度，并校验 localStorage 数据。
- 展示敏感日志和禁止外部暴露提示。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-007 为准。

### Forbidden Files

Go 后端、workspace、锁文件、启动脚本和知识库。

### Dependencies

TASK-DLV-006。

### Acceptance Criteria

见 `acceptance/TASK-DLV-007.md`。

### Required Tests

- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`
- `pnpm --filter dev-log-viewer build`

### Risks

暂停与断线不得混淆；导出不得绕过当前缓冲上限或新增服务端文件读取。

### Notes

localStorage 使用模块专用、带版本前缀的 key。

## TASK-DLV-008 - 静态嵌入、启停脚本与运行文档

### Goal

形成单进程本地运行链路，并把新增开发工具准确纳入启动脚本和知识库。

### Scope

- 实现前端构建产物的生产 `go:embed` 与开发模式静态处理。
- 完成构建/运行命令和 Go 入口集成。
- 为 `start-dev.sh`/`stop-dev.sh` 增加显式 `logs|log-viewer`，不改变无参数或 `all`。
- 编写模块 README。
- 更新 `.knowledge/manifest.yaml`、system-overview 和 local-development 中受影响的已验证事实。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-008 为准。

### Forbidden Files

业务服务、Gateway、三个 Vue 应用、根 README 和仓库根 `docs/**`。

### Dependencies

TASK-DLV-007。

### Acceptance Criteria

见 `acceptance/TASK-DLV-008.md`。

### Required Tests

- `pnpm --filter dev-log-viewer build`
- `cd dev-log-viewer && go test ./...`
- `cd dev-log-viewer && go vet ./...`
- `bash -n start-dev.sh stop-dev.sh`
- `.knowledge` validator、引用检查和 impact detection

### Risks

构建顺序错误会嵌入旧资源；知识更新必须以实际代码和测试为证据，不得写入设计推测。

### Notes

用户已确认该 TASK 可修改列明的共享脚本和知识文件。

## TASK-DLV-009 - 集成、安全与视觉验收

### Goal

验证完整本地链路满足 SPEC，并形成可审计的最终证据，不新增功能。

### Scope

- 添加缺失的跨层 fixture、集成测试和只读 smoke 脚本。
- 验证 loopback、任意路径不可达、CSP、安全文本渲染、SSE 重连和资源释放。
- 对八个设计状态及 1440×900、1366×768 基线进行人工/可重复验证记录。
- 复核知识影响和文档一致性；只允许修正已授权知识文件的事实错误。

### Allowed Files

以 `task-scope.json` 的 TASK-DLV-009 为准。

### Forbidden Files

生产实现文件、业务模块、共享配置、根 README 和仓库根 `docs/**`。

### Dependencies

TASK-DLV-008。

### Acceptance Criteria

见 `acceptance/TASK-DLV-009.md`。

### Required Tests

- `bash .spec/dev-log-viewer/scripts/agent-check.sh TASK-DLV-009`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- 完整 Go、React、脚本和 smoke 验证

### Risks

若验收暴露生产缺陷，本 TASK 不得扩大到修改生产代码；应回到对应 TASK 或使用经确认的 `fix-check-failures` 范围。

### Notes

最终报告必须逐项映射 AC-001 至 AC-018，并说明未执行的人工检查。

## Required Knowledge Review

每个 TASK 都必须执行 `.knowledge/README.md` 定义的知识影响检查并写入 Markdown 报告与 evidence。TASK-DLV-008 获准直接更新以下知识：

- `.knowledge/manifest.yaml`
- `.knowledge/architecture/system-overview.md`
- `.knowledge/runbooks/local-development.md`

其他 TASK 默认只读审查；若发现范围外知识漂移，报告 `STALE`、`CANDIDATE` 或 `coverage_gap`，不得越界修改。

## Subagent And Goal Mode Strategy

- 当用户以 Codex Goal mode 执行时，一个 Goal objective 只覆盖一个指定 TASK 的 implement → 独立 review → repair 闭环；完成后等待用户确认下一 TASK。只有用户显式调用 `harness-pipeline` 时才能串行编排多个 TASK。
- 每个 TASK 优先将仓库调研、测试设计、失败诊断、知识影响分析和独立只读复审委托给不同 subagent，以隔离上下文。
- 同一时刻只能有一个主实现 TASK。主 Agent 对 scope、工作树写入、pipeline-state、报告和 evidence 负责。
- 不把同一生产文件同时分配给多个写入型 subagent；subagent 发现范围缺口时只报告，不自行扩 scope。
- `self-review` 优先由未参与实现的 subagent 或 fresh context 执行，最终 verdict 必须回写当前 TASK 证据。
