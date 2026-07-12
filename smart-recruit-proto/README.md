# smart-recruit-proto

Central protobuf contract source for the extracted Smart Recruit services.

## Responsibility

- Own shared `.proto` definitions and Go generation strategy.
- Keep generated service contracts aligned for gateway and backend services.
- Preserve current protobuf compatibility unless a scoped TASK explicitly changes a contract.
- Provide drift checks so `logic-grpc-service`, `web-gin-service`, and extracted services use the same proto contract.

## Startup

This root is a contract module and does not run a server.

## Generation and Sync

Install `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc`, then run:

```bash
./scripts/generate-go.sh
```

Check that canonical proto files and legacy mirrors are aligned:

```bash
node ../scripts/check-proto-sync.mjs --check
```

## Monolith Relationship

During migration, `logic-grpc-service` and `web-gin-service` remain the runtime fallback. This module becomes the shared contract source used by both legacy and extracted services.
