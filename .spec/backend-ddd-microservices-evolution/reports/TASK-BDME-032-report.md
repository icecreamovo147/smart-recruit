# TASK-BDME-032 Report - AI Agent Knowledge Runtime Extraction

## Summary

Extended the AI Agent runtime boundary to own prompt/config/MCP/Skill/embedding/intelligence-facing services, added explicit adapters for Recruitment-owned source data, and redacted system prompt diagnostics by logging only character count plus SHA-256 fingerprint.

## Modified Files

- `logic-grpc-service/service/ai_agent_runtime.go`: expanded `AIAgentRuntime` to include prompt, agent config, MCP, Skill, AgentSkill, resume profile, candidate match, recruiting intelligence, embedding, and embedding config services.
- `logic-grpc-service/service/ai_agent_runtime_test.go`: verified runtime wiring for AI-derived knowledge/intelligence services.
- `logic-grpc-service/service/services.go`: exposes existing compatibility fields from `AIAgentRuntime`.
- `logic-grpc-service/internal/aiagent/infrastructure/source_adapters.go`: documented explicit transitional adapters for Recruitment-owned application/job/resume/profile source data and object storage reads.
- `logic-grpc-service/internal/aiagent/infrastructure/repositories_test.go`: verified source adapter aliases remain aligned with current repository types.
- `logic-grpc-service/service/ai_service.go`: changed system prompt diagnostics to fingerprint-only logging.
- `logic-grpc-service/service/ai_service_test.go`: added prompt fingerprint redaction coverage.
- `docs/backend-ddd-microservices-evolution-ai-agent-runtime-extraction.md`: documented ownership, source-data adapter expectations, and redaction behavior.
- `.knowledge/architecture/agent-runtime.md`: documented runtime diagnostics redaction.
- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK progress and completion.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-032-report.md`: this report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-032-evidence.json`: machine-readable evidence.

## Scope

Scope check passed. All changed files are allowed by TASK-BDME-032 scope. No frontend, dependency manifest, `go.mod`, `go.sum`, database schema, protobuf, gateway route, public API, deployment traffic, SPEC/SDD/TASK, or Harness contract files were modified.

Human confirmation was required by the TASK and is recorded as satisfied by the user's global gate override.

## SPEC / SDD / Acceptance

- SPEC comparison: aligned with FR-002 and FR-021..FR-025 by strengthening AI Agent ownership boundaries without changing external behavior.
- SDD comparison: aligned with target service architecture and shadow/cutover strategy by keeping extraction inside runtime boundaries and avoiding gateway/deployment cutover.
- Acceptance comparison: AI-derived profiles, matching artifacts, embeddings, memory/config/Skill/MCP services are runtime-owned; Recruitment source data is marked through explicit adapters; prompt diagnostics are redacted.

## Checks

- `git diff --name-only && git ls-files --others --exclude-standard`: passed; listed tracked and untracked TASK files.
- `cd logic-grpc-service && go test ./service -run 'TestNewAIAgentRuntime|TestAIAgentRuntime|TestRedactedSensitiveTextFingerprint' && go test ./internal/aiagent/infrastructure`: passed.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `TASK_BASE_TREE=af9b730aa602a13dc5e12e5b1e3e76d077df7fc8 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-032`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree af9b730aa602a13dc5e12e5b1e3e76d077df7fc8 --json`: passed; update required, no coverage gaps.

## Self-Review

Verdict: 通过.

Findings: none. No public APIs, schemas, gateway routes, MQ routing keys, or deployment traffic controls changed.

## Risks

- Recruitment source access remains a transitional in-process adapter while the monolith is still intact; later extraction tasks must replace these with service APIs/events or narrower adapter contracts.
- Tool traces still store existing raw persistence fields internally, but user-facing trace queries retain desensitization and prompt diagnostics now avoid raw prompt logging.

## Next TASK

TASK-BDME-033 can start after this TASK is committed.
