# TASK-BDME-022 Report

## TASK

- TASK ID: TASK-BDME-022
- Title: Notification Event Conversion
- Status: completed
- Self-review verdict: 通过

## Modified Files

- `.spec/backend-ddd-microservices-evolution/pipeline-state.json`: recorded TASK-BDME-022 completion, checks, evidence, and pre-seeded TASK-BDME-023.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-022-report.md`: this TASK report.
- `.spec/backend-ddd-microservices-evolution/reports/TASK-BDME-022-evidence.json`: machine-readable TASK evidence.
- `logic-grpc-service/service/application_service.go`: changed notification/email Outbox event types from transport routing names to `application.*_requested` source-domain events while preserving routing keys.
- `logic-grpc-service/service/interview_service.go`: changed interview notification/email Outbox event types to `interview.*_requested` while preserving routing keys.
- `logic-grpc-service/service/offer_service.go`: changed offer notification/email Outbox event types to `offer.*_requested` while preserving routing keys.
- `logic-grpc-service/service/notification_event_conversion_test.go`: added AST regression coverage that rejects `notification.create` or `email.send` as the Outbox domain event type in notification-producing workflows.
- `.knowledge/domains/notification-outbox.md`: documented source-domain notification event types and the routing-key compatibility constraint.

## Scope Result

- Scope check: passed.
- Out-of-scope changes: none.
- Forbidden files modified: none.
- Human confirmation: not required by this TASK.
- Public API, frontend, package, dependency, auth, database schema, security, and deployment traffic changes: none.
- Compatibility: existing MQ consumers remain compatible because routing keys and payload shapes are unchanged; only the Outbox envelope `event_type` now records the source-domain event name.

## SPEC / SDD / Acceptance Comparison

- SPEC §5 FR-006: satisfied by converting notification-producing cross-domain writes into explicit domain events.
- SPEC §5 FR-011: satisfied by keeping standardized Outbox metadata useful for event diagnostics and ownership review.
- SPEC §5 FR-012: satisfied by preserving the Outbox/Inbox event flow while separating event type from broker routing key.
- SPEC §9 Error Handling and Fallback Requirements: satisfied because existing delivery, retry, and consumer behavior remain unchanged.
- SDD §6.2 Domain Event Flow: satisfied by using source-domain event names in the Outbox envelope and transport routing keys only for broker delivery.
- SDD §9 Error Handling and Fallback Design: satisfied by preserving existing failure and retry paths while adding regression coverage for event conversion.
- Acceptance:
  - TASK goal implemented as scoped: passed.
  - Existing behavior remains compatible: passed; MQ routing keys `notification.create` and `email.send` are unchanged.
  - Report/evidence include scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact: passed.

## Checks

- `go test ./service -run 'TestNotificationOutboxWritesUseDomainEventTypes|TestBuildOutboxEvent|TestConsumeWithInbox'`: passed.
- `cd logic-grpc-service && go test ./...`: passed.
- `cd web-gin-service && go test ./...`: passed.
- `git diff --name-only && git ls-files --others --exclude-standard`: passed.
- `TASK_BASE_TREE=31b5ea1a845c2e690a7e2a210fa1a14ec03b3c75 bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-022`: passed.
- `bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh`: passed.
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 31b5ea1a845c2e690a7e2a210fa1a14ec03b3c75`: passed.
- `node .knowledge/scripts/validate-knowledge.mjs --root .`: passed.

## Knowledge Impact

- Impact result: `update_required`.
- Updated active knowledge documents:
  - `.knowledge/domains/notification-outbox.md`: UPDATED.
- Reviewed but unchanged due broad routes: `debug-recruitment-lifecycle`, `knowledge-coverage-audit`, `local-development`, `recruitment`, `recruitment-lifecycle`, `service-boundaries`, `status-notification-drift`, `system-overview`.
- Coverage gap: false.

## Risks

- Downstream analytics or ad-hoc queries that relied on Outbox `event_type = notification.create` or `email.send` must now use routing key for transport classification or the new source-domain event types for ownership classification.
- The regression test covers literal `WriteEventTx` calls in the current service files; future helper abstractions should add equivalent tests if the call shape changes.

## Next TASK

Next TASK can start: yes.
