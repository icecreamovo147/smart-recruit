# smart-recruit-recruitment-service

Independent Recruitment service source root.

## Responsibility

- Own jobs, candidates, applications, and recruitment lifecycle behavior.
- Expose gRPC service endpoints for gateway recruitment traffic after scoped cutover.
- Coordinate cross-domain changes through gRPC or events rather than direct cross-service table writes.

## Startup

This root now contains an explicit Recruitment gRPC runtime:

```bash
go run ./cmd/recruitment-service --check
go run ./cmd/recruitment-service --serve --addr :50062
```

The runtime registers Job, Candidate, and Application services against the shared MySQL schema. Gateway traffic remains on `logic` until the scoped Recruitment cutover TASK records compatibility and rollback evidence.

## Monolith Relationship

Recruitment traffic remains on `logic-grpc-service` until extraction and route-mode cutover evidence is complete.
