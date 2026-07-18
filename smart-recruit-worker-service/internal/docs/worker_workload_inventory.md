# Worker Workload Inventory

TASK-026 creates the Worker service workload profile and owner-contract
inventory. It does not split the worker binary, add new workloads, change queue
bindings, or change default workload enablement.

## Workload Toggle Contract

- `WORKER_WORKLOADS`: comma/semicolon/space separated allow-list. Empty means
  all known workloads.
- `WORKER_DISABLED_WORKLOADS`: removes workloads from the enabled set.
- All current workload profiles are `DefaultOn=true`; this preserves the
  previous default behavior.
- RabbitMQ and MySQL remain hard readiness dependencies for active workloads.

## Owner Contracts

| Workload | Owner context | Queue/source | Owned writes | Idempotency | Retry | DLQ/dead-letter |
| --- | --- | --- | --- | --- | --- | --- |
| `outbox-dispatcher` | platform | `event_outbox` | Claim pending outbox rows and mark published/dead. | outbox claim/mark-published status transition | outbox retry count and `next_retry_at` | `event_outbox` dead status |
| `notification-consumer` | notification | `RABBITMQ_NOTIFICATION_QUEUE` | Notification rows and inbox claims. | notification inbox business key | RabbitMQ retry + inbox failed attempts | notification queue DLQ + `event_inbox` dead |
| `email-consumer` | notification | `RABBITMQ_EMAIL_QUEUE` | Email logs and inbox claims. | email inbox and email log uniqueness | RabbitMQ retry + inbox failed attempts | email queue DLQ + `event_inbox` dead |
| `resume-parse-consumer` | recruitment | `RABBITMQ_RESUME_PARSE_QUEUE` | Resume parse status/text for the owning resume. | resume parse run and resume idempotency | RabbitMQ retry + inbox failed attempts | resume parse queue DLQ + `event_inbox` dead |
| `embedding-consumer` | ai-agent | `RABBITMQ_EMBEDDING_QUEUE` | AI embedding rows and inbox claims. | embedding content hash and model key | RabbitMQ retry + inbox failed attempts | embedding queue DLQ + `event_inbox` dead |
| `agent-run-consumer` | ai-agent | `RABBITMQ_AGENT_RUN_QUEUE` | Agent run status/events and final assistant message. | agent run status transition guard | RabbitMQ retry + inbox failed attempts; terminal states guarded | agent run queue DLQ + `event_inbox` dead |
| `analytics-projection-consumer` | analytics | domain-event projections | Projection events and checkpoints. | analytics projection event id ledger | projection retry/dead-letter metadata | projection dead-letter metadata |

## Boundary Rules

- Worker workloads may consume events and update their owner context only.
- Worker workloads must not directly mutate recruitment, notification,
  analytics, or AI Agent owner tables outside the contract listed above.
- Inbox claims and dead-letter metadata are worker safety records, not a license
  to bypass owner service invariants.
- Reports and logs must not include raw resume text, provider credentials,
  private object keys, email bodies, or sensitive notification payloads.
- TASK-026 keeps workload starters as controlled supervisors; concrete consumer
  cutover belongs to later Worker runtime tasks.
- TASK-028 confirms there are no direct shared business service orchestration
  imports in `smart-recruit-worker-service`; the remaining
  `smart-recruit-commons/mq` import is an infrastructure bridge until shared
  cleanup decides whether RabbitMQ helpers become commons.
