# AGENT_RULES - legacydomain-retirement

## 1. Canonical Inputs

Each TASK must read:

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/legacydomain-retirement/legacydomain-retirement-SPEC.md`
- `.spec/legacydomain-retirement/legacydomain-retirement-SDD.md`
- `.spec/legacydomain-retirement/TASKS.md`
- `.spec/legacydomain-retirement/task-scope.json`
- `.spec/legacydomain-retirement/acceptance/<TASK-ID>.md`
- `.knowledge/README.md`, `.knowledge/manifest.yaml`, `.knowledge/INDEX.md`, and routed active knowledge

## 2. Migration Rules

- Execute exactly one TASK at a time.
- Do not implement later TASKs early.
- Do not mechanically rename `legacydomain`; replace active dependencies first.
- `domain` packages must not import GORM, protobuf, Redis, RabbitMQ, HTTP, gRPC, Nacos, OSS, provider SDKs, or environment/config packages.
- `application` packages must depend on ports, not local infrastructure.
- `interfaces` packages must not directly operate persistence adapters.
- GORM records belong only in infrastructure persistence packages.
- Cross-owner writes must use owner contracts, events, or explicit application ports.
- `.knowledge/**` updates are expected when routed documents reference paths or behaviors changed by the current TASK.

## 3. Hard Stops

Stop and request confirmation if a TASK needs to:

- change public API behavior;
- change database schema, migrations, `db.sql`, or table ownership semantics;
- change auth, RBAC, data-scope, JWT, refresh-token, internal gRPC auth, TLS, or MCP security behavior;
- modify package manifests, lockfiles, dependencies, CI/CD, or global config;
- modify files outside current TASK scope;
- delete tests instead of migrating or replacing them;
- add direct cross-service writes;
- update `.knowledge` outside routed active documents without evidence;
- proceed with unreliable TASK baseline or failing scope checks.

## 4. Testing Rules

After each implementation TASK, run or explain why unable:

```bash
git diff --name-only
bash .spec/legacydomain-retirement/scripts/check-task-scope.sh <TASK-ID>
bash .spec/legacydomain-retirement/scripts/agent-check.sh
```

Also run TASK-specific checks from the acceptance file.

## 5. Knowledge Impact Rules

Every TASK has `requiredKnowledgeImpact: true`.

Reports and evidence must include:

- routed active documents reviewed;
- documents updated;
- stale/conflict/coverage-gap findings;
- validation result for `.knowledge` scripts.

If relevant knowledge cannot be updated because the scope is wrong, stop and correct the scope before implementation.

## 6. Report Rules

Each implementation TASK must create or update:

- `.spec/legacydomain-retirement/reports/<TASK-ID>-report.md`
- `.spec/legacydomain-retirement/reports/<TASK-ID>-evidence.json`

Reports must truthfully record modified files, scope result, SPEC/SDD/acceptance comparison, tests, risks, knowledge impact, and whether the next TASK can start.
