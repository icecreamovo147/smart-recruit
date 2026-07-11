# AGENT_RULES - hr-agent-trace-optimization

## 1. Authority

Follow, in order:

1. `AGENTS.md`
2. `.agents/skills/spec-harness/SKILL.md`
3. `.spec/hr-agent-trace-optimization/hr-agent-trace-optimization-SPEC.md`
4. `.spec/hr-agent-trace-optimization/hr-agent-trace-optimization-SDD.md`
5. `.spec/hr-agent-trace-optimization/TASKS.md`
6. The current TASK acceptance file
7. `.spec/hr-agent-trace-optimization/task-scope.json`

## 2. Feature-Specific Rules

- Preserve the existing `AgentTracePanel` public API: `sessionId`, `visible`, and `update:visible`.
- Preserve existing durable run and legacy tool trace compatibility.
- Keep backend desensitization authoritative. Do not request, reconstruct, display, copy, or log unmasked sensitive data.
- Keep final answer markdown sanitized.
- Use existing Vue 3 Composition API, Element Plus, and HR frontend test patterns.
- Do not add third-party dependencies.
- Do not modify package manifests or lockfiles.
- Do not modify auth, authorization, permissions, quota, MCP policy semantics, database schema, or migrations.
- Do not change public API/protobuf behavior except in `TASK-HATO-006`, and only after explicit user confirmation.
- The trace panel is an observability surface. Do not add cancel or confirm controls unless a future SPEC/SDD revision explicitly requires them.

## 3. Scope Boundaries

- Work only on the requested TASK.
- Modify only files allowed by the current TASK in `task-scope.json`.
- If a needed file is not in scope, stop and request confirmation instead of expanding scope.
- Do not refactor unrelated HR AI chat logic.
- Do not modify `useHrAgentRun` or `hr-frontend/src/api/agentRun.ts` unless a TASK explicitly allows it.
- Do not edit `.knowledge/` unless a TASK explicitly allows it.

## 4. Testing Requirements

- Every implementation TASK must run:
  - `git diff --name-only`
  - `bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh <TASK-ID>`
  - `bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh <TASK-ID>`
- Frontend TASKs must run:
  - `pnpm --filter hr-frontend typecheck`
  - `pnpm --filter hr-frontend test`
- `TASK-HATO-006` must additionally run affected Go tests if implemented.
- If a command cannot run, record the concrete reason in the TASK report and evidence.

## 5. Knowledge Impact

Every implementation TASK must follow the `.knowledge/README.md` protocol:

- use `.knowledge/manifest.yaml` routes for the current file scope;
- read only relevant active knowledge;
- verify critical claims against `source_refs`;
- run `.knowledge/scripts/detect-impact.mjs` when a reliable TASK base tree is available;
- report `knowledge_impact` in Markdown report and evidence.

If knowledge needs updating but the TASK scope does not allow it, record the debt in the TASK report instead of editing out-of-scope files.

## 6. Observability Rules

- Keep existing `debugLog.trace` load logging.
- Do not log raw input/output JSON or sensitive trace payloads.
- Log counts, IDs, and state transitions rather than payload content.
- Avoid noisy per-keystroke search logs.

## 7. Hard Stop Conditions

Stop and request confirmation if implementation requires:

- package manifest or lockfile changes;
- new dependencies;
- shared runtime or composable changes outside TASK scope;
- auth, permission, quota, or security behavior changes;
- database schema or migration changes;
- public API/protobuf changes outside `TASK-HATO-006`;
- generated protobuf changes without `TASK-HATO-006` confirmation;
- editing files outside the current TASK scope;
- weakening markdown sanitization or data desensitization;
- canceling backend runs from the trace panel.

## 8. Report Requirements

Each implemented TASK must create or update:

- `.spec/hr-agent-trace-optimization/reports/<TASK-ID>-report.md`
- `.spec/hr-agent-trace-optimization/reports/<TASK-ID>-evidence.json`

The report and evidence must agree on modified files, commands, exit codes, scope status, review status, risks, and whether the next TASK can start.
