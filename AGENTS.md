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

## Testing Guidelines

Frontend tests use Vitest and Vue Test Utils. Place tests near covered code or existing test folders, and use `*.test.ts` naming. Always run `typecheck` for touched apps.

Go tests use the standard `testing` package. Use `*_test.go` files and table-driven tests for parser, validation, and service behavior.

## Commit & Pull Request Guidelines

History follows Conventional Commit style, for example `feat: add database backed agent skills` and `fix(P1-006): increase web-gin HTTP timeout`.

Use concise commit subjects with `feat`, `fix`, `refactor`, `test`, or `docs`; include a scope when helpful. Pull requests should describe the change, list verification commands, link related issues, and include screenshots for UI changes.

## Security & Configuration Tips

Do not commit secrets, tokens, or local credentials. Keep environment-specific settings in ignored local config files. When changing auth, AI, MCP, or recruitment data flows, document risks and add focused regression tests.
