# Self Review Prompt

你要对刚完成的单个 TASK 做自查，不要实现新功能。

## 必须先读取

1. `.spec/bailian-embedding-provider/bailian-embedding-provider-SPEC.md`
2. `.spec/bailian-embedding-provider/bailian-embedding-provider-SDD.md`
3. `.spec/bailian-embedding-provider/TASKS.md`
4. `.spec/bailian-embedding-provider/AGENT_RULES.md`
5. `.spec/bailian-embedding-provider/task-scope.json`
6. `.spec/bailian-embedding-provider/acceptance/<TASK-ID>.md`

## 自查内容

- git diff 是否只包含当前 TASK 范围内文件。
- 是否修改了 forbiddenFiles。
- 是否改变公共 API 行为或公共模块；如果是，是否已人工确认。
- 是否存在敏感信息泄漏。
- 是否存在吞异常、随意使用 `any`、删除测试、弱化断言。
- 是否满足 acceptance 的自动化、手工、异常、回归检查项。
- 是否执行了 requiredCommands；未执行必须说明原因。

## 输出格式

- 结论：通过 / 不通过。
- 发现的问题，按严重程度排序。
- 必须修复的问题。
- 可后续优化的问题。
- 已执行命令与结果。
- 是否建议进入下一个 TASK。
