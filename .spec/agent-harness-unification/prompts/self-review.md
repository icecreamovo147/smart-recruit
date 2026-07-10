# Prompt - self-review

Use `spec-harness`.

```text
Mode: self-review
Feature: agent-harness-unification
Task: <TASK-ID>
```

Review 前完整读取：

1. SPEC、SDD、TASKS、AGENT_RULES
2. `task-scope.json`
3. 对应 acceptance
4. TASK report 和 evidence
5. 当前 TASK 基线到工作区的实际 git diff

Review 必须只读，不得修改任何文件。不要信任实现报告中的完成声明，必须对照实际文件、命令退出码和 evidence。

重点检查：

- 是否修改了 scope 外或 forbidden 文件；
- 是否修改 SPEC/SDD、业务、依赖、历史或本地权限；
- schema、状态和 evidence 是否可验证且一致；
- 是否保留兼容入口而没有复制第二套规则；
- Legacy 冻结是否被误写成取消；
- scope/check/review/confirmation 失败时是否仍可能 completed；
- 测试和静态检查是否真实执行；
- 是否存在绝对用户路径、权限扩大或敏感信息。

如果平台支持独立 Agent 或新上下文，优先独立 Review；否则在输出中声明 `reviewer_type: self-review`。

输出 findings，按 Critical、High、Medium、Low 排序，并使用规定表格。最后只能输出以下之一：

```text
verdict: 通过
```

或：

```text
verdict: 不通过
```

不要在本模式修复问题。
