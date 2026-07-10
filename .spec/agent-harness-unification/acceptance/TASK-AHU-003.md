# Acceptance - TASK-AHU-003

## TASK Summary

让 `harness-pipeline` 使用机器 evidence 和状态不变量决定是否可以 completed，并支持经人工批准的 `completed_with_exceptions`。

## SPEC References

- FR-010、FR-011
- AC-009、AC-011、AC-013
- Error Handling Requirements 3、7

## SDD References

- 3.7 Pipeline state 设计
- 3.9 机器证据设计
- 6.3 Pipeline 完成判定
- 11.5 Pipeline invariant tests

## Acceptance Criteria

- [ ] pipeline state 包含 schemaVersion、current_phase、task_runs、blocked_tasks。
- [ ] scope/check/review/confirmation 任一必需条件失败时不能 completed。
- [ ] 缺少 evidence 的 TASK 不能进入 completed_tasks。
- [ ] failed_tasks 或 blocked_tasks 非空时不能 completed。
- [ ] `completed_with_exceptions` 只接受完整人工批准元数据。
- [ ] `skip_human_confirm=true` 不把跳过的必需 TASK 当作完成。
- [ ] 生成 `validate-pipeline-state.mjs` 和 `pipeline-state.test.mjs`。
- [ ] 既有调用参数保持兼容。
- [ ] 历史 pipeline state 不被自动回写。

## Required Checks

```bash
git diff --name-only
bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-003
bash .spec/agent-harness-unification/scripts/agent-check.sh
node .agents/skills/harness-pipeline/scripts/pipeline-state.test.mjs
```

Fixture 必须覆盖：全部通过、scope 失败、check 失败、Review 不通过、缺失确认、缺失 evidence、批准例外和 blocked/failed 非空。

## Manual Verification, if needed

- 核对状态转换没有引入隐式自动批准。
- 核对 summary 只能从 state/evidence 得出完成结论。

## Out-of-Scope

- 修改 `spec-harness`；
- 回写历史 state；
- Claude adapter；
- 业务代码。
