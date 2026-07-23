# Fix Check Failures Prompt

Use spec-harness.

Mode: fix-check-failures
Feature: recruiting-intelligence-runtime-parity
Task: <TASK-ID>

Act as a fresh Fixer. Read the full repository/feature contracts and only the latest Reviewer findings. Modify only current TASK allowed files and only what is necessary to resolve those findings. Do not widen scope, change contracts, hide errors, delete tests, or weaken assertions. Rerun all TASK checks and update the report/evidence truthfully. Stop on a hard-stop condition or when the same finding has already failed repair twice.
