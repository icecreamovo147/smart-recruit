# Acceptance - TASK-002

## TASK Summary

Refine Recruitment runtime adapters from one catch-all native adapter into focused local adapters.

## SPEC References

- SPEC 2
- SPEC 5.5
- SPEC 5.6
- SPEC 5.7
- SPEC 11

## SDD References

- SDD 3 Recruitment
- SDD 6
- SDD 8
- SDD 12

## Acceptance Criteria

- Active Recruitment runtime no longer depends on a single broad `nativeAdapter` implementing unrelated APIs.
- Focused adapters preserve current behavior for job, taxonomy, candidate/resume, application, owner contract, collaboration, admin/invite, usage, and outbox responsibilities.
- No `legacydomain` directory/import is reintroduced.
- Relevant tests pass.
- `knowledge_impact` is included.

## Required Checks

- `cd smart-recruit-recruitment-service && go test ./...`
- common Harness and knowledge checks from AGENT_RULES.

## Manual Verification, if needed

Compare selected SQL behavior against the pre-TASK baseline when tests do not cover a moved query.

## Out-of-Scope

Schema changes, proto changes, Gateway/frontend changes.
