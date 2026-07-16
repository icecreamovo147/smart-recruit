# TASK-001 Acceptance

- `ACASE-001` / `CHECK-001`: configured Agent with empty, cleared, disabled-only, abstract-only, unknown, or unimplemented bindings exposes zero builtin business tools; one enabled concrete binding exposes only that tool.
- `ACASE-002` / `CHECK-002`: empty `skill_capability_keys` executes zero MCP tools; explicit selection executes only the matching enabled binding.
- `ACASE-003` / `CHECK-001`, `CHECK-002`: invalid arguments, unauthorized/not-found domain data, unsupported tools, and downstream non-success responses produce non-nil errors, error Trace/Step status, and never count as useful results.
- HR identity remains present on every Recruitment RPC and no direct database read is introduced.
- Scope, Harness, formatting, and knowledge-impact checks pass.
