# TASK-010 Acceptance - 新增 Embedding MQ Consumer

## 1. 任务目标

新增消费 embedding.upsert 事件的 Consumer，调用 EmbeddingService.EmbedObject 写入 ai_embeddings。

> 该任务涉及公共模块或尚需定位的业务入口，开始实现前必须先获得人工确认。

## 2. 自动化测试检查项

- 执行 `cd logic-grpc-service && go test ./...`，结果必须通过；若项目暂未配置该命令，必须在报告中说明。
- 验证：发布事件后 consumer 生成 ready embedding。
- 验证：Provider 失败进入重试或死信。
- 验证：重复事件不产生不一致数据。
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
- MQ 发布或消费失败不得造成主业务保存失败。
- 重复事件不得造成不一致写入。

## 5. 回归测试范围

- 现有 MQ consumer。
- logic-grpc-service 启停流程。

## 6. 完成后必须输出的报告内容

- TASK-010 是否完成，以及完成摘要。
- 实际修改文件列表。
- 执行过的命令、结果和失败原因。
- 未执行的命令及原因。
- 是否触碰公共模块或越界文件。
- 风险点是否已处理。
- 回滚方式。
- 后续建议执行的 TASK。
