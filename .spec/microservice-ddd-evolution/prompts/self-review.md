Use spec-harness.

Mode: self-review
Feature: microservice-ddd-evolution
Task: <TASK-ID>

Review the current TASK diff without modifying files. Read:

- .spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md
- .spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md
- .spec/microservice-ddd-evolution/TASKS.md
- .spec/microservice-ddd-evolution/AGENT_RULES.md
- .spec/microservice-ddd-evolution/task-scope.json
- .spec/microservice-ddd-evolution/acceptance/<TASK-ID>.md
- .spec/microservice-ddd-evolution/reports/<TASK-ID>-report.md
- current git diff

Prioritize scope violations, compatibility regressions, missing tests, unsafe DDD boundaries, public API changes, schema/auth/security risks, and inaccurate TASK evidence.

End with exactly one verdict:

```text
verdict: 通过
```

or:

```text
verdict: 不通过
```
