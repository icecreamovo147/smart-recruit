# Contract v2 reference

Read this file completely for every schema v2 feature.

## Artifact layout

```text
.spec/<feature>/
  contract.json
  <feature>-SPEC.md
  <feature>-SDD.md
  TASKS.md
  AGENT_RULES.md
  task-scope.json
  traceability.json
  acceptance/
  prompts/
    implement-task.md
    self-review.md
    fix-check-failures.md
    propose-amendment.md
    review-amendment.md
    reconcile-plan.md
  scripts/
    check-task-scope.sh
    agent-check.sh
  changes/
  revisions/
  reviews/
  reports/
  docs/
  pipeline-state.json
```

The migration profile also requires `baseline/behavior-manifest.json`.

## Ownership

- Contract-owned: SPEC, SDD, TASKS, rules, `contract.json`, `task-scope.json`, `traceability.json`, acceptance, baseline, CRs, and revision manifests.
- Runtime-owned: reports, evidence, `pipeline-state.json`, and pipeline summary.
- Business-owned: source, tests, schema, configuration, and runtime code allowed by a TASK.

Ordinary TASKs may modify business-owned files only. Amendment modes modify contract-owned files. The orchestrator writes runtime-owned files.

## `contract.json`

```json
{
  "schemaVersion": 2,
  "feature_name": "example",
  "profile": "feature_delivery",
  "contractRevision": 1,
  "planningStatus": "approved",
  "rollingWindow": 2,
  "requiredCompletionLevel": "behavior_verified",
  "baseline": null,
  "assumptions": [
    {
      "id": "ASM-001",
      "statement": "Existing storage supports the workflow",
      "status": "verified",
      "blocking": true,
      "evidenceRefs": ["repo://path#symbol"],
      "affects": ["FR-001", "TASK-001"]
    }
  ],
  "reviewPolicy": {
    "plan": "independent_required",
    "highRiskTask": "independent_required",
    "amendment": "independent_required",
    "deleteSource": "independent_and_human"
  }
}
```

Allowed profiles:

- `feature_delivery`
- `behavior_preserving_migration`

Allowed planning states:

- `draft`
- `review_pending`
- `approved`
- `amendment_pending`
- `blocked`

Do not use numeric model confidence as evidence. Use explicit assumption status and evidence references.

An approved initial revision requires `reviews/PLAN-REV-XXXX.json` bound to the canonical `contract_hash`, with `outcome=pass` and `verdict=通过`. When `reviewPolicy.plan=independent_required`, record distinct planner and Reviewer run IDs. Later approved revisions may use the applied CR approval and revision manifest as their plan-review authority.

## `traceability.json`

```json
{
  "schemaVersion": 2,
  "feature_name": "example",
  "contractRevision": 1,
  "requirements": [
    {
      "id": "FR-001",
      "mandatory": true,
      "status": "planned",
      "design_refs": ["example-SDD.md#workflow"],
      "task_ids": ["TASK-001"],
      "acceptance_ids": ["ACASE-001"],
      "check_ids": ["CHECK-001"],
      "evidence_refs": []
    }
  ]
}
```

Use stable IDs for functional, non-functional, API, RPC, flow, permission, event, data, and fallback requirements. Mandatory requirements must map to design, TASK, acceptance, and check before implementation. At completion they must be `verified` or `approved_delta` with evidence. `approved_delta` requires a user-approved contract reference.

## `task-scope.json`

```json
{
  "schemaVersion": 2,
  "feature_name": "example",
  "contractRevision": 1,
  "tasks": {
    "TASK-001": {
      "title": "Complete one vertical behavior",
      "lifecycle": "ready",
      "requirements": ["FR-001"],
      "behaviorSurfaces": ["FLOW-001"],
      "dependencies": [],
      "allowedFiles": ["service/**", "service/**/*_test.go"],
      "forbiddenFiles": [],
      "allowedActions": ["modify", "create", "test"],
      "requiresHumanConfirmation": false,
      "reviewPolicy": "self_allowed",
      "destructiveActions": [],
      "requiredChecks": [
        {
          "id": "CHECK-001",
          "kind": "integration",
          "blocking": true,
          "covers": ["FR-001", "FLOW-001"],
          "command": "go test ./path/...",
          "successCriteria": "The seeded workflow reaches its expected final state"
        }
      ],
      "acceptance": ".spec/example/acceptance/TASK-001.md",
      "report": ".spec/example/reports/TASK-001-report.md"
    }
  }
}
```

Allowed lifecycles:

- `draft`
- `ready`
- `in_progress`
- `review`
- `completed`
- `amendment_pending`
- `needs_revalidation`
- `superseded`
- `blocked`

Allowed check kinds:

- `static`
- `build`
- `unit`
- `integration`
- `contract`
- `differential`
- `e2e`
- `manual`

Allowed destructive actions:

- `delete_files`
- `bulk_delete`
- `delete_source`
- `cutover`
- `schema_change`
- `public_api_change`
- `security_change`

Every destructive TASK requires human confirmation and independent Review. Keep cutover and source deletion in separate TASKs.

## Evidence v2

```json
{
  "schemaVersion": 2,
  "feature_name": "example",
  "task_id": "TASK-001",
  "contract_revision": 1,
  "task_definition_hash": "...",
  "traceability_hash": "...",
  "base_sha": "...",
  "head_sha": "...",
  "changed_files": [],
  "scope": { "status": "passed", "out_of_scope": [], "forbidden": [] },
  "checks": [
    {
      "id": "CHECK-001",
      "kind": "integration",
      "covers": ["FR-001", "FLOW-001"],
      "command": "go test ./path/...",
      "exit_code": 0,
      "status": "passed",
      "started_at": "...",
      "duration_ms": 123,
      "artifact": "reports/artifacts/CHECK-001.json"
    }
  ],
  "skipped_checks": [],
  "coverage_claims": [{ "id": "FR-001", "status": "verified" }],
  "review": {
    "reviewer_type": "independent_agent",
    "implementer_run_id": "...",
    "reviewer_run_id": "...",
    "outcome": "pass",
    "verdict": "通过",
    "round": 1
  },
  "human_confirmation": { "required": false, "confirmed": false },
  "exceptions": []
}
```

Every required check ID must have evidence. Blocking checks must pass. Advisory skipped checks require a reason. Extra diagnostic checks do not compensate for missing required checks.

## Contract Amendment

Store `changes/CR-XXXX.json` and `.md`.

```json
{
  "schemaVersion": 2,
  "changeRequestId": "CR-0001",
  "feature_name": "example",
  "baseRevision": 1,
  "status": "proposed",
  "classification": "contract_gap",
  "approvalLevel": "L1",
  "reason": "Observed runtime ownership differs from the planned boundary",
  "evidenceRefs": ["reports/TASK-001-report.md#finding"],
  "trigger": { "task_id": "TASK-001", "phase": "implement" },
  "impact": {
    "requirements": ["FR-001"],
    "tasks": ["TASK-001", "TASK-002"],
    "completedTasksRequiringRevalidation": []
  },
  "revalidationPlan": ["TASK-001"],
  "resultingRevision": null,
  "updates": []
}
```

Allowed classifications:

- `contract_gap`
- `requirement_conflict`
- `scope_correction`
- `baseline_change`
- `approved_behavior_change`

An approved/applied CR sets `resultingRevision=baseRevision+1` and records approval metadata. L2 requires `approved_by=user`.

For application, stage each complete replacement under `changes/<CR-ID>/` and add:

```json
{
  "target": "task-scope.json",
  "staged_file": "changes/CR-0001/task-scope.json",
  "before_sha256": "<64 hex chars or null for a new file>",
  "after_sha256": "<64 hex chars>"
}
```

An applied CR must stage `contract.json`. The application script restricts targets to contract-owned files, validates hashes and the resulting revision, updates pipeline revalidation state, creates `revisions/REV-XXXX.json`, and restores previous files if validation fails.

Applying a CR must update affected artifacts in one logical change, create a revision manifest, bind new hashes, and mark impacted completed TASKs `needs_revalidation`.

## Review routing

| Outcome | Route |
|---|---|
| `pass` | complete TASK and reconcile plan |
| `implementation_defect` | fix and review again |
| `contract_gap` | propose Amendment |
| `requirement_conflict` | Amendment and human decision |
| `environment_blocker` | block at current completion level |

## Completion

Feature-delivery completion levels:

```text
planned → code_complete → integration_verified → behavior_verified → completed
```

The contract declares the required level. Completed state requires no open CR, no revalidation debt, verified mandatory traceability, passing blocking checks, and the required completion level.
