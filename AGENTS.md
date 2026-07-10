# Repository Guidelines

## Project Structure & Module Organization

- `hr-frontend/`, `user-frontend/`, `interviewer-frontend/`: Vue 3 + Vite apps for HR, candidate, and interviewer users. Source lives in `src/` with `api/`, `components/`, `views/`, `stores/`, `types/`, and `assets/`.
- `logic-grpc-service/`: core Go gRPC service. Business logic is mainly in `service/`, persistence in `repository/` and `model/`, protobufs in `proto/` and generated code in `recruitment/pb/`.
- `web-gin-service/`: Go HTTP gateway. Routes live in `router/`, handlers in `handler/`, backend clients in `rpc/`, middleware in `middleware/`.
- `docs/`, `deploy/`, `docker/`, `db.sql`: documentation, deployment assets, local infrastructure, and database schema.

## Build, Test, and Development Commands

- `pnpm install`: install workspace frontend dependencies.
- `pnpm --filter hr-frontend dev`: run the HR app on port `5173`.
- `pnpm --filter user-frontend dev`: run the user app on port `5174`.
- `pnpm --filter interviewer-frontend dev`: run the interviewer app on port `5175`.
- `pnpm --filter hr-frontend build`: build one frontend app; replace the filter as needed.
- `pnpm --filter hr-frontend typecheck`: run Vue TypeScript checks.
- `pnpm --filter hr-frontend test`: run Vitest.
- `go test ./...`: run Go tests from either `logic-grpc-service/` or `web-gin-service/`.
- `./start-dev.sh` / `./stop-dev.sh`: manage the local development stack.

## Coding Style & Naming Conventions

Use TypeScript and Vue single-file components. Prefer PascalCase for components, camelCase for variables/functions, and kebab-case for route paths. Keep reusable UI in `src/components/` and pages in `src/views/`.

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

## SPEC + SDD + Harness Workflow

For every non-trivial feature, create a feature directory first:

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

Required development order:

SPEC -> SDD -> TASKS -> Harness -> single TASK implementation -> Harness checks -> TASK completion report -> user confirmation -> next TASK.

Hard rules:

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
