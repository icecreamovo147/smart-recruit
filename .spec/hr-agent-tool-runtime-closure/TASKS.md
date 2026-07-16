# TASKS — hr-agent-tool-runtime-closure

TASKs execute serially through `harness-pipeline`. Only the rolling ready window is executable.

| TASK | Title | Depends on | Initial lifecycle | Human confirmation |
|---|---|---|---|---|
| TASK-001 | Fail-closed Tool governance and truthful errors | - | ready after plan approval | no |
| TASK-002 | Live-data truth gate and per-Agent iteration control | TASK-001 | draft | required for shared module |
| TASK-003 | Prompt and published Agent Skill runtime semantics | TASK-002 | draft | no |
| TASK-004 | Durable governance identity and streaming event contract | TASK-003 | draft | no |
| TASK-005 | Bounded aggregation, knowledge, and cumulative behavior verification | TASK-004 | draft | no |
| TASK-006 | Application-analysis message validity and Anthropic envelope defense | TASK-005 | ready after CR-0007 approval | required for shared/public validation behavior |

## TASK-001 - Fail-closed Tool governance and truthful errors

Remove default builtin fallback for configured Agents, require explicit MCP selection, and make executor business failures non-nil classified errors. Add negative tests for empty, disabled, unknown, abstract-only, and error cases.

## TASK-002 - Live-data truth gate and per-Agent iteration control

Apply deterministic planning to both provider branches, require successful matching read Tool evidence for live-data answers, and propagate validated Agent `max_iterations` to an additive per-call shared AI loop option. This TASK pauses for explicit approval before shared-module writes.

## TASK-003 - Prompt and published Agent Skill runtime semantics

Add allowlisted Prompt rendering and server/UI binding validation. Require exact `current_version_id` Agent Skill content and record version identity. Do not expand authorized tools through Prompt or Skill instructions.

## TASK-004 - Durable governance identity and streaming event contract

Persist effective Agent identity on durable Runs, add Prompt/Skill version evidence, classify text deltas as `assistant.delta`, and remove the duplicate final full-delta behavior.

## TASK-005 - Bounded aggregation, knowledge, and cumulative behavior verification

Bound and parallelize existing HR-wide application aggregation with deterministic order/partial-failure semantics, update routed active knowledge, and run cumulative backend/frontend/Harness behavior checks.

## TASK-006 - Application-analysis message validity and Anthropic envelope defense

Seed a canonical planner-recognizable analysis message, use it consistently in the HR frontend, reject blank durable Run requests at HTTP and gRPC boundaries, and make Anthropic system-only inputs fail locally before network dispatch. Add focused frontend, gateway, AI service, shared adapter, and planner regression evidence.


## Hard stops

Stop for Amendment/user direction if work requires a schema/migration, Proto/public API, dependency/lockfile, auth/security policy, side-effecting automatic Tool, global configuration, or out-of-scope shared module beyond the approved TASK-002 option. Blocking checks cannot be skipped.
