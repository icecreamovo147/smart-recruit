# 任务编号：P1-005

## 1. 任务目标

在 HR 管理端新增「Agent 管理」页面，对接 P1-004 创建的 Agent 配置接口，支持可视化创建/编辑 Agent，绑定模型/Prompt/工具，以及在线预览调试。

## 2. 对应缺陷

- **缺陷 2**：Agent 配置全部硬编码，无前端编排入口（前端侧）

## 3. 前置条件

- P1-004 完成（后端接口可用）
- P0-006 完成（模型配置页面可用，用于模型选择器）
- P1-002 完成（Prompt 管理页面可用，用于 Prompt 选择器）

## 4. 允许修改的文件

- `hr-frontend/src/router/index.ts`
- `hr-frontend/src/App.vue`
- `hr-frontend/src/views/hr/admin/`（新增 AgentManageView.vue）
- `hr-frontend/src/api/`（新增 agent.ts）
- `hr-frontend/src/types/`（新增 agent.ts 类型）

## 5. 禁止修改的文件

- `logic-grpc-service/` 下所有文件
- `web-gin-service/` 下所有文件
- 已有业务页面

## 6. 实现步骤

### 6.1 路由与菜单

- 新增路由 `/hr/admin/agents`，`meta.perm: PERM.SYSTEM_CONFIG_MANAGE`
- 在侧边栏菜单新增「Agent 管理」菜单项

### 6.2 页面设计

1. **列表页**：`el-table` 展示 Agent 名称/类型/绑定模型/Prompt/工具数量/状态/操作
2. **编辑页/弹窗**：
   - 基本信息：名称、描述、类型
   - 模型选择：下拉从已有的模型列表选择
   - Prompt 选择：下拉从已有的 Prompt 模板选择
   - 额外指令：textarea
   - 最大迭代次数：数字输入
   - 温度覆盖：滑块（可选，留空用模型默认值）
   - 工具绑定：多选从已有工具列表选择
3. **调试预览**（可选，本期基础版）：
   - 使用选定的 Agent 配置发送测试消息，查看回复

## 7. 数据库变更

不涉及

## 8. 接口变更

前端对接 P1-004 接口

## 9. 前端变更

- 新增 `AgentManageView.vue`
- 新增 `api/agent.ts`、`types/agent.ts`
- 修改 `router/index.ts` + `App.vue`

## 10. 权限与安全要求

- 页面需 `SYSTEM_CONFIG_MANAGE` 权限
- 调试预览仅管理员可用

## 11. 验收标准

- [ ] 左侧菜单有「Agent 管理」入口
- [ ] 列表页可查看所有 Agent 配置
- [ ] 可新增/编辑/删除 Agent
- [ ] 模型选择器从已有模型列表读取
- [ ] Prompt 选择器从已有 Prompt 列表读取
- [ ] 工具绑定可选多选
- [ ] `pnpm typecheck` 通过
- [ ] `pnpm build` 通过

## 12. 必须运行的测试命令

```bash
cd hr-frontend && pnpm install && pnpm typecheck && pnpm build
```

## 13. 完成后必须输出的内容

- 变更摘要
- UI 截图（列表页、编辑弹窗、调试预览）
- 构建验证结果

## 14. 回滚方案

- 移除新增文件
- 恢复 `router/index.ts` 和 `App.vue`

---

## 完成记录

### 完成日期
2026-06-26

### Developer Agent
@icecreamovo147

### 验证结果
- [x] pnpm typecheck: PASS
- [x] pnpm build: PASS
- [x] pnpm test: PASS (9 tests passed)
- [x] 未修改禁止文件: PASS
- [x] 无 TODO/FIXME/HACK: PASS
- [x] 无调试代码残留: PASS

