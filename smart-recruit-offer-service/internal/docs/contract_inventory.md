# Offer Service Contract Inventory

## 1. Purpose

This inventory tracks the Offer pilot migration state in `.spec/microservice-ddd-evolution`.

It records the current public contract, runtime wiring, shared-domain dependencies, table access boundary, and test coverage as Offer moves into local DDD layers.

This document does not change protobuf, runtime registration, persistence behavior, schema, or user-visible behavior.

## 2. Current Local DDD Layers

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

TASK-004 moved Offer domain model, lifecycle policy, domain events, repository port, command/query DTOs, application ports, and application service orchestration into these local packages.

TASK-005 moves runtime registration to a local `interfaces/grpc` adapter backed by the local application service. The service no longer wires `smart-recruit-commons/service.OfferService` on the active runtime path.

## 3. Protobuf and Runtime Contract

Current runtime registration:

- `smart-recruit-offer-service/internal/runtime.Runtime.RegisterGRPC`
- `pb.RegisterOfferServiceServer`
- Service descriptor: `pb.OfferService_ServiceDesc.ServiceName`

Current local `pb.OfferServiceServer` methods exposed by runtime:

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

- TASK-005 does not change protobuf request/response types, rpc names, route mode, gateway behavior, or runtime registration behavior.
- Error mapping remains compatible: actor metadata mismatch returns gRPC error, permission/scope errors map to forbidden responses, and Offer business/state errors map to bad request responses.

## 4. Current Compatibility Dependency Baseline

Current direct imports in `cmd/offer-service/main.go` after TASK-029:

- `smart-recruit-offer-service/internal/legacydomain/repository`
- local `internal/application/service`
- local `internal/infrastructure/client`
- local `internal/infrastructure/mq`
- local `internal/infrastructure/persistence`
- local `internal/interfaces/grpc`

Current direct import in runtime:

- `smart-recruit-proto/recruitment/pb`

Current local service construction:

- `buildOfferServer` constructs only the Offer dependencies required by the local application service.
- runtime dependency uses local `interfaces/grpc.Server`, not shared `service.OfferService`.

Current owner-local legacy repository/model usage retained as temporary infrastructure debt:

- `repository.NewJobRepo`
- `repository.NewApplicationRepo`
- `repository.NewOfferRepo`
- `repository.NewOutboxRepo`
- `repository.NewAuthzRepo`

Migration implication:

- TASK-005 clears direct shared `service.OfferService` wiring.
- Remaining shared `repository`/`model` imports are infrastructure-only adapters and must be revisited when shared business model cleanup tasks run.

## 5. Table Access Boundary

Source: `smart-recruit-deploy/mysql-table-ownership.json`.

Offer owner tables:

- `offers`
- `offer_events`

Allowed transitional read:

- `applications` (owner: recruitment)

Allowed shared/platform write:

- `event_outbox` (owner: platform)

No TASK-005 change modifies these ownership rules.

Risk to track in later TASKs:

- Offer infrastructure now constructs only Offer-required shared repositories instead of the full shared service graph.
- Offer lifecycle changes that affect application status currently use a local application lifecycle adapter over the existing shared MySQL transaction as transitional owner-contract debt. This must be revisited when Recruitment owner APIs/events are introduced.

## 6. Current Test Coverage

Existing Offer service tests:

- `cmd/offer-service/main_test.go`
- `internal/runtime/runtime_test.go`
- `internal/domain/model/offer_test.go`
- `internal/application/service/offer_service_test.go`
- `internal/interfaces/grpc/offer_server_test.go`

Current test focus:

- CLI/runtime check behavior.
- Runtime requires Offer dependency.
- Runtime registers `OfferService` with gRPC.
- Offer domain lifecycle/state rules.
- Offer application command/query orchestration.
- gRPC request parsing, response success mapping, business error mapping, and event DTO mapping.

Current test gap:

- Persistence adapters currently rely on shared repository tests and service-level `go test ./...`; direct DB adapter tests can be added after the next infrastructure cleanup task if this layer grows more logic.
- No local Offer persistence adapter tests yet.
- No local mapper tests yet.

Expected next tests:

- TASK-004: domain/application tests for create, update, send, withdraw, accept, reject, and event listing rules.
- TASK-005: infrastructure/interface/runtime tests for GORM adapters, outbox/client adapters, and gRPC compatibility.

## 7. Out-of-Scope Confirmation

TASK-003 intentionally does not:

- Move Offer business rules from `smart-recruit-commons`.
- Change protobuf or generated Go contracts.
- Change database schema, migrations, or table ownership.
- Change gateway routing or public HTTP behavior.
- Delete or deprecate shared legacy implementation.
- Start using the new skeleton packages from runtime.
