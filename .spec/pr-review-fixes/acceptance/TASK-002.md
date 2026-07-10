# Acceptance - TASK-002

## TASK Summary

Enforce globally unique enabled LLM default model behavior.

## SPEC References

- FR-002
- FR-003
- FR-004
- AC-002
- AC-003

## SDD References

- Section 3: Global LLM Default Model
- Section 4: Data Structure Changes
- Section 12: Migration Risks

## Acceptance Criteria

- Setting a model as default clears all other enabled default models globally.
- Runtime default lookup returns the selected global default model.
- A migration enforces at most one enabled global default model at the database layer.
- Existing duplicate defaults are cleaned deterministically before the unique constraint is added.
- `db.sql` remains consistent with the migration.
- No protobuf or frontend API shape changes are introduced.

## Required Checks

```bash
cd logic-grpc-service && go test ./repository ./service
git diff --name-only
bash .spec/pr-review-fixes/scripts/check-task-scope.sh TASK-002
bash .spec/pr-review-fixes/scripts/agent-check.sh
```

If MySQL is available:

```bash
cd logic-grpc-service && go test -tags=mysql -run 'TestMySQLMigrationConsistency|TestBaselineScenarios' ./migration/
```

## Manual Verification, if needed

Review generated-column and unique-index SQL for MySQL compatibility.

## Out-of-Scope

- Agent-specific model selection.
- Frontend UX redesign.
