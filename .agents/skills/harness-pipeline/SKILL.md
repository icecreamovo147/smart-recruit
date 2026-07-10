---
name: harness-pipeline
description: 自动串行执行某个功能点下所有 TASK：对每个 TASK 按 implement-task → self-review → fix-check-failures 循环驱动，直至全部完成或人工打断。依赖 spec-harness 的四模式定义。
---

# harness-pipeline

串行编排器。对 `.spec/<feature-name>/` 下已分解好的 TASK 列表，逐个执行：

```
for each TASK:
  loop (max_review_rounds):
    implement-task   → 实现
    self-review      → 审查
    if verdict == 通过: break
    fix-check-failures → 修复
```

人仅在以下时机介入：
- 所有 TASK 完成后审合入包
- 某 TASK 被打上 `requiresHumanConfirmation` 标记时停等确认
- 连续 `max_review_rounds` 轮审查未通过时硬暂停

## 输入

由调用方在 prompt 中提供：

| 参数 | 必填 | 默认 | 说明 |
|------|:----:|------|------|
| `feature_name` | ✅ | — | `.spec/` 下的功能目录名 |
| `start_task` | — | 第一个未完成的 TASK | 格式 `TASK-001`，用于续跑 |
| `max_review_rounds` | — | `3` | 单个 TASK 最多修复轮数 |
| `skip_human_confirm` | — | `false` | 设为 `true` 则遇见 `requiresHumanConfirmation` 不停等，直接拒绝该 TASK 并跳过 |

## 前置条件

执行前必须满足：
1. `.spec/<feature_name>/` 下已存在 SPEC.md、SDD.md、TASKS.md、task-scope.json、AGENT_RULES.md、acceptance/、prompts/、scripts/
2. `scripts/check-task-scope.sh` 和 `scripts/agent-check.sh` 已就绪
3. `git status` 干净（无与当前功能无关的未提交改动）

## 状态管理

状态文件：`.spec/<feature_name>/pipeline-state.json`

```json
{
  "feature_name": "...",
  "current_task": "TASK-003",
  "completed_tasks": ["TASK-001", "TASK-002"],
  "failed_tasks": [],
  "review_round": 2,
  "status": "in_progress"
}
```

- 每个 TASK 实现完成后立即更新状态文件
- 支持中断续跑：读取 state → 从 `current_task` 继续

## 执行流程

### Step 0：初始化

1. 读入以下文件（与 spec-harness implement-task 相同）：
   - `.spec/<feature_name>/<feature_name>-SPEC.md`
   - `.spec/<feature_name>/<feature_name>-SDD.md`
   - `.spec/<feature_name>/TASKS.md`
   - `.spec/<feature_name>/AGENT_RULES.md`
   - `.spec/<feature_name>/task-scope.json`

2. 从 `TASKS.md` 提取所有 TASK-ID 列表（按出现顺序）
3. 如果 `start_task` 指定了值，定位到该 TASK；否则从第一个开始
4. 运行 `git status --short`，确认无无关改动
5. 如果 `pipeline-state.json` 已存在，恢复状态并确认续跑

### Step 1：TASK 循环

对每个未完成的 TASK（从起始 TASK 到最后一个）：

#### Step 1.1：人工确认检查

如果 `task-scope.json` 中该 TASK 的 `requiresHumanConfirmation` 为 `true`：

- 如果 `skip_human_confirm` 为 `true`：拒绝执行，将该 TASK 标记为 `failed`（原因：requiresHumanConfirmation），跳过
- 否则：**停止并向用户发送确认请求**，等待用户回复后再继续

#### Step 1.2：implement-task

按照 spec-harness SKILL.md 中 `implement-task` 模式执行当前 TASK：

1. 读取 `.spec/<feature_name>/acceptance/<TASK-ID>.md`
2. 读取 `.spec/<feature_name>/prompts/implement-task.md`
3. 仅修改 `task-scope.json` 中该 TASK 的 `allowedFiles`
4. 完成后运行：
   - `git diff --name-only`
   - `bash .spec/<feature_name>/scripts/check-task-scope.sh <TASK-ID>`
   - `bash .spec/<feature_name>/scripts/agent-check.sh`
5. 生成 `.spec/<feature_name>/reports/<TASK-ID>-report.md`

#### Step 1.3：self-review

按照 spec-harness SKILL.md 中 `self-review` 模式审查当前 TASK：

1. 对照 SPEC、SDD、TASKS、task-scope.json、acceptance、AGENT_RULES.md
2. 检查：越界、遗漏、兼容性、安全、异常处理
3. 输出 verdict：**通过** 或 **不通过 + 问题列表**

如果 verdict 为 **通过**：
- 更新 pipeline-state：当前 TASK 加入 `completed_tasks`，`review_round` 重置为 0
- 进入下一个 TASK

如果 verdict 为 **不通过**：
- `review_round += 1`
- 如果 `review_round > max_review_rounds`：**硬暂停**，将该 TASK 标记为 `failed`，输出失败报告，停止整个 pipeline
- 否则进入 Step 1.4

#### Step 1.4：fix-check-failures

按照 spec-harness SKILL.md 中 `fix-check-failures` 模式修复：

1. 仅修复 self-review 列出的问题
2. 不实现新功能，不扩大范围
3. 修复后重新运行失败的检查
4. 如果连续两次修复同一问题失败：停止并说明根因
5. 回到 Step 1.3（self-review）

### Step 2：全部完成

所有 TASK 完成后：

1. 更新 pipeline-state：`status` = `completed`
2. 输出最终汇总报告：`.spec/<feature_name>/reports/pipeline-summary.md`

### Step 3：异常与人工门禁

- **Hard Stop 条件**（与 spec-harness 一致）：
  - 需要修改 `package.json` 或 lockfile
  - 需要修改公共模块/共享类型/全局配置
  - 需要修改公共 API 行为
  - 发现 SPEC/SDD/TASKS 冲突
- **人工中断**：运行中用户可随时停止，pipeline-state 保留，后续可按续跑方式继续

## 最终汇总报告格式

```markdown
# Pipeline Summary - <feature_name>

## 执行概况
- 起始 TASK: <TASK-ID>
- 结束 TASK: <TASK-ID>
- 完成: <N> / <M>
- 失败: <列表>

## 各 TASK 结果
| TASK | 状态 | 审查轮数 | 报告 |
|------|------|----------|------|
| TASK-001 | ✅ | 2 | link |
| TASK-002 | ✅ | 1 | link |
| TASK-003 | ❌ | 4 (超限) | link |

## 总修改文件
<git diff --name-only 汇总>

## 待确认项
- 需要人工合并的 TASK（requiresHumanConfirmation=true 已跳过）
- 失败 TASK 根因
- 建议修复方式
```

## 与 spec-harness 的关系

`harness-pipeline` **不重新定义** implement-task / self-review / fix-check-failures 的具体行为，而是直接引用 `spec-harness` SKILL.md 中各模式的规则和约束。执行每个 TASK 时，应将 `spec-harness` SKILL.md 加载到上下文中，按该 Skill 的对应模式执行。

## 续跑

用户说「继续 pipeline」「续跑」或检测到 `pipeline-state.json` 存在时：

1. 读 `pipeline-state.json`
2. 从 `current_task` 的当前阶段继续（如果是 review 阶段则从 review 继续）
3. 如果 `status` 为 `completed`，直接输出汇总报告
