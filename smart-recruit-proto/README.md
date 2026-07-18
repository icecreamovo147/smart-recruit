# smart-recruit-proto

Central protobuf contract source for the extracted Smart Recruit services.

## Responsibility

- Own shared `.proto` definitions and Go generation strategy.
- Keep generated service contracts aligned for gateway and backend services.
- Preserve current protobuf compatibility unless a scoped TASK explicitly changes a contract.

## Startup

This root is a contract module and does not run a server.

## Generation and Sync

The repository pins `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` in `scripts/tool-versions.env`. Bootstrap the pinned cross-platform toolchain once, then generate:

```bash
./scripts/bootstrap-tools.sh
./scripts/generate-go.sh
```

`generate-go.sh` automatically prefers the bootstrapped cache and fails before writing files when any active tool version differs from the repository pins. Do not regenerate contracts with an arbitrary `protoc` from `PATH`.

Validate that canonical files are present and generated headers match the pinned toolchain:

```bash
node ../scripts/check-proto-sync.mjs --check
```

For full reproducibility, run `generate-go.sh` twice and confirm the second run leaves no Git diff. Proto Lint performs that regeneration-and-diff gate in CI.

## Ownership

This module is the only protobuf source root for the Go backend runtime.
