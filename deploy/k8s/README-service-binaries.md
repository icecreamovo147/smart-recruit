# Service Binary Deployment Convention

This directory still contains the active gateway, logic service, and logic worker manifests. TASK-BDME-026 adds convention only; it does not add a routed service deployment.

Future extracted backend service manifests should follow these labels:

```yaml
app.kubernetes.io/part-of: smart-recruit
app.kubernetes.io/component: backend
backend.smart-recruit/unit: <unit-name>
backend.smart-recruit/role: gateway|service|worker
backend.smart-recruit/cutover-mode: shadow|dual-run|dual-read|routed|none
```

Use `backend.smart-recruit/cutover-mode: none` for compiled skeletons that must not receive production traffic. Use `shadow`, `dual-run`, `dual-read`, or `routed` only in a scoped cutover TASK with rollback evidence.

The current active examples remain:

- `deploy/k8s/logic-deployment.yaml`: gRPC backend with background workers disabled.
- `deploy/k8s/worker-deployment.yaml`: worker-only process using `logic-grpc-service --worker-only`.

