# Offer Gateway Cutover

## Scope

TASK-BDME-043 adds gateway routing controls for Offer lifecycle and candidate decision APIs. It preserves public HTTP behavior and existing protobuf contracts.

## Routing Modes

Default rollback-safe mode:

```text
OFFER_ROUTE_MODE=logic
OFFER_GRPC_ADDR=
```

Cutover mode:

```text
OFFER_ROUTE_MODE=offer
OFFER_GRPC_ADDR=dns:///offer-service:50051
```

When `OFFER_ROUTE_MODE=offer`, the gateway fails fast if `OFFER_GRPC_ADDR` is empty.

## Routed Clients

The gateway routes this generated client to the Offer target in cutover mode:

- `OfferService`

All other generated clients remain on their existing targets. Recruitment, Interview, Identity, Notification, and AI Agent route switches continue to control only their own service surfaces.

## Rollback

Set:

```text
OFFER_ROUTE_MODE=logic
OFFER_GRPC_ADDR=
```

Then restart or roll the gateway. No database schema, protobuf, frontend, or public HTTP contract change is required for rollback.

## Verification

Run:

```bash
cd web-gin-service && go test ./config ./rpc
cd web-gin-service && go test ./...
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-043 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
