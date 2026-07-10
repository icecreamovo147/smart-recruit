# 任务编号：P1-002

## 1. 任务目标

在 HR 管理端新增「Prompt 管理」页面，对接 P1-001 创建的 Prompt 管理接口，支持可视化管理 Prompt 模板、查看版本历史、回滚到历史版本、变量预览。

## 2. 对应缺陷

- **缺陷 3**：System Prompt 硬编码，无 Prompt 管理/版本（前端侧）

## 3. 前置条件

- P1-001 完成（后端接口可用）

## 4. 允许修改的文件

- `hr-frontend/src/router/index.ts`
- `hr-frontend/src/App.vue`
- `hr-frontend/src/views/hr/admin/`（新增 PromptManageView.vue）
- `hr-frontend/src/api/`（新增 prompt.ts）
- `hr-frontend/src/types/`（新增 prompt.ts 类型）

## 5. 禁止修改的文件

- `logic-grpc-service/` 下所有文件
- `web-gin-service/` 下所有文件
- 已有业务页面

## 6. 实现步骤

### 6.1 路由与菜单

- 新增路由 `/hr/admin/prompts`，`meta.perm: PERM.SYSTEM_CONFIG_MANAGE`
- 在侧边栏菜单「基础数据」分组或独立新增「Prompt 管理」菜单项

### 6.2 页面设计 (`PromptManageView.vue`)

1. **列表页**：
   - `el-table` 展示：名称、角色（系统/用户）、Agent 类型（HR/候选人）、当前版本、状态、更新时间
   - 操作：编辑、版本历史、删除、启用/禁用

2. **编辑弹窗/页面**：
   - `el-form`：名称、角色选择（system/user）、Agent 类型
   - `el-input type="textarea"` 用于 Prompt 内容编辑（等宽字体，支持多行）
   - 变量高亮/提示：`{{variable}}` 格式高亮显示
   - 变量列表展示：从模板中自动提取 `{{...}}` 占位符
   - 保存时自动递增版本

3. **版本历史**：
   - `el-dialog` 展示所有历史版本列表
   - 每行：版本号、变更人、变更时间、变更备注
   - 可查看历史版本内容（只读）
   - 支持回滚到任意历史版本（二次确认弹窗）

4. **变量预览**（可选）：
   - 输入测试变量值，预览渲染后效果

## 7. 数据库变更

不涉及

## 8. 接口变更

前端对接 P1-001 接口

## 9. 前端变更

- 新增 `PromptManageView.vue`
- 新增 `api/prompt.ts`、`types/prompt.ts`
- 修改 `router/index.ts` + `App.vue`

## 10. 权限与安全要求

- 页面需 `SYSTEM_CONFIG_MANAGE` 权限
- 候选人 Prompt 模板不暴露给非管理员

## 11. 验收标准

- [ ] 左侧菜单有「Prompt 管理」入口
- [ ] 列表页可查看所有 Prompt 模板
- [ ] 可新增/编辑/删除 Prompt 模板
- [ ] 编辑后版本号自动递增
- [ ] 版本历史可查看，可回滚
- [ ] 回滚有二次确认
- [ ] `pnpm typecheck` 通过
- [ ] `pnpm build` 通过

## 12. 必须运行的测试命令

```bash
cd hr-frontend && pnpm install && pnpm typecheck && pnpm build
```

## 13. 完成后必须输出的内容

- 变更摘要
- UI 截图（列表页、编辑弹窗、版本历史弹窗、回滚确认）
- 构建验证结果

## 14. 回滚方案

- 移除新增文件
- 恢复 `router/index.ts` 和 `App.vue`
