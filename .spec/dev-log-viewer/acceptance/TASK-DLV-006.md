# Acceptance - TASK-DLV-006

## TASK Summary

实现 Canonical 高密度 UI、响应式布局、详情分栏和虚拟日志列表。

## SPEC References

- UI-001 至 UI-019
- UI-020 至 UI-024
- NFR-006 至 NFR-009
- AC-007、AC-009、AC-010、AC-014
- D-010、D-011

## SDD References

- 1.4 设计基线
- 2.4 UI 一致性难点
- 3.4、3.5 前端职责与状态机
- 6.6 客户端数据流
- 11.2 React 测试

## Acceptance Criteria

- 页面具备顶部工具栏、服务侧栏、筛选栏、日志主区域、详情分栏和状态栏。
- 服务名、分组、端口严格使用 SPEC UI-007，不复制设计稿虚构数据。
- 1440×900 完整布局和 1366×768 紧凑布局无页面级双滚动或正文遮挡。
- 日志列表使用 `@tanstack/react-virtual`，10,000 条状态不创建 10,000 个 DOM 行。
- 详情面板可调整宽度但不覆盖日志列；缺失字段不伪造。
- 键盘焦点可见，状态不只依靠颜色；日志正文以文本节点渲染。
- 生产构建不含 CDN 或远程字体请求。

## Required Checks

- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`
- `pnpm --filter dev-log-viewer build`
- Harness scope 与 agent-check。

## Manual Verification, if needed

- 对照 `design/screens/Live Logs - Canonical 1440p.png` 与 compact 截图检查两个目标视口。
- 记录差异和理由，不以 Stitch HTML 的示例数据覆盖仓库事实。

## Out-of-Scope

导出、布局持久化、全部异常状态、后端和启停脚本。
