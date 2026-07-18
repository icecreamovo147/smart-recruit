# Legacydomain Retirement SPEC

## 1. Background

The completed `microservice-ddd-evolution` feature localized active legacy business code under service-private `internal/legacydomain` packages. That made shared business cleanup possible, but four service roots still carry active legacy implementation debt:

- `smart-recruit-offer-service/internal/legacydomain`
- `smart-recruit-interview-service/internal/legacydomain`
- `smart-recruit-recruitment-service/internal/legacydomain`
- `smart-recruit-ai-agent-service/internal/legacydomain`

The repository knowledge base also contains active references to these paths. If the code is migrated without knowledge updates, future agents may continue routing to legacy paths.

## 2. Goals

- Remove all service-local `internal/legacydomain` directories from active service roots.
- Remove all non-test imports of `internal/legacydomain`.
- Replace active legacy model/repository/service/AI dependencies with local DDD, infrastructure, owner-contract, or read-model adapters.
- Preserve existing frontend behavior, gateway routes, protobuf-visible behavior, database schema, local runtime startup, and service ownership rules unless a TASK explicitly receives human confirmation.
- Make affected `.knowledge/**` updates part of every implementation TASK scope and require `knowledge_impact` evidence.
- Tighten boundary and ownership checks so future `legacydomain` reintroduction fails validation.

## 3. Non-Goals

- Do not rename `legacydomain` to another compatibility package as a substitute for debt removal.
- Do not change database schema or migrations for this retirement.
- Do not change public HTTP routes, frontend APIs, or user-visible labels.
- Do not move business logic into the gateway.
- Do not introduce new third-party dependencies.
- Do not perform broad module rename or package manifest changes.

## 4. User-Facing Behavior

End users should observe no behavioral change. HR, candidate, and interviewer workflows must continue to work through the existing gateway and protobuf contracts.

## 5. Functional Requirements

- FR-001: New work must live under `.spec/legacydomain-retirement`.
- FR-002: `task-scope.json` for this feature must allow relevant `.knowledge/**` updates in every implementation TASK.
- FR-003: Every implementation TASK must report `knowledge_impact` with reviewed documents, update paths, and stale/conflict/coverage-gap status.
- FR-004: All active references to `internal/legacydomain` in Offer must be replaced by local infrastructure adapters or owner contracts before deleting Offer legacy files.
- FR-005: All active references to `internal/legacydomain` in Interview must be replaced by local infrastructure adapters or owner contracts before deleting Interview legacy files.
- FR-006: Recruitment runtime must stop constructing legacy service/repository graphs and must use local application, infrastructure, and interface adapters.
- FR-007: AI Agent runtime must stop constructing legacy service/repository/AI graphs and must use local application, provider, MQ, persistence, MCP, and gRPC adapters.
- FR-008: Cross-owner writes must not continue through copied legacy repositories. They must use owner service contracts, events, or explicit application ports.
- FR-009: GORM records may exist only in infrastructure packages; domain models must remain transport- and persistence-neutral.
- FR-010: Final validation must fail if any service root contains `internal/legacydomain` or non-test code imports it.

## 6. Non-Functional Requirements

- NFR-001: Preserve DDD dependency direction: domain has no infrastructure imports, application has no local infrastructure imports, interfaces do not import persistence directly.
- NFR-002: Keep each TASK independently reviewable and scoped.
- NFR-003: Use existing Go modules and standard tooling only.
- NFR-004: Keep tests fake/env-gated for AI provider and MQ behavior.
- NFR-005: Do not commit secrets, production logs, credentials, or raw candidate data.

## 7. Compatibility Requirements

- CR-001: Existing protobuf service names and public RPC behavior must remain compatible unless explicitly confirmed for an internal owner contract TASK.
- CR-002: Gateway route modes and frontend API behavior must not change.
- CR-003: MySQL table ownership from `smart-recruit-deploy/mysql-table-ownership.json` remains authoritative.
- CR-004: Local commands `./start-dev.sh`, service `--check`, and service `--serve` behavior must remain compatible.
- CR-005: Existing status keys, permission keys, outbox routing semantics, and candidate-sensitive data handling must be preserved.

## 8. Observability and Debug Requirements

- ODR-001: Service runtime logs must continue using existing logging setup.
- ODR-002: Long-running AI Agent workers must preserve explicit startup/failure visibility.
- ODR-003: TASK reports must record validation commands and results truthfully.
- ODR-004: Knowledge updates must keep source references current after path changes.

## 9. Error Handling and Fallback Requirements

- EHF-001: Existing business error mappings in gRPC adapters must remain compatible.
- EHF-002: AI provider fallback, embedding fallback, MCP policy denial, and stream cancellation behavior must remain explicit.
- EHF-003: Owner-contract failures must fail closed and avoid partial cross-owner writes.
- EHF-004: TASKs must stop on required public API, schema, auth, dependency, or global config changes unless explicitly confirmed.

## 10. Security and Safety Requirements

- SSR-001: Authentication, authorization, RBAC, data scope, JWT, refresh-token, internal gRPC auth, and TLS behavior must not change incidentally.
- SSR-002: MCP private-network restrictions, command allowlists, confirmation policy, and audit behavior must remain compatible.
- SSR-003: Candidate-sensitive data must not be newly logged or persisted outside existing contracts.
- SSR-004: Provider credentials and encryption keys must never be logged or committed.

## 11. Acceptance Criteria

- AC-001: `.spec/legacydomain-retirement` contains SPEC, SDD, TASKS, AGENT_RULES, task scope, acceptance files, scripts, prompts, reports, and docs directories.
- AC-002: Every task allows `.knowledge/**` updates and requires knowledge impact reporting.
- AC-003: Final repository scan finds no `internal/legacydomain` directories in the four target services.
- AC-004: Final repository scan finds no non-test `legacydomain` imports.
- AC-005: Boundary and table ownership checks pass after final cleanup.
- AC-006: Touched Go service modules pass `go test ./...`.
- AC-007: Knowledge validation and reference checks pass after knowledge updates.

## 12. Out of Scope

- Frontend feature changes.
- Database migrations or schema edits.
- Production deployment changes.
- Dependency upgrades.
- Repository-root feature documentation outside `.spec/legacydomain-retirement`.

## 13. Assumptions Requiring Confirmation

- ARC-001: Internal owner-contract protobuf additions are acceptable only in TASKs that set `requiresHumanConfirmation: true`.
- ARC-002: If an existing public protobuf method cannot safely express an owner-internal lifecycle operation, a new internal RPC is preferred over direct cross-owner table writes.

## 14. Open Questions

- OQ-001: The final exact wire shape for Recruitment application lifecycle and Identity authorization contracts must be decided inside the confirmed owner-contract TASK.
