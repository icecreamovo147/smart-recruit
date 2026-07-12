# Implement Task Prompt - microservice-runtime-implementation

使用 `spec-harness`。

Mode: `implement-task`

Feature: `microservice-runtime-implementation`

Task: `<TASK-ID>`

必须阅读：

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.knowledge/README.md`
- `.spec/microservice-runtime-implementation/microservice-runtime-implementation-SPEC.md`
- `.spec/microservice-runtime-implementation/microservice-runtime-implementation-SDD.md`
- `.spec/microservice-runtime-implementation/TASKS.md`
- `.spec/microservice-runtime-implementation/AGENT_RULES.md`
- `.spec/microservice-runtime-implementation/task-scope.json`
- `.spec/microservice-runtime-implementation/acceptance/<TASK-ID>.md`

执行要求：

- 一次只执行一个 TASK。
- 必须真实修改当前 TASK 对应的代码、配置、部署、脚本、测试或运行证据。
- 禁止用“创建下游 spec”替代实现。
- 严格遵守 `task-scope.json`。
- 运行必需检查，生成 report 和 evidence。
- 自审通过后才能进入下一 TASK。
