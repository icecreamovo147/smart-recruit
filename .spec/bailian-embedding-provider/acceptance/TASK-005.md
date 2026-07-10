# TASK-005 Acceptance - 实现 Embedding Provider Factory

## 1. 任务目标

根据默认 embedding model 和 provider 配置构造真实 Provider 或降级 Provider。

## 2. 自动化测试检查项

- 执行 `cd logic-grpc-service && go test ./...`，结果必须通过；若项目暂未配置该命令，必须在报告中说明。
- 验证：配置缺失返回 unavailable。
- 验证：配置正确返回 Bailian provider。
- 验证：解密失败不 panic，不泄漏密钥。
- 验证：go test 通过。

## 3. 手工验收检查项

- 对照 TASKS.md 确认修改文件未超出允许范围。
- 对照 task-scope.json 检查 git diff 中的文件是否越界。
- 确认实现符合 SPEC/SDD，不引入非目标范围能力。
- 确认日志、错误返回和测试数据没有泄漏 API Key 或完整敏感文本。

## 4. 异常场景检查项

- Provider 未配置或不可用时，不应阻塞主业务流程。
- 外部接口失败时，错误必须可观测且脱敏。
- 输入为空、配置缺失、权限不足或数据不存在时，应返回明确错误或进入降级状态。
- 401/403 不应重试。
- 429、5xx、timeout 应按策略重试或降级。

## 5. 回归测试范围

- EmbeddingService 初始化前置依赖。

## 6. 完成后必须输出的报告内容

- TASK-005 是否完成，以及完成摘要。
- 实际修改文件列表。
- 执行过的命令、结果和失败原因。
- 未执行的命令及原因。
- 是否触碰公共模块或越界文件。
- 风险点是否已处理。
- 回滚方式。
- 后续建议执行的 TASK。
