# smart-recruit-proto

Central protobuf contract source for the extracted Smart Recruit services.

## Responsibility

- Own shared `.proto` definitions and Go generation strategy.
- Keep generated service contracts aligned for gateway and backend services.
- Preserve current protobuf compatibility unless a scoped TASK explicitly changes a contract.

## Startup

This root is a contract module and does not run a server.

## Generation and Sync

Install `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc`, then run:

```bash
./scripts/generate-go.sh
```

Check that generated Go files are aligned with the canonical proto source:

```bash
node ../scripts/check-proto-sync.mjs --check
```

## Ownership

This module is the only protobuf source root for the Go backend runtime.
