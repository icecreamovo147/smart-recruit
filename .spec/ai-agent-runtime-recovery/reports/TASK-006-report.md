# TASK Report - TASK-006

## 1. TASK ID

TASK-006 - Connect Agent, Prompt, Skill, and Capability Governance to Runtime

## 2. Status

Completed.

Independent self-review required two fix rounds. Round 1 found request-selected capability keys restricted tools but not Agent Skill selection. Round 2 found `temperature_override` did not affect provider execution. Both were fixed. Round 3 returned `通过`.

## 3. Modified File List

- `.spec/ai-agent-runtime-recovery/pipeline-state.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-006-evidence.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-006-report.md`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store_owner_role_test.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_ai_chat_test.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`

## 4. Change Summary by File

- `native_servers.go`: HR runtime now loads default enabled `hr_recruiting_agent` AgentConfig per request, applies bound prompt templates and instructions, filters native tools by capability/tool bindings and request `skill_capability_keys`, selects manual/auto Agent Skills through existing policy, persists selected skill metadata on messages, and passes AgentConfig `temperature_override` into provider execution through a backward-compatible optional provider interface.
- `llm_runtime.go`: NativeStore provider now implements `CompleteWithOptions` and applies runtime `temperature_override` to selected LLM config before constructing the client.
- `native_store.go`: chat message persistence now round-trips existing `agent_skill_ids_json` and `agent_skill_names_json`; adds prompt-template lookup by ID for runtime AgentConfig binding.
- `native_ai_chat_test.go`: adds targeted runtime tests for AgentConfig prompt/instruction/temperature, capability filtering, request-selected capability filtering for tools and Agent Skills, manual Agent Skill selection, and auto rule selection.
- `native_store_owner_role_test.go`: verifies Agent Skill metadata persists and reads back from existing chat history JSON columns.
- `pipeline-state.json`: records TASK-006 runtime status.

## 5. Scope Check Result

Passed.

Command:

```bash
TASK_BASE_TREE=b805f67514efcef0f325b36931cb4634a0d573b9 bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-006
```

Result:

```text
scope_result: PASS (TASK-006)
```

## 6. SPEC Comparison Result

Passed.

The implementation connects existing AgentConfig, prompt templates, capability bindings, selected capability keys, and Agent Skill metadata to HR runtime without proto, schema, auth/RBAC, dependency, global config, K8s, or root docs changes.

## 7. SDD Comparison Result

Passed.

The runtime now consumes governance data per request and remains within current microservice boundaries. MCP live execution and embedding-provider semantic retrieval remain out of scope for later TASKs.

## 8. Acceptance Comparison Result

Passed.

- Default enabled `hr_recruiting_agent` AgentConfig is loaded per request: implemented and tested.
- Prompt templates and AgentConfig instruction changes affect later runtime requests: implemented and tested.
- Capability bindings and selected `skill_capability_keys` restrict tools and Agent Skills: fixed after review and tested.
- Manual Agent Skill IDs and rule fallback selection are handled consistently: implemented and tested. Embedding-backed semantic retrieval remains out of scope for TASK-008.
- Skill confirmation metadata is persisted where current schema supports it: existing durable run metadata remains covered; chat history skill IDs/names now round-trip existing JSON columns.
- `temperature_override` affects provider execution through an optional provider interface and NativeStore implementation: fixed after review and tested.

## 9. Test Commands and Results

| Command | Result |
|---|---|
| `GOWORK=off go test ./internal/interfaces/grpc -run 'TestHRChatRuntimeAppliesAgentPromptAndManualAgentSkills|TestHRChatRuntimeCapabilityBindingsRestrictApplicationTool|TestHRChatRuntimeAutoSelectsEligibleAgentSkill|TestHRChatRuntimeSelectedCapabilitiesRestrictAgentSkillSelection|TestConfirmAgentRun' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./internal/infrastructure/persistence -run 'TestNativeStoreChatMessagePersistsAgentSkillMetadata|TestNativeStoreChatOwnerRoleIsolationWithHRIDCollision' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service` | Passed |
| `git diff --name-only` | Passed |
| `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-006` | Passed |
| `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh` | Passed |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree b805f67514efcef0f325b36931cb4634a0d573b9` | Passed; result `update_required` |

Gateway tests were not required by TASK-006 acceptance because no Gateway files changed in TASK-006; `agent-check.sh` still ran Gateway tests due existing staged TASK-004 Gateway changes and they passed.

## 10. Knowledge Impact

Result: `update_required`.

No `.knowledge/**` files were edited because they are outside TASK-006 scope. Candidate knowledge debt was recorded for Agent runtime, AI configuration governance, Agent Skill selection, semantic retrieval boundaries, persistence metadata, and debug runbooks.

## 11. Review Rounds

- Round 1: `不通过`; request-selected `skill_capability_keys` restricted tools but not Agent Skill selection.
- Round 2: `不通过`; AgentConfig `temperature_override` was not applied to provider execution.
- Round 3: `通过`; confirmed default AgentConfig loading, prompt/instruction rendering, capability/tool/skill filtering, metadata persistence, and temperature override application.

## 12. Risks

- Semantic Agent Skill scoring is limited to existing rule/policy selection in this TASK. Embedding-backed retrieval remains TASK-008 scope.
- MCP/SKILL live execution remains TASK-007 scope; TASK-006 only constrains availability and prompt/runtime metadata.

## 13. Follow-up Items

- Update relevant `.knowledge/**` documents in a future knowledge-maintenance scope for HR runtime governance wiring and temperature override behavior.

## 14. Whether the Next TASK Can Start

Yes. TASK-006 checks passed and independent self-review returned `通过`; TASK-007 can start.
