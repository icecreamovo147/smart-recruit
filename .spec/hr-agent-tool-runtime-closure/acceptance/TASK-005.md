# TASK-005 Acceptance

- `ACASE-010` / `CHECK-008`: HR-wide application aggregation enforces 100 jobs, 10 pages/job, 5,000 rows, and four workers; honors cancellation; orders by job ID/application ID; returns successful rows with partial metadata when some jobs fail; and returns a non-nil error/no useful result when all jobs fail.
- `ACASE-011` / `CHECK-012`: all routed active knowledge accurately describes the final Tool, Prompt, Skill, Run, MCP, and service-boundary behavior and validates.
- `ACASE-020` / `CHECK-009`, `CHECK-010`, `CHECK-011`: cumulative AI Agent, shared AI, and HR frontend suites verify all mandatory requirements, including “现在有哪些岗位” returning only Tool data.
- All feature/traceability/pipeline validators pass with no open CR, revalidation debt, failed checks, or unverified mandatory requirement.
