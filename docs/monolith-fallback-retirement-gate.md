# Monolith Fallback Retirement Gate

The default decision is to retain monolith fallback until live compose smoke evidence proves that every migrated Gateway route is healthy through the extracted services.

Run:

```bash
node scripts/check-monolith-fallback-retirement.mjs --output .spec/microservice-runtime-implementation/reports/monolith-fallback-retirement-gate.json
```

The gate audits Gateway route modes, rollback settings, cutover evidence, service discovery seed data, compose direct targets, and full compose smoke evidence. It may pass while still recommending `retain_monolith_fallback`; that means the control is functioning and retirement is intentionally blocked.
