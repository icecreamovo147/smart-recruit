# GitHub CI Current State Refresh SDD

## Design

The CI workflow is updated in place because the required behavior is purely orchestration-level and does not require product code changes.

## Go CI

Use a matrix over the current Go modules:

- `smart-recruit-proto`
- `smart-recruit-platform-go`
- `smart-recruit-commons`
- `smart-recruit-gateway`
- `smart-recruit-identity-service`
- `smart-recruit-recruitment-service`
- `smart-recruit-interview-service`
- `smart-recruit-offer-service`
- `smart-recruit-notification-service`
- `smart-recruit-ai-agent-service`
- `smart-recruit-analytics-service`
- `smart-recruit-worker-service`

The MySQL migration consistency test is scoped to `smart-recruit-commons`, where the migration package now lives.

## Frontend CI

Install once from the repository root with `pnpm install --frozen-lockfile`, then execute package scripts through `pnpm --filter <app>`.

## Proto CI

The canonical proto source and generated Go contracts live in `smart-recruit-proto`. CI installs generator versions matching the committed generated files, runs `./scripts/generate-go.sh`, then verifies the canonical proto and generated files have an empty diff.

## Knowledge Validation

The knowledge validation workflow validates the current knowledge refresh feature and this CI refresh feature so `.spec/**` changes remain structurally checked in PRs.

## Risks

- Go CI fan-out increases job count.
- MySQL tag tests depend on GitHub Actions service readiness.
