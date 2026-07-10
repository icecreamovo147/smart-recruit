---
description: Execute exactly one SPEC + SDD + Harness task
---

你现在执行 SPEC + SDD + Harness 工作流。

功能点：$1
任务编号：$2

本轮只执行 `$2`，不要执行其他任务。

必须读取并遵守：

- AGENTS.md
- .agents/skills/spec-harness/SKILL.md
- .spec/$1/$1-SPEC.md
- .spec/$1/$1-SDD.md
- .spec/$1/TASKS.md
- .spec/$1/AGENT_RULES.md
- .spec/$1/task-scope.json
- .spec/$1/acceptance/$2.md
- .spec/$1/prompts/implement-task.md

执行要求：

1. 先确认 `$2` 的目标、允许修改范围、禁止修改范围和验收标准；
2. 运行 `git status --short`，如果存在与 `$2` 无关的改动，停止并说明；
3. 只实现 `$2`；
4. 不修改 `$2` 范围之外的文件；
5. 不顺手重构；
6. 不修改 package.json、lockfile、全局配置，除非 `$2` 明确允许；
7. 如果需要扩大范围，立即停止并请求确认；
8. 完成后运行：
   - `git diff --name-only`
   - `bash .spec/$1/scripts/check-task-scope.sh $2`
   - `bash .spec/$1/scripts/agent-check.sh`
9. 创建或更新报告：
   - `.spec/$1/reports/$2-report.md`

报告必须包含：

- 任务目标
- 修改文件列表
- 每个文件修改说明
- 是否超出 task-scope.json
- SPEC 对照结果
- SDD 对照结果
- acceptance 对照结果
- 检查命令与结果
- 未完成内容
- 风险点
- 是否建议进入下一个 TASK

完成 `$2` 后停止，不要继续执行后续任务。