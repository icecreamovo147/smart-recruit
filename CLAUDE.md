# Claude Code Adapter

This repository's durable project rules live in `AGENTS.md`. Read `AGENTS.md` first and treat it as the repository constitution.

Claude Code is a provider adapter for the canonical Agent workflow:

1. Use `.agents/skills/spec-harness/SKILL.md` for SPEC, SDD, Harness, single-TASK implementation, review, and repair.
2. Use `.agents/skills/harness-pipeline/SKILL.md` only when the user explicitly requests serial multi-TASK orchestration.
3. Use `.spec/<feature-name>/` as the only executable feature contract and runtime evidence source.

Claude-specific files under `.claude/` may help route commands or subagents, but they must not redefine task sources, review verdicts, state transitions, completion rules, branch policy, or merge behavior.

Legacy and reference material may provide historical context, but pending work must be migrated into a current `.spec/<feature-name>/` contract before implementation.

