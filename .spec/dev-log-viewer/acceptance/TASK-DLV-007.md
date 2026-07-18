# Acceptance - TASK-DLV-007

## TASK Summary

完成暂停/跟随、异常状态、复制/导出和受限布局持久化。

## SPEC References

- UI-018 至 UI-035
- FR-019 至 FR-024
- SS-005 至 SS-009
- AC-007 至 AC-011、AC-016、AC-017、AC-018
- D-007 至 D-010

## SDD References

- 3.5 页面状态机
- 6.6 客户端数据流
- 9.2 前端降级
- 11.2 React 单元/组件测试
- 15 已确认技术决策

## Acceptance Criteria

- 暂停继续接收但冻结可见快照；恢复、跳到最新和 unseen 计数行为确定。
- reconnecting、paused、high-volume、no-results、no-services 与 reset 分隔状态符合设计基线。
- 清空视图只清客户端内存；复制失败和手动重连有可见反馈。
- 导出当前筛选顺序的 UTF-8 `.log`，不超过客户端缓冲，不调用新后端 API。
- 只保存合法的侧栏折叠和详情宽度；非法、旧版本或不可用 localStorage 安全回退。
- 页面持续显示敏感日志及禁止局域网/公网暴露提示。
- 不实现自动脱敏、JSONL、服务端导出或远程资源请求。

## Required Checks

- `pnpm --filter dev-log-viewer typecheck`
- `pnpm --filter dev-log-viewer test`
- `pnpm --filter dev-log-viewer build`
- Harness scope 与 agent-check。

## Manual Verification, if needed

- 对照剩余六张设计状态截图逐一验证，并记录目标视口和结果。

## Out-of-Scope

静态嵌入、启动脚本、自动日志脱敏、服务端导出。
