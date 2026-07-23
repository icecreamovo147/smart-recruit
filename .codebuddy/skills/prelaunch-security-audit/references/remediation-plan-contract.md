# Agent-Ready Remediation Plan Contract

## Purpose

Create a plan that another Codex can execute without guessing scope, security intent, acceptance criteria, or verification. The plan is not authorization to modify production or broaden repository scope.

## Task schema

Each item in `remediation.tasks` must use:

```json
{
  "id": "REM-001",
  "finding_ids": ["SEC-001"],
  "priority": "P0",
  "title": "Restrict Gateway management endpoints",
  "objective": "Remove public access while preserving internal monitoring.",
  "rationale": "Resolves the confirmed release blocker.",
  "status": "PENDING",
  "owner_hint": "platform/security",
  "dependencies": [],
  "parallel_safe": false,
  "human_confirmation_required": true,
  "scope": {
    "allowed_paths": ["smart-recruit-gateway/**", "deploy/**"],
    "excluded_paths": ["smart-recruit-proto/**"]
  },
  "steps": [
    "Split public and management listeners.",
    "Restrict the management listener through deployment policy."
  ],
  "acceptance_criteria": [
    "Public ingress cannot reach metrics, Swagger, or detailed readiness.",
    "Prometheus can still scrape the internal endpoint."
  ],
  "tests": [
    "Run Gateway unit tests.",
    "Verify public and internal endpoint behavior in staging."
  ],
  "rollback": "Restore the previous listener configuration only in an isolated environment.",
  "security_notes": ["Do not replace network isolation with obscurity."],
  "deliverables": ["Code/config change", "Focused regression test", "Deployment evidence"]
}
```

## Priority

| Priority | Use |
|---|---|
| `P0` | Active `CRITICAL` or release-blocking `HIGH`; complete before launch |
| `P1` | Other release blockers and material `HIGH`; complete before launch unless formally accepted by authorized owners |
| `P2` | `MEDIUM` risk or material evidence gap; schedule with a named owner and date |
| `P3` | `LOW` hardening or long-term maturity improvement |

## Planning rules

- Map every open `CRITICAL`, `HIGH`, and release blocker to at least one task.
- Prefer one cohesive security outcome per task. Combine findings only when they share the same root cause and verification path.
- Order tasks by dependency and risk reduction, not by file path.
- Keep `allowed_paths` narrow and name shared/public-contract/config changes explicitly.
- Set `human_confirmation_required: true` for public API changes, schema/migration changes, shared auth policy, production configuration, secret rotation, data lifecycle decisions, external-system changes, or destructive operations.
- Include negative regression tests, not only happy-path tests.
- State whether tasks are safe to run in parallel. Authorization, schema, protobuf, shared packages, global configuration, and deployment changes are normally not parallel-safe without coordination.
- Include rollback or containment guidance, but never recommend restoring a known exploitable configuration in production.
- Do not embed secret values, exploit payloads, production identifiers, or personal data.

## Markdown shape

The rendered remediation plan must contain:

1. Audit linkage and verdict.
2. Execution rules and non-goals.
3. Ordered dependency/priority summary.
4. One self-contained section per task.
5. Finding-to-task traceability matrix.
6. Final cumulative verification gate.
7. A fenced `prelaunch-security-remediation-manifest` JSON block generated from the same canonical audit JSON.

The embedded manifest keeps the artifact in Markdown format while allowing `prelaunch-security-fix` to parse task IDs, dependencies, scope, acceptance criteria, and confirmation gates without guessing from prose. Do not hand-edit the manifest; regenerate the three audit artifacts from the canonical audit JSON.

The default execution meaning is to run every task continuously and serially in dependency order and then `P0 -> P1 -> P2 -> P3` priority order. A user may explicitly select a smaller task or priority scope.

The cumulative gate should require:

- all P0/P1 tasks complete or explicitly accepted by authorized owners;
- affected unit/integration/security tests pass;
- dependency, secret, image, and IaC scans rerun where relevant;
- staging authorization/tenant/AI/MCP/payment negative tests pass where applicable;
- all three audit artifacts regenerated from updated evidence;
- final `GO`, `CONDITIONAL_GO`, or `NO_GO` reassessment.
