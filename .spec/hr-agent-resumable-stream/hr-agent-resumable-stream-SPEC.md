# HR Agent Resumable Stream SPEC

## 1. Background

The HR Agent assistant currently binds a running AI response to a browser-owned streaming request. Several HR entry points each manage their own stream lifecycle, cancellation, message mutation, and process trace updates. When the browser refreshes or the HTTP/SSE connection drops, the gateway request context is canceled, the logic service treats the Agent run as canceled or interrupted, and the user loses a coherent running state.

Current repository inspection found the repeated streaming behavior in `hr-frontend/src/views/hr/AIChatView.vue`, stream helpers in `hr-frontend/src/api/ai.ts`, gateway streaming in `web-gin-service/handler/hr/ai.go`, and run/chat persistence behavior in `logic-grpc-service/service/ai_service.go`, `logic-grpc-service/repository/agent_run_repo.go`, and `logic-grpc-service/model/model.go`.

## 2. Goals

- Consolidate duplicated HR Agent streaming logic behind one frontend runtime abstraction.
- Decouple Agent execution lifetime from the browser streaming connection.
- Restore running state after refresh, reconnect, or route remount.
- Persist ordered run events and replay missing events idempotently.
- Keep explicit user cancellation distinct from network disconnect.
- Preserve current HR chat behavior, including analysis sessions, skill confirmation, retry, process trace, candidate/action options, and result metadata.
- Keep ownership boundaries: HR frontend owns UI state, web-gin owns HTTP transport, logic-grpc owns execution, persistence, and run lifecycle.
- Support one active Agent run per HR chat session unless later product requirements expand this.

## 3. Non-Goals

- Do not change candidate or interviewer frontend behavior.
- Do not add new AI tools, Agent Skills, prompt strategy, or model selection behavior.
- Do not promise exact token-level continuation after logic-service crash or provider-side interruption.
- Do not require Redux, React Reducer, Pinia, or a global frontend state library.
- Do not redesign the HR chat UI.
- Do not immediately delete legacy chat stream APIs.
- Do not introduce new infrastructure dependencies beyond the existing MySQL, RabbitMQ/Outbox, and optional Redis stack.

## 4. User-Facing Behavior

- Starting HR Agent work creates or reuses an idempotent durable run using a client request id.
- The run continues on the backend if the browser refreshes, closes the tab, route-remounts, or temporarily loses the SSE connection.
- The HR frontend can query the active run for a session and subscribe from `last_event_seq` or `Last-Event-ID`.
- Visible answer text, process trace, confirmation state, candidate/action options, and result metadata restore after refresh.
- Explicit cancel requests transition the run through cancel-requested/canceled states and are the only normal path for user-initiated cancellation.
- Skill selection pauses the same logical run in a waiting-confirmation state and resumes it after confirmation.
- Completion persists a single coherent assistant message and final run status.
- Legacy streaming behavior remains available during rollout.

## 5. Functional Requirements

- FR-001: The backend must expose a command to create an HR Agent run using `session_id`, user input or action payload, and `client_request_id`.
- FR-002: Create-run must be idempotent for the same HR user, session, and client request id.
- FR-003: Agent execution must be owned by the logic service and must not depend on the browser request context after command acceptance.
- FR-004: The run state machine must support at least `queued`, `planning`, `running`, `waiting_confirmation`, `cancel_requested`, `succeeded`, `partial`, `failed`, and `canceled`.
- FR-005: The backend must persist ordered per-run events with a monotonically increasing sequence number.
- FR-006: The backend must persist enough snapshot state to restore assistant text, process text, status, result metadata, confirmation request, active option context, and last event sequence.
- FR-007: The gateway must expose a subscription endpoint that can replay events after a sequence and then stream live updates.
- FR-008: Subscription disconnect must not cancel the run.
- FR-009: The backend must expose active-run lookup by session for refresh recovery.
- FR-010: The backend must expose explicit cancel for an active run.
- FR-011: Skill confirmation must resume the same logical run when possible and must not rely on a still-open browser stream.
- FR-012: The frontend must introduce a unified HR Agent runtime composable and pure TypeScript reducer for stream/run events.
- FR-013: The frontend reducer must ignore stale events for non-current runs and deduplicate already applied event sequence numbers.
- FR-014: Existing HR Agent entry points must migrate to the unified runtime: normal submit, retry, candidate analysis from route, candidate option actions, and skill confirmation.
- FR-015: On page refresh, the HR frontend must restore the active run for the current session and resume event subscription without duplicating assistant text.
- FR-016: On completion, the frontend must reconcile transient stream state with persisted chat history and final run metadata.
- FR-017: The old chat stream path must remain compatible during rollout unless a later TASK explicitly removes it.

## 6. Non-Functional Requirements

- NFR-001: Refresh restore should show a coherent running state within 2 seconds under normal local development conditions.
- NFR-002: Replayed events must not duplicate visible assistant content or process trace lines.
- NFR-003: Event persistence must avoid writing one database row per tiny token when batching is available and behavior remains replayable.
- NFR-004: The design must not add new package dependencies unless a TASK explicitly allows dependency changes.
- NFR-005: Logs should include `run_id`, `session_id`, HR user id, and request id where available, without logging raw candidate PII or full prompts.
- NFR-006: Backend state transitions must be safe under duplicate create, duplicate subscribe, duplicate cancel, and duplicate confirm requests.
- NFR-007: The frontend runtime should remain local to HR Agent flows and avoid unnecessary global state.

## 7. Compatibility Requirements

- Existing HR chat display and message ordering must remain stable.
- Legacy `ChatStream` behavior must remain usable during rollout.
- Gateway APIs must remain transport-oriented and must not become the source of truth for run state.
- Proto updates must be mirrored in logic and gateway trees with generated code kept in sync.
- Schema changes must include forward migration, rollback, `db.sql`, model, repository, and tests.

## 8. Observability and Debug Requirements

- Backend logs should carry run id, session id, HR user id, and request id when available.
- Run events and snapshots should make replay and duplicate suppression debuggable.
- Terminal failures should preserve structured error type and user-safe message.
- TASK reports must record test results, scope status, and knowledge impact.

## 9. Error Handling and Fallback Requirements

- Browser disconnect must not be treated as user cancellation.
- Duplicate create, cancel, confirm, and subscribe requests must be idempotent or safely rejected.
- Provider or worker interruption may end a run as `partial` or `failed` with persisted state.
- If active-run restore fails, the frontend should surface a recoverable error and avoid corrupting local chat state.
- Stale events for an old run/session must be ignored by the frontend reducer.

## 10. Security and Safety Requirements

- HR authorization must be enforced for create, get, subscribe, cancel, and confirm operations.
- A user must not be able to subscribe to or cancel another HR user's run.
- Event payloads must not expose internal secrets, tool credentials, or unredacted provider diagnostics.
- Public API and proto changes require explicit human confirmation before implementation.
- Database migrations and schema baseline changes require explicit human confirmation before implementation.

## 11. Acceptance Criteria

- AC-001: A running HR Agent response continues on the backend after browser refresh.
- AC-002: After refresh, the HR chat restores the active running answer and process trace without duplicate text.
- AC-003: A dropped subscription can reconnect with `last_event_seq` and receive only missing events plus future events.
- AC-004: Clicking cancel stops the run and persists a canceled status.
- AC-005: Closing or refreshing the browser does not mark the run canceled.
- AC-006: Skill confirmation can survive refresh before and after the HR user submits the confirmation.
- AC-007: Submit, retry, candidate analysis, option actions, and skill confirmation all use the same frontend runtime path.
- AC-008: Legacy chat stream behavior remains usable during rollout.
- AC-009: Scope checks prevent TASK implementations from modifying files outside their declared scope.
- AC-010: Tests cover backend state transitions, event replay, frontend reducer behavior, and refresh restoration behavior.

## 12. Out of Scope

- Candidate frontend changes.
- Interviewer frontend changes.
- New Agent tools, Agent Skill ranking, prompt design, or model routing.
- UI redesign.
- Removing legacy streaming APIs.
- Exact token continuation across provider or logic-service crash.
- New infrastructure dependencies.

## 13. Assumptions Requiring Confirmation

- A-001: The feature is limited to HR frontend Agent assistant flows.
- A-002: One active run per chat session is sufficient for current product behavior.
- A-003: Skill selection confirmation should resume the same logical run rather than create a separate visible assistant response.
- A-004: MySQL remains the source of truth; RabbitMQ/Outbox may dispatch work; Redis is optional acceleration only.
- A-005: The legacy `/ai/chat/stream` flow remains during rollout.
- A-006: Provider or logic-service crashes may end a run as partial or failed; exact continuation is out of scope.

## 14. Open Questions

- OQ-001: What retention period should apply to detailed run events after a run reaches a terminal state?
- OQ-002: Should multiple browser tabs show shared cancel/confirmation controls, or should only the most recent tab own controls?
- OQ-003: Should rollout be guarded by an existing feature flag/config mechanism or enabled directly after validation?
- OQ-004: Should process trace replay expose exactly the same detail as the current live trace for all HR users?
