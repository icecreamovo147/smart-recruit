# Self Review Prompt

你要对刚完成的单个 TASK 做自查，不要实现新功能。

## 必须先读取

1. `.spec/skill-memory-ranking/skill-memory-ranking-SPEC.md`
2. `.spec/skill-memory-ranking/skill-memory-ranking-SDD.md`
3. `.spec/skill-memory-ranking/TASKS.md`
4. `.spec/skill-memory-ranking/AGENT_RULES.md`
5. `.spec/skill-memory-ranking/task-scope.json`
6. `.spec/skill-memory-ranking/acceptance/<TASK-ID>.md`
7. `.spec/skill-memory-ranking/reports/<TASK-ID>-report.md`

## 自查内容

- `git diff` 是否只包含当前 TASK 范围内文件。
- 是否修改了 `forbiddenFiles`。
- 是否改变公共 API 行为或公共模块；如果是，是否已人工确认。
- 是否存在敏感信息泄漏（API Key、token、Memory 完整内容、Skill markdown 完整内容、embedding vector 原值、query 原文）。
- 是否存在吞异常、随意使用 `any`、删除测试、弱化断言。
- 是否满足 `acceptance` 的自动化、手工、异常、回归检查项。
- 是否执行了 `requiredCommands`；未执行必须说明原因。
- 权重 / 阈值常量是否集中在 `skill_memory_ranking.go` 顶部。
- 旧 `Score int` 字段值映射是否正确。
- proto 字段是否仅追加，未修改 / 删除 / 重用现有字段编号。
- Skill 池 / Memory 池是否互不污染。
- Memory gating 是否在阈值前不生效。
- top1/top2 gap 与 confidence 枚举是否覆盖 4 种情况。
- 降级路径下分数是否在合法范围，无 NaN / 负数 / 超过 1.5。

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
