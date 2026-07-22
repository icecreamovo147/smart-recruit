# Security Remediation Task State Machine

## States

| State | Terminal | Satisfies dependency | Meaning |
|---|---:|---:|---|
| `PENDING` | No | No | Not evaluated for readiness |
| `READY` | No | No | Dependencies satisfied |
| `AWAITING_CONFIRMATION` | No | No | Exact task authorization is required |
| `IN_PROGRESS` | No | No | Finding revalidated and implementation started |
| `VERIFYING` | No | No | Implementation exists and security checks are running |
| `PASSED` | Yes | Yes | All implementation, acceptance, security, and scope gates passed |
| `ALREADY_RESOLVED` | Yes | Yes | No change needed and the original negative verification passes |
| `FAILED` | Yes | No | In-scope implementation/verification exhausted without success |
| `BLOCKED` | Yes | No | Required environment, authority, dependency, or evidence unavailable |
| `STALE` | Yes | No | Plan no longer matches current code or architecture |
| `SCOPE_GAP` | Yes | No | Correct repair requires unauthorized paths or contract expansion |
| `DEFERRED` | Yes | No | Authorized owner postponed work |
| `ACCEPTED_RISK` | Yes | Yes | Authorized owner accepted the risk with owner and expiry evidence |
| `ABORTED` | Yes | No | User or integrity stop ended the task |

## Allowed transitions

```text
PENDING -> READY | BLOCKED | STALE | DEFERRED
READY -> AWAITING_CONFIRMATION | IN_PROGRESS | ALREADY_RESOLVED | BLOCKED | STALE | SCOPE_GAP
AWAITING_CONFIRMATION -> IN_PROGRESS | DEFERRED | ABORTED
IN_PROGRESS -> VERIFYING | BLOCKED | STALE | SCOPE_GAP | ABORTED
VERIFYING -> PASSED | IN_PROGRESS | FAILED | BLOCKED | SCOPE_GAP | ABORTED
FAILED -> READY
BLOCKED -> READY
STALE -> READY
SCOPE_GAP -> READY
DEFERRED -> READY | ACCEPTED_RISK
```

Reopening a terminal failure requires a note explaining what changed. Never transition directly from `PENDING`, `READY`, or `IN_PROGRESS` to `PASSED`.

## Scheduling rules

- Recalculate `READY` after every terminal transition.
- A dependency is satisfied only by `PASSED`, `ALREADY_RESOLVED`, or `ACCEPTED_RISK`.
- Mark descendants effectively blocked in scheduling output when a dependency is terminal and unsatisfied; do not rewrite them to `BLOCKED` unless the run closes.
- Continue independent tasks after a local blocker.
- Stop the whole run for plan hash mismatch, unsafe worktree overlap, repository corruption, uncontained new P0/P1 risk, or a foundational control failure that invalidates later verification.

## Run state

Use:

- `IN_PROGRESS`
- `AWAITING_CONFIRMATION`
- `REMEDIATION_COMPLETE_PENDING_REAUDIT`
- `REMEDIATION_PARTIAL`
- `REMEDIATION_BLOCKED`
- `REMEDIATION_ABORTED`

Do not use audit verdicts as run states.

