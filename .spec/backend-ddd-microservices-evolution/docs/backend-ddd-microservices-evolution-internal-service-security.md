# Backend DDD Microservices Evolution Internal Service Security

Last verified: 2026-07-12

This document defines the internal gRPC service-to-service security baseline for the backend DDD/microservices evolution. It keeps existing local behavior compatible while allowing production and staged extracted services to fail closed.

## Runtime Controls

| Variable | Owner | Required behavior |
| --- | --- | --- |
| `GRPC_INTERNAL_AUTH` | logic server | `required` in production so non-health RPCs must carry `x-internal-token`. |
| `GRPC_INTERNAL_TOKEN` | gateway and logic | Shared secret sent by the gateway and verified by logic/extracted services. Production rejects missing, placeholder, or too-short values. |
| `GRPC_INTERNAL_TLS` | gateway and logic | `required` in production/staging, `optional` for local development without mounted certs. |
| `GRPC_TLS_CERT_FILE` | logic server | Server certificate path used when internal TLS is enabled. |
| `GRPC_TLS_KEY_FILE` | logic server | Server private key path used when internal TLS is enabled. |
| `GRPC_TLS_CA_FILE` | gateway client | CA bundle path used to verify logic or extracted service certificates. |
| `GRPC_TLS_SERVER_NAME` | gateway client | Optional verification name for Kubernetes DNS or certificate SAN alignment. |

## Behavior

- Logic gRPC server loads TLS credentials only when `GRPC_TLS_CERT_FILE` and `GRPC_TLS_KEY_FILE` are both configured.
- Gateway gRPC clients use TLS credentials only when `GRPC_TLS_CA_FILE` is configured.
- `GRPC_INTERNAL_TLS=required` fails fast when the required server certificate/key or gateway CA file is missing.
- Health checks remain allowed through the existing internal token interceptor to preserve Kubernetes readiness/liveness behavior.
- Kubernetes manifests that enable internal TLS use TCP socket probes for the logic gRPC port unless a TLS-capable gRPC health probe is introduced.
- The internal token is still required for non-health RPCs when `GRPC_INTERNAL_AUTH=required`; TLS does not replace request authentication.
- Local Docker Compose keeps `GRPC_INTERNAL_TLS=optional` by default so developers do not need local certificates.

## Kubernetes Contract

`deploy/k8s/configmap.yaml` sets internal TLS to `required` and points logic/web pods to `/etc/recruitment/grpc-tls`.

The `recruitment-grpc-tls` secret must provide:

- `ca.crt`
- `tls.crt`
- `tls.key`

`logic-grpc-service` mounts the same secret for the server certificate and key. `web-gin-service` mounts it for the trusted CA. A production deployment may split CA and server key material into separate secrets as long as the mounted paths match config.

## Rollback

If TLS rollout fails before traffic cutover, restore the previous compatible behavior by setting:

```text
GRPC_INTERNAL_TLS=optional
GRPC_TLS_CA_FILE=
GRPC_TLS_CERT_FILE=
GRPC_TLS_KEY_FILE=
```

Keep `GRPC_INTERNAL_AUTH=required` and `GRPC_INTERNAL_TOKEN` enabled during rollback. Do not roll back to unauthenticated service-to-service traffic outside local development.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./...
cd web-gin-service && go test ./...
```

Focused tests cover:

- logic production config requiring token and TLS files when TLS is required;
- logic gRPC server TLS option validation;
- gateway production config requiring token and CA file when TLS is required;
- gateway gRPC client CA loading and fail-fast behavior.
