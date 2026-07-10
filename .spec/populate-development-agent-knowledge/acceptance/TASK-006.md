# Acceptance - TASK-006

## TASK Summary

Add resume intelligence and candidate matching knowledge.

## SPEC References

- FR-006
- Security and Safety Requirements

## SDD References

- Section 7 Configuration Design
- Section 9 Error Handling and Fallback Design

## Acceptance Criteria

- Knowledge covers resume upload/storage, parsing, structured profiles, extractors, matching, evidence, quotas, and sensitive data boundaries.
- Routes cover relevant backend and frontend paths.
- Examples use placeholders and contain no personal data.
- Validators pass.

## Required Checks

- Knowledge validators.
- Harness scope and agent check.
- Targeted source inspection evidence recorded in the report.

## Manual Verification, if needed

Required if sensitive data handling policy is proposed.

## Out-of-Scope

- Changing resume parsing, storage, matching, or quota behavior.
