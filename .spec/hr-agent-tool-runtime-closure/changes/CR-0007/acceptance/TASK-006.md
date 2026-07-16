# TASK-006 Acceptance

- `ACASE-012` / `CHECK-013`: HR route-entry and in-chat candidate analysis submit a non-empty planner-recognizable message even when a legacy backend returns no messages.
- `ACASE-012` / `CHECK-014`: blank or whitespace-only durable Run HTTP requests are rejected before RPC dispatch while valid requests remain compatible.
- `ACASE-012` / `CHECK-015`: a newly created application-analysis session persists and returns exactly one non-empty user message requesting resume-to-job match evaluation; blank gRPC Runs are rejected before governance loading, persistence, or dispatch.
- `ACASE-012` / `CHECK-016`: Anthropic System-only input fails locally without HTTP dispatch; valid System + User input serializes a non-empty `messages` array; the canonical analysis message is classified as `candidate_match_evaluation` when an application ID is present.
- The implementation adds no Proto, schema, dependency, lockfile, configuration, or secret changes.
