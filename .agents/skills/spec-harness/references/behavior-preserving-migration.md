# Behavior-preserving migration profile

Read this file completely for microservice extraction, DDD restructuring, module/runtime replacement, gateway cutover, storage migration, or legacy retirement when existing behavior must remain compatible.

## Principle

Treat the pinned source implementation as an executable behavior contract. Architecture may change; behavior may change only through an L2 user-approved `approved_delta`.

Do not redefine an incomplete intermediate migration as the new baseline.

## Baseline

`contract.json` and `baseline/behavior-manifest.json` must pin:

```json
{
  "baseline": {
    "ref": "dev",
    "sha": "<immutable commit>",
    "tree": "<tree id>",
    "immutable": true
  }
}
```

Also record applicable fixture, schema, protobuf, configuration, and test hashes. Read the pinned tree with `git show`, a read-only worktree, or equivalent. Never write to the source baseline.

Inventory all externally or operationally reachable surfaces:

- HTTP routes, middleware, callers, and response fields;
- gRPC methods and routing ownership;
- MQ topics/routing keys, producers, consumers, retry and DLQ;
- scheduled jobs and background workers;
- database reads, writes, joins, table names, transactions, and locking;
- OSS, cache, email, notification, audit, and provider effects;
- authentication, ownership, role, and data-scope rules;
- errors, fallback, idempotency, crash recovery, and temporal behavior.

## Behavior manifest

```json
{
  "schemaVersion": 2,
  "feature_name": "example-migration",
  "contractRevision": 1,
  "baseline": { "ref": "dev", "sha": "...", "tree": "...", "immutable": true },
  "capabilities": [
    {
      "id": "CAP-RESUME-UPLOAD-PARSE",
      "mandatory": true,
      "source": {
        "entrypoints": ["HTTP-RESUME-UPLOAD"],
        "implementation_refs": ["dev:path/file.go#Function"],
        "tests": ["dev:path/file_test.go"]
      },
      "target": { "implementation_refs": [] },
      "dimensions": {
        "transport": { "status": "unknown", "evidence_refs": [] },
        "fields": { "status": "unknown", "evidence_refs": [] },
        "state": { "status": "unknown", "evidence_refs": [] },
        "persistence": { "status": "unknown", "evidence_refs": [] },
        "side_effects": { "status": "unknown", "evidence_refs": [] },
        "security": { "status": "unknown", "evidence_refs": [] },
        "failure_behavior": { "status": "unknown", "evidence_refs": [] },
        "operational_behavior": { "status": "unknown", "evidence_refs": [] }
      },
      "events": [
        {
          "routing_key": "resume.parse",
          "producer_refs": ["target:producer.go"],
          "consumer_refs": ["target:consumer.go"],
          "final_effect_checks": ["CHECK-RESUME-PARSED-TEXT"]
        }
      ]
    }
  ]
}
```

Parity statuses:

- `verified_equal`
- `approved_delta`
- `missing`
- `unknown`
- `blocked`
- `not_applicable`

`approved_delta` requires a user-approved CR reference. Mandatory capabilities may complete only with `verified_equal`, `approved_delta`, or justified `not_applicable` in every dimension, plus evidence.

## TASK design

Partition by a closed user or operational journey, not only by architecture layer.

Prefer:

```text
upload → outbox → consumer → parsing → persisted text → query response
```

Avoid claiming migration completion for:

```text
domain skeleton → repository skeleton → service registration
```

Tasks may intentionally span Gateway, service, worker, persistence, and tests when that is the smallest behaviorally closed slice. Keep write scope explicit; allow read-only verification across the full journey.

Separate these concerns:

1. source characterization;
2. target implementation;
3. differential verification;
4. cutover and rollback;
5. observed stability;
6. source retirement.

Do not combine cutover and source deletion.

## Test lineage and differential evidence

Treat source tests as migration assets. Before replacing behavior:

- preserve or black-box source characterization tests;
- map every source test to a target/differential check;
- do not replace a working source expectation with a target test that declares `unsupported` correct.

Run the same fixture against source and target. Compare applicable dimensions:

- status/error and every response field;
- streaming sequence, duration, replay, and terminal state;
- DB before/after and transaction outcome;
- outbox/event type, payload, ordering, consumer effect, retry, and DLQ;
- OSS/cache/email/notification/audit effects;
- actor × role × data-scope allow/deny matrix;
- idempotency and process restart recovery.

AI behavior may use a deterministic fake or recorded provider. Text may use structural invariants where exact equality is inappropriate; persistence, permissions, state, routing, and side effects remain exact.

Bind evidence to source commit, target commit, fixture hash, capability revision, normalization rules, source result, target result, and diff.

## Cutover gate

Before `cutover_ready`, require:

- all affected mandatory capabilities parity-verified;
- complete Gateway-to-owner route mapping;
- production wiring tests, not noop registration;
- closed producer/consumer graph and final-effect assertions;
- exact schema/table mapping and seeded integration;
- security and data-scope differential checks;
- no reachable stub, placeholder, fixed 404/501, empty success, or conservative unavailable response;
- an executable rollback and reconciliation plan;
- no open contract gap or high-risk follow-up.

Static checks, build, registration, any-4xx smoke, and mock-only tests cannot satisfy this gate.

## Source-retirement gate

Before deleting the source implementation, require:

- `cutover_observed` evidence or an explicitly approved equivalent environment;
- all mandatory parity dimensions verified;
- source tests retained through the observation period;
- no traffic or dependency on the source path;
- independent parity Review and user confirmation;
- deletion-size anomaly Review;
- no unapproved risk, follow-up, stub, or unknown capability.

More than 20 deleted files requires `bulk_delete` or `delete_source` declaration. All destructive actions require independent Review and human confirmation.

## Completion levels

```text
discovered
→ baseline_locked
→ implemented
→ parity_verified
→ cutover_ready
→ cutover_observed
→ retirement_ready
→ retired
```

Set `requiredCompletionLevel` to the actual feature objective. A migration implementation feature may stop at `parity_verified`; a feature that promises legacy retirement must reach `retired`.

If a live environment is unavailable, preserve the achieved level and set the pipeline to blocked with `environment_blocker`. Do not report ordinary completion.
