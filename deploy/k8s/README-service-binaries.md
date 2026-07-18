# Service Binary Deployment Convention

This directory contains shared Kubernetes conventions and configuration examples for Smart Recruit services. Deploy concrete microservice workloads from the `smart-recruit-*` source roots.

Backend service manifests should follow these labels:

```yaml
app.kubernetes.io/part-of: smart-recruit
app.kubernetes.io/component: backend
backend.smart-recruit/unit: <unit-name>
backend.smart-recruit/role: gateway|service|worker|domain-library
backend.smart-recruit/cutover-mode: routed|none
```

Use `backend.smart-recruit/cutover-mode: routed` for services that are called by `smart-recruit-gateway`; use `none` for helper workloads or shared domain libraries.
