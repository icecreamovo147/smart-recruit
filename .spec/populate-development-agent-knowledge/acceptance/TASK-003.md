# Acceptance - TASK-003

## TASK Summary

Add gateway, public contract, and persistence knowledge.

## SPEC References

- FR-003
- Compatibility Requirements

## SDD References

- Section 5 API and Interface Changes
- Section 7 Configuration Design

## Acceptance Criteria

- Gateway/API contract knowledge is grounded in router, handler, middleware, rpc, proto, and generated-code locations.
- Persistence knowledge is grounded in migrations, model, repository, and tests.
- Route additions do not break reference checks.
- Validators pass.

## Required Checks

- Knowledge validators.
- Harness scope and agent check.
- Targeted source inspection evidence recorded in the report.

## Manual Verification, if needed

Required if the TASK proposes public-contract policy changes.

## Out-of-Scope

- Editing proto, generated code, database schema, migrations, models, repositories, or gateway code.
