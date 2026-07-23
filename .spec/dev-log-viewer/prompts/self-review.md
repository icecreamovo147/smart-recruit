# Self Review Prompt

使用 `spec-harness`。

Mode: `self-review`

Feature: `dev-log-viewer`

Task: `<TASK-ID>`

读取：

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/dev-log-viewer/dev-log-viewer-SPEC.md`
- `.spec/dev-log-viewer/dev-log-viewer-SDD.md`
- `.spec/dev-log-viewer/TASKS.md`
- `.spec/dev-log-viewer/AGENT_RULES.md`
- `.spec/dev-log-viewer/task-scope.json`
- `.spec/dev-log-viewer/acceptance/<TASK-ID>.md`
- `.spec/dev-log-viewer/reports/<TASK-ID>-report.md`
- `.spec/dev-log-viewer/reports/<TASK-ID>-evidence.json`
- 当前 TASK 的 Git diff 和知识影响文档

优先由未参与实现的独立 subagent 或 fresh context 进行只读复审。禁止修改任何文件。

检查：

- scope、SPEC、SDD、acceptance、依赖顺序和 Hard Stop；
- 测试是否真实执行，报告/evidence/exit code/timing 是否一致；
- 安全、有界性、错误隔离、兼容性和可观测性；
- knowledge impact 是否按路由逐文档给出固定 verdict；
- 是否出现未批准行为、隐藏公共 API、无关重构或测试缺口。

按 Critical/High/Medium/Low 输出 findings 和 Required Fixes，最后只能使用：

```text
verdict: 通过
```

或：

```text
verdict: 不通过
```
