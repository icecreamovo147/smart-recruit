# Acceptance - TASK-004

## TASK Summary

创建本地开发、Agent 召回调试 Runbook，以及 Proto、Embedding fallback、HR 管理页一致性 Pitfall。

## SPEC References

- FR-003 至 FR-005
- FR-013
- AC-002、AC-003、AC-009、AC-013、AC-014

## SDD References

- Section 3, Proposed Design
- Section 7, Configuration Design
- Section 11.3, Content Acceptance
- Section 13, TASK-004

## Acceptance Criteria

- 创建 SPEC/TASKS 指定的 5 篇 runbook/pitfall 文档。
- 所有命令、路径、端口、降级行为和同步约束可从当前仓库验证。
- Runbook 区分通用步骤和 OS 差异，不保存本地实际配置。
- Pitfall 描述触发条件、风险、验证方式和来源。
- INDEX/manifest 能将 Proto、Embedding 和 HR 管理页路径路由到对应文档。
- 不重复定义 AGENTS 的持久规则，不包含真实凭据。

## Required Checks

- 知识工具测试、结构校验和引用检查。
- 对 Proto、Embedding、HR 管理页代表性路径运行影响检测测试。
- `git diff --check`
- Harness scope 与 agent check。

## Manual Verification, if needed

- 按 Runbook 在当前设备核对命令存在性；需要外部服务的步骤可说明未实际启动。
- Reviewer 检查示例配置只使用占位符。

## Out-of-Scope

- 业务代码修复。
- Root README、AGENTS、Harness、CI 和部署配置修改。
