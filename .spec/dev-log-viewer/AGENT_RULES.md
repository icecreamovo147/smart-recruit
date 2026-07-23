# Dev Log Viewer Agent Rules

## Mode And Authority

本功能由 `spec-harness` 管理。权威顺序为：根 `AGENTS.md`、`.agents/skills/spec-harness/SKILL.md`、本功能 SPEC/SDD、`TASKS.md`、`task-scope.json`、当前 TASK acceptance、运行证据。

不得在未指定 TASK 时实施代码，不得一次实施多个 TASK。`pipeline-state.json` 是运行状态源，`task-scope.json` 的 `status` 只是生成时初始值。

## Required Reading

每个非平凡 TASK 开始前必须读取：

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/dev-log-viewer/dev-log-viewer-SPEC.md`
- `.spec/dev-log-viewer/dev-log-viewer-SDD.md`
- `.spec/dev-log-viewer/TASKS.md`
- `.spec/dev-log-viewer/AGENT_RULES.md`
- `.spec/dev-log-viewer/task-scope.json`
- `.spec/dev-log-viewer/acceptance/<TASK-ID>.md`
- `.knowledge/README.md`、`.knowledge/manifest.yaml`、`.knowledge/INDEX.md` 及 TASK 路由的 active 文档

## Implementation Rules

- 只执行当前 TASK，只修改其 `allowedFiles`。
- 不修改 SPEC、SDD、TASKS 或 acceptance；发现缺口时 Hard Stop。
- 不修改业务服务、Gateway、Proto、数据库、三个 Vue 前端、部署或 CI。
- 新应用根目录固定为 `dev-log-viewer/`，不得迁回 `packages/` 或接入业务 Gateway。
- 服务目录只允许 SPEC 指定的 12 个 ID、PID 和日志文件，不接受任意路径。
- HTTP 只监听 loopback；日志正文始终作为不可信文本处理。
- 状态文案只表达 process/log/stream state，不把 PID 存活称为健康。
- 文件、事件环、客户端队列和浏览器记录都必须有界；禁止静默丢弃。
- 不加载 CDN、Google Fonts 或其他第三方 UI 资源。
- 不新增 SPEC 第 13 节之外的依赖；若必须新增则 Hard Stop。
- `start-dev.sh all` 和无参数行为不得改变。
- 不记录被查看日志正文、关联 ID 值、环境变量值或用户绝对路径到查看器自身日志。

## Subagent And Goal Mode Rules

- 普通 Goal mode 只持续推进一个指定 TASK 的 implement → independent self-review → fix-check-failures（如需）→ evidence；完成后等待用户确认。只有显式 `harness-pipeline` 可串行推进多个 TASK。
- 优先用 subagent 隔离独立调研、测试设计、失败诊断、知识影响分析和只读 review。
- 主 Agent 是唯一 TASK 状态所有者，负责建立 baseline、批准写入分工、合并结果、运行 scope check 和更新 evidence。
- 禁止多个写入型 Agent 同时修改相同文件或相互覆盖工作树。
- review subagent 不得修改文件；修复必须切换到 `fix-check-failures`。
- subagent 不得创建自己的 TASK 来源、状态文件或替代验收标准。

## Knowledge Rules

- 每个 TASK 设置 `requiredKnowledgeImpact: true`，报告固定格式的 knowledge impact。
- 允许直接修改知识的范围仅以当前 TASK 的 `knowledge.modify` 和 `allowedFiles` 交集为准。
- TASK-DLV-008 可依据完成代码更新 manifest、system-overview 和 local-development。
- L2 描述更新必须有代码/测试证据和通过的独立复审；L3 决策仍需人类确认或进入 inbox/ADR。
- 不读取或修改 inbox、archive、stale、deprecated 内容，除非当前 TASK 明确允许。
- 禁止写入真实日志、凭据、个人数据或 Agent 内部推理。

## Required Checks

每个 TASK 至少运行：

```bash
git diff --name-only
bash .spec/dev-log-viewer/scripts/check-task-scope.sh <TASK-ID>
bash .spec/dev-log-viewer/scripts/agent-check.sh <TASK-ID>
```

并运行 acceptance 列出的 TASK 专用检查。无法运行时必须在报告和 evidence 中记录真实原因，不得标记通过。

## Report And Evidence

每个 TASK 必须生成：

- `.spec/dev-log-viewer/reports/<TASK-ID>-report.md`
- `.spec/dev-log-viewer/reports/<TASK-ID>-evidence.json`

Markdown 报告必须包含 TASK ID、修改文件、逐文件摘要、scope、SPEC/SDD/acceptance 对比、真实命令结果、knowledge impact、风险、后续项和下一 TASK 是否可开始。Evidence 必须满足 spec-harness schema，并与报告一致。

## Compatibility Rules

- 现有 12 个服务、Gateway 和三个 Vue 前端行为保持不变。
- 不改变现有日志格式、Proto、数据库或业务 API。
- 查看器退出不得影响任何业务进程。
- 旧日志文件不存在、不可读、截断或替换只影响对应服务。
- 无前端构建产物时必须给出明确构建错误，不得嵌入陈旧或远程资源。

## Hard Stops

出现以下任一条件立即停止并请求用户处理：

- Harness 不是 `current`，TASK ID 或 acceptance 不一致，或缺少可靠 TASK base tree。
- 工作树包含无法与当前 TASK 区分的既有未提交改动。
- 所需文件不在当前 `allowedFiles`，或命中 `forbiddenFiles`。
- 需要新增未批准依赖、改变公网监听、开放任意文件路径或修改业务公共契约。
- 需要改变默认/`all` 启动语义、SPEC/SDD 或设计源事实。
- 知识文档与高优先级来源冲突，或 L3 决策缺少确认。
- 同一检查连续两轮修复仍失败。
