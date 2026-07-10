# Acceptance - TASK-003

## TASK Summary

Synchronize Agent SKILL status changes with semantic embedding lifecycle.

## SPEC References

- FR-005
- FR-006
- FR-007
- FR-008
- AC-004
- AC-005

## SDD References

- Section 3: Agent SKILL Embedding Lifecycle
- Section 6: Algorithm or Workflow Changes

## Acceptance Criteria

- Disabling a SKILL marks existing `agent_skill` embeddings inactive.
- Historical embedding rows are not physically deleted.
- Search excludes inactive embeddings.
- Enabling a SKILL publishes or invokes an upsert path for indexable SKILL content.
- Version activation and metadata updates continue refreshing embeddings.
- Operations are idempotent and testable without RabbitMQ.

## Required Checks

```bash
cd logic-grpc-service && go test ./repository ./service
git diff --name-only
bash .spec/pr-review-fixes/scripts/check-task-scope.sh TASK-003
bash .spec/pr-review-fixes/scripts/agent-check.sh
```

## Manual Verification, if needed

Review that no migration was added. If schema changes appear necessary, stop for user confirmation.

## Out-of-Scope

- Physical deletion of embeddings.
- Protobuf changes.
- RabbitMQ infrastructure changes.
