---
description: Fix only failed checks for one SPEC + SDD + Harness task
---

你现在执行 fix-check-failures。

功能点：$1
任务编号：$2

只修复上一轮检查失败的问题。

必须读取：

- .spec/$1/TASKS.md
- .spec/$1/AGENT_RULES.md
- .spec/$1/task-scope.json
- .spec/$1/acceptance/$2.md
- .spec/$1/reports/$2-report.md
- .spec/$1/prompts/fix-check-failures.md

限制：

1. 不实现新功能；
2. 不执行后续 TASK；
3. 不扩大修改范围；
4. 不重构无关代码；
5. 不修改 package.json、lockfile、全局配置；
6. 如果必须扩大范围，立即停止并说明原因；
7. 如果同一个问题连续两次修复失败，停止并说明根因。
8. **禁止以最小更改或 MVP 思路修复**。每次修复必须采用适合长期维护的企业级生产标准：
   - 完整消除根因而非仅屏蔽症状；
   - 补充必要的错误处理、日志、可观测性；
   - 代码风格、命名、注释与项目现有最佳实践一致；
   - 如果修复涉及判断逻辑，必须保证边界情况（空值、零值、超时、取消）均被妥善处理；
   - 不得为了"最小 diff"而保留遗留的冗余代码、死分支或潜在的 panic/NPE 路径。

修复完成后：

1. 只重新运行失败的检查命令；
2. 如果无法确定失败命令，则运行：
   `bash .spec/$1/scripts/agent-check.sh`
3. 更新：
   `.spec/$1/reports/$2-report.md`

报告补充：

- 失败原因
- 修复内容
- 修改文件
- 是否仍在 `$2` 范围内
- 重新运行的命令
- 新检查结果
- 是否仍存在问题

完成后停止。