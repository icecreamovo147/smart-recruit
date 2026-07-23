# Repository Guidelines

## Project Structure & Module Organization

- `hr-frontend/`, `user-frontend/`, `platform-frontend/`: Vue 3 + Vite apps for the staff workspace, candidate portal, and platform console. App-specific source lives in `src/` with `api/`, `components/`, `views/`, `stores/`, `types/`, and `assets/`.
- `packages/shared/`: explicit cross-app frontend package for shared components, types, utilities, and brand assets imported through the `@shared/*` alias.
- `smart-recruit-gateway/`: Go HTTP gateway. Routes live in `router/`, handlers in `handler/`, backend clients in `rpc/`, middleware in `middleware/`.
- `smart-recruit-identity-service/`, `smart-recruit-recruitment-service/`, `smart-recruit-interview-service/`, `smart-recruit-offer-service/`, `smart-recruit-notification-service/`, `smart-recruit-ai-agent-service/`, `smart-recruit-analytics-service/`, `smart-recruit-billing-service/`, `smart-recruit-worker-service/`: independently buildable Go service roots.
- `smart-recruit-commons/`: shared domain support, migrations, MQ, OSS, AI, email, authz/JWT helpers, resume parsing, and command-line utilities such as the migration runner.
- `smart-recruit-platform-go/`: shared runtime platform packages for config, Nacos, logging, health, metrics, trace, metadata, and gRPC helpers.
- `smart-recruit-proto/`: canonical protobuf source and generated Go contracts.
- `dev-log-viewer/`: independently buildable local Go/React log-viewing utility and pnpm workspace package.
- `smart-recruit-deploy/`, `deploy/`, `docker/`: microservice deployment assets, Kubernetes manifests, and the local/full-stack Docker Compose setup.
- `docs/`, `db.sql`: documentation and the baseline database schema.

## Build, Test, and Development Commands

- `pnpm install`: install workspace frontend dependencies.
- `pnpm --filter hr-frontend dev`: run the HR app on port `5173`.
- `pnpm --filter user-frontend dev`: run the user app on port `5174`.
- `pnpm --filter platform-frontend dev`: run the platform console on port `5175`.
- `pnpm --filter hr-frontend build`: build one frontend app; replace the filter as needed.
- `pnpm --filter hr-frontend typecheck`: run Vue TypeScript checks.
- `pnpm --filter hr-frontend test`: run Vitest.
- `go test ./...`: run Go tests from any `smart-recruit-*` Go module.
- `smart-recruit-proto/scripts/bootstrap-tools.sh`: install the repository-pinned protobuf compiler and Go plugins into the local tool cache.
- `smart-recruit-proto/scripts/generate-go.sh`: regenerate canonical Go protobuf contracts; it refuses unpinned tool versions.
- `./start-dev.sh` / `./stop-dev.sh`: manage the local development stack.

## Coding Style & Naming Conventions

Use TypeScript and Vue single-file components. Prefer PascalCase for components, camelCase for variables/functions, and kebab-case for route paths. Keep app-specific reusable UI in that app's `src/components/` and pages in `src/views/`. Put deliberately cross-app components, types, utilities, and assets in `packages/shared/src/`; do not import source directly from another frontend app.

When adding a new HR left-menu page, keep the `page-header` styling consistent with the existing Agent Management and Prompt Management pages. Reuse the same layout, spacing, title/description treatment, and primary/secondary action placement before introducing page-specific variations.

Use `gofmt` for Go code. Go package names should be short, lowercase, and aligned with existing directories. Keep transport concerns in handlers/routers and business rules in service packages.

## Implementation Guidelines

For feature work or refactoring, first analyze the current code structure, ownership boundaries, and established patterns. Implement the best-practice solution for this repository rather than a casual minimum-change patch. Keep the scope focused, but choose designs that remain maintainable as the product grows.

When asked to analyze current code and propose an implementation plan, provide an enterprise-grade best-practice plan grounded in the existing frontend/backend architecture, data model, API contracts, testing strategy, and operational risks. Do not frame the plan as a minimal version, quick fix, or smallest possible change unless the user explicitly asks for that tradeoff.

## Testing Guidelines

Frontend tests use Vitest and Vue Test Utils. Place tests near covered code or existing test folders, and use `*.test.ts` naming. Always run `typecheck` for touched apps.

Go tests use the standard `testing` package. Use `*_test.go` files and table-driven tests for parser, validation, and service behavior.

## Commit & Pull Request Guidelines

History follows Conventional Commit style, for example `feat: add database backed agent skills` and `fix(P1-006): increase web-gin HTTP timeout`.

Use concise commit subjects with `feat`, `fix`, `refactor`, `test`, or `docs`; include a scope when helpful. Pull requests should describe the change, list verification commands, link related issues, and include screenshots for UI changes.

## Security & Configuration Tips

Do not commit secrets, tokens, or local credentials. Keep environment-specific settings in ignored local config files. When changing auth, AI, MCP, or recruitment data flows, document risks and add focused regression tests.

## Optional SPEC + SDD + Harness Workflow

### Explicit activation only

`spec-harness` is opt-in. Do not proactively invoke or read the `spec-harness` skill, create a `.spec/<feature-name>/` feature, or impose the SPEC/SDD/TASK/Harness lifecycle merely because a request is non-trivial, asks for analysis or an implementation plan, or involves implementing, testing, reviewing, or fixing code.

Use `spec-harness` only when the user explicitly names `spec-harness` (including `$spec-harness`) or explicitly instructs the Agent to use the repository Harness workflow. A request for a SPEC, SDD, implementation plan, task breakdown, acceptance criteria, code implementation, review, or fix does not by itself activate `spec-harness`. The existence of an applicable `.spec` directory or a reference to an existing feature/TASK does not activate the skill unless the user also asks to use the Harness workflow. `harness-pipeline` is likewise used only when the user explicitly requests multi-TASK pipeline execution.

Without explicit activation, follow the normal repository analysis, implementation, review, and testing guidelines directly in the current task. Do not create Harness artifacts or require Harness phases, reports, self-review rounds, or fixer rounds.

### Authority and source of truth

When the user explicitly activates the Harness workflow, use this canonical control plane:

1. `AGENTS.md` defines durable repository-wide rules.
2. `.agents/skills/spec-harness/SKILL.md` defines the feature and single-TASK lifecycle.
3. `.agents/skills/harness-pipeline/SKILL.md` defines serial multi-TASK orchestration only when the user explicitly invokes the pipeline.
4. `.spec/<feature-name>/` is the only executable feature contract and runtime evidence location.

Provider-specific files for Codex, Claude Code, or other agents may adapt to this control plane, but must not redefine TASK sources, review verdicts, state transitions, or completion rules. Historical plans and execution logs are reference material unless they have been migrated into a current `.spec/<feature-name>/` contract.

`pipeline-state.json` is the runtime status source for a feature. A `status` value in `task-scope.json` is only its generated initial state and must not override pipeline evidence.

### Development knowledge protocol

`.knowledge/` is the canonical control plane's downstream project knowledge layer for coding Agents, developers, and reviewers. It may guide navigation, impact review, runbooks, and recurring pitfalls, but ordinary knowledge cannot override `AGENTS.md`, active SPEC/SDD/TASK/acceptance files, source code, tests, schema, protobuf definitions, or runtime evidence.

For every non-trivial task:

1. Read `AGENTS.md` and `.knowledge/README.md`. Read an active `.spec/<feature-name>/` contract only when the user explicitly activated its Harness workflow.
2. Use `.knowledge/manifest.yaml` routes and `.knowledge/INDEX.md` to select only relevant active knowledge.
3. Verify critical knowledge claims against each document's `source_refs`.
4. For an explicitly activated Harness TASK, run knowledge impact detection when `.knowledge/scripts/detect-impact.mjs` is available and a reliable TASK base tree exists, then report `knowledge_impact` in TASK evidence.

For an explicitly activated Harness TASK, if TASK scope does not allow updating affected knowledge, record `STALE`, `CANDIDATE`, or `coverage_gap` debt in the report instead of editing scope-out files. For ordinary non-Harness work, report discovered stale knowledge or coverage gaps directly to the user and update them only when the request authorizes documentation changes. Do not read `inbox/`, `archive/`, stale, deprecated, or archived knowledge by default. Do not create provider-specific knowledge copies; adapters such as `CLAUDE.md` must keep pointing at this canonical entry.

### Rules when explicitly activated

Only after explicit Harness activation, create or use the feature directory:

`.spec/<feature-name>/`

The feature directory should contain:

- `<feature-name>-SPEC.md`
- `<feature-name>-SDD.md`
- `TASKS.md`
- `AGENT_RULES.md`
- `task-scope.json`
- `acceptance/`
- `prompts/`
- `scripts/`
- `reports/`

Required Harness development order:

SPEC -> SDD -> TASKS -> Harness -> single TASK implementation -> Harness checks -> TASK completion report -> user confirmation -> next TASK.

Harness hard rules:

- Do not implement without SPEC and SDD.
- Do not implement without `TASKS.md`.
- Execute only one TASK at a time.
- Do not modify files outside the current TASK scope.
- Do not refactor unrelated code opportunistically.
- Do not mass-format unrelated files.
- Do not modify `package.json`, lockfiles, or global config unless the current TASK explicitly allows it.
- Do not use `any` casually to pass type checks.
- Do not swallow exceptions.
- Do not delete existing tests.
- If a change requires modifying shared modules, shared types, global config, or public API behavior, stop and request confirmation.

After each TASK, run or explain why unable to run:

- `git diff --name-only`
- `bash .spec/<feature-name>/scripts/check-task-scope.sh <TASK-ID>`
- `bash .spec/<feature-name>/scripts/agent-check.sh`

After each TASK, output a report with:

- TASK ID
- Modified file list
- Change summary for each file
- Whether the changes exceed task scope
- SPEC comparison result
- SDD comparison result
- Acceptance comparison result
- Test commands and results
- Risks
- Whether the next TASK can start
