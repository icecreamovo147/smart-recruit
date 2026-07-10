# Agent Skill Selection Confirmation SPEC

## 1. Background

HR AI chat currently selects database-backed Agent Skills before executing the ADK or legacy tool-calling flow. The selector may automatically match multiple Skills for one user question because `maxAgentSkillsPerRequest` allows up to three selected Skills and the ranking path accepts positively scored candidates. In questions such as candidate/job fit evaluation, this can cause adjacent but not necessarily intended Skills to be injected into the prompt.

The frontend already supports manually selected `agent_skill_ids` in `ChatRequestPayload`, and the backend already accepts `agent_skill_ids` through `ChatRequest`. The missing behavior is an explicit confirmation step when the backend is uncertain because multiple automatic Skill candidates are available.

## 2. Goals

- Let HR users confirm which automatically matched Agent Skill or Skills should be used before the backend executes the AI answer.
- Preserve the existing manual Skill selection flow: if the user explicitly selects Skills before sending, backend execution should proceed using those Skills.
- Keep the feature generic across Agent Skills; do not hard-code business-specific Skill names such as Offer, interview, or candidate matching.
- Provide enough candidate metadata for the frontend to display concise labels, recommendation state, and selection reasons.
- Keep ADK and legacy runtime behavior consistent.
- Keep execution trace understandable by recording recommended and user-confirmed Skill selections.

## 3. Non-Goals

- Do not redesign the Skill ranking algorithm itself.
- Do not change Skill embedding generation, scoring weights, or priority boost logic.
- Do not add new dependencies.
- Do not implement candidate-side AI Skill selection.
- Do not change authentication, authorization, or permission behavior.
- Do not change database schema unless later explicitly confirmed.
- Do not remove the existing manual Skill selector in the chat composer.

## 4. User-Facing Behavior

- When an HR user sends a message and the backend finds multiple automatic Agent Skill candidates requiring confirmation, the assistant response area should show a Skill selection confirmation UI instead of a generated answer.
- The confirmation UI should list candidate Skills with:
  - display name
  - short selection reason
  - whether it is recommended by default
  - optional score or confidence text if available
- The user can select one, multiple, or none of the candidates, then submit the confirmation.
- After confirmation, the frontend resubmits the original message with the selected `agent_skill_ids`; the backend then executes the normal AI response.
- If the user cancels or selects none, the backend should execute without Agent Skills.
- If only one strong Skill is selected by the backend, the system may continue without prompting, preserving current fast path.
- If no Skill candidates are available, the system should continue without prompting.

## 5. Functional Requirements

- FR-001: Backend must distinguish manual Skill IDs from automatically matched Skill candidates.
- FR-002: Backend must support a "selection required" outcome before prompt injection and model execution.
- FR-003: Backend must not inject Agent Skill instruction blocks when returning a selection-required outcome.
- FR-004: Backend must include a structured candidate list with Skill ID, display name, internal name, short reason, score fields if available, recommended flag, and selection metadata.
- FR-005: Backend must use generic selection policy inputs such as number of candidates, relative score gap, confidence, and manual selection state.
- FR-006: Frontend must detect the selection-required stream event and render a confirmation UI tied to the pending user message.
- FR-007: Frontend must resubmit the original message, session/model context, and confirmed `agent_skill_ids`.
- FR-008: Frontend retry behavior must preserve confirmed Skill choices when retrying a failed confirmed request.
- FR-009: Execution trace must show selected/confirmed Skill details after final execution.
- FR-010: The existing composer manual Skill selection must continue to populate `agent_skill_ids` and bypass automatic confirmation.

## 6. Non-Functional Requirements

- The confirmation flow must not introduce duplicate persisted user messages for the same logical request.
- UI state must be recoverable from stream completion or failure without leaving the chat permanently loading.
- TypeScript and Go changes must remain type-safe without casual `any`.
- The feature must avoid large unrelated refactors.
- The confirmation UI must be compact and usable in the existing HR chat layout.

## 7. Compatibility Requirements

- Existing `/api/v1/hr/ai/chat-stream` clients that do not handle selection-required events should not receive that event unless they are HR frontend clients using the updated flow.
- Existing `agent_skill_ids` request behavior must continue to execute directly.
- Existing `ChatRequest` fields must remain backward compatible.
- Existing ADK and legacy runtime execution must continue after confirmed selection.
- Existing AgentTracePanel should continue to display completed runs.

## 8. Observability and Debug Requirements

- Logs should identify when Skill confirmation is required, including session ID, candidate count, recommended IDs, and whether manual IDs were present.
- Agent run plan or step trace should record recommended candidates and confirmed IDs where safe to do so.
- Frontend debug logging should record selection-required receipt and confirmation submission without logging sensitive message content beyond existing debug norms.

## 9. Error Handling and Fallback Requirements

- If Skill candidate generation fails, backend should fall back to existing behavior of executing without confirmation or without Agent Skills, consistent with current error handling.
- If confirmation submission fails, frontend should restore the message input or pending confirmation state.
- If selected Skill IDs become unavailable between recommendation and confirmation, backend should ignore invalid IDs or return a clear error according to existing manual Skill validation behavior.
- If the user aborts while waiting for confirmation, no AI execution should start.

## 10. Security and Safety Requirements

- The confirmation candidate list must only include Skills available to the HR agent and current runtime capabilities.
- Do not expose raw Skill body markdown or hidden instruction content in the frontend candidate list.
- Do not expose unauthorized Skill IDs or disabled Skills.
- Do not bypass existing rate limit, auth, permission, or model availability checks.

## 11. Acceptance Criteria

- AC-001: A message with multiple automatic Skill candidates can return a frontend confirmation UI before AI execution.
- AC-002: Confirming one or more Skills resubmits the original message and produces a normal AI answer using those Skill IDs.
- AC-003: Selecting none proceeds without Agent Skill instruction injection.
- AC-004: Manually selected Skills from the composer execute directly without an extra confirmation step.
- AC-005: ADK and legacy runtime paths use the same confirmed Skill behavior.
- AC-006: Typecheck and relevant Go tests pass.
- AC-007: Execution trace remains readable and identifies confirmed Skill selections.

## 12. Out of Scope

- Ranking algorithm redesign.
- New database tables for pending confirmations.
- Candidate-side chat support.
- Full conversation branching or undo.
- Model prompt quality tuning beyond Skill instruction inclusion/exclusion.

## 13. Assumptions Requiring Confirmation

- The confirmation state can live entirely in the frontend between the initial selection-required event and the confirmation resubmit.
- It is acceptable for the backend not to persist the initial unexecuted selection-required request as a completed assistant answer.
- The first implementation may use SSE event payloads instead of a separate REST preview endpoint.
- Existing `agent_skill_ids` can represent confirmed user choices without adding a new `confirmed_agent_skill_ids` field.

## 14. Open Questions

- What exact score-gap threshold should trigger confirmation instead of auto-execution?
- Should the default selection include only Top1 or all recommended candidates?
- Should selection-required events be enabled by default or behind a runtime config flag?
- Should "select none" be shown as a first-class option in the UI copy?
