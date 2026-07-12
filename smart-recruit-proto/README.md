# smart-recruit-proto

Central protobuf contract source for the extracted Smart Recruit services.

## Responsibility

- Own shared `.proto` definitions and Go generation strategy.
- Keep generated service contracts aligned for gateway and backend services.
- Preserve current protobuf compatibility unless a scoped TASK explicitly changes a contract.

## Startup

This root is a contract module and does not run a server. Later TASKs add proto sync and generation commands.

## Monolith Relationship

During migration, `logic-grpc-service` and `web-gin-service` remain the runtime fallback. This module becomes the shared contract source used by both legacy and extracted services.
