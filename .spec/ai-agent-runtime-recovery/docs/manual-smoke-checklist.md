# Manual Smoke Checklist - ai-agent-runtime-recovery

## Purpose

This checklist is the feature-owned manual evidence plan for restored AI Agent runtime behavior. It is intentionally kept under `.spec/ai-agent-runtime-recovery/docs/` because repository-root `docs/**` is out of scope for this feature.

Use this after automated regression checks pass and live local services, seeded data, and safe non-production AI credentials are available. No live manual checks were run during TASK-010 because the workspace did not include running service dependencies or safe provider credentials.

## Evidence Rules

- Capture request path, tenant/user role, test data identifiers, response status, and relevant UI screenshot or log excerpt.
- Do not capture secrets, raw provider tokens, raw candidate PII, or production data.
- Mark each row as `Not run locally`, `Passed`, `Failed`, or `Blocked`.
- Record skipped checks honestly with the skip reason and required prerequisite.

## Prerequisites

| Item | Required state | TASK-010 local status |
|---|---|---|
| Backend stack | Gateway, AI Agent service, and dependent services running against local data | Not run locally |
| Frontend apps | HR frontend and user frontend running from local workspace | Not run locally |
| AI credentials | Safe non-production LLM and embedding credentials configured | Not available |
| Test data | Local HR, candidate, application, resume, interview, offer, job, skill, MCP, and embedding records | Not verified |
| Audit/log access | Local service logs or DB read access for traces/events | Not verified |

## HR AI Chat

| Check | Expected result | Status | Evidence to attach | Skip reason |
|---|---|---|---|---|
| Open HR AI Chat and send a recruitment-aware question | Stream starts and returns status, delta, tool/context, usage, and done events without falling back to a raw prompt-only provider call | Not run locally | Browser screenshot, gateway request, AI Agent logs | No live stack or provider credentials in TASK-010 |
| Ask for application or candidate context | Response includes relevant recruitment context and safe tool trace metadata | Not run locally | Tool trace record or log excerpt | No seeded local recruitment data verified |
| Trigger fallback by disabling provider or using invalid config | UI receives fallback/error status without a false success response | Not run locally | UI screenshot and service log | No live stack |
| Verify Agent Skill influence | Selected skills or context usage metadata are visible in trace/debug output | Not run locally | Trace/debug record | No live stack |

## Durable Agent Run

| Check | Expected result | Status | Evidence to attach | Skip reason |
|---|---|---|---|---|
| Create a durable HR Agent Run | Run is queued, transitions to running, and persists request payload and events | Not run locally | Run id, event stream, DB/event excerpt | No live stack |
| Subscribe/replay events | Subscriber receives replayed events by sequence and live events afterward | Not run locally | Event sequence excerpt | No live stack |
| Confirm a waiting run | Confirmation continuation does not duplicate prior user message and transitions legally | Not run locally | Confirmation request and events | No live stack |
| Cancel queued or running run | Run reaches canceled terminal state and no later success event is emitted | Not run locally | Status/event excerpt | No live stack |
| Verify frontend context usage | HR frontend displays restored context usage metadata from normalized run result events | Not run locally | UI screenshot and SSE payload excerpt | No live frontend/backend stack |

## Candidate AI

| Check | Expected result | Status | Evidence to attach | Skip reason |
|---|---|---|---|---|
| Candidate streaming chat | Stream uses candidate-owned applications, resumes, jobs, interviews, and offers only | Not run locally | Request, stream excerpt, trace | No live candidate session/data |
| Candidate non-stream chat | `POST /api/v1/candidate/ai/chat` returns a compatibility response instead of 404 | Not run locally | Gateway response | No live gateway stack |
| Suggested questions | Suggestions come from parsed model output or deterministic fallback | Not run locally | Response payload | No live provider |
| Isolation check | Candidate cannot access another candidate's data | Not run locally | Negative response and audit log | No live auth/test data |

## Recruiting Intelligence Parse/Evaluate

| Check | Expected result | Status | Evidence to attach | Skip reason |
|---|---|---|---|---|
| Parse resume profile for fresh data | Profile snapshot is generated and persisted through AI Agent service behavior | Not run locally | Profile payload/DB excerpt | No live stack or seeded resume data |
| Evaluate candidate match | Evaluation and evidence are generated and persisted for candidate/job | Not run locally | Evaluation payload/DB excerpt | No live stack |
| Compare or list missing intelligence | Existing read APIs show generated entries and missing entries accurately | Not run locally | API response screenshot | No live stack |
| Permission-sensitive access | HR-only intelligence routes remain unavailable to non-HR users | Not run locally | Negative auth response | No live auth stack |

## MCP

| Check | Expected result | Status | Evidence to attach | Skip reason |
|---|---|---|---|---|
| MCP server CRUD | Server config can be created, updated, activated, deactivated, and listed | Not run locally | UI/API screenshots | No live stack |
| Test connection and tool discovery | Runtime runner reports connection status and discovered tools or explicit failure | Not run locally | Test response and logs | No safe MCP endpoint configured |
| Execute allowed tool | Policy-approved tool call returns sanitized result and audit/log record | Not run locally | Runtime log and audit row | No safe MCP endpoint configured |
| Denied tool/policy path | Disallowed call returns denial and does not execute the tool | Not run locally | Error payload and audit row | No live stack |
| Confirmation path | Tool requiring confirmation waits and resumes only after confirmation | Not run locally | Event stream and confirmation payload | No live stack |
| Credential redaction | UI/logs do not expose secrets or raw tokens | Not run locally | Redacted response/log excerpt | No live stack |
| Unsupported runner error | Unsupported transport/config produces explicit error, not silent success | Not run locally | Error payload | No live stack |

## Embedding

| Check | Expected result | Status | Evidence to attach | Skip reason |
|---|---|---|---|---|
| Provider/model CRUD | Embedding provider and model can be configured without exposing credentials | Not run locally | UI/API screenshots | No live stack |
| Model test success | Configured safe model returns dimension/latency metadata | Not run locally | Test response | No safe embedding credentials |
| Model test failure | Invalid model/config returns explicit fallback/error reason | Not run locally | Error response | No live stack |
| Agent Skill backfill | Active Agent Skill versions get ready embeddings after backfill/upsert | Not run locally | `ai_embeddings` row summary | No live DB/provider |
| Stale invalidation | Skill content/version changes invalidate prior ready embeddings before new ready row | Not run locally | DB row status excerpt | No live DB/provider |

## Semantic Debug

| Check | Expected result | Status | Evidence to attach | Skip reason |
|---|---|---|---|---|
| Agent Skill semantic debug | Response includes provider, model, dimension, latency, candidate count, and score breakdown | Not run locally | Debug API/UI screenshot | No live stack |
| Fallback behavior | Provider failure or missing embeddings returns explicit fallback reason and rule-based results where available | Not run locally | Debug response | No live provider |
| HR AI skill selection | Runtime uses semantic skill scores when embeddings are ready and safe fallback otherwise | Not run locally | Trace/debug record | No live stack |

## TASK-010 Automated Evidence

| Check | Status |
|---|---|
| `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./...` from `smart-recruit-gateway` | Passed |
| `pnpm --filter hr-frontend typecheck` | Passed |
| `pnpm --filter user-frontend typecheck` | Passed |
| `pnpm --filter hr-frontend test -- --run src/api/agentRun.test.ts` | Passed; current script forwarding ran all HR Vitest files |

