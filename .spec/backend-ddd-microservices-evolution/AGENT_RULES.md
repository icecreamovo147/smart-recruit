# Agent Rules - backend-ddd-microservices-evolution

## Feature-Specific Implementation Rules

- Use `spec-harness` for every TASK and execute exactly one TASK at a time unless `harness-pipeline` explicitly orchestrates serial execution.
- Read SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, the current acceptance file, the active prompt, AGENTS.md, and relevant active knowledge before editing.
- Preserve existing frontend-visible HTTP behavior unless the current TASK explicitly scopes and confirms a contract change.
- Keep protobuf source and generated code synchronized whenever proto files are touched.
- Do not introduce a transitional Analytics service read API path. Analytics must use domain-event projections/read models directly.
- Do not require an interim `logic-grpc-service` facade. Gateway may route directly to extracted services only in cutover TASKs.

## Scope Boundaries

- Modify only files allowed by current TASK scope.
- Do not modify SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, acceptance files, prompts, or scripts during implementation tasks.
- Do not modify frontend applications in this feature unless a later approved SPEC revision adds frontend scope.
- Do not edit package manifests, lockfiles, `go.mod`, or `go.sum` unless this Harness is explicitly revised and approved.

## Forbidden Changes

- Big-bang rewrite or all-service extraction in one TASK.
- Hidden public API, database, authentication, authorization, security, deployment, or traffic-routing changes.
- Distributed transactions as the default cross-service consistency mechanism.
- New infrastructure dependencies such as service mesh, Kafka, or a new gateway product without confirmed SPEC revision.
- Secrets, tokens, credentials, private keys, or production-like sensitive values in tracked files, logs, metrics, traces, events, reports, or evidence.

## Testing Requirements

- Run `git diff --name-only`, the TASK scope check, and `agent-check.sh` after every implementation.
- Run backend Go tests for every touched Go service directory.
- Run route/proto contract tests when gateway, handlers, middleware, rpc clients, proto, or generated code are touched.
- Run migration/model consistency checks when migrations, models, repositories, `db.sql`, or schema ownership are touched.
- Reports must include skipped checks with reasons and approval status; skipped checks are not passing checks.

## Compatibility Rules

- Candidate, HR, and interviewer workflows remain compatible throughout migration.
- Auth cookies, refresh, logout, current principal, RBAC, scopes, and token invalidation remain compatible until a confirmed Identity TASK changes them.
- Notification and AI extraction must use shadow, dual-run, controlled routing, or equivalent cutover evidence.
- Shared database access is transitional only with table ownership, exception records, and removal plans.

## Logging and Debug Rules

- Propagate request id or trace id through gateway, gRPC, events, and workers.
- Use structured logs and safe correlation identifiers.
- Do not log secrets, raw tokens, raw credentials, private keys, prompts, or unnecessary candidate personal data.
- Cutover reports must include before/after metrics, rollback readiness, and residual risks.

## Report Requirements

Every TASK must create or update:

- `.spec/backend-ddd-microservices-evolution/reports/<TASK-ID>-report.md`
- `.spec/backend-ddd-microservices-evolution/reports/<TASK-ID>-evidence.json`

Reports must include modified files, per-file summary, scope result, SPEC comparison, SDD comparison, acceptance comparison, test commands/results, knowledge impact, risks, and whether the next TASK can start.

## Hard Stop Conditions

Stop and request confirmation if implementation needs to:

- Modify files outside current TASK scope.
- Change public API behavior.
- Change auth/RBAC/security behavior.
- Change database schema or production data ownership.
- Change deployment or traffic routing.
- Add dependencies.
- Edit package manifests, lockfiles, `go.mod`, or `go.sum`.
- Weaken or delete tests.
- Continue through contradictory SPEC/SDD/TASK/acceptance requirements.
