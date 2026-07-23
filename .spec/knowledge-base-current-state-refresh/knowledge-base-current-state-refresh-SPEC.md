# Knowledge Base Current State Refresh SPEC

## 1. Background

The repository has completed substantial backend DDD and microservice extraction work. Several active `.knowledge/` documents still reference the former `web-gin-service/` and `logic-grpc-service/` layout, and current knowledge validation fails because those `source_refs` no longer exist.

## 2. Goals

- Refresh active formal knowledge to reflect current source code boundaries.
- Update knowledge routing so future TASK impact checks point at current modules.
- Keep draft inbox and archived historical material out of this refresh.
- Restore local knowledge validation and reference checks.

## 3. Non-Goals

- Do not modify business code, frontend code, runtime behavior, protobuf contracts, database schema, package manifests, or lockfiles.
- Do not rewrite inbox drafts or archived historical documents.
- Do not create new architecture decisions.

## 4. User-Facing Behavior

There is no product UI or API behavior change. The user-facing result is a reliable developer/Agent knowledge base for future work.

## 5. Functional Requirements

- Active knowledge must describe current module ownership using `smart-recruit-gateway/`, `smart-recruit-*-service/`, `smart-recruit-proto/`, `smart-recruit-platform-go/`, `smart-recruit-commons/`, and `smart-recruit-deploy/`.
- `source_refs` in active formal knowledge must point to existing repository files.
- The manifest routes must no longer route current work through deleted `web-gin-service/` or `logic-grpc-service/` paths.
- Knowledge tooling fixtures must stop using deleted legacy paths when those fixtures are part of active validation.
- Reports must explicitly record that inbox/archive legacy references remain out of scope.

## 6. Non-Functional Requirements

- Changes must be deterministic, repository-relative, ASCII-compatible, and validation-friendly.
- The update must remain concise enough for Agents to route future tasks without loading the entire knowledge base.

## 7. Compatibility Requirements

- Preserve existing `.knowledge` schema version and frontmatter format.
- Preserve existing validation commands and their CLI names.
- Preserve existing harness validation behavior for current feature packages.

## 8. Observability and Debug Requirements

- The TASK report and evidence must include validation command results.
- Legacy reference checks must be recorded with their scope and outcome.

## 9. Error Handling and Fallback Requirements

- If a knowledge claim cannot be verified against current code, remove or narrow the claim instead of migrating stale text.
- If draft or archived documents still contain legacy paths, report them as intentionally out of scope.

## 10. Security and Safety Requirements

- Do not add secrets, credentials, production logs, candidate data, HR data, or hidden Agent reasoning.
- Do not weaken security, auth, or public-contract policy.

## 11. Acceptance Criteria

- `node .knowledge/scripts/knowledge-validator.test.mjs` passes.
- `node .knowledge/scripts/validate-knowledge.mjs --root .` passes.
- `node .knowledge/scripts/check-references.mjs --root .` passes.
- Active `.knowledge` and knowledge tooling no longer contain `web-gin-service` or `logic-grpc-service`.
- TASK scope check and feature agent check pass.

## 12. Out of Scope

- `.knowledge/inbox/**`
- `.knowledge/archive/**`
- Business source code changes
- Public API, protobuf, schema, package, or lockfile changes

## 13. Assumptions Requiring Confirmation

None. The user approved active-only knowledge refresh.

## 14. Open Questions

None.
