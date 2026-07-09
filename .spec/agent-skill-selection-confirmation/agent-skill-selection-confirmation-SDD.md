# Agent Skill Selection Confirmation SDD

## 1. Existing Architecture Summary

- HR frontend chat lives in `hr-frontend/src/views/hr/AIChatView.vue`.
- Streaming requests use `sendMessageStream` from `hr-frontend/src/api/ai.ts`.
- Request payload already supports `agent_skill_ids`.
- Stream payload already supports generic status events through `event_type` and `event_message`.
- Backend HTTP gateway maps HR chat stream requests in `web-gin-service/handler/hr/ai.go`.
- gRPC request/response structures are defined in both `logic-grpc-service/proto/recruitment.proto` and `web-gin-service/proto/recruitment.proto`.
- Backend Skill selection is performed in `logic-grpc-service/service/agent_skill_selector.go`.
- ADK and legacy paths both call `s.selectAgentSkills(...)` before injecting Agent Skill instructions in `logic-grpc-service/service/ai_service.go`.
- Agent run trace is recorded through `logic-grpc-service/service/agent_run_recorder.go`.

## 2. Problem Analysis

The current selector returns up to three selected Agent Skills. This is useful for composition, but the backend immediately injects all selected Skill instructions into the model prompt. When multiple adjacent Skills match, the user has no chance to correct the selection before execution. Tightening ranking with hard-coded business rules would not generalize across future Skills.

The better boundary is to separate automatic candidate recommendation from user-confirmed execution when the system is uncertain.

## 3. Proposed Design

Introduce a generic two-phase flow:

1. Candidate selection phase:
   - Backend computes Agent Skill candidates using the existing selector.
   - Backend evaluates whether confirmation is required.
   - If required, backend returns a stream event and stops before AI execution.

2. Confirmed execution phase:
   - Frontend renders a Skill confirmation UI.
   - User confirms selected IDs.
   - Frontend resubmits the original message with `agent_skill_ids`.
   - Backend treats these IDs as manual choices and executes normally.

Confirmation policy should be generic:

- No manual IDs are present.
- Automatic selection returns more than one candidate.
- The score/confidence profile indicates ambiguity, such as close Top1/Top2 gap or mixed candidate categories.
- Initial implementation may use "more than one automatic candidate" as the trigger if detailed confidence gating is deferred, but this must be documented in code and tests.

## 4. Data Structure Changes

Add frontend TypeScript types for Skill confirmation candidates and selection-required payloads in `hr-frontend/src/types/ai.ts`.

Potential backend structures:

- `AgentSkillSelectionCandidate`
  - `id`
  - `name`
  - `display_name`
  - `reason`
  - `score`
  - `priority`
  - `category`
  - `scenario`
  - `recommended`
  - optional ranking breakdown fields

The feature should avoid exposing Skill content/body.

## 5. API and Interface Changes

Preferred low-impact interface:

- Reuse `ChatStreamResponse.EventType` with a new value: `agent_skill_selection_required`.
- Encode the candidate list in an additional structured field if proto change is approved, or in `EventMessage` JSON as a transitional option.
- HTTP gateway should pass the stream payload to the browser as JSON.
- Frontend `StreamPayload` should include `agent_skill_selection` or equivalent structured candidate data.

Changing public proto fields is a Hard Stop unless explicitly accepted for the implementation TASK.

## 6. Algorithm or Workflow Changes

Backend flow:

1. Validate user/session/model as today.
2. Before AI execution, compute available capabilities.
3. Select Agent Skill candidates.
4. If request has manual `agent_skill_ids`, execute directly.
5. If no confirmation is required, execute directly using selected automatic Skills.
6. If confirmation is required:
   - send stream event `agent_skill_selection_required`
   - include candidate list and original session ID
   - mark stream done without generating model output
   - do not inject Skill instructions

Frontend flow:

1. Submit user message as today.
2. On stream event `agent_skill_selection_required`, stop pending assistant generation.
3. Render confirmation UI as an assistant-side pending action.
4. On confirm, call the same submit path with the original message and selected IDs.
5. On cancel/select none, call the same submit path with an empty selected ID array and a flag or local state indicating not to request confirmation again.

The "do not ask again for this resubmission" mechanism needs confirmation. Options:

- Treat an explicit empty `agent_skill_ids` plus a frontend-only state as enough if backend can distinguish confirmation submission.
- Add a request flag such as `agent_skill_selection_confirmed`.
- Add a signed or opaque confirmation token.

## 7. Configuration Design

Initial implementation can avoid new global config. If thresholds are introduced, they should be localized near Skill selection policy and default to safe values. Adding global config is a Hard Stop unless confirmed.

## 8. Compatibility Strategy

- Existing manual Skill selections remain direct execution.
- Existing sessions and message history remain readable.
- Existing AgentTracePanel display remains compatible.
- If frontend does not understand the new event, it may show no answer; therefore backend should only emit it once frontend support is delivered in the same feature.
- Both logic and web proto copies must remain in sync if proto fields are added.

## 9. Error Handling and Fallback Design

- Selector errors should follow current fallback behavior.
- Confirmation UI failure should restore the user's message and choices.
- Confirmation resubmit with unavailable IDs should use existing manual Skill filtering behavior, and the trace should show resulting selected IDs.
- Stream abort before confirmation should not create model output.

## 10. Observability and Debug Output Design

- Add logs around confirmation decision:
  - candidate count
  - selected/recommended IDs
  - manual IDs present
  - decision reason
- Agent run recorder may record a non-executed selection-required step if a run is started before the decision. Prefer deciding before starting model execution.
- Frontend debug logs should include event receipt and confirm action metadata.

## 11. Testing Strategy

- Go unit tests for selection confirmation policy:
  - multiple automatic candidates require confirmation
  - manual IDs bypass confirmation
  - zero/one candidate bypasses confirmation
  - candidate payload omits Skill body content
- Go tests for ADK/legacy shared selection path if factored.
- HTTP/SSE handler tests if payload mapping changes.
- Frontend typecheck.
- Frontend component tests for confirmation UI if existing Vitest patterns make this practical.

## 12. Migration Risks

- Proto changes require regenerating pb files in both services.
- If no request flag indicates confirmed empty selection, the backend may repeatedly ask for confirmation after "select none".
- Duplicate user messages may appear if frontend persists both the initial unexecuted request and confirmed resubmit incorrectly.
- Existing dirty worktree changes may interfere with scope scripts.

## 13. Implementation Boundaries

- Keep Skill selection confirmation policy in backend service layer near Agent Skill selection and AI execution orchestration.
- Keep frontend UI inside HR chat components.
- Backend emission of `agent_skill_selection_required` is owned by TASK-ASC-004, because it must modify AI service orchestration after the API contract and frontend UI exist.
- Do not modify ranking weights, embedding provider, database schema, package manifests, or unrelated Agent Skill admin pages.
- Do not refactor AI chat streaming beyond the narrow event handling required.

## 14. Alternatives Considered

- Hard-code intent gates for specific Skill names: rejected because it lacks generality.
- Always auto-select only Top1: rejected because it removes legitimate multi-Skill composition.
- Frontend-only filtering after execution trace: rejected because prompt injection has already happened.
- Separate REST preview endpoint: viable but heavier; stream event is lower-friction with current architecture.

## 15. Assumptions Requiring Confirmation

- A new request flag to indicate confirmed selection is acceptable if needed to avoid repeated confirmation.
- Proto changes are acceptable if structured payload cannot be cleanly represented in existing fields.
- The confirmation UI should live in the assistant message area, not as a modal.

## 16. Open Questions

- Should "select none" bypass confirmation on resubmit via a new field?
- Should the backend persist the initial selection-required event in chat history?
- What exact UI copy should explain why confirmation is needed?
- Should confirmation be limited to HR agent type only?
