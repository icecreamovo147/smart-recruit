# Self Review Prompt - microservice-runtime-implementation

使用 `spec-harness`。

Mode: `self-review`

Feature: `microservice-runtime-implementation`

Task: `<TASK-ID>`

只审查，不修改文件。重点检查：

- 是否真实落地，而不是只写文档。
- 是否越过 TASK scope。
- 是否破坏 HTTP/protobuf 兼容。
- 是否提交 secrets 或真实 `.env`。
- 是否违反单 MySQL 约束。
- 是否缺少测试、scope check、agent-check 或 evidence validation。

结尾必须使用：

```text
verdict: 通过
```

或：

```text
verdict: 不通过
```
