# TASKS - semantic-retrieval-score-fixes

| TASK | Title | Status |
| --- | --- | --- |
| TASK-SRF-001 | Clarify HR debug score display | pending |
| TASK-SRF-002 | Fix embedding result dedupe and candidate breadth | pending |
| TASK-SRF-003 | Allow semantic-only Skill candidates | pending |
| TASK-SRF-004 | Expand Skill embedding text and sync proto source | pending |
| TASK-SRF-005 | Verify targeted backend and frontend checks | pending |

## Scope

Allowed files:

- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`
- `logic-grpc-service/service/embedding_service.go`
- `logic-grpc-service/service/embedding_service_test.go`
- `logic-grpc-service/service/skill_memory_ranking.go`
- `logic-grpc-service/service/skill_memory_ranking_test.go`
- `logic-grpc-service/service/embedding_text_builder.go`
- `logic-grpc-service/service/embedding_text_builder_test.go`
- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/service/embedding_backfill_service.go`
- `web-gin-service/proto/recruitment.proto`
- `.spec/semantic-retrieval-score-fixes/**`

Forbidden files:

- `package.json`, lockfiles, generated pb files, unrelated frontend/backend modules.
