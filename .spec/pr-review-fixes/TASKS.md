# TASKS - pr-review-fixes

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | Enable CI for PRs targeting dev | completed | CI workflow only | acceptance/TASK-001.md |
| TASK-002 | Enforce globally unique LLM default model | completed | LLM service, repo, migrations, tests | acceptance/TASK-002.md |
| TASK-003 | Synchronize Agent SKILL status with embeddings | completed | Agent SKILL and embedding lifecycle | acceptance/TASK-003.md |
| TASK-004 | Validate and mask provider extra headers | completed | LLM and embedding provider config security | acceptance/TASK-004.md |
| TASK-005 | Clamp management list page size to 100 | completed | Shared pagination normalization and service callers | acceptance/TASK-005.md |

## TASK-001 - Enable CI for PRs targeting dev

### Goal

Ensure pull requests targeting `dev` run the repository CI workflow.

### Scope

Modify GitHub Actions branch filters only.

### Allowed Files

- `.github/workflows/ci.yml`

### Forbidden Files

- Business code
- Tests
- Package manifests and lockfiles
- Migration files

### Dependencies

None.

### Acceptance Criteria

- `pull_request.branches` includes both `dev` and `main`.
- Existing CI jobs remain intact.

### Required Tests

- `git diff --name-only`
- `bash .spec/pr-review-fixes/scripts/check-task-scope.sh TASK-001`
- `bash .spec/pr-review-fixes/scripts/agent-check.sh`

### Risks

- YAML syntax errors would prevent workflow execution.

### Notes

No business code changes are permitted.

## TASK-002 - Enforce globally unique LLM default model

### Goal

Make enabled LLM default model globally unique and runtime-consistent.

### Scope

Update LLM model repository/service behavior, add migration constraints, and add targeted tests.

### Allowed Files

- `logic-grpc-service/service/llm_config_service.go`
- `logic-grpc-service/service/llm_config_service_test.go`
- `logic-grpc-service/repository/model_config_repo.go`
- `logic-grpc-service/repository/model_config_repo_test.go`
- `logic-grpc-service/migrations/*llm*default*.sql`
- `logic-grpc-service/migrations/*llm*default*.down.sql`
- `db.sql`

### Forbidden Files

- Frontend files
- Web gateway handlers
- Protobuf files
- Auth/RBAC code
- Package manifests and lockfiles

### Dependencies

TASK-001 should be completed first so CI protects later changes.

### Acceptance Criteria

- Creating or updating a model as default clears all other enabled defaults globally.
- The runtime default lookup returns the most recently selected global default.
- A DB constraint prevents multiple enabled global defaults.
- Existing fallback to first enabled model remains only when no enabled default exists.

### Required Tests

- `go test ./repository ./service` from `logic-grpc-service`
- MySQL migration consistency test if a MySQL DSN is available
- Harness checks

### Risks

- Existing DBs with duplicate defaults must be cleaned deterministically before adding the constraint.
- `db.sql` must stay consistent with migrations.

### Notes

Do not change protobuf messages or frontend API payloads.

## TASK-003 - Synchronize Agent SKILL status with embeddings

### Goal

Keep Agent SKILL semantic embeddings consistent when SKILLs are enabled, disabled, updated, or version-activated.

### Scope

Add logical invalidation for SKILL embeddings and refresh embeddings on enable.

### Allowed Files

- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/service/agent_skill_service_test.go`
- `logic-grpc-service/service/embedding_service.go`
- `logic-grpc-service/service/embedding_service_test.go`
- `logic-grpc-service/repository/ai_embedding_repo.go`
- `logic-grpc-service/repository/ai_embedding_repo_test.go`

### Forbidden Files

- Migration files unless an implementation blocker proves existing `status` cannot support logical invalidation
- Protobuf files
- Frontend files
- Package manifests and lockfiles

### Dependencies

TASK-001 should be completed first.

### Acceptance Criteria

- Disabling a SKILL marks existing `agent_skill` embeddings inactive without deleting rows.
- Search does not return inactive SKILL embeddings.
- Enabling a SKILL publishes or invokes an upsert path for indexable SKILL text.
- Updating indexable metadata and activating versions continue refreshing embeddings.
- Operations are idempotent.

### Required Tests

- `go test ./repository ./service` from `logic-grpc-service`
- Harness checks

### Risks

- Best-effort event publishing may leave embeddings stale until the worker runs.
- Tests should not require RabbitMQ.

### Notes

Do not physically delete rows from `ai_embeddings`.

## TASK-004 - Validate and mask provider extra headers

### Goal

Prevent raw provider header secrets from being stored invalidly or returned unmasked.

### Scope

Introduce shared header validation/masking in logic service and apply it to LLM and embedding provider config services.

### Allowed Files

- `logic-grpc-service/service/llm_config_service.go`
- `logic-grpc-service/service/embedding_config_service.go`
- `logic-grpc-service/service/provider_headers.go`
- `logic-grpc-service/service/provider_headers_test.go`
- `logic-grpc-service/service/llm_config_service_test.go`
- `logic-grpc-service/service/embedding_config_service_test.go`

### Forbidden Files

- Frontend files unless backend response compatibility is impossible without a UI adjustment
- Protobuf files
- Package manifests and lockfiles
- Auth/RBAC code

### Dependencies

TASK-001 should be completed first.

### Acceptance Criteria

- Create/update rejects invalid `extra_headers_json`.
- Only JSON objects with string values are accepted.
- Header names are validated.
- Provider responses return masked header values.
- Invalid legacy stored data is never returned raw.
- Provider connection tests fail clearly when stored headers are invalid.

### Required Tests

- `go test ./service` from `logic-grpc-service`
- Harness checks

### Risks

- Existing invalid legacy rows may become non-editable until corrected.

### Notes

Do not return raw extra header values for edit forms.

## TASK-005 - Clamp management list page size to 100

### Goal

Apply a shared, maintainable pagination limit to management list APIs.

### Scope

Add or reuse a shared pagination helper and update management service list methods.

### Allowed Files

- `logic-grpc-service/service/helpers.go`
- `logic-grpc-service/service/llm_config_service.go`
- `logic-grpc-service/service/embedding_config_service.go`
- `logic-grpc-service/service/mcp_service.go`
- `logic-grpc-service/service/mcp_policy_api.go`
- `logic-grpc-service/service/mcp_log_api.go`
- `logic-grpc-service/service/prompt_service.go`
- `logic-grpc-service/service/skill_service.go`
- `logic-grpc-service/service/agent_service.go`
- `logic-grpc-service/service/*_test.go`

### Forbidden Files

- Frontend files
- Protobuf files
- Package manifests and lockfiles
- Database migrations

### Dependencies

TASK-001 should be completed first.

### Acceptance Criteria

- `page <= 0` normalizes to `1`.
- `page_size <= 0` normalizes to `20`.
- `page_size > 100` clamps to `100`.
- Updated management services use the shared helper.
- Existing callers requesting `page_size <= 100` are unaffected.

### Required Tests

- `go test ./service` from `logic-grpc-service`
- Harness checks

### Risks

- Overly broad helper replacement could accidentally alter public candidate/job pagination. Keep scope to management services only.

### Notes

Do not add export or bulk-list behavior in this task.
