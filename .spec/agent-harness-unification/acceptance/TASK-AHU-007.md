# Acceptance - TASK-AHU-007

## TASK Summary

只读审计全部控制面并生成最终报告、机器证据和 pipeline summary 草案；发现共享问题时阻塞并返回对应 TASK。

## SPEC References

- AC-001 至 AC-013
- FR-013
- Observability and Debug Requirements
- Compatibility Requirements

## SDD References

- 10 Observability and Debug Output Design
- 11 Testing Strategy
- 13 Implementation Boundaries

## Acceptance Criteria

- [ ] 常规和 `FINAL_AUDIT=1` agent-check 均通过。
- [ ] AC-001 至 AC-013 有逐项证据。
- [ ] 所有可执行入口只以 `.agents/.spec` 为权威。
- [ ] Claude adapters 不含旧固定 branch、旧任务状态、第三套 verdict 或绝对路径。
- [ ] Legacy inventory 与实际 pending/phase/unsupported feature 对齐。
- [ ] 全仓 `.spec` audit 只读且分类结果可解释。
- [ ] pipeline invariant fixtures 全部通过。
- [ ] 最终 diff 没有业务、依赖、部署、CI 或未允许历史文件。
- [ ] 发现共享问题时 verdict 不通过，不在本 TASK 修改共享文件。

## Required Checks

```bash
git diff --name-only
bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-007
bash .spec/agent-harness-unification/scripts/agent-check.sh
FINAL_AUDIT=1 bash .spec/agent-harness-unification/scripts/agent-check.sh
```

同时运行 TASK-AHU-002/003 提供的共享 audit、scope/evidence fixture 和 pipeline invariant tests。

## Manual Verification, if needed

- 抽查 Markdown report 与 evidence JSON 是否一致。
- 抽查 `completed_with_exceptions` 样例是否包含完整批准元数据。
- 核对最终 summary 没有把阻塞项描述为完成。

## Out-of-Scope

- 修改任何 shared control-plane 文件；
- 修复前序 TASK 的问题；
- 修改业务或历史资料；
- commit、push、merge 或 PR。
