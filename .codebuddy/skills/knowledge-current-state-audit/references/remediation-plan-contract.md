# Agent-Ready Knowledge Remediation Plan Contract

## Purpose

Create a repair plan that another Agent can execute without guessing the evidence, resolution target, authority level, file scope, dependencies, acceptance criteria, or validation commands. The plan does not authorize source-code, production, external-system, or out-of-scope changes.

## Task schema

Each `remediation.tasks` item must use:

```json
{
  "id": "KREM-001",
  "finding_ids": ["KNO-001"],
  "priority": "P1",
  "title": "Restore formal knowledge document validity",
  "objective": "Make the candidate-profile knowledge participate in deterministic validation and routing.",
  "rationale": "Resolves the structural blocker recorded by KNO-001.",
  "status": "PENDING",
  "resolution_target": "KNOWLEDGE",
  "knowledge_level": "L1",
  "owner_hint": "engineering-platform",
  "dependencies": [],
  "parallel_safe": false,
  "human_confirmation_required": true,
  "scope": {
    "allowed_paths": [".knowledge/domains/candidate-profile-roadmap.md", ".knowledge/manifest.yaml", ".knowledge/INDEX.md"],
    "excluded_paths": ["smart-recruit-*-service/**"]
  },
  "source_evidence": [".knowledge/README.md", ".knowledge/schemas/frontmatter.schema.json"],
  "steps": [
    "Determine whether the file is active knowledge, an Inbox candidate, or historical material.",
    "Apply the matching lifecycle and metadata rules without changing unverified policy."
  ],
  "acceptance_criteria": [
    "The file has an approved lifecycle location and valid metadata.",
    "All knowledge structural and strict-reference checks pass."
  ],
  "tests": [
    "node .knowledge/scripts/validate-knowledge.mjs --root .",
    "node .knowledge/scripts/check-references.mjs --root . --strict-routes"
  ],
  "rollback": "Restore the previous knowledge file and routing entries from the task diff if validation or review rejects the lifecycle decision.",
  "deliverables": ["Validated knowledge update", "Updated route/index when required", "Verification evidence"]
}
```

## Priority and levels

Priority:

- `P0`: CRITICAL Agent harm, security/privacy/public-contract corruption, or a blocker that makes the knowledge layer broadly untrustworthy.
- `P1`: HIGH findings and structural blockers that disable deterministic validation/impact routing.
- `P2`: MEDIUM drift or significant coverage/evidence gaps.
- `P3`: LOW navigation, wording, and long-term maintainability work.

Knowledge level:

- `L1`: mechanical facts such as paths, commands, references, route entries, and verified operational steps.
- `L2`: behavior, architecture, data flow, failure behavior, or domain invariants; require code/test/contract evidence and review.
- `L3`: decisions, security/privacy policy, data lifecycle, public API policy, or accepted ADR changes; require human confirmation and normally start as an Inbox candidate or proposed ADR.

## Planning rules

- Map every open finding to at least one task; combine findings only when they share a resolution target and verification path.
- Order tasks by dependency, restoration of auditability, severity, then knowledge level.
- Set `human_confirmation_required: true` for every L3 task, accepted ADR change, ambiguous resolution target, source-code change, public contract/schema/configuration policy, security/privacy/data lifecycle decision, or destructive move.
- Set `resolution_target` consistently with linked findings. Never turn a possible source defect into a documentation edit merely to make them agree.
- Keep `allowed_paths` explicit and narrow. A glob is acceptable only for a cohesive module whose exact touched files cannot yet be known.
- Include cumulative validation, not only per-file inspection.
- Updating `last_verified` requires substantive verification of the document, not a formatting or single-link fix.
- Preserve historical explanation by archiving/superseding when appropriate; do not delete accepted ADR history.
- Do not read draft/archived content as current truth while implementing tasks.

## Markdown shape

The rendered remediation plan must contain:

1. Audit linkage, verdict, scope, commit, and working-tree state.
2. Execution rules and non-goals.
3. Strategy and ordered priority/dependency summary.
4. One self-contained section per task.
5. Finding-to-task traceability matrix.
6. Final cumulative verification gate.
7. A fenced `knowledge-remediation-manifest` JSON block generated from the same canonical audit JSON.

The embedded manifest must contain task IDs, findings, priority, target, level, dependencies, confirmation gates, scope, evidence, steps, acceptance criteria, tests, rollback, and deliverables. Do not hand-edit it.

## Cumulative gate

- All planned P0/P1 tasks are complete or explicitly accepted by an authorized owner.
- Every modified active document has been substantively reverified and has a defensible verdict.
- Knowledge validator tests, repository validation, strict references, and relevant business tests pass.
- Manifest routes and index references match the repaired catalog.
- No source/knowledge conflict is closed without deciding and recording the authoritative side.
- Regenerate all three audit artifacts from fresh evidence and reassess the audit verdict.
