Use spec-harness.

Mode: self-review
Feature: legacydomain-retirement
Task: <TASK-ID>

Review the current TASK diff without modifying files. Read:

- .spec/legacydomain-retirement/legacydomain-retirement-SPEC.md
- .spec/legacydomain-retirement/legacydomain-retirement-SDD.md
- .spec/legacydomain-retirement/TASKS.md
- .spec/legacydomain-retirement/AGENT_RULES.md
- .spec/legacydomain-retirement/task-scope.json
- .spec/legacydomain-retirement/acceptance/<TASK-ID>.md
- .spec/legacydomain-retirement/reports/<TASK-ID>-report.md
- current git diff

Prioritize scope violations, compatibility regressions, missing tests, unsafe DDD boundaries, public API changes, schema/auth/security risks, stale knowledge references, and inaccurate TASK evidence.

End with exactly one verdict:

```text
verdict: 通过
```

or:

```text
verdict: 不通过
```
