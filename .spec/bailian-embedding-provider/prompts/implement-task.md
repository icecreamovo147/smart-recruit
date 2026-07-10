# Implement Single Task Prompt

你要执行 `.spec/bailian-embedding-provider/` 下的单个 TASK。不要同时执行多个 TASK。

## 必须先读取

1. `.spec/bailian-embedding-provider/bailian-embedding-provider-SPEC.md`
2. `.spec/bailian-embedding-provider/bailian-embedding-provider-SDD.md`
3. `.spec/bailian-embedding-provider/TASKS.md`
4. `.spec/bailian-embedding-provider/AGENT_RULES.md`
5. `.spec/bailian-embedding-provider/task-scope.json`
6. `.spec/bailian-embedding-provider/acceptance/<TASK-ID>.md`

## 执行输入

- TASK-ID：`<填写 TASK-XXX>`

## 执行规则

- 只允许修改 task-scope.json 中该 TASK 的 allowedFiles。
- 不允许修改 forbiddenFiles。
- 如果该 TASK 的 `requiresHumanConfirmation=true`，开始改代码前必须停止并向用户确认。
- 如果发现必须修改范围外文件，停止并说明原因，不要擅自修改。
- 不允许修改依赖文件、全局配置、公共 API 行为，除非当前 TASK 明确允许。
- 不允许删除已有测试，不允许用 `any` 绕过类型问题，不允许吞异常。
- 后端代码必须遵循 gofmt；前端代码必须通过 typecheck。

## 完成要求

1. 实现当前 TASK。
2. 执行 acceptance 中要求的自测命令。
3. 运行 `.spec/bailian-embedding-provider/scripts/check-task-scope.sh` 并核对越界风险。
4. 输出完成报告：
   - TASK 编号和名称；
   - 修改文件；
   - 自测命令和结果；
   - 未执行命令及原因；
   - 是否越界；
   - 是否触碰公共模块；
   - 风险和回滚方式；
   - 下一步建议。
