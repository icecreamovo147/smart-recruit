# Fix Check Failures Prompt - microservice-runtime-implementation

使用 `spec-harness`。

Mode: `fix-check-failures`

Feature: `microservice-runtime-implementation`

Task: `<TASK-ID>`

只修复当前 TASK 的失败项：

- 不扩大 scope。
- 不新增无关功能。
- 不跳过失败检查。
- 修复后重新运行失败检查和必需 Harness 检查。
- 更新 TASK report。
