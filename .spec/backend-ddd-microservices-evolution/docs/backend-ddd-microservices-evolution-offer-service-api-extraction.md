# Offer Service API Extraction

## Scope

TASK-BDME-042 creates the compile-safe `offer-service` command and extracts the Offer-owned generated gRPC surface into `internal/offer/runtime`. This TASK does not route gateway traffic, change protobuf contracts, change database schema, or alter public HTTP behavior.

## Extracted APIs

The Offer runtime can explicitly register:

- `OfferService.CreateOffer`
- `OfferService.UpdateOffer`
- `OfferService.GetOffer`
- `OfferService.ListOffersByApplication`
- `OfferService.SendOffer`
- `OfferService.WithdrawOffer`
- `OfferService.AcceptOffer`
- `OfferService.RejectOffer`
- `OfferService.ListMyOffers`
- `OfferService.ListOfferEvents`

## Runtime Behavior

`logic-grpc-service/cmd/offer-service` supports:

```bash
go run ./cmd/offer-service --check
go run ./cmd/offer-service --describe
```

Default execution exits non-zero and does not bind a listener. Runtime registration requires an explicit `OfferAPI` dependency and a gRPC registrar. The runtime descriptor keeps `TrafficEnabled=false` and `CutoverMode=none`.

## Compatibility

Current Offer lifecycle behavior remains in the monolith path. Offer creation, send, withdraw, accept, reject, event history, notification/outbox effects, and application lifecycle coordination are unchanged by this TASK.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./internal/offer/interfaces ./internal/offer/runtime ./cmd/offer-service
cd logic-grpc-service && go run ./cmd/offer-service --check
cd logic-grpc-service && go run ./cmd/offer-service --describe
cd logic-grpc-service && go test ./...
```
