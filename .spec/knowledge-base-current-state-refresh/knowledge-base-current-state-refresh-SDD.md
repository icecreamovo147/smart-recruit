# Knowledge Base Current State Refresh SDD

## 1. Existing Architecture Summary

The active backend is split across `smart-recruit-gateway/`, independent `smart-recruit-*-service/` modules, shared protocol definitions in `smart-recruit-proto/`, platform utilities in `smart-recruit-platform-go/`, shared domain/runtime support in `smart-recruit-commons/`, and deployment assets in `smart-recruit-deploy/`.

## 2. Problem Analysis

Many active knowledge entries and manifest routes still describe the deleted `web-gin-service/` and `logic-grpc-service/` layout. This causes missing `source_ref` validation failures and can route future TASKs to stale knowledge.

## 3. Proposed Design

Refresh active knowledge at the document level:

- Replace deleted path references with current module owners.
- Rewrite stale monolith/cutover language to current routed microservice behavior.
- Keep statements grounded in source files, service runtime files, proto definitions, ownership manifest, and deployment compose files.
- Adjust validation fixtures so they exercise current paths.

## 4. Data Structure Changes

No product data structure changes. Knowledge frontmatter remains schema version 1.

## 5. API and Interface Changes

No product API changes. Knowledge scripts keep the same command-line interface.

## 6. Algorithm or Workflow Changes

Knowledge validation should treat active formal knowledge as the default source of truth and should not fail because intentionally excluded draft inbox or archived documents contain historical references.

## 7. Configuration Design

No runtime configuration changes.

## 8. Compatibility Strategy

Existing consumers continue to read `.knowledge/manifest.yaml`, `.knowledge/INDEX.md`, active documents, and validation scripts using the current schema.

## 9. Error Handling and Fallback Design

Unverified legacy statements are removed or replaced with narrower current-code statements. Out-of-scope draft/archive legacy references are reported but not edited.

## 10. Observability and Debug Output Design

The TASK evidence records command exit codes, changed files, scope results, and knowledge impact.

## 11. Testing Strategy

Run knowledge validator tests, active knowledge validation, reference checks, legacy grep for active/default scope, TASK scope check, and feature agent check.

## 12. Migration Risks

- Replacing too much prose could lose useful historical context. Mitigation: keep history only where it is still operationally relevant and sourced.
- Validation could hide draft issues. Mitigation: record inbox/archive legacy debt in the TASK report.

## 13. Implementation Boundaries

Allowed implementation files are `.knowledge/**` and `.spec/knowledge-base-current-state-refresh/**`. No business source files, manifests, generated code, or root documentation may be changed.

## 14. Alternatives Considered

- Full archive/inbox rewrite: rejected because it conflicts with the active-only scope and can rewrite historical evidence.
- Mechanical path replacement only: rejected because several route-mode and boundary statements are semantically stale.

## 15. Assumptions Requiring Confirmation

None.

## 16. Open Questions

None.
