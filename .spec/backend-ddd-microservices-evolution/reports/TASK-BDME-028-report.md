# TASK Report - TASK-BDME-028

## 1. TASK ID

- TASK ID: TASK-BDME-028
- Title: Notification Service Runtime Extraction
- Status: completed
- Self-review verdict: 通过

## 2. Modified File List

- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-028-report.md`
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-028-evidence.json`
- `logic-grpc-service/service/notification_runtime.go`
- `logic-grpc-service/service/notification_runtime_test.go`
- `logic-grpc-service/service/services.go`
- `logic-grpc-service/main.go`
- `docs/backend-ddd-microservices-evolution-notification-runtime-extraction.md`
- `docs/backend-ddd-microservices-evolution-notification-service-skeleton.md`
- `.knowledge/domains/notification-outbox.md`
- `.knowledge/runbooks/service-binary-convention.md`
- `.knowledge/manifest.yaml`

## 3. Change Summary by File

- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK-BDME-028 baseline, human-gate override, and completion/check evidence.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-028-report.md`: this TASK report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-028-evidence.json`: machine-readable TASK evidence.
- `logic-grpc-service/service/notification_runtime.go`: added `NotificationRuntime` composition for Notification persistence/unread behavior, async write worker, outbox dispatch, Notification consumer, and Email consumer startup.
- `logic-grpc-service/service/notification_runtime_test.go`: covered runtime component wiring and explicit nil runtime/MQ startup diagnostics.
- `logic-grpc-service/service/services.go`: changed `NewServices` to create Notification-related services/consumers/outbox publisher through `NewNotificationRuntime`.
- `logic-grpc-service/main.go`: changed background worker startup to start Notification outbox/notification/email components through `services.NotificationRuntime.Start`, preserving the remaining worker startup flow.
- `docs/backend-ddd-microservices-evolution-notification-runtime-extraction.md`: documented the runtime boundary, compatibility guarantees, verification, and future cutover work.
- `docs/backend-ddd-microservices-evolution-notification-service-skeleton.md`: linked the skeleton to the extracted runtime while keeping skeleton startup unrouted.
- Knowledge files: updated Notification/outbox knowledge, service binary runbook, and routing manifest for the runtime extraction.

## 4. Scope Check Result

- Scope check: passed.
- Out-of-scope changes: none.
- Forbidden files modified: none.
- Human confirmation: required by task-scope and satisfied by the user's global gate override in this thread.
- Public API, frontend, package, dependency, auth/security, database schema, protobuf, Dockerfile, Kubernetes manifest, and traffic routing changes: none.
- Runtime compatibility: Notification outbox publisher, Notification consumer, and Email consumer still start in the existing background worker block; the change centralizes their composition/startup behind `NotificationRuntime`.

## 5. SPEC Comparison Result

- SPEC §5 FR-002: satisfied by making Notification runtime a separable backend service runtime boundary.
- SPEC §5 FR-021..FR-023: satisfied by keeping gateway/deployment cutover out of this TASK and documenting future shadow/dual-run/routed cutover requirements.
- SPEC §5 FR-024: supported by tests and docs proving the runtime boundary while preserving existing monolith behavior.
- SPEC §5 FR-025: supported by preparing a runtime that future gateway cutover can call directly, without adding a logic facade.
- SPEC §11 AC-009..AC-014: staged for Notification runtime build/run behavior without altering user-visible APIs.

## 6. SDD Comparison Result

- SDD §3.1 Target Service Architecture: satisfied for Notification runtime composition.
- SDD §6.4 Shadow and Cutover: satisfied by preserving current worker startup and deferring gateway/deployment routing to later cutover TASKs.

## 7. Acceptance Comparison Result

- TASK goal implemented as scoped: passed.
- Existing behavior remains compatible: passed.
- Report/evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact: passed.

## 8. Test Commands and Results

- `cd logic-grpc-service && go test ./service -run 'TestNewNotificationRuntime|TestNotificationRuntime'`: passed.
- `cd logic-grpc-service && go test ./cmd/notification-service ./internal/notification/runtime`: passed.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `TASK_BASE_TREE=4217b3a936dbfa44ca5c71d29de7bbd895119910 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-028`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 4217b3a936dbfa44ca5c71d29de7bbd895119910 --json`: passed.
- `git diff --name-only && git ls-files --others --exclude-standard`: passed.

## 9. Knowledge Impact

- Impact result: `update_required`.
- Updated active knowledge documents:
  - `.knowledge/domains/notification-outbox.md`: UPDATED.
  - `.knowledge/runbooks/service-binary-convention.md`: UPDATED.
  - `.knowledge/manifest.yaml`: UPDATED.
- Reviewed but unchanged:
  - `.knowledge/runbooks/knowledge-coverage-audit.md`: UNCHANGED.
  - `.knowledge/runbooks/local-development.md`: UNCHANGED.
  - `.knowledge/architecture/service-boundaries.md`: UNCHANGED.
  - `.knowledge/architecture/system-overview.md`: UNCHANGED.
- Coverage gap: false.
- Knowledge validation: passed.

## 10. Risks

- `NotificationRuntime` centralizes existing startup; future changes to it affect both current monolith worker startup and extracted-service plans.
- `cmd/notification-service` still does not start this runtime; a later deployment/cutover TASK must connect the command to runtime startup with readiness and rollback controls.
- Nil MQ handling for the Notification runtime is now explicit; normal MQ-enabled behavior is unchanged.

## 11. Follow-up Items

- TASK-BDME-029 can own gateway notification route cutover and rollback controls.
- A later worker/deployment TASK must define how `cmd/notification-service` starts `NotificationRuntime` outside the monolith.

## 12. Whether the Next TASK Can Start

Next TASK can start: yes.

