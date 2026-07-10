# Acceptance - TASK-FU-002

## TASK Summary

在 `.github/workflows/ci.yml` 新增 `proto-lint` job，protoc 25.x 重生成两侧 `recruitment.pb.go` 并与 committed diff 必须为空。

## SPEC References

- SPEC §5 FR-FU-002：CI 流水线 protoc 步骤
- SPEC §7 兼容性
- SPEC §11 AC-FU-002

## SDD References

- SDD §3.2 CI protoc 步骤
- SDD §11 测试策略

## Acceptance Criteria

- [ ] AC-001：`.github/workflows/ci.yml` 包含 `proto-lint` job
- [ ] AC-002：protoc 25.x 显式安装（wget / unzip）
- [ ] AC-003：protoc-gen-go 安装（go install + cache）
- [ ] AC-004：重生成两侧 pb 步骤与本地 TASK-005 完全一致
- [ ] AC-005：`git diff --exit-code` 对两侧 pb 必须为空；失败时输出 diff 行号
- [ ] AC-006：不修改现有 3 个 job（go-test / frontend / secret-scan）

## Required Checks

- [ ] `bash -n .github/workflows/ci.yml` 语法正确
- [ ] `cat .github/workflows/ci.yml | grep -c "jobs:"` = 4（go-test / frontend / secret-scan / proto-lint）
- [ ] `grep -c "protoc-25" .github/workflows/ci.yml` >= 1
- [ ] `grep -c "protoc-gen-go" .github/workflows/ci.yml` >= 1
- [ ] `grep -c "diff --exit-code" .github/workflows/ci.yml` >= 1

## Manual Verification, if needed

- 阅读 `proto-lint` job 步骤，确认 protoc 25.x 与本地一致。
- 在干净 working tree 下手动执行 protoc 步骤（如果环境支持），确认 diff 为空。
- 验证 CI 配置不会阻塞其他 job（不显式 `needs`）。

## Out-of-Scope

- 不得修改 `proto/**` / `pb/**`。
- 不得修改其他 workflows。
- 不得新增第三方 action（仅 `actions/checkout` / `actions/setup-go` / `wget` / `unzip`）。
