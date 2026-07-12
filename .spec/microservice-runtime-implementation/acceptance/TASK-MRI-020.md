# Acceptance - TASK-MRI-020

## 验收标准

- `smart-recruit-ai-agent-service` 可独立构建并启动。
- 注册 AI、Prompt、AgentConfig、MCP、Skill、AgentSkill、RecruitingIntelligence、EmbeddingConfig 等 AI-owned 服务。
- 长任务通过 worker/RabbitMQ 或受控 runtime 处理。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-020`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
