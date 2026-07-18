# Implement TASK Prompt

使用 `spec-harness`。

Mode: `implement-task`

Feature: `dev-log-viewer`

Task: `<TASK-ID>`

开始修改前完整读取：

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/dev-log-viewer/dev-log-viewer-SPEC.md`
- `.spec/dev-log-viewer/dev-log-viewer-SDD.md`
- `.spec/dev-log-viewer/TASKS.md`
- `.spec/dev-log-viewer/AGENT_RULES.md`
- `.spec/dev-log-viewer/task-scope.json`
- `.spec/dev-log-viewer/acceptance/<TASK-ID>.md`
- `.knowledge/README.md`、`.knowledge/manifest.yaml`、`.knowledge/INDEX.md`
- 当前 TASK `knowledge.review` 路由的 active 文档

执行规则：

- 先将 feature 分类为 current/legacy-compatible/unsupported，并建立能区分本 TASK 与既有改动的可靠 Git base tree。
- 只执行 `<TASK-ID>`，不提前执行后续 TASK，不修改 SPEC/SDD/TASKS/acceptance。
- 严格遵守 `allowedFiles`、`forbiddenFiles`、依赖和 acceptance；范围不足时 Hard Stop。
- 如果在 Codex Goal mode 中运行，保持 Goal 持续推进当前 TASK 的 implement → review → repair 闭环，但不得绕过单 TASK 状态机。
- 尽可能使用 subagent 隔离仓库调研、测试设计、知识影响分析和失败诊断；主 Agent 统一管理写入、baseline、scope、pipeline-state、报告和 evidence。
- 不让多个写入型 subagent 同时修改相同文件；subagent 不得自行扩 scope 或推进下一 TASK。
- `requiresHumanConfirmation` 必须在写入前有明确确认并记录到 evidence。TASK-DLV-001/008 的已批准事项见 SPEC 第 13 节和 task notes；仍需记录本次用户确认文本。
- 实施后运行 TASK required checks、scope check 和 agent-check，生成一致的 Markdown report 与 evidence JSON。
- 完成实现后停止在当前 TASK，交给未参与实现的 subagent/fresh context 执行 `self-review`。
