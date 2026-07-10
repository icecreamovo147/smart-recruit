# Bailian Embedding Provider Agent Rules

## 1. 基本执行规则

1. 每次只能执行一个 TASK，不允许在同一轮合并多个 TASK。
2. 执行 TASK 前必须读取：
   - `.spec/bailian-embedding-provider/bailian-embedding-provider-SPEC.md`
   - `.spec/bailian-embedding-provider/bailian-embedding-provider-SDD.md`
   - `.spec/bailian-embedding-provider/TASKS.md`
   - `.spec/bailian-embedding-provider/AGENT_RULES.md`
   - `.spec/bailian-embedding-provider/task-scope.json`
   - `.spec/bailian-embedding-provider/acceptance/<TASK-ID>.md`
3. 不允许修改任务范围之外的文件。
4. 不允许修改 `package.json`、`pnpm-lock.yaml`、`pnpm-workspace.yaml`、全局配置文件，除非当前 TASK 明确允许。
5. 不允许大规模格式化无关文件。
6. 不允许为了通过类型检查随意使用 `any`。
7. 不允许吞掉异常；必须保留错误语义、日志和可观测性。
8. 不允许删除已有测试或弱化已有断言。
9. 不允许改变公共 API 行为；新增字段必须保持向后兼容。
10. 如果必须修改公共模块，必须先停止并说明原因，等待人工确认。
11. 每个 TASK 完成后必须输出完成报告。

## 2. 安全规则

- 不得提交真实 API Key、Bearer Token、Cookie 或本地凭证。
- 日志和测试 fixture 不得包含完整简历、完整 Memory 内容、完整 SKILL.md 或真实敏感文本。
- API Key 只能加密存储，不得通过前端响应返回明文。
- 前端不得使用 `v-html` 展示调试内容。

## 3. 代码质量规则

- Go 代码必须 `gofmt`。
- Vue/TypeScript 改动必须通过 `pnpm --filter hr-frontend typecheck`。
- 后端改动必须在对应服务目录运行 `go test ./...`。
- Provider、Factory、MQ、Backfill 逻辑必须补充针对失败路径的测试。
- 不能把业务事件发布放入 Repository 层。

## 4. 完成报告格式

每个 TASK 完成后必须输出：

- TASK 编号和名称；
- 实际修改文件；
- 未修改但检查过的关键文件；
- 已执行的自测命令和结果；
- 未执行命令及原因；
- 是否触碰公共模块；
- 是否存在越界修改；
- 风险、回滚方式和后续任务建议。
