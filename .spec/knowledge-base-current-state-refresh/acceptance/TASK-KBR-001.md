# Acceptance - TASK-KBR-001

## Required Outcomes

- Active `.knowledge` documents describe current microservice module ownership.
- `.knowledge/manifest.yaml` routes current paths instead of deleted legacy service paths.
- Active formal `source_refs` point to existing files.
- Knowledge scripts use current-path fixtures.
- Draft inbox and archive legacy references are left unchanged and reported as out of scope.

## Required Verification

- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `rg -n "web-gin-service|logic-grpc-service" .knowledge --glob '!archive/**' --glob '!inbox/**'`
- `bash .spec/knowledge-base-current-state-refresh/scripts/check-task-scope.sh TASK-KBR-001`
- `bash .spec/knowledge-base-current-state-refresh/scripts/agent-check.sh`
