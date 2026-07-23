# Knowledge Fix Verification Checklist

## Finding and authority

- Reproduce or re-check the original claim before editing.
- Confirm the current higher-authority source: repository instructions, explicit active contract, schema/Proto/code/tests, accepted ADR, then Active knowledge.
- Preserve `resolution_target`; do not repair the opposite side for convenience.
- Identify whether current worktree drift changed the original finding, target, scope, or dependencies.

## Knowledge changes

- Keep frontmatter valid and repository-relative source references resolvable.
- Update `last_verified` only after substantive verification; keep `review_after` defensible.
- Keep Manifest routes narrow and deterministic; update INDEX only for stable navigation.
- Keep Inbox drafts non-authoritative and exclude archive content from current truth.
- Do not rewrite accepted ADR history; propose a superseding decision.
- Do not store secrets, personal data, raw resumes/logs, hidden prompts, private provider payloads, or Agent reasoning.

## Source changes

- Fix the higher-authority root cause, not only a failing fixture or downstream prose.
- Preserve compatibility and behavior outside the finding.
- Add or update focused regression coverage when practical.
- For schema changes, reconcile migrations, baseline schema, models, ownership, fixtures, and deployment assumptions.
- For public contracts, reconcile canonical Proto/schema, generated code, clients, Gateway, and consumers.
- For CI/operations, prove commands actually execute and fail visibly on empty selection or invalid configuration.

## Scope and worktree

- Inspect all pre-existing dirty overlaps before editing.
- Capture the task fingerprint baseline immediately before changes.
- Confirm every changed/added/deleted path matches an allowed pattern and no excluded pattern.
- Treat unexplained concurrent changes as a stop condition, not as task output.
- Do not reset, clean, discard, auto-rollback, stage, commit, push, or deploy.

## Verification

- Evaluate every acceptance criterion explicitly.
- Run task-specific commands and relevant impacted checks.
- Run knowledge validator tests, repository validation, and strict references after knowledge changes when structurally possible.
- Run impact detection with a reliable base tree; record `BLOCKED` and use manual routing when impossible.
- Run relevant business tests after SOURCE changes.
- Run `git diff --check` and inspect the final diff for accidental scope expansion.
- Record every skipped check as `NOT_RUN` or `BLOCKED` with a reason.

## Completion

- Produce the per-task report before transitioning to `PASSED`.
- Preserve residual risk and external evidence gaps.
- Render the cumulative summary.
- Request or execute a fresh `knowledge-current-state-audit` before claiming verified current state.
