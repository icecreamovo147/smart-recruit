# Acceptance - TASK-005

## TASK Summary

Add AI platform and configuration governance knowledge.

## SPEC References

- FR-005
- Security and Safety Requirements

## SDD References

- Section 7 Configuration Design
- Section 9 Error Handling and Fallback Design

## Acceptance Criteria

- Knowledge distinguishes AI runtime behavior from admin configuration surfaces.
- LLM, prompt, agent config, MCP, Skill, Agent Skill, embedding, and audit concerns are source-backed.
- No real provider credentials or sensitive configuration are included.
- Validators pass.

## Required Checks

- Knowledge validators.
- Harness scope and agent check.
- Targeted source inspection evidence recorded in the report.

## Manual Verification, if needed

Required for any proposed governance policy beyond current-code description.

## Out-of-Scope

- Changing AI runtime, provider, prompt, MCP, Skill, or embedding behavior.
