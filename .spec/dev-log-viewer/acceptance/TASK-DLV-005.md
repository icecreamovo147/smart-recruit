# Acceptance - TASK-DLV-005

## TASK Summary

实现前端 API 类型、唯一流连接、10,000 条日志环和纯状态逻辑。

## SPEC References

- UI-010、UI-020 至 UI-025、UI-029 至 UI-034
- FR-019 至 FR-022
- AC-006 至 AC-009、AC-011

## SDD References

- 3.4 前端职责
- 3.5 页面状态机
- 4 数据结构
- 5 API 契约
- 6.6 客户端数据流
- 9.2 前端降级

## Acceptance Criteria

- TypeScript 类型与服务目录及 SSE envelope 契约一致，无 `any` 逃逸。
- reducer 按 EventID 去重，最多保留 10,000 条，淘汰最旧项并累计 dropped。
- 服务、级别、文本和关联 ID 筛选采用 AND 语义且不修改源缓冲。
- reset、dropped、malformed event 和连接状态转换可测试且不会使状态崩溃。
- 只维护一个有效 EventSource；服务选择变化和卸载会清理旧连接。
- 暂停渲染、停止自动跟随、unseen 和连接断开是独立状态。

## Required Checks

- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`
- Harness scope 与 agent-check。

## Manual Verification, if needed

无；使用 mock EventSource 和 fake timers 自动验证。

## Out-of-Scope

正式布局、导出、localStorage、Go 代码和视觉验收。
