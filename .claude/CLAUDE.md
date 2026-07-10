# Claude Code Project Rules

## 当前项目目标

本项目正在进行“智能招聘 Agent 平台化改造”。

当前系统已有招聘业务系统、AI 问答、Eino/ADK Agent、SSE、工具调用 Trace、Token 估算、RBAC、审计等基础能力，但仍需要逐步补齐模型配置、Prompt 管理、Trace 可视化、MCP、Skills、RAG、成本统计、安全治理等 Agent 平台能力。

## 强制规则

1. 每次只能执行一个任务文件。
2. 没有任务文件，不允许开发。
3. 没有测试结果，不允许声称完成。
4. 不允许顺手开发任务范围外的功能。
5. 不允许直接修改 main 分支。
6. 所有任务从 `integration/agent-platform` 创建任务分支。
7. 每个任务必须在独立 git worktree 中完成。
8. Developer Agent 负责开发。
9. Reviewer Agent 只负责审查，不允许改代码。
10. Fixer Agent 只允许修复 Reviewer 指出的问题。
11. 每个任务最多允许 2 轮自动修复。
12. 测试失败、合并冲突、任务范围不清晰时必须停止。
13. 任务状态变更时，必须同步更新 `docs/agent-harness/EXECUTION_LOG.md` 中对应任务的状态、分支、Review 结论、备注等字段。

## 每次开发前必须读取

- docs/agent-harness/00-HARNESS.md
- docs/agent-harness/03-ARCHITECTURE_GUARDRAILS.md
- docs/agent-harness/04-TEST_COMMANDS.md
- docs/agent-harness/05-DEFINITION_OF_DONE.md
- docs/agent-harness/06-REVIEW_CHECKLIST.md
- 当前任务文件

## 分支规则

- 主开发集成分支：`integration/agent-platform`
- 任务分支命名：`agent/<task-id>-<task-name>`
- 不允许直接在 main 开发
- 不允许自动 push 到远程
- 不允许自动合并到 main

## 停止条件

遇到以下情况必须停止并报告：

1. 需要修改任务范围外的文件。
2. 需要新增依赖，但任务文件没有允许。
3. 需要新增数据库 migration，但任务文件没有设计。
4. 测试失败且无法在当前任务范围内修复。
5. 发现任务文件与实际代码不一致。
6. 合并冲突无法安全解决。
7. 可能影响现有招聘业务主流程。
8. 涉及 API Key、简历、手机号、邮箱、薪资等敏感信息但没有脱敏方案。