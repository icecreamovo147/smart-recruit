# smart-recruit-offer-service

Independent Offer service source root.

## Responsibility

- Own offer creation, updates, send/revoke behavior, candidate response handling, and offer lifecycle state.
- Coordinate application lifecycle changes through gRPC or events.
- Avoid direct writes to Recruitment-owned tables.

## Startup

This module now contains an independently buildable Offer gRPC runtime:

```bash
GOWORK=off go test ./...
GOWORK=off go run ./cmd/offer-service --check
GOWORK=off go run ./cmd/offer-service --serve --addr :50064
```

At runtime it reuses the shared Smart Recruit MySQL schema through existing repositories, registers `OfferService`, exposes gRPC health, starts the shared metrics endpoint, initializes tracing, and registers the `offer` instance through Nacos discovery when configured.

Gateway traffic targets this service directly through discovery or `OFFER_GRPC_ADDR`.
