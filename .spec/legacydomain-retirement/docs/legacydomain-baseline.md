# Legacydomain Retirement Baseline

## Purpose

TASK-001 records the current retirement baseline before active service code is changed. It is evidence for later TASKs and does not change runtime behavior.

TASK-009 records the final retirement status after all implementation tasks completed. The initial baseline remains below as historical evidence; the final status is authoritative for the completed feature.

## Final Retirement Status

| Service | Retired root | Final status |
|---|---|---|
| Offer | `smart-recruit-offer-service/internal/legacydomain` | Deleted; active runtime uses local `internal/infrastructure/**` adapters and owner contracts. |
| Interview | `smart-recruit-interview-service/internal/legacydomain` | Deleted; active runtime uses local `internal/infrastructure/**` adapters and owner contracts. |
| Recruitment | `smart-recruit-recruitment-service/internal/legacydomain` | Deleted; active runtime uses local native persistence and application-owner contract adapters. |
| AI Agent | `smart-recruit-ai-agent-service/internal/legacydomain` | Deleted; active runtime uses native gRPC and AI-owned persistence adapters. |

Final enforcement:

- `scripts/check-backend-boundaries.mjs` fails on any `internal/legacydomain` directory or non-test Go import.
- `scripts/check-mysql-table-ownership.mjs` fails if any targeted service reintroduces `internal/legacydomain`.
- Historical `.spec` reports may still mention legacy paths as evidence, but active source, scripts, and `.knowledge` no longer depend on them.

## Target Roots

| Service | Legacy root | Go files | Current role |
|---|---|---:|---|
| Offer | `smart-recruit-offer-service/internal/legacydomain` | 45 | Repository/model compatibility backing local Offer adapters. |
| Interview | `smart-recruit-interview-service/internal/legacydomain` | 45 | Repository/model compatibility backing local Interview adapters. |
| Recruitment | `smart-recruit-recruitment-service/internal/legacydomain` | 140 | Active runtime service/repository/model/AI compatibility graph. |
| AI Agent | `smart-recruit-ai-agent-service/internal/legacydomain` | 140 | Active runtime service/repository/model/AI compatibility graph. |

## Active Non-Test Import Sites

Current non-test files outside legacy roots that import `internal/legacydomain`:

- `smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/legacy_servers.go`
- `smart-recruit-interview-service/cmd/interview-service/main.go`
- `smart-recruit-interview-service/internal/infrastructure/client/application_adapter.go`
- `smart-recruit-interview-service/internal/infrastructure/client/authorizer.go`
- `smart-recruit-interview-service/internal/infrastructure/client/staff_directory.go`
- `smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher.go`
- `smart-recruit-interview-service/internal/infrastructure/persistence/interview_repository.go`
- `smart-recruit-offer-service/cmd/offer-service/main.go`
- `smart-recruit-offer-service/internal/infrastructure/client/application_adapter.go`
- `smart-recruit-offer-service/internal/infrastructure/client/authorizer.go`
- `smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher.go`
- `smart-recruit-offer-service/internal/infrastructure/persistence/offer_repository.go`
- `smart-recruit-recruitment-service/cmd/recruitment-service/main.go`

Test files currently importing legacy packages:

- `smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher_test.go`
- `smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher_test.go`

## Knowledge References

Active knowledge currently references AI Agent legacy paths in:

- `.knowledge/architecture/agent-runtime.md`
- `.knowledge/architecture/semantic-retrieval.md`
- `.knowledge/domains/agent-skill.md`
- `.knowledge/domains/ai-configuration-governance.md`
- `.knowledge/domains/mcp-tool-governance.md`
- `.knowledge/domains/memory-and-context.md`
- `.knowledge/domains/resume-intelligence.md`
- `.knowledge/pitfalls/embedding-fallback.md`
- `.knowledge/pitfalls/mcp-policy-audit.md`
- `.knowledge/pitfalls/resume-sensitive-data.md`
- `.knowledge/runbooks/debug-agent-retrieval.md`
- `.knowledge/runbooks/debug-ai-configuration.md`
- `.knowledge/runbooks/debug-resume-intelligence.md`

`service-boundaries.md` also explicitly names `internal/legacydomain` as current service-local debt. TASK-001 updates that document to point at this feature as the active retirement control plane. AI-specific references remain current until TASK-007/TASK-008 move their source files.

## Script Baseline

`scripts/check-mysql-table-ownership.mjs` currently scans legacy repository/service roots because legacy repositories remain active. Later retirement tasks must remove these roots from the scanner as each service root is deleted.

`scripts/check-backend-boundaries.mjs` now reports legacydomain directories and import sites as a non-blocking staging signal. TASK-009 must convert absence into a hard validation rule after all four roots are removed.

## Verification Commands

Baseline gathered from:

```bash
find smart-recruit-offer-service smart-recruit-interview-service smart-recruit-recruitment-service smart-recruit-ai-agent-service -path '*/internal/legacydomain' -type d
find <service>/internal/legacydomain -type f -name '*.go' | wc -l
rg -l '"smart-recruit-[^"]*/internal/legacydomain' smart-recruit-*-service -g '*.go'
rg -n 'legacydomain' .knowledge smart-recruit-*-service/internal/docs scripts -g '*.md' -g '*.mjs'
```
