# 实时开发日志查看器 SDD

## 1. 现有架构概览

### 1.1 本地进程与日志

仓库根目录 `start-dev.sh` 是非 Docker 本地开发入口：

1. 创建 `.dev/pids`、`.dev/logs`、`.dev/bin`；
2. 编译选中的 Go 二进制；
3. 安装选中的前端依赖；
4. 通过 `nohup` 启动进程；
5. 使用 `> .dev/logs/<name>.log 2>&1` 合并并覆盖写入 stdout/stderr；
6. 将子进程 PID 写入 `.dev/pids/<name>.pid`。

`stop-dev.sh` 根据同一服务目录和 PID 文件停止进程。日志查看器可复用这些文件作为只读事实源，但不能把 PID 存活等同于业务健康。

### 1.2 日志格式

Go 业务服务通过 `smart-recruit-platform-go/logger` 输出 Zap 日志，本地配置默认启用彩色 console 格式，实际 `.dev/logs` 中包含 ANSI 颜色控制序列、时间、级别、caller、消息和可选 JSON 字段。Gateway 同时包含 Zap console、Gin debug 路由输出和多行文本。三个前端日志主要是 pnpm/Vite 文本。

因此第一版必须支持混合文本而不是假设统一 JSON；解析器只能增强展示，不能决定日志是否保留。

### 1.3 前端和工作区

仓库当前三个产品前端均为 Vue 3 + Vite。用户已明确选择 React + TypeScript + Vite + Tailwind CSS，并确认日志查看器放在仓库根目录 `dev-log-viewer/`。该工具作为独立应用，不复用或修改现有 Vue 应用，并作为根 pnpm workspace 的新增成员统一管理依赖；对应 workspace 和锁文件变更必须纳入专门 TASK scope。

### 1.4 设计基线

`.spec/dev-log-viewer/design/screens` 包含默认、紧凑屏幕、选中详情、SSE 重连、暂停、高流量、无结果和无服务八种状态。`DESIGN.md` 定义 Kinetic Terminal 深色高密度设计系统；`html/` 仅作布局结构参考。

## 2. 问题分析

### 2.1 文件读取难点

- 启动脚本使用覆盖写，已有 inode 可能被截断为 0；只监听 append 会永久停在旧 offset。
- 编辑器、未来日志轮转或进程行为可能删除并重建文件；需识别文件身份变化。
- 写入可能跨系统调用，读取块可能落在 UTF-8 字符或换行中间。
- Go 错误堆栈跨多行，而 Vite/Gin 输出不遵循统一时间前缀。
- 12 个文件同时活跃，不能让某个大文件或慢订阅者阻塞其他文件。

### 2.2 实时传输难点

- 浏览器需要单向实时数据，WebSocket 的双向协议和连接管理超出需求。
- 断线期间可能有新日志；简单重连会重复或漏掉数据。
- 浏览器和服务端内存不能随日志无限增长。
- 暂停渲染、停止自动跟随和连接断开是三种不同状态，必须分别建模。

### 2.3 安全难点

- 本地日志可能含候选人、员工、请求或第三方调用信息。
- 任意路径 API 会把本地工具变成本机文件读取漏洞。
- 日志正文是不可信输入，可能包含 HTML、控制序列或超长文本。
- 无鉴权只在强制 loopback 监听的前提下成立。

### 2.4 UI 一致性难点

设计稿部分样例文本存在虚构服务名或旧端口；实现必须以仓库服务目录为准。详情面板、高流量横幅和紧凑布局需共享同一应用外壳，不能分别复制为互相漂移的页面。

## 3. 建议设计

### 3.1 模块位置

在仓库根目录创建独立模块：

```text
dev-log-viewer/
  go.mod
  package.json
  cmd/dev-log-viewer/
    main.go
  internal/
    catalog/
    tailer/
    parser/
    stream/
    server/
  web/
    index.html
    src/
      api/
      components/
      hooks/
      state/
      types/
      styles/
      App.tsx
    dist/
  embed.go
  embed_dev.go
  vite.config.ts
  tsconfig.json
```

该目录同时作为前端 package 根和独立 Go module 根。将其加入根 pnpm workspace 后，可继续复用仓库的 pnpm 依赖管理方式；Go `embed` 只能嵌入 module 目录内文件，因此前端产物放在 `web/dist`。

### 3.2 组件关系

```text
.dev/logs/*.log ──> TailCoordinator ──> RecordParser ──> EventHub
                         │                                  │
.dev/pids/*.pid ──> ServiceCatalog                         SSE
                         │                                  │
                         └──── GET /api/v1/services ────────┤
                                                            ▼
                                             React Log Store / Ring Buffer
                                                            │
                                      Filters ── LogTable ── DetailPanel
```

### 3.3 后端职责

- `catalog`：固定白名单、仓库根定位、PID/文件元数据状态。
- `tailer`：末尾快照、增量读取、半行缓存、截断/替换检测。
- `parser`：ANSI 清理、级别/caller/时间识别、JSON 尾部与关联字段提取、多行组装。
- `stream`：全局事件 ID、服务器环形缓冲、订阅、补发、有界队列和丢弃统计。
- `server`：HTTP 路由、安全响应头、SSE 编码、静态资源和优雅退出。

### 3.4 前端职责

- `api`：服务目录请求、EventSource 生命周期和事件解码。
- `state`：服务、连接、日志环、筛选、选择、暂停、跟随状态和受限 UI 偏好的 reducer。
- `hooks`：`useLogStream`、`useServiceStatus`、`useAutoFollow`、`useClipboard`、`usePersistedLayout` 和客户端纯文本导出。
- `components`：`AppShell`、`ServiceSidebar`、`TopToolbar`、`FilterBar`、`LogTable`、`LogRow`、`LogDetailPanel`、`StatusBar` 和状态横幅/空状态。
- `styles`：将 `DESIGN.md` Token 映射为 Tailwind theme/CSS variables。

### 3.5 页面状态机

连接状态：

```text
connecting -> connected -> reconnecting -> connected
                    \-> disconnected -> reconnecting
```

视图状态独立于连接状态：

```text
followLatest: true/false
renderPaused: true/false
selectedLogId: string/null
sidebarCollapsed: true/false
detailPanelOpen: true/false
```

连接断开不清空日志；滚动离开底部只改变 `followLatest`；点击暂停只改变 `renderPaused`。

## 4. 数据结构变更

不修改数据库、Proto、业务服务模型或共享类型。新增类型只存在于日志查看器模块。

### 4.1 Go 服务目录模型

```go
type ServiceDefinition struct {
    ID       string
    Name     string
    Group    string
    Port     int
    LogFile  string
    PIDFile  string
}

type ServiceStatus struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`
    Group         string    `json:"group"`
    Port          int       `json:"port"`
    ProcessState  string    `json:"process_state"` // running/offline/unknown
    LogState      string    `json:"log_state"`     // ready/missing/unreadable
    LogSizeBytes  int64     `json:"log_size_bytes"`
    LogUpdatedAt  *time.Time `json:"log_updated_at,omitempty"`
}
```

API 不返回绝对日志/PID 文件路径，避免泄露本机目录。

### 4.2 日志记录模型

```go
type LogRecord struct {
    ID         uint64            `json:"id"`
    Generation uint64            `json:"generation"`
    Service    string            `json:"service"`
    ObservedAt time.Time         `json:"observed_at"`
    SourceTime *time.Time        `json:"source_time,omitempty"`
    Level      string            `json:"level"`
    Caller     string            `json:"caller,omitempty"`
    Message    string            `json:"message"`
    Lines      []string          `json:"lines"`
    Fields     map[string]string `json:"fields,omitempty"`
    RequestID  string            `json:"request_id,omitempty"`
    TraceID    string            `json:"trace_id,omitempty"`
    SpanID     string            `json:"span_id,omitempty"`
}
```

`Fields` 只保存日志中实际存在且成功解析的标量字段。无法解析的 JSON 或重复字段不影响 `Message/Lines`；重复键采用最后成功解析值，并保留原始文本行。

### 4.3 SSE 事件

```go
type StreamEnvelope struct {
    Type       string      `json:"type"` // snapshot_start/log/snapshot_end/reset/dropped/status
    EventID    uint64      `json:"event_id"`
    Service    string      `json:"service,omitempty"`
    Payload    interface{} `json:"payload,omitempty"`
    Dropped    uint64      `json:"dropped,omitempty"`
    Recoverable bool       `json:"recoverable,omitempty"`
}
```

TypeScript 使用同字段的 discriminated union，不单独维护与 API 不一致的宽松 `any` 类型。

## 5. API 与接口变更

所有接口由日志查看器自己的 `127.0.0.1:8090` 提供，不修改 Gateway。

### 5.1 `GET /healthz`

用途：确认查看器 HTTP 进程存活。

成功响应：

```json
{"status":"ok","service":"dev-log-viewer"}
```

### 5.2 `GET /api/v1/services`

用途：获取固定服务目录和当前状态。

成功响应：

```json
{
  "services": [
    {
      "id": "identity-service",
      "name": "identity-service",
      "group": "backend",
      "port": 50061,
      "process_state": "running",
      "log_state": "ready",
      "log_size_bytes": 22081,
      "log_updated_at": "2026-07-15T14:50:29+08:00"
    }
  ]
}
```

状态轮询建议间隔为 2 秒。接口失败时前端保留上次成功状态并显示“状态暂不可用”。

### 5.3 `GET /api/v1/logs/stream`

查询参数：

- `service`：可重复，只接受白名单 ID；省略表示全部服务；
- `tail`：每服务初始行数，默认 300，范围 0～1000。

请求头：浏览器重连时可带 `Last-Event-ID`。

响应头：

```text
Content-Type: text/event-stream
Cache-Control: no-cache, no-store
Connection: keep-alive
X-Content-Type-Options: nosniff
```

SSE 示例：

```text
id: 1042
event: log
data: {"type":"log","event_id":1042,"service":"identity-service","payload":{...}}

event: heartbeat
data: {"type":"heartbeat"}
```

非法服务返回 400；超范围 `tail` 返回 400，不静默放宽。

### 5.4 静态资源路由

- `/` 和前端静态路径由嵌入文件系统提供；
- 未匹配的前端路由回退到 `index.html`；
- `/api/` 和 `/healthz` 不执行 SPA fallback；
- 静态资源使用内容哈希长缓存，`index.html` 使用 no-cache。

## 6. 算法与工作流变更

### 6.1 仓库根定位

启动器显式向 Go 进程传递仓库根，后端将 `.dev/logs`、`.dev/pids` 解析为绝对路径后进行 containment 校验。禁止根据当前工作目录猜测后直接开放路径。

### 6.2 初始 tail

对每个文件：

1. `Stat` 获取大小和文件身份；
2. 从文件末尾按固定块反向读取，直到获得所需换行数或到达文件开头；
3. 丢弃起始处不完整的半行；
4. 对读取结果执行 UTF-8 合法化和记录组装；
5. 保存当前文件 offset 和 identity；
6. 将快照事件发给当前新订阅者，而不是重新广播给所有客户端。

单个快照读取字节数设置硬上限，防止单条超长日志造成无界内存；超过限制时发送截断标记。

### 6.3 增量跟随

采用 Go 标准库的集中轮询协调器，默认 200ms～250ms 周期扫描 12 个固定文件，不新增文件监听依赖：

- `size > offset`：使用 `ReadAt` 读取新增范围；
- `size == offset`：无操作；
- `size < offset`：视为截断，generation +1，从头读取并发送 reset；
- identity 变化：完成旧句柄剩余读取后重开，generation +1；
- 文件缺失：关闭旧句柄并等待下一轮出现；
- 权限/暂时 I/O 错误：按服务记录错误并退避，不中断协调器。

每轮对单文件读取量设置上限并采用轮转顺序，避免繁忙文件饿死其他服务。

### 6.4 行和记录组装

1. tailer 只按字节和换行生成物理行；跨块半行保存在服务私有缓冲。
2. parser 去除 ANSI CSI/OSC 控制序列，保留普通 Unicode 文本。
3. 能识别时间 + 级别 + caller 的行作为新记录开始。
4. 以空白开头、Go 源路径、goroutine/堆栈模式开头的后续行追加到当前记录。
5. Gin/Vite 等无法归类行独立成记录；不得错误吞并到上一个长时间未完成记录。
6. 当前记录在下一条明确起始行或短暂空闲超时后 flush。
7. 若消息尾部是合法 JSON object，则提取标量字段；解析失败保留原文。

单条记录总字节数和行数必须有限制，超限后截断并增加 `truncated=true` 字段。

### 6.5 事件环和重连

- EventHub 为实时记录分配进程内全局单调 EventID。
- 服务器维护 20,000 条固定容量事件环；允许通过受上下限校验的环境变量覆盖。
- 新连接无 `Last-Event-ID`：发送所选服务有界快照，再进入实时订阅。
- 有 `Last-Event-ID` 且仍在事件环：按 ID 顺序补发匹配服务事件。
- ID 早于事件环最小值或属于上一次进程：发送不可恢复 reset，再发送新快照。
- 每客户端队列固定容量；发布采用非阻塞写入，溢出累加该客户端 dropped。
- dropped 通知本身需合并/限频，避免告警加剧拥塞。

### 6.6 客户端数据流

1. 首次加载服务目录，默认选择全部服务；
2. 建立唯一 EventSource；
3. reducer 按 EventID 去重并将记录写入 10,000 条环形缓冲；
4. 用户筛选在 memoized selector 中执行；
5. `renderPaused=true` 时继续入缓冲，但冻结当前可见快照；
6. 用户滚动离开底部设置 `followLatest=false` 并累计 unseen；
7. Jump to latest 清零 unseen、恢复跟随并滚至末尾；
8. 服务选择变化时关闭旧 EventSource，带最新可恢复 ID 建立新连接。

日志列表使用 `@tanstack/react-virtual` 进行固定/可测量行高的窗口化渲染，避免 10,000 个 DOM 节点同时存在。导出逻辑从当前 memoized 筛选结果生成 UTF-8 纯文本 Blob，不调用新的后端接口。侧栏折叠状态和详情面板宽度使用带版本前缀的专用 localStorage key，并对读取值做类型与范围校验。

## 7. 配置设计

后端配置通过命令行参数或查看器专用环境变量读取，不复用业务 `CONFIG_PATH`：

| 配置 | 默认值 | 说明 |
| --- | --- | --- |
| `DEV_LOG_VIEWER_ADDR` | `127.0.0.1:8090` | 必须校验为 loopback |
| `DEV_LOG_VIEWER_ROOT` | 启动脚本传入仓库根 | 仓库根 |
| `DEV_LOG_VIEWER_TAIL_LINES` | `300` | 每服务初始行数 |
| `DEV_LOG_VIEWER_MAX_TAIL_LINES` | `1000` | API tail 上限 |
| `DEV_LOG_VIEWER_POLL_INTERVAL` | `250ms` | 文件扫描周期 |
| `DEV_LOG_VIEWER_EVENT_BUFFER` | `20000` | 服务端事件环容量 |
| `DEV_LOG_VIEWER_CLIENT_QUEUE` | `2048` | 单客户端发送队列 |
| `DEV_LOG_VIEWER_HEARTBEAT` | `15s` | SSE 心跳周期 |

所有数值配置必须有合理上下限并在启动时校验；非法配置导致查看器启动失败并输出明确错误。第一版不支持修改日志目录或白名单文件名。

前端构建时只需要同源 API，不注入远程服务 URL，不加载 CDN 或网络字体。Vite 开发代理目标固定为本机 Go 开发地址。tail 和缓冲容量不在 UI 中开放配置。

## 8. 兼容性策略

- 新模块不导入 `smart-recruit-commons`、`smart-recruit-platform-go` 或业务 Proto，避免形成共享模块依赖。
- Go 后端优先使用标准库 `net/http`、`embed`、`os`、`io`、`encoding/json`，不要求 Gateway/Gin。
- 不改造现有日志输出；parser 同时支持当前 Zap console、Gin 和 Vite 文本。
- `start-dev.sh` 和 `stop-dev.sh` 只新增独立 target/alias，既有 target 展开结果保持不变。
- 查看器的 PID/日志可沿用 `.dev/pids/dev-log-viewer.pid` 和 `.dev/logs/dev-log-viewer.log`，但查看器自身日志不得加入被查看白名单，避免递归流。
- 现有 Vue 应用、业务 Gateway 和后端服务不引用日志查看器代码。
- 设计样例中的旧服务名、旧端口和虚构状态不得进入运行时目录。

## 9. 错误处理与降级设计

### 9.1 后端

- 目录缺失：启动成功，定期等待；服务目录返回日志 missing。
- 文件权限错误：记录服务级错误，目录 API 返回 unreadable，SSE 其他服务正常。
- 读取错误：关闭并延迟重开该文件，避免忙循环。
- 解析错误：生成 `UNKNOWN` 记录，原物理行保留。
- 客户端队列满：非阻塞丢弃并聚合 dropped 事件。
- JSON 编码/SSE 写失败：关闭该客户端，不影响 hub。
- 静态资源缺失：构建失败；运行时若意外缺失返回 500 和纯文本错误。
- 退出：停止接收连接、取消 tailer、关闭订阅，设置短超时优雅关闭 HTTP。

### 9.2 前端

- 服务目录失败：保留最后成功值并显示状态警告。
- EventSource error：进入 reconnecting，保留日志和筛选条件。
- reset：插入系统分隔记录，按新快照继续，不把旧新代次静默混合。
- malformed event：忽略该 envelope、增加协议错误计数并输出开发诊断；不得让 reducer 崩溃。
- Clipboard 失败：显示非阻塞 toast。
- 高流量：淘汰最旧数据、累计 dropped、停止昂贵动画并显示横幅。
- localStorage 非法或不可用：忽略持久化值并使用默认布局，不影响日志读取。
- 导出失败：显示非阻塞错误提示，不清空当前筛选结果。

## 10. 可观测性与调试输出设计

查看器自身使用标准库 `log/slog` 输出结构化或文本日志，字段包括：

- `component`：catalog/tailer/parser/stream/http；
- `service`：只记录白名单服务 ID；
- `event`：opened/reset/reopen_failed/client_connected/client_disconnected/dropped/shutdown；
- `error`：系统错误，不附业务日志正文；
- `count`、`offset`、`generation` 等诊断数值。

禁止记录：

- 被查看的完整日志行；
- `request_id`、`trace_id` 的具体值；
- 本地凭据、环境变量内容或完整绝对用户目录。

`/healthz` 只报告查看器进程。服务侧栏的 process state 和 log state 分开显示，避免“System Health 100%”等误导性文案。

## 11. 测试策略

### 11.1 Go 单元测试

| 测试域 | 覆盖内容 | 对应验收 |
| --- | --- | --- |
| catalog | 12 服务事实、PID 缺失/陈旧/非法、绝对路径不外泄 | AC-001、AC-011、AC-012 |
| tailer snapshot | 末尾 N 行、空文件、超长行、UTF-8、半行 | AC-002、AC-004 |
| tailer follow | append、truncate、rename/recreate、delete/reappear、公平读取 | AC-003 |
| parser | ANSI、Zap、JSON 尾部、Gin、Vite、堆栈、解析降级 | AC-004 |
| stream hub | EventID、补发、缺口 reset、慢客户端、dropped、取消订阅 | AC-005、AC-006 |
| HTTP | 白名单参数、tail 上限、SSE headers、healthz、静态 fallback、安全头 | AC-006、AC-012 |

测试使用 `t.TempDir()` 构造 `.dev/logs`/`.dev/pids`，不得读取开发者真实日志。

### 11.2 React 单元/组件测试

| 测试域 | 覆盖内容 | 对应验收 |
| --- | --- | --- |
| reducer/ring | EventID 去重、10,000 上限、dropped/reset | AC-006、AC-009 |
| selectors | 服务、级别、文本 AND 筛选、关联 ID | AC-007 |
| stream hook | connecting/connected/reconnecting、手动重连、清理旧连接 | AC-006、AC-011 |
| follow/pause | 滚动停止跟随、unseen、Jump to latest、暂停/恢复 | AC-008 |
| components | 空状态、高流量横幅、详情面板、复制反馈 | AC-009、AC-010、AC-011 |
| security | 日志中的 HTML/script 只作为文本显示 | AC-012 |
| export | 当前筛选顺序、UTF-8 文本、10,000 条边界、失败反馈 | AC-016 |
| persisted layout | 合法/非法值、版本 key、只恢复侧栏和详情宽度 | AC-017 |
| offline assets | 生产构建不产生第三方字体或 CDN 请求 | AC-018 |

### 11.3 集成和手工测试

- `go test ./...`
- `go vet ./...`
- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`
- `pnpm --filter dev-log-viewer build`
- `bash -n start-dev.sh stop-dev.sh`
- 使用临时日志目录启动后端，验证 curl SSE、截断、替换和重连。
- 在 1440×900 和 1366×768 浏览器视口对照八张设计截图验收。
- 验证现有脚本原 target 的 help、展开和停止语义未变化。

### 11.4 安全测试

- 传入未知服务、`../`、绝对路径和 URL 编码穿越，必须返回 400/404 且不读取文件。
- 符号链接指向允许目录外时必须拒绝或标记不可读。
- 注入 `<script>`、事件属性、ANSI OSC 链接和超长字段时不得执行或破坏布局。
- 尝试配置非 loopback 地址时启动失败。

## 12. 迁移风险

- R-001：新增 React/Tailwind package 会修改 manifest 和锁文件；用户已批准第 15 节所列依赖，但仍需在独立 TASK 中审查版本、许可证和依赖供应链。
- R-002：根目录新增 `dev-log-viewer/` 不在当前 `pnpm-workspace.yaml` 的匹配范围内；用户已批准加入 workspace，实施时必须把 workspace 配置和锁文件列入同一受控 TASK。
- R-003：静态资源嵌入要求先构建前端；若脚本构建顺序错误，Go build 会失败或嵌入旧资源。
- R-004：文件轮询在网络盘或极高频日志下可能增加 I/O；需以限频、每轮上限和基准测试控制。
- R-005：混合日志多行组装是启发式，可能误合并；必须保证原始行可见且有长度/超时边界。
- R-006：PID 可能复用，`kill -0` 只能说明某进程存在；UI 文案必须限定为 process state。
- R-007：SSE 事件环只在进程内，查看器重启后不能连续恢复，必须显式 reset。
- R-008：设计 HTML 含 CDN 和远程占位资源，直接复制会违反本地只读和离线目标。
- R-009：现有 `.spec/dev-log-viewer` 为未跟踪目录；后续 TASK 基线必须在提交或明确基准后建立，不能把既有设计资料误计入单个实现 TASK。

## 13. 实现边界

### 13.1 允许涉及的区域

- `dev-log-viewer/**`：新增独立 Go + React 模块；
- `start-dev.sh`、`stop-dev.sh`：已确认可在脚本集成 TASK 中增加显式 `logs|log-viewer` target；
- `pnpm-workspace.yaml`：已确认可在依赖脚手架 TASK 中增加 `dev-log-viewer` workspace 条目；
- `pnpm-lock.yaml`：已确认可在依赖脚手架 TASK 中随已批准依赖更新；
- `.spec/dev-log-viewer/**`：功能契约、验收、报告和设计资料；
- `.gitignore`：仅在确有新生成物需要忽略且 TASK 明确授权时修改。

### 13.2 禁止涉及的区域

- `smart-recruit-gateway/**`；
- `smart-recruit-*-service/**`；
- `smart-recruit-platform-go/**`；
- `smart-recruit-commons/**`；
- `smart-recruit-proto/**`；
- 三个现有前端应用；
- 数据库、迁移、Docker/部署和 CI 配置。

### 13.3 建议实现切分边界

后续 `prepare-harness` 应按依赖顺序至少拆分为：

1. 独立模块脚手架和依赖基线；
2. 服务目录与 PID 状态；
3. 文件 tail 与 parser；
4. EventHub 与 SSE API；
5. React 状态模型和 API hooks；
6. Canonical UI 和响应式布局；
7. 详情、筛选、暂停、重连、导出、布局持久化、空状态和高流量状态；
8. 静态嵌入与启动/停止脚本集成；
9. README、集成、安全和视觉回归验收。

每个 TASK 必须限定文件范围。用户已确认新增模块 manifest、已列依赖、workspace 条目、锁文件以及显式启动/停止 target；这些变更仍须分别写入对应 TASK scope。未列出的共享配置或依赖继续设置 `requiresHumanConfirmation: true`。

## 14. 备选方案

### 14.1 Vector + Loki + Grafana

优点：成熟采集、查询、历史存储和可观测性关联。缺点：对仅查看本机 12 个文件过重，引入多个常驻组件和配置，不符合当前小型本地工具目标。暂不采用。

### 14.2 Dozzle

优点：轻量实时 Web 日志体验成熟。缺点：主要面向 Docker/Podman/Kubernetes 日志，当前 `start-dev.sh` 是非容器进程。暂不采用。

### 14.3 WebSocket

优点：支持双向控制。缺点：当前日志流是单向，SSE 原生重连和标准 HTTP 语义更匹配；第一版也不允许 Web 控制服务。暂不采用。

### 14.4 复用业务 Gateway

优点：减少一个端口。缺点：把本机文件读取能力放入业务传输边界，扩大权限和部署风险，并耦合业务启动顺序。明确不采用。

### 14.5 终端 `tail -f` 聚合

优点：实现成本低。缺点：难以提供多服务筛选、结构化详情、响应式状态和安全的浏览器体验，不能满足设计目标。仅作为临时人工 fallback。

### 14.6 Go 标准库轮询与 fsnotify

标准库轮询依赖少、对 12 个固定文件足够，且容易测试截断和替换。`fsnotify` 延迟更低，但增加依赖且仍需轮询兜底。第一版采用标准库轮询；若实测 NFR-001 不达标，再通过独立方案变更评估 fsnotify。

## 15. 已确认技术决策

- D-001：根目录 `dev-log-viewer/` 同时作为前端 package 和独立 Go module 根，Go 通过 `go:embed` 嵌入 `web/dist`。
- D-002：模块纳入根 pnpm workspace；允许 React、TypeScript、Vite、Tailwind CSS、Vitest、Testing Library 与 `@tanstack/react-virtual`，并允许在受控 TASK 中更新 workspace 配置和锁文件。
- D-003：默认监听 `127.0.0.1:8090`，Vite 开发端口为 `8091`；生产运行仅 loopback、同源且离线。
- D-004：固定 UI 容量为 300/1000 行 tail 和 10,000 条浏览器缓冲；服务端事件环 20,000 条、客户端队列 2,048 条作为初始容量，后端参数均需校验上下限。
- D-005：服务状态只表示 PID 对应进程存活，不主动探测端口，不称为健康状态。
- D-006：只新增显式 `logs|log-viewer` 启停 target，不改变无参数或 `all`。
- D-007：纯文本 `.log` 导出在客户端完成；不提供服务端导出或 JSONL。
- D-008：第一版不做自动脱敏，依靠 loopback-only、只读白名单和醒目敏感信息提示控制风险。
- D-009：只持久化侧栏折叠状态和详情面板宽度。
- D-010：不使用网络字体或 CDN，采用系统字体回退。
- D-011：虚拟列表使用 `@tanstack/react-virtual`，模块提供 README。
- D-012：tail 第一版采用标准库约 250ms 轮询；NFR-001 不达标时再评估 `fsnotify`。

## 16. 开放问题

无。所有原开放问题已按第 15 节关闭；任何偏离都应先修订 SPEC/SDD，再更新对应 TASK 和验收证据。
