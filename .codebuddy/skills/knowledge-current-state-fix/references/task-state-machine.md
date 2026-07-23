# Knowledge Remediation Task State Machine

## States

| State | Terminal | Satisfies dependency | Meaning |
|---|---:|---:|---|
| `PENDING` | No | No | Waiting for dependencies |
| `READY` | No | No | Dependencies are satisfied |
| `REVALIDATING` | No | No | Current evidence is being compared with the audit |
| `AWAITING_CONFIRMATION` | No | No | Exact authorization or a policy decision is required |
| `IN_PROGRESS` | No | No | Finding remains valid and in-scope repair started |
| `VERIFYING` | No | No | Repair exists and gates are running |
| `PASSED` | Yes | Yes | Finding, acceptance, tests, scope, diff, and review passed |
| `ALREADY_RESOLVED` | Yes | Yes | No edit was needed and all verification gates passed |
| `ACCEPTED` | Yes | Yes | Authorized owner accepted the unresolved item with evidence |
| `FAILED` | Yes | No | In-scope repair or verification was exhausted |
| `BLOCKED` | Yes | No | Required evidence, environment, authority, or dependency is unavailable |
| `STALE_PLAN` | Yes | No | Current evidence no longer matches the task contract |
| `SCOPE_GAP` | Yes | No | Correct repair needs unauthorized paths |
| `DEFERRED` | Yes | No | Authorized owner postponed work |
| `ABORTED` | Yes | No | User or integrity stop ended the task |

## Allowed transitions

```text
PENDING -> READY | BLOCKED | STALE_PLAN | DEFERRED
READY -> REVALIDATING | BLOCKED | STALE_PLAN
REVALIDATING -> AWAITING_CONFIRMATION | IN_PROGRESS | ALREADY_RESOLVED | BLOCKED | STALE_PLAN | SCOPE_GAP
AWAITING_CONFIRMATION -> IN_PROGRESS | DEFERRED | ABORTED
IN_PROGRESS -> VERIFYING | BLOCKED | STALE_PLAN | SCOPE_GAP | ABORTED
VERIFYING -> PASSED | IN_PROGRESS | FAILED | BLOCKED | SCOPE_GAP | ABORTED
FAILED -> READY
BLOCKED -> READY
STALE_PLAN -> READY
SCOPE_GAP -> READY
DEFERRED -> READY | ACCEPTED
```

Entering `IN_PROGRESS` for an effective-gated task requires a recorded confirmation. `PASSED` and `ALREADY_RESOLVED` require all six named PASS gates. `ACCEPTED` requires an authorized-owner note and an expiry/review condition; the Agent cannot self-accept.

After each success transition, promote dependency-satisfied `PENDING` tasks to `READY`. Unsuccessful terminal dependencies leave descendants pending and report them as dependency-blocked. Never skip a dependency or mark it satisfied based only on written code.

## Run states

- `IN_PROGRESS`
- `AWAITING_CONFIRMATION`
- `REMEDIATION_COMPLETE_PENDING_REAUDIT`
- `REMEDIATION_PARTIAL`
- `REMEDIATION_BLOCKED`
- `REMEDIATION_ABORTED`

Audit verdicts are not execution states.
