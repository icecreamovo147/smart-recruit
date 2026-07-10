# Agent Rules - populate-development-agent-knowledge

## 1. Authority

Follow this order:

1. `AGENTS.md`
2. `.agents/skills/spec-harness/SKILL.md`
3. `.spec/populate-development-agent-knowledge/`
4. `.knowledge/README.md`
5. Current source code, proto, migrations, tests, and existing active knowledge as evidence

If a rule conflicts with a higher authority, stop and report the conflict.

## 2. Feature Boundary

This feature fills the coding-Agent knowledge layer. It must not change product behavior, runtime configuration, public contracts, database schema, generated code, package manifests, lockfiles, deployment assets, or CI configuration.

## 3. Knowledge Writing Rules

- Active knowledge must be verified against current repository sources.
- Use repository-relative POSIX paths only.
- Keep documents concise and task-routable.
- Prefer source anchors and boundaries over copying implementation details.
- Put uncertain, policy-level, security-decision, public-contract, or data-lifecycle conclusions in `.knowledge/inbox/` unless explicitly confirmed.
- Do not include secrets, credentials, live tokens, personal data, raw logs, database snapshots, hidden prompts, or internal Agent reasoning.
- Update `INDEX.md` and `manifest.yaml` only when routing changes are necessary.

## 4. Required Reads Before Each TASK

Read:

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- this feature SPEC and SDD
- `TASKS.md`
- `task-scope.json`
- the current acceptance file
- `.knowledge/README.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- existing active knowledge routed to the planned files

Then inspect only the source files needed to verify the TASK's claims.

## 5. Scope Rules

- Execute exactly one TASK at a time.
- Modify only files allowed by that TASK.
- Do not broaden scope to fix unrelated knowledge.
- If a required source file is outside allowed modification scope, it may be read but not edited.
- If a useful knowledge update is outside scope, record `STALE`, `CANDIDATE`, or `coverage_gap`.

## 6. Required Checks

After implementation run:

```bash
git diff --name-only
bash .spec/populate-development-agent-knowledge/scripts/check-task-scope.sh <TASK-ID>
bash .spec/populate-development-agent-knowledge/scripts/agent-check.sh
```

Also run any TASK-specific checks from the acceptance file.

## 7. Reports and Evidence

Each implementation TASK must create or update:

- `.spec/populate-development-agent-knowledge/reports/<TASK-ID>-report.md`
- `.spec/populate-development-agent-knowledge/reports/<TASK-ID>-evidence.json`

Reports must include:

- modified files;
- summary by file;
- source references inspected;
- scope check result;
- SPEC/SDD/acceptance comparison;
- test commands and real results;
- Knowledge Impact section;
- risks and follow-up items;
- whether the next TASK can start.

Evidence must be machine-readable and include `knowledgeImpact`.

## 8. Hard Stop Conditions

Stop and request confirmation if the TASK requires:

- modifying business code;
- modifying package manifests or lockfiles;
- modifying proto, generated code, database schema, or migrations;
- modifying CI/CD configuration;
- changing authentication, authorization, permission, security, or public API behavior;
- changing `.agents/**`, `AGENTS.md`, or historical feature contracts;
- adding dependencies;
- creating active policy knowledge beyond describing current code;
- editing outside current TASK scope.
