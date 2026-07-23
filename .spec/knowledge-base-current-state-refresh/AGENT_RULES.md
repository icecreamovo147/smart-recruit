# Agent Rules - knowledge-base-current-state-refresh

## Scope Rules

- Execute only `TASK-KBR-001`.
- Modify only `.knowledge/**` and `.spec/knowledge-base-current-state-refresh/**`.
- Do not modify product source code, generated code, database schemas, root docs, package manifests, lockfiles, or deployment behavior.
- Do not update `.knowledge/inbox/**` or `.knowledge/archive/**`; report their remaining legacy references instead.

## Knowledge Rules

- Current source code, tests, proto files, deploy files, and table ownership manifests outrank old knowledge prose.
- Replace unverifiable claims with narrower verified statements.
- Keep source references exact and repository-relative.
- Keep active documents concise and route-focused.

## Validation Rules

Run and report:

- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- legacy reference grep for `.knowledge` excluding inbox/archive
- `git diff --name-only`
- TASK scope check
- feature agent check

## Hard Stops

Stop before changing public APIs, protobuf definitions, database schema, package manifests, lockfiles, business code, root docs, or archived/inbox knowledge content.
