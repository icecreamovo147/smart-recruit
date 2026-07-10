# Prompt & Agent 管理对接运行时 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 Prompt 管理页面和 Agent 管理页面的配置数据在 AI 对话运行时真正生效——Prompt 管理同时影响 HR 端和候选人端，Agent 管理控制运行时工具列表、模型参数和系统提示词。

**Architecture:** 在 Chat 运行时注入 `AgentConfigService`（或直接注入 `AgentConfigRepo` + `PromptTemplateRepo`），从 `agent_configs` 表读取可用工具列表和参数配置，从 `prompt_templates` 表读取系统提示词模板。HR 端和候选人端各自查询对应 `agent_type` 的配置。保留硬编码兜底：当表中无对应配置时回退到现有硬编码常量。

**Tech Stack:** Go (gRPC service), GORM, Eino Agent框架

---

## 当前状态总结

| 模块 | 当前行为 | 目标行为 |
|------|---------|---------|
| HR Prompt | 从 DB 读取 `agent_type="hr_agent"` 的 system prompt | 不变（已工作） |
| 候选人 Prompt | 使用硬编码 `candidateSystemPrompt` 常量 | 从 DB 读取 `agent_type="candidate_assistant"` 的 system prompt |
| HR Agent 配置 | `agent_configs` 表从不读取，工具列表和参数硬编码 | 从 `agent_configs` 读取工具绑定、max_iterations、temperature |
| 候选人 Agent 配置 | 同上，硬编码 | 同上，对接 `agent_configs` 表 |

---

### Task 1: 候选人 Prompt 接入 DB

**Files:**
- Modify: `logic-grpc-service/service/candidate_ai_service.go` — 注入 PromptTemplateRepo，替换硬编码 system prompt
- Modify: `logic-grpc-service/service/candidate_agent_context.go` — `buildCandidateAgentMessages` 支持外部传入 system prompt

- [ ] **Step 1: 在 CandidateAIService 结构体中注入 PromptTemplateRepo**

在 `logic-grpc-service/service/candidate_ai_service.go` 找到 `CandidateAIService` 结构体定义，添加 `promptRepo` 字段和构造函数参数。

先读取结构体定义位置：
```bash
grep -n 'type CandidateAIService struct' logic-grpc-service/service/candidate_ai_service.go
```

修改结构体，添加字段：
```go
type CandidateAIService struct {
    // ... existing fields ...
    promptRepo *repository.PromptTemplateRepo  // optional: nil-safe when not injected
}
```

在 `NewCandidateAIService` 构造函数中添加可选参数（使用 functional options 模式或简单地在现有参数后追加）。

- [ ] **Step 2: 修改 `buildCandidateAgentMessages` — 接受外部 system prompt 参数**

修改 `logic-grpc-service/service/candidate_agent_context.go:22`：

修改函数签名，添加参数：
```go
func (s *CandidateAIService) buildCandidateAgentMessages(
    ctx context.Context,
    userID int64,
    sessionID int64,
    currentMessage string,
    systemPrompt string,  // 新增：DB 模板或回退到硬编码
) ([]*schema.Message, error) {
```

将第 37 行的 `schema.SystemMessage(candidateSystemPrompt)` 改为：
```go
    prompt := systemPrompt
    if prompt == "" {
        prompt = candidateSystemPrompt  // 兜底：DB 无数据时使用硬编码
    }
    messages := []*schema.Message{
        schema.SystemMessage(prompt),
    }
```

- [ ] **Step 3: 在调用点查询 DB 并传入 system prompt**

更新所有调用 `buildCandidateAgentMessages` 的地方。先定位：
```bash
grep -n 'buildCandidateAgentMessages' logic-grpc-service/service/candidate_ai_service.go
```

在每个调用点之前添加 DB 查询：
```go
systemPrompt := candidateSystemPrompt  // 兜底默认值
if s.promptRepo != nil {
    tmpl, err := s.promptRepo.GetActiveByAgentType(ctx, "candidate_assistant", "system")
    if err == nil && tmpl != nil {
        systemPrompt = tmpl.Content
    }
}

messages, err := s.buildCandidateAgentMessages(ctx, userID, sessionID, message, systemPrompt)
```

同时更新 `logic-grpc-service/service/candidate_agent_context.go:22` 的调用签名，移除对 `candidateSystemPrompt` 常量的直接引用（该常量保留作为兜底）。

- [ ] **Step 4: 编译验证**

```bash
cd logic-grpc-service && go build ./...
```
Expected: 编译通过，无报错。

- [ ] **Step 5: Commit**

```bash
git add logic-grpc-service/service/candidate_ai_service.go logic-grpc-service/service/candidate_agent_context.go
git commit -m "feat: candidate AI reads system prompt from prompt_templates table with hardcoded fallback"
```

---

### Task 2: Agent 管理配置对接 HR 运行时

**Files:**
- Modify: `logic-grpc-service/service/ai_service.go` — `runADKChat` 和 `runLegacyChat` 注入 AgentConfig
- Modify: `logic-grpc-service/service/agent_service.go` — 确保 `GetAgentConfig` 可在 service 内部被调用
- Check: `logic-grpc-service/repository/agent_config_repo.go` — 确认 repo 方法

- [ ] **Step 1: 确认 AgentConfigRepo 的读取方法**

读取 `logic-grpc-service/repository/agent_config_repo.go`，确认 `GetByAgentType` 方法签名和返回值。确认返回的 `*model.AgentConfig` 包含：
- `ToolBindings`（已 eager-load 或需要单独查询）
- `PromptTemplateID`（外键指向 prompt_templates）
- `MaxIterations`
- `TemperatureOverride`
- `IsEnabled`

- [ ] **Step 2: 在 AIService 中注入 AgentConfigRepo 和 PromptTemplateRepo**

在 `logic-grpc-service/service/ai_service.go` 中找到 `AIService` 结构体，添加：
```go
type AIService struct {
    // ... existing fields ...
    agentConfigRepo *repository.AgentConfigRepo
    promptRepo      *repository.PromptTemplateRepo  // 可能已有
}
```

检查是否已有 `promptRepo` 字段（`contextBuilder` 中已使用）。如果没有则在构造函数中添加。

同时注入 `agentConfigRepo`。

- [ ] **Step 3: 创建 `getAgentRuntimeConfig` 辅助方法**

在 `logic-grpc-service/service/ai_service.go` 中添加新方法：

```go
// agentRuntimeConfig holds the resolved runtime configuration for an agent.
type agentRuntimeConfig struct {
    SystemPrompt       string
    ToolNames           []string
    MaxIterations       int
    TemperatureOverride *float64
}

func (s *AIService) getAgentRuntimeConfig(ctx context.Context, agentType string) *agentRuntimeConfig {
    cfg := &agentRuntimeConfig{
        MaxIterations: 0, // 默认：ADK 内部默认值
    }
    
    // 1. 从 agent_configs 读取配置
    if s.agentConfigRepo != nil {
        agentCfg, err := s.agentConfigRepo.GetByAgentType(ctx, agentType)
        if err == nil && agentCfg != nil && agentCfg.IsEnabled == 1 {
            if agentCfg.MaxIterations > 0 {
                cfg.MaxIterations = int(agentCfg.MaxIterations)
            }
            if agentCfg.TemperatureOverride != nil {
                cfg.TemperatureOverride = agentCfg.TemperatureOverride
            }
            
            // 2. 读取绑定的工具列表
            bindings, err := s.agentConfigRepo.ListToolBindings(ctx, agentCfg.ID)
            if err == nil {
                for _, b := range bindings {
                    if b.IsEnabled == 1 {
                        cfg.ToolNames = append(cfg.ToolNames, b.ToolName)
                    }
                }
            }
            
            // 3. 读取绑定的 Prompt 模板
            if agentCfg.PromptTemplateID != nil && *agentCfg.PromptTemplateID > 0 && s.promptRepo != nil {
                tmpl, err := s.promptRepo.GetByID(ctx, *agentCfg.PromptTemplateID)
                if err == nil && tmpl != nil {
                    cfg.SystemPrompt = tmpl.Content
                }
            }
        }
    }
    
    return cfg
}
```

- [ ] **Step 4: 修改 `runADKChat` — 根据 AgentConfig 过滤工具列表**

修改 `logic-grpc-service/service/ai_service.go:395` 的 `runADKChat` 方法。

在调用 `getOrInitADKTools()` 之后，添加工具过滤逻辑：

```go
adkTools, err := s.getOrInitADKTools()
if err != nil {
    // ... existing fallback ...
}

// 读取 Agent 运行时配置
runtimeCfg := s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent")

// 如果有配置的工具白名单，则过滤工具列表
if len(runtimeCfg.ToolNames) > 0 {
    adkTools = filterToolsByName(adkTools, runtimeCfg.ToolNames)
}

// 使用配置的 maxIterations
maxIterations := 0
if runtimeCfg.MaxIterations > 0 {
    maxIterations = runtimeCfg.MaxIterations
}

return aiClient.ChatWithADKAgent(ctx, ai.AgentRunInput{
    AgentName:     "hr_recruiting_agent",
    Instruction:   extractSystemInstruction(messages),
    Messages:      messages,
    Tools:         adkTools,
    MaxIterations: maxIterations,
    // ... existing fields ...
}, onDelta, traceFn, onStatus)
```

- [ ] **Step 5: 添加工具过滤辅助函数**

在 `logic-grpc-service/service/ai_service.go` 中添加：

```go
// filterToolsByName returns tools whose name is in the allowlist.
// If allowlist is empty, returns all tools unchanged.
func filterToolsByName(tools []tool.BaseTool, allowlist []string) []tool.BaseTool {
    if len(allowlist) == 0 {
        return tools
    }
    allowed := make(map[string]bool, len(allowlist))
    for _, name := range allowlist {
        allowed[name] = true
    }
    filtered := make([]tool.BaseTool, 0, len(tools))
    for _, t := range tools {
        info, err := t.Info(context.Background())
        if err == nil && allowed[info.Name] {
            filtered = append(filtered, t)
        }
    }
    return filtered
}
```

- [ ] **Step 6: 同样修改 `runLegacyChat` 路径**

修改 `logic-grpc-service/service/ai_service.go:439` 的 `runLegacyChat` 方法。

当前硬编码使用所有工具：
```go
tools := ai.RecruitingTools()
```

改为从 AgentConfig 读取配置的工具白名单来过滤 `RecruitingTools()` 返回的列表。与 ADK 路径使用相同的 `getAgentRuntimeConfig` 方法：

```go
runtimeCfg := s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent")
tools := ai.RecruitingTools()
if len(runtimeCfg.ToolNames) > 0 {
    tools = filterToolInfosByName(tools, runtimeCfg.ToolNames)
}
```

添加 `filterToolInfosByName` 辅助函数（操作 `[]*schema.ToolInfo`）。

- [ ] **Step 7: 编译验证**

```bash
cd logic-grpc-service && go build ./...
```
Expected: 编译通过。

- [ ] **Step 8: Commit**

```bash
git add logic-grpc-service/service/ai_service.go
git commit -m "feat: HR agent reads tool bindings and params from agent_configs table at runtime"
```

---

### Task 3: Agent 管理配置对接候选人运行时

**Files:**
- Modify: `logic-grpc-service/service/candidate_ai_service.go` — ADK 和 Legacy 路径使用 agent_configs
- Check: `logic-grpc-service/ai/candidate_adk_tools.go` — 确认工具列表

- [ ] **Step 1: 在 CandidateAIService 中注入 AgentConfigRepo**

在 `CandidateAIService` 结构体中添加：
```go
type CandidateAIService struct {
    // ... existing fields ...
    agentConfigRepo *repository.AgentConfigRepo
}
```

在构造函数中添加对应参数。

- [ ] **Step 2: 创建候选人专用的 `getCandidateAgentRuntimeConfig`**

在 `candidate_ai_service.go` 中添加，逻辑与 Task 2 的 `getAgentRuntimeConfig` 相同，但使用 `agent_type="candidate_assistant"`：

```go
func (s *CandidateAIService) getCandidateAgentRuntimeConfig(ctx context.Context) *agentRuntimeConfig {
    cfg := &agentRuntimeConfig{MaxIterations: 0}
    
    if s.agentConfigRepo != nil {
        agentCfg, err := s.agentConfigRepo.GetByAgentType(ctx, "candidate_assistant")
        if err == nil && agentCfg != nil && agentCfg.IsEnabled == 1 {
            if agentCfg.MaxIterations > 0 {
                cfg.MaxIterations = int(agentCfg.MaxIterations)
            }
            if agentCfg.TemperatureOverride != nil {
                cfg.TemperatureOverride = agentCfg.TemperatureOverride
            }
            
            bindings, _ := s.agentConfigRepo.ListToolBindings(ctx, agentCfg.ID)
            for _, b := range bindings {
                if b.IsEnabled == 1 {
                    cfg.ToolNames = append(cfg.ToolNames, b.ToolName)
                }
            }
            
            if agentCfg.PromptTemplateID != nil && *agentCfg.PromptTemplateID > 0 && s.promptRepo != nil {
                tmpl, _ := s.promptRepo.GetByID(ctx, *agentCfg.PromptTemplateID)
                if tmpl != nil {
                    cfg.SystemPrompt = tmpl.Content
                }
            }
        }
    }
    return cfg
}
```

- [ ] **Step 3: 修改候选人 ADK 路径 — 过滤工具 + 读取 Prompt**

在 `candidate_ai_service.go` 的 ADK 路径（约第 195-217 行）修改：

```go
adkTools, toolErr := s.getOrInitCandidateADKTools()
if toolErr != nil {
    // ... existing fallback ...
} else {
    // 读取 Agent 运行时配置
    runtimeCfg := s.getCandidateAgentRuntimeConfig(ctx)
    
    // 过滤工具列表
    if len(runtimeCfg.ToolNames) > 0 {
        adkTools = filterToolsByName(adkTools, runtimeCfg.ToolNames)
    }
    
    // 确定 system prompt 来源（DB 配置优先于 Task 1 的 agent_type 查询）
    systemPrompt := runtimeCfg.SystemPrompt
    if systemPrompt == "" {
        systemPrompt = candidateSystemPrompt  // 兜底：Task 1 的 agent_type 查询在外层已处理
    }
    messages, err := s.buildCandidateAgentMessages(ctx, userID, sessionID, message, systemPrompt)
    
    maxIter := 0
    if runtimeCfg.MaxIterations > 0 {
        maxIter = runtimeCfg.MaxIterations
    }
    
    reply, metadata, execErr = s.aiClient.ChatWithADKAgent(adkCtx, ai.AgentRunInput{
        AgentName:     "candidate_assistant",
        Instruction:   extractSystemInstruction(messages),
        Messages:      messages,
        Tools:         adkTools,
        MaxIterations: maxIter,
        // ... existing fields ...
    }, streamFilter.Write, traceFn, nil)
}
```

- [ ] **Step 4: 修改候选人 Legacy 路径 — 同样过滤工具**

在 Legacy 路径（约第 220-227 行）：
```go
} else {
    runtimeCfg := s.getCandidateAgentRuntimeConfig(ctx)
    tools := ai.CandidateTools()
    if len(runtimeCfg.ToolNames) > 0 {
        tools = filterToolInfosByName(tools, runtimeCfg.ToolNames)
    }
    // ... rest unchanged ...
}
```

- [ ] **Step 5: 提取共享的 `filterToolInfosByName` 到公共文件**

将 Task 2 Step 6 的 `filterToolInfosByName` 和本 Task 使用的 `filterToolsByName` 提取到一个公共位置，避免候选人服务重复定义。

在 `logic-grpc-service/service/` 下已有 `helpers.go`，将两个过滤函数放入该文件：

```go
// filterToolsByName filters ADK tools by name allowlist.
func filterToolsByName(tools []tool.BaseTool, allowlist []string) []tool.BaseTool {
    if len(allowlist) == 0 {
        return tools
    }
    allowed := make(map[string]bool, len(allowlist))
    for _, name := range allowlist {
        allowed[name] = true
    }
    filtered := make([]tool.BaseTool, 0, len(tools))
    for _, t := range tools {
        info, err := t.Info(context.Background())
        if err == nil && allowed[info.Name] {
            filtered = append(filtered, t)
        }
    }
    return filtered
}

// filterToolInfosByName filters schema.ToolInfo by name allowlist.
func filterToolInfosByName(tools []*schema.ToolInfo, allowlist []string) []*schema.ToolInfo {
    if len(allowlist) == 0 {
        return tools
    }
    allowed := make(map[string]bool, len(allowlist))
    for _, name := range allowlist {
        allowed[name] = true
    }
    filtered := make([]*schema.ToolInfo, 0, len(tools))
    for _, t := range tools {
        if allowed[t.Name] {
            filtered = append(filtered, t)
        }
    }
    return filtered
}
```

需要的 import：
```go
import (
    "context"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/schema"
)
```

- [ ] **Step 6: 编译验证**

```bash
cd logic-grpc-service && go build ./...
```
Expected: 编译通过。

- [ ] **Step 7: Commit**

```bash
git add logic-grpc-service/service/candidate_ai_service.go logic-grpc-service/service/helpers.go
git commit -m "feat: candidate agent reads tool bindings and params from agent_configs table at runtime"
```

---

### Task 4: Prompt 管理补充 — HR 端 Agent 配置中的 Prompt 模板优先

**Files:**
- Modify: `logic-grpc-service/service/agent_context.go`

- [ ] **Step 1: 修改 `AgentContextBuilder.Build` — Agent 配置中的 Prompt 模板优先**

在当前 `agent_context.go:146-151`，HR 端通过 `promptRepo.GetActiveByAgentType(ctx, "hr_agent", "system")` 查询 system prompt。

现在需要在 Agent 配置中也支持指定 prompt_template_id。如果 Agent 配置中有 `prompt_template_id`，则优先使用该模板；否则回退到当前按 agent_type 查询的逻辑。

由于 `AgentContextBuilder` 目前不持有 `agentConfigRepo`，有两种方式：
- A) 注入 `agentConfigRepo` 到 `AgentContextBuilder`
- B) 在调用 `Build` 之前由 `runToolCallingChat` 查询并传入

选择方案 B（侵入性更小）：

在 `runToolCallingChat`（约第 296 行）中，在调用 `contextBuilder.Build` 之前查询 Agent 配置中的 prompt 模板：

```go
func (s *AIService) runToolCallingChat(...) {
    // 查询 Agent 配置中的 prompt 模板
    runtimeCfg := s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent")
    
    actx, err := s.contextBuilder.Build(ctx, AgentContextInput{
        HrID:           req.HrId,
        SessionID:      session.ID,
        ApplicationID:  req.ApplicationId,
        CurrentMessage: req.Message,
    })
    
    // 如果 Agent 配置中有 prompt 模板，覆盖 contextBuilder 从 DB 读取的值
    if runtimeCfg.SystemPrompt != "" {
        actx.SystemPromptTemplate = runtimeCfg.SystemPrompt
    }
    // ... rest unchanged ...
}
```

这样 Agent 配置中的 prompt_template_id 绑定的模板会覆盖通过 agent_type 查询到的 system prompt。

- [ ] **Step 2: 编译验证**

```bash
cd logic-grpc-service && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add logic-grpc-service/service/ai_service.go
git commit -m "feat: HR agent config prompt_template_id overrides agent_type system prompt"
```

---

### Task 5: 端到端验证

- [ ] **Step 1: 确认种子数据一致**

验证 `SeedDefaultPrompts` 和 `SeedDefaultAgents` 的种子数据：
- `SeedDefaultPrompts` 中的 `agent_type` 值应与运行时查询一致：`"hr_agent"` / `"candidate_assistant"`
- `SeedDefaultAgents` 中的 `AgentType` 值应匹配：`"hr_recruiting_agent"` / `"candidate_assistant"`

检查 `logic-grpc-service/service/prompt_service.go` 的 `SeedDefaultPrompts` 方法中 agent_type 字段：
```bash
grep -n 'agent_type\|AgentType\|agentType' logic-grpc-service/service/prompt_service.go logic-grpc-service/service/agent_service.go
```

如果不一致（一个是 `"hr_agent"`，另一个是 `"candidate_assistant"`），需要统一。

- [ ] **Step 2: 运行现有测试**

```bash
cd logic-grpc-service && go test ./... 2>&1 | tail -20
```
Expected: 所有测试通过。

- [ ] **Step 3: 手动测试场景**

1. **Prompt 管理 → HR 对话**：在 Prompt 管理页面修改 HR Agent 的 system prompt → 发送 HR 端 AI 对话 → 验证回复风格与修改一致
2. **Prompt 管理 → 候选人对话**：在 Prompt 管理页面修改候选人 Agent 的 system prompt → 发送候选人端 AI 对话 → 验证回复风格与修改一致
3. **Agent 管理 → 工具控制**：在 Agent 管理页面禁用某个工具（如 `query_total_applications`）→ 发送对话 → 验证该工具不会被调用
4. **Agent 管理 → MaxIterations**：修改 max_iterations 为 2 → 验证 Agent 不会超过 2 轮工具调用
5. **硬编码兜底**：删除数据库中对应 agent_type 的 agent_config 记录 → 验证系统回退到硬编码默认值，不崩溃

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test: end-to-end verification for prompt and agent runtime integration"
```

---

## 风险与回滚

| 风险 | 缓解措施 |
|------|---------|
| Agent 配置错误导致工具列表为空 | 空 allowlist 不过滤，使用全部工具（兜底逻辑） |
| DB 中 prompt_templates 内容有误 | 保留硬编码 `candidateSystemPrompt` / `buildToolCallingMessages` 兜底 |
| agent_type 命名不一致导致查不到配置 | Task 5 Step 1 统一命名 |
| 性能影响 | AgentConfig 查询每次对话只执行一次，工具列表缓存 |

回滚方式：每个 task 是独立 commit，可以单独 revert。所有修改都有兜底逻辑，删除 DB 中对应记录即可恢复硬编码行为。
