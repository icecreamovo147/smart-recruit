# Backend DDD Microservices Evolution Event Replay And Dead Letter Runbook

Last verified: 2026-07-11

This runbook covers safe inspection, replay preparation, consumer repair, dead-letter review, and retention cleanup for the transitional `event_outbox` and `event_inbox` tables.

## Safety Rules

- Treat every replay as a data repair operation. Run during a bounded maintenance or incident window.
- Capture counts and affected event ids before preparing SQL.
- Never edit event payloads in place unless a separate, approved data repair plan exists.
- Do not replay events until the poison-message root cause is fixed or explicitly accepted.
- Do not expose payloads, raw personal data, secrets, tokens, or credentials in tickets, logs, or reports.
- Prefer the checked-in script in `scripts/event-replay-dead-letter.sh`; it defaults to read-only inspection or SQL generation.

## Current Status Values

`event_outbox.status`:

- `0`: pending
- `1`: published
- `2`: dead
- `3`: processing

`event_inbox.status`:

- `0`: processing
- `1`: processed
- `2`: failed
- `3`: dead

Retention defaults from the repositories:

- Published outbox events: 30 days.
- Processed inbox records: 30 days.
- Dead-letter outbox and inbox records: 90 days.

## Environment For Read-Only Inspection

The helper script can execute read-only `SELECT` checks through the MySQL CLI when these variables are set:

```bash
export MYSQL_HOST=127.0.0.1
export MYSQL_PORT=3306
export MYSQL_USER=smart_recruit
export MYSQL_PWD='<local password>'
export MYSQL_DATABASE=smart_recruit
```

Keep credentials in the shell environment only. Do not commit them to files.

## Inspect Backlog And Dead Letters

```bash
scripts/event-replay-dead-letter.sh outbox-stats --execute
scripts/event-replay-dead-letter.sh inbox-stats --execute
scripts/event-replay-dead-letter.sh outbox-dead --limit 50 --execute
scripts/event-replay-dead-letter.sh inbox-dead --consumer notification-worker --limit 50 --execute
```

If DB access is unavailable, omit `--execute`; the script prints the SQL to run through an approved database console.

## Prepare Outbox Replay

Use this only after confirming the target events are safe to republish and downstream consumers are idempotent.

```bash
scripts/event-replay-dead-letter.sh outbox-replay-sql \
  --event-id evt_123 \
  --event-id evt_456
```

The generated SQL:

- locks matching rows for review;
- moves dead outbox rows back to pending;
- clears stale locks and `dead_lettered_at`;
- leaves payload and historical retry count intact.

Review the generated SQL with another engineer before execution. After execution, watch `event_outbox` pending, retrying, published, and dead-letter counts.

## Repair Inbox Dead Letters

Use this when a consumer dead-letter record blocks reprocessing but the original event can be redelivered or replayed from outbox.

```bash
scripts/event-replay-dead-letter.sh inbox-repair-sql \
  --consumer notification-worker \
  --event-id evt_123
```

The generated SQL changes dead inbox rows back to failed and clears `dead_lettered_at`. The next delivery can then be claimed by the consumer helper, which increments `attempt_count` and moves the record through processing.

## Unlock Stale Processing Outbox Rows

Use this only when the worker that claimed an event is confirmed dead and the lock is stale.

```bash
scripts/event-replay-dead-letter.sh outbox-unlock-sql --event-id evt_123
```

The generated SQL moves processing rows back to pending and clears `locked_at` and `locked_by`.

## Retention Cleanup

Preview retention SQL:

```bash
scripts/event-replay-dead-letter.sh retention-sql
```

The generated statements delete in batches:

- outbox `published` rows older than 30 days;
- inbox `processed` rows older than 30 days;
- outbox and inbox `dead` rows older than 90 days.

Adjust days and batch size only when the incident or maintenance plan explicitly calls for it:

```bash
scripts/event-replay-dead-letter.sh retention-sql \
  --published-days 45 \
  --processed-days 45 \
  --dead-days 120 \
  --batch-size 500
```

## Post-Repair Verification

1. Re-run `outbox-stats` and `inbox-stats`.
2. Confirm target event ids no longer sit in dead-letter state unless intentionally retained.
3. Confirm downstream side effects are idempotent and duplicated effects did not occur.
4. Record event ids, SQL reviewed, SQL executed, time window, actor, and verification output in the incident or TASK report.
5. If payload repair, schema change, consumer behavior change, or retention policy change is needed, create a separate scoped TASK.

