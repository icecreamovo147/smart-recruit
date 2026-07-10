# Claude Code Local Adapter

This file is not an independent rule source.

Before acting, read the root `CLAUDE.md`, then `AGENTS.md`, then the relevant canonical skill:

- `.agents/skills/spec-harness/SKILL.md`
- `.agents/skills/harness-pipeline/SKILL.md` only for explicit pipeline runs

Executable feature work must come from `.spec/<feature-name>/`.

Provider-specific Claude skills and agents in this directory are thin adapters. If they conflict with `AGENTS.md`, `.agents/skills/**`, or `.spec/<feature-name>/`, the canonical sources win.

