# Agent Rules - ai-agent-runtime-recovery

## 1. Authority

Follow this order:

1. Repository `AGENTS.md`.
2. `.agents/skills/spec-harness/SKILL.md`.
3. `.spec/ai-agent-runtime-recovery/ai-agent-runtime-recovery-SPEC.md`.
4. `.spec/ai-agent-runtime-recovery/ai-agent-runtime-recovery-SDD.md`.
5. `.spec/ai-agent-runtime-recovery/TASKS.md`.
6. `.spec/ai-agent-runtime-recovery/task-scope.json`.
7. The current TASK acceptance file.
8. Current source code, tests, protobuf definitions, migrations, and runtime evidence.

`origin/dev` is a behavior reference, not an executable source of truth. Use it to restore semantics while preserving current microservice boundaries.

## 2. Feature-Specific Rules

- Preserve the current `smart-recruit-ai-agent-service` and `smart-recruit-gateway` architecture.
- Do not reintroduce `logic-grpc-service` or `web-gin-service` as runtime dependencies.
- Do not modify K8s manifests for this feature.
- Do not modify repository-root `docs/**`; use `.spec/ai-agent-runtime-recovery/docs/**` for feature-owned notes.
- Do not modify protobuf definitions, database migrations, shared modules, auth/RBAC behavior, package manifests, or lockfiles unless the current TASK explicitly allows the path and the required confirmation has been recorded.
- If a TASK discovers that proto, schema, auth, or shared module changes are required outside its explicit scope, stop and request confirmation.
- Do not hide unsupported runtime behavior behind success responses.
- Do not log or persist secrets, tokens, raw credentials, or raw candidate-sensitive data in reports, tool traces, or debug output.

## 3. Goal Mode and Subagent Rules

The user intends to execute this feature through Codex Goal mode and prefers subagents for context isolation.

- The controlling agent must read `AGENTS.md`, `spec-harness`, SPEC, SDD, TASKS, `AGENT_RULES.md`, `task-scope.json`, acceptance, prompts, and required knowledge files itself.
- Subagents may be used for scoped implementation, read-only review, dev-branch comparison, or failed-check investigation.
- Give each subagent only the current TASK, allowed files, forbidden files, acceptance criteria, relevant dev reference paths, and required checks.
- Subagents must not redefine requirements, expand scope, skip Harness checks, or make Hard Stop decisions.
- Prefer an independent read-only subagent for `self-review` when available.
- The controlling agent must run checks, update reports/evidence, and decide whether the next TASK can start.

## 4. Required Preflight for Implementation

Before any TASK implementation:

1. Classify the feature as `current` using `node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/ai-agent-runtime-recovery`.
2. Verify the TASK exists in `TASKS.md`, `task-scope.json`, and `acceptance/<TASK-ID>.md`.
3. Establish a reliable TASK base tree and export it as `TASK_BASE_TREE`.
4. Check `requiresHumanConfirmation`; if true, record explicit user confirmation before editing.
5. Read `.knowledge/README.md`, `.knowledge/manifest.yaml`, `.knowledge/INDEX.md`, and routed active knowledge for the TASK scope.
6. Stop if the worktree contains unrelated dirty changes that make the TASK baseline unreliable.

## 5. Required Checks After Implementation

Every implemented TASK must run or explicitly explain why unable to run:

```bash
git diff --name-only
bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh <TASK-ID>
bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh
```

Also run every command listed in the TASK acceptance file.

## 6. Knowledge Impact

Every non-trivial TASK must include knowledge impact in its report and evidence.

- Use `.knowledge/manifest.yaml` and `.knowledge/INDEX.md` to select active knowledge.
- Verify critical claims against each document's `source_refs`.
- Run `node .knowledge/scripts/detect-impact.mjs --root . --base-tree <git-tree>` when available and a reliable base tree exists.
- If affected knowledge is outside TASK scope, report `STALE`, `CANDIDATE`, or `coverage_gap`; do not edit out-of-scope knowledge.
- Do not read `.knowledge/inbox/**` or `.knowledge/archive/**` by default.

## 7. Dev Branch Reference Protocol

When restoring behavior from `origin/dev`:

- Use commands such as `git show origin/dev:<path>` for exact reference code.
- Cite the dev reference files in the TASK report.
- Rebuild behavior inside current service boundaries instead of copying monolith runtime dependencies.
- If dev behavior requires data or contracts missing from current service boundaries, stop and record the gap.

Recommended reference paths:

- `logic-grpc-service/service/ai_service.go`
- `logic-grpc-service/service/candidate_ai_service.go`
- `logic-grpc-service/service/recruiting_intelligence_service.go`
- `logic-grpc-service/service/agent_run_*`
- `logic-grpc-service/service/agent_skill_*`
- `logic-grpc-service/ai/*`
- `logic-grpc-service/repository/*`
- `logic-grpc-service/server/mcp*`

## 8. Report Requirements

Each implementation TASK must create or update:

- `.spec/ai-agent-runtime-recovery/reports/<TASK-ID>-report.md`
- `.spec/ai-agent-runtime-recovery/reports/<TASK-ID>-evidence.json`

Reports must include:

- TASK ID.
- Modified file list.
- Change summary by file.
- Scope check result.
- SPEC comparison result.
- SDD comparison result.
- Acceptance comparison result.
- Test commands and results with exit codes.
- Knowledge impact.
- Risks and follow-up items.
- Whether the next TASK can start.

Do not describe failed, skipped, or unavailable checks as passing.

## 9. Hard Stop Conditions

Stop and request confirmation if any TASK needs:

- Protobuf changes.
- Database schema or migration changes.
- Auth, RBAC, permission, quota, risk-control, or security policy changes.
- New dependencies or dependency upgrades.
- `package.json` or lockfile changes.
- Shared module changes not explicitly allowed by the current TASK.
- K8s manifest changes.
- Repository-root `docs/**` changes.
- Files outside current TASK scope.
- Public API behavior changes not already approved as compatibility restoration in SPEC.
