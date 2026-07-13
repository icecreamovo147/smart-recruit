# Offer Service Contract Inventory

## 1. Purpose

This inventory is the TASK-003 baseline for the Offer pilot migration in `.spec/microservice-ddd-evolution`.

It records the current public contract, runtime wiring, shared-domain dependencies, table access boundary, and test coverage before moving Offer business rules into local DDD layers.

This document does not change protobuf, runtime registration, persistence behavior, schema, or user-visible behavior.

## 2. Current Local DDD Skeleton

The Offer service now has the target DDD package shape under `internal/`:

```text
internal/
  domain/
    model/
    repository/
    service/
    event/
  application/
    command/
    query/
    port/
    service/
  infrastructure/
    persistence/
    client/
    mq/
  interfaces/
    grpc/
    mapper/
  runtime/
```

Current skeleton packages contain only package-level documentation and no behavior. Runtime still uses the existing shared `smart-recruit-domain-go/service.OfferService` path.

## 3. Protobuf and Runtime Contract

Current runtime registration:

- `smart-recruit-offer-service/internal/runtime.Runtime.RegisterGRPC`
- `pb.RegisterOfferServiceServer`
- Service descriptor: `pb.OfferService_ServiceDesc.ServiceName`

Current `OfferAPI` methods exposed by runtime:

- `CreateOffer`
- `UpdateOffer`
- `GetOffer`
- `ListOffersByApplication`
- `SendOffer`
- `WithdrawOffer`
- `AcceptOffer`
- `RejectOffer`
- `ListMyOffers`
- `ListOfferEvents`

Compatibility rule:

- TASK-003 does not change protobuf request/response types, rpc names, error mapping, route mode, gateway behavior, or runtime registration behavior.

## 4. Current Shared Dependency Baseline

Current direct imports in `cmd/offer-service/main.go`:

- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`

Current direct import in runtime:

- `smart-recruit-proto/recruitment/pb`

Current shared service construction:

- `buildDomainServices` returns `*service.Services`
- runtime dependency uses `services.Offer`

Current repositories constructed for the Offer runtime:

- `repository.NewUserRepo`
- `repository.NewRefreshTokenRepo`
- `repository.NewJobRepo`
- `repository.NewProfileRepo`
- `repository.NewResumeRepo`
- `repository.NewApplicationRepo`
- `repository.NewInterviewRepo`
- `repository.NewOfferRepo`
- `repository.NewNotificationRepo`
- `repository.NewOutboxRepo`
- `repository.NewAuthzRepo`
- `repository.NewChatRepo`
- `repository.NewSessionSummaryRepo`
- `repository.NewToolTraceRepo`
- `repository.NewAgentRunRepo`
- `repository.NewMemoryRepo`
- `repository.NewInviteCodeRepo`
- `repository.NewDepartmentRepo`
- `repository.NewJobLocationRepo`
- `repository.NewDepartmentLocationRepo`
- `repository.NewUsageLogRepo`
- `repository.NewEmailLogRepo`

Migration implication:

- TASK-004 should replace Offer lifecycle rules, repository ports, and application orchestration locally while keeping the runtime path compatible.
- TASK-005 should replace runtime/interface/infrastructure wiring so the service no longer directly depends on shared `service.OfferService`, or records any remaining dependency as temporary debt.

## 5. Table Access Boundary

Source: `smart-recruit-deploy/mysql-table-ownership.json`.

Offer owner tables:

- `offers`
- `offer_events`

Allowed transitional read:

- `applications` (owner: recruitment)

Allowed shared/platform write:

- `event_outbox` (owner: platform)

No TASK-003 change modifies these ownership rules.

Risk to track in later TASKs:

- Current shared service construction also builds repository dependencies for non-Offer owner tables. During TASK-004/TASK-005, Offer application ports must clearly separate required application snapshots from direct non-owner table writes.
- Offer lifecycle changes that affect application status must go through a recruitment owner contract, domain event, or explicitly approved process-manager path.

## 6. Current Test Coverage

Existing Offer service tests:

- `cmd/offer-service/main_test.go`
- `internal/runtime/runtime_test.go`

Current test focus:

- CLI/runtime check behavior.
- Runtime requires Offer dependency.
- Runtime registers `OfferService` with gRPC.

Current test gap:

- No local Offer domain unit tests yet.
- No local Offer application command/query tests yet.
- No local Offer persistence adapter tests yet.
- No local mapper tests yet.

Expected next tests:

- TASK-004: domain/application tests for create, update, send, withdraw, accept, reject, and event listing rules.
- TASK-005: infrastructure/interface/runtime tests for GORM adapters, outbox/client adapters, and gRPC compatibility.

## 7. Out-of-Scope Confirmation

TASK-003 intentionally does not:

- Move Offer business rules from `smart-recruit-domain-go`.
- Change protobuf or generated Go contracts.
- Change database schema, migrations, or table ownership.
- Change gateway routing or public HTTP behavior.
- Delete or deprecate shared legacy implementation.
- Start using the new skeleton packages from runtime.
