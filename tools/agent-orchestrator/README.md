# Agent Orchestrator

> **历史只读 / 不受支持。** 本工具已于 2026-07-30 原地隔离，不能作为当前 Agent 控制面执行任务。它依赖的 `docs/agent-harness/tasks/` 已不存在，分支与自动提交模型也不符合当前 `AGENTS.md` 的显式激活规则。代码、任务清单和日志仅为历史审计证据。

本工具曾用于串行处理 `docs/agent-harness/tasks/` 下 23 个任务，按 Developer → Review → Fix（可选）→ Merge 推进。当前控制面由根 `AGENTS.md` 和用户明确激活的仓库技能决定；历史工具不能替代或自动启动该流程。

CLI 现在只允许：

```bash
python tools/agent-orchestrator/orchestrate_serial.py --dry-run
```

该命令仅打印历史任务清单并明确说明执行已禁用。任何非 dry-run 参数都会在读取 Agent 命令、切换分支、写日志或创建提交之前返回非零退出码。`--yes`、环境变量或旧分支均不能绕过隔离。

## 目录结构

```
tools/agent-orchestrator/
├── tasks.yaml              # 任务清单（23 个任务）
├── orchestrate_serial.py   # 主脚本
├── README.md               # 本文件
├── prompts/
│   ├── developer.md        # Developer Agent 提示词模板
│   ├── reviewer.md         # Reviewer Agent 提示词模板
│   └── fixer.md            # Fixer Agent 提示词模板
└── logs/
    ├── .gitkeep
    ├── status.jsonl        # 每次运行自动生成，记录每个任务的结果
    └── <task-id>-*.log     # 每个任务的 Developer/Reviewer/Fixer 输出
```

## 历史使用前提（不再构成运行授权）

1. **Python 3.10+**，已安装 PyYAML：
   ```bash
   pip install pyyaml
   ```

2. **Agent 命令可用**。脚本通过环境变量 `DEV_AGENT_CMD` / `REVIEW_AGENT_CMD` / `FIX_AGENT_CMD` 调用外部 Agent。

   默认值为 `claude -p`（将 prompt 通过 stdin 管道传入）。如果你的环境中 `claude -p` 不支持管道输入，请改用其他命令。

   例如通过文件方式调用：
   ```bash
   export DEV_AGENT_CMD="claude --print < /path/to/prompt.txt"
   ```

   或使用自定义脚本包装：
   ```bash
   export DEV_AGENT_CMD="python scripts/call_claude_agent.py"
   ```

   **注意**：Agent 命令必须能从 stdin 读取 prompt 文本，并将输出写到 stdout。退出码为 0 表示成功，非 0 会被记录到日志。

3. **历史实现要求**位于 `integration/agent-platform` 分支且工作区干净；当前工具在检查或切换该分支前即拒绝实际执行。

4. 每个历史 `task_file` 指向的 `.md` 文件原本必须存在；这些路径当前已缺失，因此任务清单只可用于审计。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DEV_AGENT_CMD` | `claude -p` | Developer Agent 命令 |
| `REVIEW_AGENT_CMD` | `claude -p` | Reviewer Agent 命令 |
| `FIX_AGENT_CMD` | `claude -p` | Fixer Agent 命令 |

Agent 命令调用方式：
```bash
echo "<prompt text>" | <agent_cmd>
```

如果你的 Agent 不支持管道输入，可以在命令中包装，例如：
```bash
export DEV_AGENT_CMD="claude -p --input-file /dev/stdin"
```

## 历史命令记录（当前禁止执行）

### Dry-Run（唯一保留的只读操作）

```bash
cd tools/agent-orchestrator
python orchestrate_serial.py --dry-run
```

输出历史任务列表、分支名和 commit message；不会把清单描述为可执行计划。

### 旧执行参数（全部 fail closed）

以下命令仅保留为历史接口说明，当前都会返回“工具已归档”的错误，不会切换分支、调用 Agent、写日志或创建提交。

#### 只执行一个任务

```bash
python orchestrate_serial.py --only P0-002
```

#### 从指定任务开始执行后续

```bash
python orchestrate_serial.py --from-task P0-002
```

从 P0-002 开始，依次执行到最后一个任务。

#### 限制一次最多执行几个

```bash
python orchestrate_serial.py --max-tasks 3
```

执行前 3 个 enabled 任务。

可与 `--from-task` 组合：
```bash
python orchestrate_serial.py --from-task P0-005 --max-tasks 3
```

从 P0-005 开始，最多执行 3 个任务。

#### 不自动合并

```bash
python orchestrate_serial.py --no-merge
```

Review PASS 后任务分支保留在原地，不 squash-merge 到 `integration/agent-platform`，方便手动检查后再合并。

#### 跳过人工确认

```bash
python orchestrate_serial.py --yes
```

不弹出 `Proceed? [y/N]` 提示，直接开始执行。

#### 失败后继续

```bash
python orchestrate_serial.py --continue-on-failure
```

某个任务 BLOCKED 或失败后不停止，继续执行后续任务。

#### 组合使用

```bash
python orchestrate_serial.py \
  --from-task P0-002 \
  --max-tasks 5 \
  --no-merge \
  --yes \
  --continue-on-failure
```

## 历史执行流程（当前已禁用）

```
1. git checkout integration/agent-platform
2. git checkout -b agent/<task-id>-<name>
3. 检查 task_file 是否存在
4. 运行 Developer Agent（prompt → stdin, output → logs/<id>-developer.log）
5. 检查是否有领先 integration 的提交（无提交 → BLOCKED）
6. 运行 Reviewer Agent（output → logs/<id>-review-1.log）
7. 解析 REVIEW_VERDICT:
   - PASS        → squash-merge → 删除任务分支 → 下一个任务
   - NEEDS_FIX   → 运行 Fixer Agent → 重新 Review（最多 max_fix_rounds 轮）
   - BLOCKED     → 停止（或继续，取决于 --continue-on-failure）
8. 写入 logs/status.jsonl
```

## 查看日志

所有日志文件在 `tools/agent-orchestrator/logs/`：

```bash
# 查看任务状态摘要
cat tools/agent-orchestrator/logs/status.jsonl | python -m json.tool

# 查看某个 Developer 的输出
cat tools/agent-orchestrator/logs/P0-002-developer.log

# 查看某个任务的第一次 Review
cat tools/agent-orchestrator/logs/P0-002-review-1.log

# 查看 Fix 输出
cat tools/agent-orchestrator/logs/P0-002-fix-1.log
```

## 历史 BLOCKED 处理记录

任务被 BLOCKED 时，脚本会保留任务分支（不会删除）。处理步骤：

1. **查看日志确认原因**：
   ```bash
   cat tools/agent-orchestrator/logs/<task_id>-review-*.log
   ```

2. **切换到任务分支手动处理**：
   ```bash
   git checkout agent/<task_id>-<name>
   ```

3. **修复后手动合并**：
   ```bash
   git checkout integration/agent-platform
   git merge --squash agent/<task_id>-<name>
   git commit -m "<commit_message>"
   git branch -D agent/<task_id>-<name>
   ```

4. **从下一个任务继续**：
   ```bash
   python orchestrate_serial.py --from-task <next_task_id>
   ```

## 安全规则

- **任何非 `--dry-run` 调用都会在发生 Git 或 Agent 副作用之前拒绝**
- **不会自动 push**
- **不会合并 main**
- **不会处理远程分支**
- **不会自动解决 merge conflict** — 冲突发生时标记 BLOCKED 并停止
- **不会修改 `docs/agent-harness/`**、`CLAUDE.md`、`.claude/`、业务代码目录
