---
name: knowledge-current-state-audit
description: "Perform an evidence-based, read-only audit of the Smart Recruit `.knowledge/` layer against the current repository, identify stale or conflicting knowledge and coverage gaps, and generate exactly three synchronized Chinese deliverables: a Markdown audit report, a self-contained HTML audit report, and an agent-ready Markdown remediation plan. Use when Codex is asked to audit or review the project knowledge base, check whether `.knowledge` matches current code, find outdated/stale knowledge, assess knowledge drift or route coverage, or produce a knowledge repair plan."
---

# Knowledge Current-State Audit

## Mission

Audit the current Smart Recruit knowledge layer against higher-authority repository evidence. Distinguish structural defects, stale claims, direct conflicts, evidence gaps, and missing coverage. Produce exactly three synchronized deliverables:

1. `*-knowledge-audit-report.md`
2. `*-knowledge-audit-report.html`
3. `*-knowledge-remediation-plan.md`

Keep the audit read-only. Do not repair knowledge or source code during this workflow.

## Authority and Safety

- Follow repository `AGENTS.md` and `.knowledge/README.md` authority order.
- Treat current code, tests, protobuf definitions, schemas, migrations, and runtime evidence as higher authority than ordinary knowledge.
- Treat accepted ADRs as protected decisions. Report implementation divergence; do not rewrite accepted ADRs.
- Do not activate `spec-harness` unless the user explicitly requests the Harness workflow.
- Do not read `.knowledge/inbox/` or `.knowledge/archive/` as current truth. Inventory their metadata only; inspect content only when needed to explain a duplicate, supersession, or user-requested review.
- Preserve user changes. Never repair, format, stage, or overwrite source or knowledge files during the audit.
- Redact secrets, personal data, resumes, raw production logs, private prompts, credentials, and provider payloads from evidence and reports.

## Required Reading

1. Read `AGENTS.md`, `.knowledge/README.md`, `.knowledge/manifest.yaml`, and `.knowledge/INDEX.md` completely.
2. Read [references/audit-playbook.md](references/audit-playbook.md) before evidence collection.
3. Read [references/audit-contract.md](references/audit-contract.md) before triage and canonical JSON construction.
4. Read [references/remediation-plan-contract.md](references/remediation-plan-contract.md) before creating remediation tasks.
5. Read `.knowledge/runbooks/knowledge-coverage-audit.md` when it exists and is active.

## Workflow

### 1. Establish the baseline

- Record repository root, branch, `HEAD`, working-tree status, generated time, and scope.
- Default the audited actual state to the current working tree, including tracked and untracked files.
- Use `HEAD` as the attribution baseline so the report distinguishes pre-existing drift from drift introduced by uncommitted work.
- If the user names a commit or range, audit that exact snapshot/range and state how uncommitted changes were handled.
- Record dirty files without assuming they are defects or modifying them.

### 2. Validate structural integrity

Run the repository-native checks when present:

```bash
node .knowledge/scripts/knowledge-validator.test.mjs
node .knowledge/scripts/validate-knowledge.mjs --root . --json
node .knowledge/scripts/check-references.mjs --root . --json --strict-routes
```

Record every command and status. A passing structural check proves only metadata and reference integrity, not semantic correctness. If validation fails, continue the read-only audit where possible and record any blocked automation.

### 3. Inventory and prioritize

- Inventory every formal document by path, ID, kind, status, owner, `last_verified`, and `review_after`.
- Give every active document exactly one final verdict.
- Use drafts only as non-authoritative inventory unless explicitly in scope.
- Prioritize security/auth/privacy/payment/AI-MCP, public contracts/schema, cross-service workflows, service boundaries, and operations before low-risk navigation knowledge.
- Treat `last_verified` and `review_after` as scheduling signals, never proof of correctness.

### 4. Detect changed and uncovered surfaces

- Use Git history since each document's `last_verified` conservatively because the metadata stores a date, not a verified commit hash.
- Compare changed paths with `manifest.yaml` routes and document `applies_to`/`source_refs`.
- Run `.knowledge/scripts/detect-impact.mjs` only with a reliable base tree and structurally valid knowledge. Otherwise reproduce the route review manually and record the limitation.
- Reverse-check current top-level modules, routes, protobufs, migrations, configuration, sensitive-data flows, deployment roots, and operational commands for missing or excessively broad routes.

### 5. Verify semantic claims

- Break each active document into testable claims: ownership, entry points, routes, state transitions, schema, configuration, fallback/retry behavior, commands, ports, data flow, authorization, and operational behavior.
- Verify critical claims against exact `source_refs` and, when those references are insufficient, the authoritative neighboring code/tests/contracts.
- Prefer repository-relative path plus line or symbol evidence.
- Use focused tests or command help only when static evidence is insufficient and execution is safe.
- Never infer `UNCHANGED` merely because a referenced file exists or the document was recently verified.

### 6. Reconcile cross-document consistency

Review the mandatory coverage dimensions in `references/audit-playbook.md`. Check duplicated facts across system overview, service boundaries, Gateway/API, persistence, auth/RBAC, recruitment/notification, AI/MCP/retrieval, resume privacy, billing, frontends, and development/deployment knowledge.

### 7. Triage findings

- Assign stable IDs `KNO-001`, `KNO-002`, and so on.
- Use document verdicts `UNCHANGED`, `STALE`, `CONFLICT`, `CANDIDATE`, or `INVALID` exactly as defined in the audit contract.
- Require claim-level evidence for confirmed stale/conflict findings.
- Distinguish knowledge defects from possible source-code defects. Set `resolution_target` to `KNOWLEDGE`, `SOURCE`, or `DECISION_REQUIRED` without silently choosing policy.
- Assign severity from agent/developer harm, security/data impact, change frequency, and blast radius.
- Record positive controls and verified-current documents as well as problems.

### 8. Build the remediation plan

- Map every open finding to at least one `KREM-NNN` task.
- Keep tasks executable by another Agent without rediscovering evidence, scope, authority level, acceptance criteria, or validation commands.
- Apply knowledge writing levels: L1 mechanical facts, L2 behavior/architecture, L3 protected policy/decision.
- Require human confirmation for L3 work, accepted ADR changes, security/privacy/data-lifecycle policy, public API policy, and tasks whose correct resolution target is uncertain.
- Do not use the plan as authorization to change source code, production systems, or files outside its explicit task scope.

### 9. Render and validate exactly three deliverables

Create the canonical JSON in a temporary directory outside the report output directory, then run:

```bash
node .agents/skills/knowledge-current-state-audit/scripts/render_knowledge_audit.mjs \
  --input <audit.json>
```

Optional arguments:

- `--output-dir <dir>`: defaults to `.knowledge-review/`.
- `--basename <name>`: defaults to `knowledge-current-state-audit-YYYYMMDD-HHMMSS`.
- `--sample`: render contract-safe sample artifacts for a smoke test.

The renderer validates document verdict coverage, finding-to-task traceability, dependencies, and the relationship between verdict and open findings. Do not hand-edit one artifact independently; update the canonical JSON and render all three again.

### 10. Deliver

- Return clickable paths to all three files.
- State the audit verdict, active document counts by verdict, findings by severity/type, coverage gaps, blocked checks, and the first remediation task.
- State that conclusions apply to the recorded commit and working-tree snapshot.

## Output Location

Default to timestamped files under `.knowledge-review/`. Keep this directory ignored by Git. Never overwrite an existing report unless the user explicitly asks.
