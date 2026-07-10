# Acceptance - TASK-AHU-002

## TASK Summary

实现共享只读 preflight/audit、确定性 TASK scope checker 和 evidence validator，不改写任何被扫描 feature。

## SPEC References

- FR-007、FR-008、FR-010、FR-013
- AC-006、AC-007、AC-008、AC-011、AC-013
- Non-Functional Requirements 6.3、6.4、6.5

## SDD References

- 3.6 Canonical Harness schema
- 3.8 TASK 基线与 scope 检查
- 3.9 机器证据设计
- 3.10 共享 validator 与脚本生成
- 11.2 至 11.4 Testing Strategy

## Acceptance Criteria

- [ ] current、legacy-compatible、unsupported 三类 feature 均有确定结果。
- [ ] unsupported 和缺失 Harness 返回非零退出码并列出原因。
- [ ] unknown TASK、缺失 acceptance、无效 schema/pattern 失败关闭。
- [ ] scope checker 覆盖 committed、staged、unstaged、untracked。
- [ ] scope checker 同时执行 allowed 和 forbidden 判断。
- [ ] TASK 基线不会把前序已完成 TASK 误算为当前 TASK。
- [ ] evidence validator 检查命令退出码、scope、review、confirmation 和异常。
- [ ] 全仓 audit 只读，不修改任何 `.spec` feature。
- [ ] 生成 `validate-feature.mjs`、`check-task-scope.mjs`、`validate-evidence.mjs`、`audit-specs.mjs` 和 `validator.test.mjs`。
- [ ] 不引入新依赖，不修改 shared Skill 文档或业务代码。

## Required Checks

```bash
git diff --name-only
bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-002
bash .spec/agent-harness-unification/scripts/agent-check.sh
node .agents/skills/spec-harness/scripts/validator.test.mjs
node .agents/skills/spec-harness/scripts/audit-specs.mjs --root .spec
```

还必须在临时 fixture 中验证：allowed、forbidden、out-of-scope、staged、unstaged、untracked、前序 TASK 基线、unknown TASK、legacy-compatible 和 unsupported。

## Manual Verification, if needed

- 检查 audit 前后 `git status --short` 一致。
- 检查错误输出足以定位 feature、TASK、文件和 pattern。

## Out-of-Scope

- 修改任何历史 `.spec` 内容；
- pipeline 最终状态迁移；
- Claude adapter；
- 业务测试。
