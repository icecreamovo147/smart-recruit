# Recruitment Runtime Convergence

TASK-018 routes the active Recruitment runtime through local
`internal/interfaces/grpc` adapters. The adapters form the protobuf-facing
boundary for Job, Candidate, Application, Collaboration, Taxonomy/Admin, and
UsageStats surfaces without changing protobuf, schema, gateway, health,
metrics, tracing, Nacos, OSS, MySQL, or Redis runtime behavior.

## Active Boundary

- `cmd/recruitment-service/main.go` constructs local interface adapters before
  passing dependencies to `internal/runtime.New`.
- `internal/runtime` continues to register the canonical Job, Admin, Candidate,
  Application, and Collaboration gRPC services.
- TASK-016 and TASK-017 local domain/application services remain the target
  business implementation boundary for later adapter replacement.

## Compatibility Debt

- Persistence, OSS, outbox, cache, invite-code, usage-log, and collaboration
  compatibility still delegate to shared runtime implementations in this TASK.
- The remaining shared dependency is intentional compatibility debt because
  removing it fully requires local GORM/OSS/outbox adapters for every protobuf
  method, including AdminService split behavior.
- No shared source, protobuf contract, database schema, package manifest, or
  lockfile is changed by this convergence step.

## Follow-up Adapter Cut Points

- Replace `JobAdapter`, `CandidateAdapter`, and `ApplicationAdapter` delegates
  with local application services plus local persistence/OSS/outbox adapters.
- Replace taxonomy/admin/usage delegates after AdminService split ownership is
  finalized between Identity, Recruitment, and Analytics.
- Keep Collaboration cross-context reads as explicit ports for Interview and
  Offer snapshots before removing shared collaboration service usage.
