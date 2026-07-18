# TASK-002 Acceptance

- `ACASE-004` / `CHECK-004`: a model-native fake provider that returns fabricated free text cannot bypass required evidence groups. Tests cover job inventory, application listing, candidate lookup, analytics metric families, candidate comparison/match, interview preparation, and offer support across success, missing-parameter, disabled, failure, empty, and direct-fabrication cases.
- Job inventory needs one matching job Tool; candidate detail needs search plus detail; comparison needs job plus candidates/applications; interview/offer needs candidate/application detail plus job detail; every requested analytics metric needs its matching metric Tool. Unsatisfied groups cannot yield business facts.
- Status-change proposal/action tools are never pre-executed and explicit confirmation remains mandatory.
- `ACASE-005` / `CHECK-003`: per-call Agent max iterations controls the actual model/tool loop, is clamped by a safety ceiling, and leaves existing default-call behavior compatible.
- Explicit user confirmation for the `smart-recruit-commons/**` write is recorded in pipeline state/evidence before implementation.
- Scope, Harness, formatting, and knowledge-impact checks pass.
