# 任务编号：P0-004

## 1. 任务目标

在 HR 端聊天页右侧新增「Agent 执行轨迹」面板，调用 P0-003 创建的 `GetToolTraces` 接口，以时间线形式展示工具调用的工具名、入参、结果、耗时。

## 2. 对应缺陷

- **缺陷 6**：Agent 执行轨迹后端已落库但前端零展示（前端侧补齐）

## 3. 前置条件

- P0-003 完成（`GetToolTraces` 接口可用）
- 前端 API 层对接 `GetToolTraces`

## 4. 允许修改的文件

- `hr-frontend/src/views/hr/AIChatView.vue`
- `hr-frontend/src/api/ai.ts`（新增 `getToolTraces`）
- `hr-frontend/src/types/ai.ts`（新增类型定义）
- `hr-frontend/src/components/`（如需新建 AgentTracePanel.vue）

## 5. 禁止修改的文件

- `logic-grpc-service/` 下所有文件
- `web-gin-service/` 下所有文件
- `user-frontend/`、`interviewer-frontend/`
- 其他已有业务页面

## 6. 实现步骤

1. 在 `types/ai.ts` 中新增类型：
   ```ts
   interface ToolTraceItem {
     id: number
     session_id: number
     tool_name: string
     args_json: string
     result_content: string
     duration_ms: number
     error_msg: string
     created_at: string
   }
   ```
2. 在 `api/ai.ts` 中新增 `getToolTraces(sessionId: number)` 方法
3. 在 `AIChatView.vue` 右侧新增抽屉/面板组件：
   - 使用 `el-drawer` 或固定侧边面板
   - 使用 Element Plus `el-timeline` 时间线组件
   - 每条时间线项展示：工具名称、入参（JSON 格式化显示）、结果概要（前 200 字符）、耗时（ms）、错误信息（红色标注）
   - 面板入口：在聊天页顶部工具栏增加「执行轨迹」按钮
4. 面板数据按会话过滤，切换会话时自动刷新

## 7. 数据库变更

不涉及

## 8. 接口变更

前端新增调用 P0-003 的 `GetToolTraces` gRPC 接口（通过 HTTP 网关）

## 9. 前端变更

- `AIChatView.vue`：增加轨迹面板入口 + 面板组件引用
- 新增 `AgentTracePanel.vue`：轨迹展示面板
- `api/ai.ts`：新增 `getToolTraces`
- `types/ai.ts`：新增类型

## 10. 权限与安全要求

- 接口调用已有权限校验（P0-003 已实现）
- 入参 JSON 格式化后需注意不暴露完整简历正文（后端已脱敏）

## 11. 验收标准

- [ ] 聊天页有「执行轨迹」入口按钮
- [ ] 点击后展示当前会话的工具调用时间线
- [ ] 时间线展示：工具名称、入参、结果摘要、耗时
- [ ] 切换会话时轨迹自动刷新
- [ ] 无工具调用时显示「本次会话无工具调用」
- [ ] 错误调用以红色/警告色标识
- [ ] 面板不影响已有聊天功能
- [ ] `pnpm typecheck` 通过
- [ ] `pnpm build` 通过

## 12. 必须运行的测试命令

```bash
cd hr-frontend && pnpm install && pnpm typecheck && pnpm build
```

## 13. 完成后必须输出的内容

- 变更摘要
- UI 截图（时间线面板展示）
- 构建验证结果

## 14. 回滚方案

- 移除新增组件文件
- 恢复 `AIChatView.vue` 修改前版本
- 恢复 `api/ai.ts` 和 `types/ai.ts` 修改

---

## 完成记录

**完成日期**: 2026-06-26

### 变更摘要

| 文件 | 变更 |
|------|------|
| `hr-frontend/src/types/ai.ts` | 新增 `ToolTraceItem` 接口 |
| `hr-frontend/src/api/ai.ts` | 新增 `getToolTraces(sessionId)` 方法，导入 `ToolTraceItem` 类型 |
| `hr-frontend/src/components/AgentTracePanel.vue` | **新建**，Agent 执行轨迹面板组件，使用 `el-drawer` + `el-timeline`，展示工具名、入参（JSON 格式化）、结果摘要（前 200 字符，可展开）、耗时、错误信息（红色标注），空状态显示"本次会话无工具调用" |
| `hr-frontend/src/views/hr/AIChatView.vue` | 导入 `AgentTracePanel` 组件，状态栏新增「执行轨迹」按钮，集成面板组件，面板按当前会话 ID 自动加载 |

### 验证结果

- [x] `pnpm install`: PASS
- [x] `pnpm typecheck`: PASS
- [x] `pnpm test`: PASS (9 tests, all passed)
- [x] `pnpm build`: PASS
- [x] 无 TODO/FIXME/HACK 残留
- [x] 无 console.log 调试代码
- [x] 未修改禁止文件

### 任务状态

- [x] 聊天页有「执行轨迹」入口按钮
- [x] 点击后展示当前会话的工具调用时间线
- [x] 时间线展示：工具名称、入参、结果摘要、耗时
- [x] 切换会话时轨迹自动刷新
- [x] 无工具调用时显示「本次会话无工具调用」
- [x] 错误调用以红色/警告色标识
- [x] 面板不影响已有聊天功能
