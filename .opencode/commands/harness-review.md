---
description: Review one SPEC + SDD + Harness task diff
---

你现在执行 self-review。

功能点：$1
任务编号：$2

本轮只做 Review，不要修改代码。

必须对照：

- AGENTS.md
- .agents/skills/spec-harness/SKILL.md
- .spec/$1/$1-SPEC.md
- .spec/$1/$1-SDD.md
- .spec/$1/TASKS.md
- .spec/$1/AGENT_RULES.md
- .spec/$1/task-scope.json
- .spec/$1/acceptance/$2.md
- .spec/$1/reports/$2-report.md

请检查：

1. 是否只完成了 `$2`；
2. 是否修改了 `$2` 范围之外的文件；
3. 是否违反 AGENT_RULES.md；
4. 是否满足 SPEC；
5. 是否符合 SDD；
6. 是否满足 acceptance；
7. 是否存在类型、Lint、运行时、异常处理、兼容性风险；
8. 是否影响公共模块或已有功能；
9. 是否需要补充测试。

输出：

- Review 结论：通过 / 不通过
- 问题列表
- 严重级别
- 是否必须修复后才能进入下一个 TASK
- 建议修复方式

本轮不要修改代码。