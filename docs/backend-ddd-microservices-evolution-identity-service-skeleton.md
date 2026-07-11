# Identity Service Skeleton

## Scope

TASK-BDME-034 creates a compile-safe Identity service binary skeleton without production auth traffic cutover.

The skeleton follows the service binary convention already registered as `identity-service`:

- command: `logic-grpc-service/cmd/identity-service`
- image convention: `recruitment/identity-service`
- config prefix: `IDENTITY_`
- health convention: gRPC health before traffic routing
- cutover state: no traffic until a later Identity cutover TASK

## Runtime Behavior

`cmd/identity-service` supports:

- `--check`: validates the skeleton descriptor and exits 0.
- `--describe`: prints the registered service unit, cutover mode, startup mode, and safety notes.

Running without flags exits non-zero and prints that the skeleton is intentionally unrouted.

The skeleton does not:

- bind a network listener;
- register auth, principal, RBAC, or audit gRPC handlers;
- rotate refresh tokens;
- mutate token-version state;
- receive gateway traffic.

## Compatibility

Existing authentication, authorization, cookie, refresh-token, token-version, RBAC, data-scope, and audit behavior remains in the current monolith paths. The gateway continues to route Identity-related HTTP requests to the existing logic service.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./internal/identity/runtime ./cmd/identity-service ./internal/platform/servicebinary
cd logic-grpc-service && go test ./...
```

Harness verification for TASK-BDME-034 also runs the feature scope check, `agent-check.sh`, and knowledge validation.
