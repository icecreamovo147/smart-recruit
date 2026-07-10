# Self Review Prompt (Followups)

你要对刚完成的单个 TASK 做自查，不要实现新功能。

## 必须先读取

1. `.spec/skill-memory-ranking-followups/skill-memory-ranking-followups-SPEC.md`
2. `.spec/skill-memory-ranking-followups/skill-memory-ranking-followups-SDD.md`
3. `.spec/skill-memory-ranking-followups/TASKS.md`
4. `.spec/skill-memory-ranking-followups/AGENT_RULES.md`
5. `.spec/skill-memory-ranking-followups/task-scope.json`
6. `.spec/skill-memory-ranking-followups/acceptance/<TASK-ID>.md`
7. `.spec/skill-memory-ranking-followups/reports/<TASK-ID>-report.md`

## 自查内容

- `git diff` 是否只包含当前 TASK 范围内文件。
- 是否修改了 `forbiddenFiles`。
- 是否改变 TASK-001..008 已落地的算法公式 / proto 字段 / 函数签名；如果是，是否已人工确认。
- 是否存在敏感信息泄漏（API Key、token、Memory 完整内容、Skill markdown 完整内容、embedding vector 原值、query 原文）。
- 是否存在吞异常、随意使用 `any`、删除测试、弱化断言。
- 是否满足 `acceptance` 的自动化、手工、异常、回归检查项。
- 是否执行了 `requiredCommands`；未执行必须说明原因。
- 是否破坏了 TASK-001..008 的兼容性（const → var 转换、ctx helper 替换、config 加载等）。

## 输出格式

- 结论：通过 / 不通过。
- 发现的问题，按严重程度排序（Critical / High / Medium / Low）。
- 必须修复的问题。
- 可后续优化的问题。
- 已执行命令与结果。
- 是否建议进入下一个 TASK。

## Verdict

- `verdict: 通过` —— 所有 Hard Stop 条件未触发，所有 Acceptance 检查项通过。
- `verdict: 不通过` —— 至少一个 Hard Stop 条件触发，或至少一个 Acceptance 检查项不通过，需提供具体问题清单。
