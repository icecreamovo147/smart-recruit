# Fix Check Failures Prompt

你要针对检查失败进行定向修复，不要扩大范围，不要顺手实现其他 TASK。

## 必须先读取

1. `.spec/skill-memory-ranking/skill-memory-ranking-SPEC.md`
2. `.spec/skill-memory-ranking/skill-memory-ranking-SDD.md`
3. `.spec/skill-memory-ranking/TASKS.md`
4. `.spec/skill-memory-ranking/AGENT_RULES.md`
5. `.spec/skill-memory-ranking/task-scope.json`
6. `.spec/skill-memory-ranking/acceptance/<TASK-ID>.md`
7. `.spec/skill-memory-ranking/reports/<TASK-ID>-report.md`

## 输入

- TASK-ID：`<填写 TASK-XXX>`
- 失败命令：`<填写命令>`
- 失败日志摘要：`<粘贴关键错误>`

## 修复规则

- 只修复导致该检查失败的直接原因。
- 只修改当前 TASK 的 `allowedFiles`。
- 如果修复需要改 `forbiddenFiles` 或其他 TASK 文件，停止并说明原因。
- 不允许通过删除测试、弱化断言、吞异常、使用 `any` 绕过类型检查。
- 修复后重新执行失败命令；必要时执行当前 TASK 的完整 `requiredCommands`。
- 权重 / 阈值常量不要分散到其他文件；如有调整需同步更新 SPEC / SDD。
- 旧 `Score int` 字段兼容映射不要破坏。
- proto 字段不要修改 / 删除 / 重用现有字段编号。

## 输出格式

- 失败原因定位。
- 实际修复内容。
- 修改文件。
- 重新执行的命令和结果。
- 是否仍有残留风险。
- 是否需要再次进入 `self-review`。
