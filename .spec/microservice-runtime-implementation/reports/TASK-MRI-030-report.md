# TASK-MRI-030 Report

## TASK ID

TASK-MRI-030

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-030-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-030-report.md`
- `.spec/microservice-runtime-implementation/scripts/agent-check.sh`
- `scripts/export-microservice-repos.mjs`
- `smart-recruit-deploy/repo-export-manifest.json`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-030` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-031`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-030-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-030-report.md`: 记录本 TASK 的人工可读验收报告。
- `.spec/microservice-runtime-implementation/scripts/agent-check.sh`: 接入 export manifest check。
- `scripts/export-microservice-repos.mjs`: 新增独立 repo export 脚本，默认 dry-run，显式 `--execute --output-dir` 才会创建本地 Git repo。
- `smart-recruit-deploy/repo-export-manifest.json`: 记录 11 个 smart-recruit 源码根的 repo 名、源目录、默认分支和构建命令。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=b0d539fc2d203739dd3461ee8097c6c0e4092b11 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-030
```

所有变更均匹配 `smart-recruit-deploy/**`、`scripts/**` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。Export 脚本可将每个 smart-recruit 源码根导出为本地独立 Git repo；不会创建远程 repo。

## SDD 对照结果

通过。Repo extraction manifest 明确每个源码根的构建命令和默认分支，实际导出通过 `/tmp` 验证。

## Acceptance 对照结果

通过。Manifest 覆盖 11 个源码根；dry-run 可运行；execute 在 `/tmp/smart-recruit-export-test` 创建 11 个本地 Git repo，默认分支均为 `main`。

## 测试命令和结果

- `node scripts/export-microservice-repos.mjs --check`: passed
- `node scripts/export-microservice-repos.mjs --dry-run --output-dir /tmp/smart-recruit-export-preview`: passed
- `node scripts/export-microservice-repos.mjs --execute --output-dir /tmp/smart-recruit-export-test`: passed
- `verify /tmp/smart-recruit-export-test contains 11 .git repositories on main branch`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree b0d539fc2d203739dd3461ee8097c6c0e4092b11 --json`: passed
- `TASK_BASE_TREE=b0d539fc2d203739dd3461ee8097c6c0e4092b11 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-030`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-030-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增 export 脚本、manifest、dry-run、execute 验证和 agent-check 门禁。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP routes、handlers、protobuf 或生成代码。
- 是否提交 secrets 或真实 `.env`：通过，脚本排除 `.env`，未创建或修改真实 `.env`。
- 是否违反单 MySQL 约束：通过，本 TASK 仅涉及 repo export 工具，未修改数据库。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，export check/dry-run/execute、scope check、agent-check 和 evidence validation 均已通过。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/system-overview.md` 和 `.knowledge/runbooks/local-development.md`，结论均为 `UNCHANGED`；本 TASK 新增 extraction tooling，不改变运行时或开发启动流程。

## 风险

- Exported repos rely on sibling checkout layout for local `replace ../...` paths until dependency publishing strategy is finalized.

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-031。
