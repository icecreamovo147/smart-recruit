#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/event-replay-dead-letter.sh <action> [options]

Actions:
  outbox-stats                 Print or execute outbox status count SQL.
  inbox-stats                  Print or execute inbox status count SQL.
  outbox-dead                  Print or execute dead outbox inspection SQL.
  inbox-dead                   Print or execute dead inbox inspection SQL.
  outbox-replay-sql            Generate SQL to move dead outbox events back to pending.
  outbox-unlock-sql            Generate SQL to unlock stale processing outbox events.
  inbox-repair-sql             Generate SQL to move dead inbox rows back to failed.
  retention-sql                Generate batched retention cleanup SQL.

Options:
  --event-id <id>              Event id. Repeat for multiple ids.
  --consumer <name>            Inbox consumer name.
  --limit <n>                  Inspection row limit. Default: 50.
  --published-days <n>         Outbox published retention days. Default: 30.
  --processed-days <n>         Inbox processed retention days. Default: 30.
  --dead-days <n>              Dead-letter retention days. Default: 90.
  --batch-size <n>             Delete batch size for retention SQL. Default: 500.
  --execute                    Execute read-only inspection SQL with mysql CLI.
  -h, --help                   Show this help.

Read-only execution requires MYSQL_HOST, MYSQL_USER, MYSQL_DATABASE, and MYSQL_PWD.
Mutation actions always print SQL only; review and execute through the approved DB change path.
USAGE
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

is_positive_integer() {
  [[ "${1:-}" =~ ^[1-9][0-9]*$ ]]
}

sql_quote() {
  local value=${1//\'/\'\'}
  printf "'%s'" "$value"
}

join_event_ids() {
  local joined=""
  local event_id
  for event_id in "${EVENT_IDS[@]}"; do
    if [[ -n "$joined" ]]; then
      joined+=", "
    fi
    joined+="$(sql_quote "$event_id")"
  done
  printf '%s' "$joined"
}

print_sql() {
  printf '%s\n' "$1"
}

run_select() {
  local sql="$1"
  [[ "$EXECUTE" == "1" ]] || {
    print_sql "$sql"
    return 0
  }
  command -v mysql >/dev/null 2>&1 || die "mysql CLI is required for --execute"
  : "${MYSQL_HOST:?MYSQL_HOST is required for --execute}"
  : "${MYSQL_USER:?MYSQL_USER is required for --execute}"
  : "${MYSQL_DATABASE:?MYSQL_DATABASE is required for --execute}"
  : "${MYSQL_PWD:?MYSQL_PWD is required for --execute}"
  MYSQL_PWD="$MYSQL_PWD" mysql \
    --host="$MYSQL_HOST" \
    --port="${MYSQL_PORT:-3306}" \
    --user="$MYSQL_USER" \
    --database="$MYSQL_DATABASE" \
    --batch \
    --raw \
    --table \
    --execute="$sql"
}

outbox_stats_sql() {
  cat <<'SQL'
SELECT 'pending' AS state, COUNT(*) AS total FROM event_outbox WHERE status = 0 AND retry_count = 0
UNION ALL
SELECT 'retrying' AS state, COUNT(*) AS total FROM event_outbox WHERE status = 0 AND retry_count > 0
UNION ALL
SELECT 'processing' AS state, COUNT(*) AS total FROM event_outbox WHERE status = 3
UNION ALL
SELECT 'published' AS state, COUNT(*) AS total FROM event_outbox WHERE status = 1
UNION ALL
SELECT 'dead' AS state, COUNT(*) AS total FROM event_outbox WHERE status = 2;
SQL
}

inbox_stats_sql() {
  if [[ -n "$CONSUMER" ]]; then
    local consumer
    consumer=$(sql_quote "$CONSUMER")
    cat <<SQL
SELECT 'processing' AS state, COUNT(*) AS total FROM event_inbox WHERE status = 0 AND consumer_name = $consumer
UNION ALL
SELECT 'processed' AS state, COUNT(*) AS total FROM event_inbox WHERE status = 1 AND consumer_name = $consumer
UNION ALL
SELECT 'failed' AS state, COUNT(*) AS total FROM event_inbox WHERE status = 2 AND consumer_name = $consumer
UNION ALL
SELECT 'dead' AS state, COUNT(*) AS total FROM event_inbox WHERE status = 3 AND consumer_name = $consumer;
SQL
  else
    cat <<'SQL'
SELECT 'processing' AS state, COUNT(*) AS total FROM event_inbox WHERE status = 0
UNION ALL
SELECT 'processed' AS state, COUNT(*) AS total FROM event_inbox WHERE status = 1
UNION ALL
SELECT 'failed' AS state, COUNT(*) AS total FROM event_inbox WHERE status = 2
UNION ALL
SELECT 'dead' AS state, COUNT(*) AS total FROM event_inbox WHERE status = 3;
SQL
  fi
}

outbox_dead_sql() {
  cat <<SQL
SELECT id, event_id, event_type, aggregate_type, aggregate_id, routing_key, retry_count,
       dead_lettered_at, LEFT(COALESCE(last_error, ''), 256) AS last_error
FROM event_outbox
WHERE status = 2
ORDER BY dead_lettered_at DESC, id DESC
LIMIT $LIMIT;
SQL
}

inbox_dead_sql() {
  local where="status = 3"
  if [[ -n "$CONSUMER" ]]; then
    where+=" AND consumer_name = $(sql_quote "$CONSUMER")"
  fi
  cat <<SQL
SELECT id, consumer_name, event_id, event_type, attempt_count,
       dead_lettered_at, LEFT(COALESCE(last_error, ''), 256) AS last_error
FROM event_inbox
WHERE $where
ORDER BY dead_lettered_at DESC, id DESC
LIMIT $LIMIT;
SQL
}

outbox_replay_sql() {
  local ids
  ids=$(join_event_ids)
  cat <<SQL
START TRANSACTION;
SELECT id, event_id, event_type, aggregate_type, aggregate_id, routing_key, status, retry_count,
       dead_lettered_at, LEFT(COALESCE(last_error, ''), 256) AS last_error
FROM event_outbox
WHERE event_id IN ($ids)
FOR UPDATE;

UPDATE event_outbox
SET status = 0,
    next_retry_at = NOW(),
    locked_at = NULL,
    locked_by = '',
    dead_lettered_at = NULL,
    updated_at = NOW()
WHERE event_id IN ($ids)
  AND status = 2;

SELECT ROW_COUNT() AS replay_prepared_rows;
COMMIT;
SQL
}

outbox_unlock_sql() {
  local ids
  ids=$(join_event_ids)
  cat <<SQL
START TRANSACTION;
SELECT id, event_id, event_type, status, locked_at, locked_by,
       LEFT(COALESCE(last_error, ''), 256) AS last_error
FROM event_outbox
WHERE event_id IN ($ids)
FOR UPDATE;

UPDATE event_outbox
SET status = 0,
    next_retry_at = NOW(),
    locked_at = NULL,
    locked_by = '',
    updated_at = NOW()
WHERE event_id IN ($ids)
  AND status = 3;

SELECT ROW_COUNT() AS unlocked_rows;
COMMIT;
SQL
}

inbox_repair_sql() {
  local ids consumer
  ids=$(join_event_ids)
  consumer=$(sql_quote "$CONSUMER")
  cat <<SQL
START TRANSACTION;
SELECT id, consumer_name, event_id, event_type, status, attempt_count,
       dead_lettered_at, LEFT(COALESCE(last_error, ''), 256) AS last_error
FROM event_inbox
WHERE consumer_name = $consumer
  AND event_id IN ($ids)
FOR UPDATE;

UPDATE event_inbox
SET status = 2,
    processing_at = NULL,
    dead_lettered_at = NULL,
    updated_at = NOW()
WHERE consumer_name = $consumer
  AND event_id IN ($ids)
  AND status = 3;

SELECT ROW_COUNT() AS repaired_rows;
COMMIT;
SQL
}

retention_sql() {
  cat <<SQL
-- Review counts before each DELETE. Repeat batched DELETE statements until ROW_COUNT() returns 0.
SELECT COUNT(*) AS outbox_published_expired
FROM event_outbox
WHERE status = 1
  AND published_at IS NOT NULL
  AND published_at < DATE_SUB(NOW(), INTERVAL $PUBLISHED_DAYS DAY);

DELETE FROM event_outbox
WHERE status = 1
  AND published_at IS NOT NULL
  AND published_at < DATE_SUB(NOW(), INTERVAL $PUBLISHED_DAYS DAY)
ORDER BY published_at ASC, id ASC
LIMIT $BATCH_SIZE;

SELECT COUNT(*) AS inbox_processed_expired
FROM event_inbox
WHERE status = 1
  AND processed_at IS NOT NULL
  AND processed_at < DATE_SUB(NOW(), INTERVAL $PROCESSED_DAYS DAY);

DELETE FROM event_inbox
WHERE status = 1
  AND processed_at IS NOT NULL
  AND processed_at < DATE_SUB(NOW(), INTERVAL $PROCESSED_DAYS DAY)
ORDER BY processed_at ASC, id ASC
LIMIT $BATCH_SIZE;

SELECT COUNT(*) AS outbox_dead_expired
FROM event_outbox
WHERE status = 2
  AND dead_lettered_at IS NOT NULL
  AND dead_lettered_at < DATE_SUB(NOW(), INTERVAL $DEAD_DAYS DAY);

DELETE FROM event_outbox
WHERE status = 2
  AND dead_lettered_at IS NOT NULL
  AND dead_lettered_at < DATE_SUB(NOW(), INTERVAL $DEAD_DAYS DAY)
ORDER BY dead_lettered_at ASC, id ASC
LIMIT $BATCH_SIZE;

SELECT COUNT(*) AS inbox_dead_expired
FROM event_inbox
WHERE status = 3
  AND dead_lettered_at IS NOT NULL
  AND dead_lettered_at < DATE_SUB(NOW(), INTERVAL $DEAD_DAYS DAY);

DELETE FROM event_inbox
WHERE status = 3
  AND dead_lettered_at IS NOT NULL
  AND dead_lettered_at < DATE_SUB(NOW(), INTERVAL $DEAD_DAYS DAY)
ORDER BY dead_lettered_at ASC, id ASC
LIMIT $BATCH_SIZE;
SQL
}

ACTION="${1:-}"
if [[ -z "$ACTION" || "$ACTION" == "-h" || "$ACTION" == "--help" ]]; then
  usage
  exit 0
fi
shift

EVENT_IDS=()
CONSUMER=""
LIMIT=50
PUBLISHED_DAYS=30
PROCESSED_DAYS=30
DEAD_DAYS=90
BATCH_SIZE=500
EXECUTE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --event-id)
      [[ $# -ge 2 ]] || die "--event-id requires a value"
      EVENT_IDS+=("$2")
      shift 2
      ;;
    --consumer)
      [[ $# -ge 2 ]] || die "--consumer requires a value"
      CONSUMER="$2"
      shift 2
      ;;
    --limit)
      [[ $# -ge 2 ]] || die "--limit requires a value"
      is_positive_integer "$2" || die "--limit must be a positive integer"
      LIMIT="$2"
      shift 2
      ;;
    --published-days)
      [[ $# -ge 2 ]] || die "--published-days requires a value"
      is_positive_integer "$2" || die "--published-days must be a positive integer"
      PUBLISHED_DAYS="$2"
      shift 2
      ;;
    --processed-days)
      [[ $# -ge 2 ]] || die "--processed-days requires a value"
      is_positive_integer "$2" || die "--processed-days must be a positive integer"
      PROCESSED_DAYS="$2"
      shift 2
      ;;
    --dead-days)
      [[ $# -ge 2 ]] || die "--dead-days requires a value"
      is_positive_integer "$2" || die "--dead-days must be a positive integer"
      DEAD_DAYS="$2"
      shift 2
      ;;
    --batch-size)
      [[ $# -ge 2 ]] || die "--batch-size requires a value"
      is_positive_integer "$2" || die "--batch-size must be a positive integer"
      BATCH_SIZE="$2"
      shift 2
      ;;
    --execute)
      EXECUTE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
done

case "$ACTION" in
  outbox-stats)
    run_select "$(outbox_stats_sql)"
    ;;
  inbox-stats)
    run_select "$(inbox_stats_sql)"
    ;;
  outbox-dead)
    run_select "$(outbox_dead_sql)"
    ;;
  inbox-dead)
    run_select "$(inbox_dead_sql)"
    ;;
  outbox-replay-sql)
    [[ "$EXECUTE" == "0" ]] || die "--execute is not allowed for mutation SQL generation"
    [[ "${#EVENT_IDS[@]}" -gt 0 ]] || die "outbox-replay-sql requires at least one --event-id"
    print_sql "$(outbox_replay_sql)"
    ;;
  outbox-unlock-sql)
    [[ "$EXECUTE" == "0" ]] || die "--execute is not allowed for mutation SQL generation"
    [[ "${#EVENT_IDS[@]}" -gt 0 ]] || die "outbox-unlock-sql requires at least one --event-id"
    print_sql "$(outbox_unlock_sql)"
    ;;
  inbox-repair-sql)
    [[ "$EXECUTE" == "0" ]] || die "--execute is not allowed for mutation SQL generation"
    [[ -n "$CONSUMER" ]] || die "inbox-repair-sql requires --consumer"
    [[ "${#EVENT_IDS[@]}" -gt 0 ]] || die "inbox-repair-sql requires at least one --event-id"
    print_sql "$(inbox_repair_sql)"
    ;;
  retention-sql)
    [[ "$EXECUTE" == "0" ]] || die "--execute is not allowed for retention SQL generation"
    print_sql "$(retention_sql)"
    ;;
  *)
    die "unknown action: $ACTION"
    ;;
esac

