# Agent Rules - recruiting-intelligence-runtime-parity

## Authority and sequence

Read repository `AGENTS.md`, this feature's SPEC, SDD, TASKS, current acceptance file, `task-scope.json`, `.knowledge/README.md`, `.knowledge/manifest.yaml`, and routed active knowledge before acting. Execute one TASK only:

`implement-task -> checks/evidence -> independent self-review -> fix-check-failures when needed -> independent re-review`

The root agent alone maintains `pipeline-state.json`, baselines, confirmation gates, and Goal status. Developer and Fixer agents may edit only the current TASK's `allowedFiles`; Reviewer is read-only. No agent may self-approve its own changes.

## Baseline and scope

- Record a reliable `base_sha` and `base_tree` before every TASK.
- Scope is calculated from that same `base_tree`, including uncommitted changes from completed TASKs.
- Do not edit feature contracts, prompts, scripts, acceptance files, or task scope during TASK execution. Runtime evidence under `reports/**` and state updates are allowed.
- Do not opportunistically refactor or mass-format unrelated code.
- A forbidden match wins over an allowed match.

## Human confirmation

TASK-001 is authorized only by the user's explicit statement:

> 用户明确选择“授权修改”，仅新增内部结构化 System/User 消息调用能力，不修改依赖、Proto 或公共 HTTP API。

Record that grant in state and TASK-001 evidence. It authorizes only the named internal `smart-recruit-commons/ai` structured-message capability and cannot authorize any later shared, public, security, schema, or dependency change.

## Required completion evidence for every TASK

- Markdown report at `reports/<TASK-ID>-report.md`.
- Machine-readable evidence at `reports/<TASK-ID>-evidence.json` that validates with the canonical validator.
- `git diff --name-only` against the TASK base tree.
- `bash .spec/recruiting-intelligence-runtime-parity/scripts/check-task-scope.sh <TASK-ID>`.
- `bash .spec/recruiting-intelligence-runtime-parity/scripts/agent-check.sh`.
- TASK-specific tests from its acceptance contract.
- Knowledge impact detection/report with per-document verdicts and verified source refs.
- Independent Reviewer output with the exact final line `verdict: 通过`.

Reports and evidence must state modified files, per-file summary, scope result, SPEC/SDD/acceptance comparison, commands with truthful exit codes/results, risks, knowledge impact, and whether the next TASK may start. Never place complete prompts, resume text, raw model output, sensitive evidence, secrets, or candidate PII in logs or artifacts.

## Knowledge protocol

For TASK-001 through TASK-006, do not edit `.knowledge`; record relevant future changes as `CANDIDATE` deferred to TASK-007 and unrelated documents as `UNCHANGED`. TASK-007 may update only the six knowledge files explicitly allowed. If impact detection proves another knowledge file requires an in-scope update, record a coverage gap and hard-stop instead of silently widening scope.

## Review and repair

- Use a fresh Reviewer agent with no Developer context; it reviews code, tests, scope, contracts, and evidence read-only.
- If review fails, use a fresh Fixer limited to the findings, rerun all required checks, then use another fresh Reviewer.
- Maximum three review rounds per TASK.
- Stop if the same issue fails repair twice.

## Hard stops

Immediately stop for any required change to dependencies/lockfiles/go.mod/go.sum, Proto/public HTTP or gRPC behavior, migration/schema/db.sql, auth/RBAC/security, shared config definitions, CI/deployment/global config, root docs, a file outside TASK scope, conflicting contracts, an unreliable baseline/evidence, or an unapproved shared-module expansion.
