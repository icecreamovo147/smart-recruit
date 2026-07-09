# Implement Single Task Prompt

你要执行 `.spec/skill-memory-ranking/` 下的单个 TASK。不要同时执行多个 TASK。

## 必须先读取

1. `.spec/skill-memory-ranking/skill-memory-ranking-SPEC.md`
2. `.spec/skill-memory-ranking/skill-memory-ranking-SDD.md`
3. `.spec/skill-memory-ranking/TASKS.md`
4. `.spec/skill-memory-ranking/AGENT_RULES.md`
5. `.spec/skill-memory-ranking/task-scope.json`
6. `.spec/skill-memory-ranking/acceptance/<TASK-ID>.md`

## 执行输入

- TASK-ID：`<填写 TASK-XXX>`

## 执行规则

- 只允许修改 `task-scope.json` 中该 TASK 的 `allowedFiles`。
- 不允许修改 `forbiddenFiles`。
- 如果该 TASK 的 `requiresHumanConfirmation=true`，开始改代码前必须停止并向用户确认。
- 如果发现必须修改范围外文件，停止并说明原因，不要擅自修改。
- 不允许修改依赖文件、全局配置、公共 API 行为，除非当前 TASK 明确允许。
- 不允许删除已有测试，不允许用 `any` 绕过类型问题，不允许吞异常。
- 后端代码必须遵循 `gofmt`；前端代码必须通过 `typecheck`。
- 权重 / 阈值常量集中在 `logic-grpc-service/service/skill_memory_ranking.go` 顶部，不要分散到其他文件。
- 旧 `Score int` 字段值 = `int(round(final_rank_score * 100))`，保持兼容映射。
- proto 仅追加新字段，不修改 / 删除 / 重用现有字段编号。

## 完成要求

1. 实现当前 TASK。
2. 执行 `acceptance` 中要求的自测命令：
   - 后端：`cd logic-grpc-service && go test ./...`
   - 前端（如涉及）：`pnpm --filter hr-frontend typecheck`
3. 运行 `.spec/skill-memory-ranking/scripts/check-task-scope.sh <TASK-ID>` 并核对越界风险。
4. 运行 `.spec/skill-memory-ranking/scripts/agent-check.sh` 并确认通过。
5. 输出完成报告：
   - TASK 编号和名称；
   - 修改文件；
   - 自测命令和结果；
   - 未执行命令及原因；
   - 是否越界；
   - 是否触碰公共模块；
   - 风险和回滚方式；
   - 下一步建议。

## Hard Stop

- 需要修改 `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`。
- 需要修改 `model/**` / `repository/**`。
- 需要修改 `selectAgentSkills` / `selectAgentSkillsWithSemantic` / `rankMemories` / `retrieveMemories` / `Build` / `DebugSemanticRetrieval` 函数签名。
- 需要修改 proto 现有字段编号、类型或语义。
- 需要新增第三方依赖。
- `requiresHumanConfirmation: true` 的 TASK 在开始编码前。
