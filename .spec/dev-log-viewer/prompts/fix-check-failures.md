# Fix Check Failures Prompt

使用 `spec-harness`。

Mode: `fix-check-failures`

Feature: `dev-log-viewer`

Task: `<TASK-ID>`

读取当前 TASK 的完整 Harness 合同、报告、evidence，以及上一轮失败的 Harness/test/self-review 原文。

只修复已报告失败：

- 不扩展 TASK scope，不添加新功能，不重构无关代码，不修改 SPEC/SDD/TASKS/acceptance。
- 可以把不同 finding 的原因分析或独立测试委托给 subagent，但写入必须由主 Agent 协调，禁止修改相同文件的并发写入。
- 修复需要范围外文件、新依赖、共享配置或新决策时 Hard Stop。
- 同一问题连续两轮修复失败时停止并说明根因。
- 重新运行原失败命令、TASK required checks、scope check 和 agent-check。
- 更新 Markdown report、evidence、review round 和 `pipeline-state.json`；不得把失败或跳过描述为通过。
- 修复后再次交给独立只读 subagent/fresh context 复审，未通过前不得开始下一 TASK。
