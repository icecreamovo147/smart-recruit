-- 000005_add_notification_event_id.sql
-- Adds database-backed idempotency for notification consumers.
-- MySQL DDL performs implicit commits, so this migration is intentionally
-- written as an ordered sequence rather than a transactional script.
-- Uses IF NOT EXISTS / IF EXISTS for idempotency when db.sql is pre-imported.

ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS event_id VARCHAR(64) NULL COMMENT '来源 outbox 事件ID，用于 MQ 重复投递幂等' AFTER id;

-- Keep the earliest row for historical duplicate idempotent notifications
-- before adding the unique key.
DELETE n
FROM notifications n
JOIN (
  SELECT id
  FROM (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY receiver_id, receiver_role, COALESCE(biz_type, ''), biz_id, type
        ORDER BY created_at ASC, id ASC
      ) AS rn
    FROM notifications
    WHERE COALESCE(biz_type, '') <> '' AND biz_id <> 0
  ) ranked
  WHERE ranked.rn > 1
) dup ON dup.id = n.id;

ALTER TABLE notifications
  DROP INDEX IF EXISTS idx_receiver_biz_type;

ALTER TABLE notifications
  DROP INDEX IF EXISTS uk_notification_event_id,
  ADD UNIQUE KEY uk_notification_event_id (event_id);

ALTER TABLE notifications
  DROP INDEX IF EXISTS uk_notification_once,
  ADD UNIQUE KEY uk_notification_once (receiver_id, receiver_role, biz_type, biz_id, type);
