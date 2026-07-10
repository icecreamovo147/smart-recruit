# Eino ADK Agent 代码审查问题修复计划

> 基于 2026-05-30 代码审查的 10 项发现，按优先级排列。

---

## P0 — 候选人端缺失工具调用审计记录

### 问题

`CandidateAIService` 结构体没有 `toolTraces` 字段，ADK 路径调用 `ChatWithADKAgent` 时 `onToolExecuted` 传了 `nil`，候选人端所有工具调用完全没有持久化到 `tool_traces` 表。而 HR 端已完整实现了此能力。

涉及文件：
- `service/candidate_ai_service.go:27-36` — 结构体缺少 `toolTraces` 字段
- `service/candidate_ai_service.go:156-163, 358-365` — `onToolExecuted` 为 nil

### 方案

1. 在 `CandidateAIService` 结构体中添加 `toolTraces *repository.ToolTraceRepo`
2. 在 `NewCandidateAIService` 构造函数中添加 `toolTraces` 参数
3. 在 `StreamChat` 和 `StreamChatGRPC` 的 ADK 路径中，构造与 HR 端一致的 `traceFn` 回调，异步写入 tool_traces（注意候选人端 owner_type 应记录为 candidate 而非 hr）
4. 更新 `server/server.go` 中 `NewCandidateAIService` 的调用点，传入 toolTraceRepo

### 预期效果

候选人端工具调用行为与 HR 端一致，可追溯、可审计。`tool_traces` 表覆盖全部 AI 助手使用场景。

---

## P1 — ADK 与 Legacy 双路径工具实现完全重复

### 问题

HR 端 14 个工具 + 候选人端 6 个工具的业务逻辑各实现了两遍：

| 路径 | HR 工具文件 | 候选人工具文件 |
|------|-----------|-------------|
| Legacy（map 参数） | `tool_executor.go`（745 行） | `candidate_tool_executor.go`（249 行） |
| ADK（类型化 struct） | `hr_adk_tools.go`（738 行） | `candidate_adk_tools.go`（314 行） |

两个路径中的工具业务逻辑完全一致，仅参数绑定方式不同（Legacy 从 `map[string]any` 取值，ADK 从类型化 struct 取值）。任何行为变更必须在两处同步。

### 方案

分两步走：

**第一步**（本次修复）：将每个工具的核心执行逻辑抽取为包内私有函数，Legacy 和 ADK 路径共同调用。

- `tool_executor.go` 中已有的 `queryTotal`、`queryToday` 等方法保持不变，ADK 路径的工具闭包改为调用 `ToolExecutor` 的对应方法
- `hr_adk_tools.go` 改为接收 `*ToolExecutor` 而非直接持有 `RecruitingToolDeps`，闭包内部委托给 `executor.Execute(ctx, hrID, toolName, args)` → 但这样失去了类型安全
- **更好的方案**：在 `hr_adk_tools.go` 的每个工具闭包中，直接委托给 `ToolExecutor` 的同名方法，map 参数从类型化 struct 转换而来

具体重构方式——为 `ToolExecutor` 和 `CandidateToolExecutor` 已有的每个方法创建一个接收 typed struct 的方法（或 adapter），ADK 工具直接调用它们。Legacy 路径保持不变但标记为 `// Deprecated`。

**第二步**（后续版本）：移除 Legacy 路径，删除 `tools.go` / `candidate_tools.go`（`schema.ToolInfo` 定义），清理 `ChatWithTools` / `ChatWithToolsLegacy` 方法。

### 预期效果

工具逻辑收敛到单点维护，修复 bug 只需改一处。Legacy 路径代码被显式标记为待移除，降低后续清理的认知负担。

---

## P1 — `ToolMetadata.merge` 覆盖而非追加

### 问题

`tool_executor.go:69-76` 中 `merge` 方法直接覆盖 `CandidateOptions`：

```go
if len(other.CandidateOptions) > 0 {
    m.CandidateOptions = other.CandidateOptions  // ← 覆盖，不是追加
}
```

当 Agent 在同一轮对话中多次调用 `search_candidates`（例如先用"张三"搜再用"李四"搜），只有最后一次的结果保留，前端候选人选项列表不完整。

### 方案

将覆盖改为追加：

```go
if len(other.CandidateOptions) > 0 {
    m.CandidateOptions = append(m.CandidateOptions, other.CandidateOptions...)
}
```

对 `Action` 字段保持覆盖语义（一轮对话只应有一个待确认动作），并添加注释说明。

### 预期效果

同一轮对话中多次工具调用的 `CandidateOptions` 全部保留，前端可以展示完整的候选人选项列表。

---

## P2 — ADK Agent 创建失败无 Legacy 降级

### 问题

`service/ai_service.go:322-325` — `NewRecruitingADKTools` 失败时直接返回错误：

```go
adkTools, err := ai.NewRecruitingADKTools(deps, req.HrId, state)
if err != nil {
    return "", ai.ToolMetadata{}, fmt.Errorf("create adk tools: %w", err)
}
```

`service/candidate_ai_service.go:152-155` 同样问题。

### 方案

在 `runADKChat` 中捕获工具创建失败后，自动降级到 Legacy 路径并记录 Warn 日志：

```go
adkTools, err := ai.NewRecruitingADKTools(deps, req.HrId, state)
if err != nil {
    logger.L().Warn("[ADK降级] 工具创建失败，自动切换到 Legacy 路径", zap.Error(err))
    return s.runLegacyChat(ctx, req, session, messages, onDelta, onStatus)
}
```

候选人端同理。

### 预期效果

框架升级或工具变更导致的短暂不兼容不会中断用户对话，系统自动降级并记录日志，运维可据此排查。

---

## P2 — ADK 工具闭包未校验 HR/User ID

### 问题

`NewRecruitingADKTools(hrID)` 和 `NewCandidateADKTools(userID)` 直接将 ID 捕获到闭包中，没有 `<= 0` 校验。如果服务层传入零值 ID，所有工具会以无效作用域查询。

### 方案

在两个构造函数入口处添加校验：

```go
func NewRecruitingADKTools(deps RecruitingToolDeps, hrID int64, state *AgentRunState) ([]tool.BaseTool, error) {
    if hrID <= 0 {
        return nil, fmt.Errorf("hrID must be positive, got %d", hrID)
    }
    // ... 原有逻辑
}
```

### 预期效果

防御性编程——非法 ID 在最早入口被拦截，带有明确的错误信息，而非下游 repository 层以"未找到"静默处理。

---

## P2 — 候选人端缺失记忆/摘要生命周期

### 问题

HR 端在对话完成后异步刷新会话摘要和写入长期记忆（`ai_service.go:298-301`）。候选人端完全没有这些调用，多轮对话的 prompt token 消耗随会话长度线性增长，最终超出模型窗口。

### 方案

为 `CandidateAIService` 添加 `summaries` 和 `memories` 字段，在 `StreamChatGRPC` 的 ADK 路径完成后（成功保存 assistant 消息之后）异步调用：

```go
go s.maybeRefreshSummary(session.ID, userID)
```

摘要刷新逻辑与 HR 端一致。记忆写入（候选人端主要为简历分析结论、岗位偏好等）可后续迭代。

同时保持 `maxCandidateContextMessages = 20` 作为硬截断兜底。

### 预期效果

候选人长时间多轮对话场景下，上下文保持在可控范围内，避免超出 token 窗口导致对话质量下降。

---

## P3 — `stripSystemMessage` 不加区分移除所有 System 消息

### 问题

`adk_agent.go:223-231` 的 `stripSystemMessage` 函数移除消息列表中**全部** System 消息，因为 ADK 的 `defaultGenModelInput` 会自行将 Instruction 作为 System 消息注入。

如果未来 AgentContextBuilder 注入多层 System 消息（如基础 prompt + 长期记忆 + 合规约束），非首条 System 消息也会被丢弃。

### 方案

改为只移除**第一条** System 消息（即原始 instruction prompt），保留后续注入的上下文：

```go
func stripSystemMessage(msgs []*schema.Message) []*schema.Message {
    out := make([]*schema.Message, 0, len(msgs))
    skippedFirst := false
    for _, m := range msgs {
        if m.Role == schema.System && !skippedFirst {
            skippedFirst = true
            continue
        }
        out = append(out, m)
    }
    return out
}
```

### 预期效果

保留多层 System 消息的能力，上下文注入（如长期记忆、合规规则）不会被静默丢弃。

---

## P3 — 每次请求重建全部工具实例

### 问题

`runADKChat` 每次调用 `NewRecruitingADKTools`（14 个 `utils.InferTool`），候选人端每次调用 `NewCandidateADKTools`（6 个）。高 QPS 下反射和分配开销累积。

### 方案

分两步：

**第一步**：将 `ToolExecutor` / `CandidateToolExecutor` 的方法提取为纯函数（不依赖 executor struct 的字段），ADK 工具直接调用纯函数，移除 `NewRecruitingADKTools` 中的反射开销——直接使用 `tool.NewBaseTool` 构造而不依赖 `utils.InferTool`。

**第二步**：如果 `utils.InferTool` 的开销可接受（Eino 框架内部可能有缓存），则在服务启动时预创建工具模板，运行时通过 `tool.WithOption` 动态注入 `hrID`/`userID`。

### 预期效果

减少每次请求的分配压力，降低 GC 频率。

---

## P3 — `AgentRunState` 主循环读取无锁

### 问题

`adk_agent.go:189,200,211` 直接读取 `state.Metadata.ToolTraces` 和 `state.Metadata` 字段，而工具闭包通过 `state.Merge()` / `state.RecordTrace()` 持锁写入。

当前 ADK 同步执行工具，实际不存在并发冲突。但 Go race detector 会报告，且未来 ADK 支持并行工具执行时会成为真实 bug。

### 方案

为 `AgentRunState` 添加带锁的读方法：

```go
func (s *AgentRunState) ReadMetadata() ToolMetadata {
    if s == nil {
        return ToolMetadata{}
    }
    s.Mu.Lock()
    defer s.Mu.Unlock()
    return s.Metadata  // 返回副本（值拷贝）
}
```

主循环改为调用 `state.ReadMetadata()`。

### 预期效果

Go race detector 零告警，并发安全由类型系统保证，不依赖"ADK 同步执行"的实现假设。

---

## P3 — 流式调用中 `time.After` 未主动停止

### 问题

`eino_client.go:526-533` 使用 `time.After` 做慢响应告警。当 LLM 快速完成（如 2 秒内），timer 被创建但未被主动停止，需要等待 GC 回收。

### 方案

改用 `time.NewTimer` + `defer timer.Stop()`：

```go
if c.slowThreshold > 0 {
    timer := time.NewTimer(c.slowThreshold)
    go func() {
        defer timer.Stop()
        select {
        case <-timer.C:
            sendStatus(onStatus, "timeout_warning", "AI 响应较慢，请稍候...", "", "")
        case <-callCtx.Done():
        }
    }()
}
```

### 预期效果

LLM 快速响应时 timer 立即回收，避免高频请求下 pending timer 堆积。

---

## 实施建议

| 批次 | 包含项 | 预估改动文件数 | 风险 |
|------|--------|:----------:|------|
| 第 1 批（P0+P1） | #2, #1, #3 | 6 | 中 — 涉及结构体字段变更和工具调用路径重构 |
| 第 2 批（P2） | #5, #6, #10 | 4 | 低 — 防御性补丁 + 新增功能 |
| 第 3 批（P3） | #4, #7, #8, #9 | 4 | 低 — 小范围修改 |

**建议先做第 1 批**（解决审计缺口和代码重复这两个最核心的问题），第 2、3 批可以在日常迭代中逐步消化。第 1 批中 `#1`（双路径合并）改动最大，建议单独一个 PR，`#2`（审计补全）和 `#3`（merge 语义）可以合入同一个 PR。
