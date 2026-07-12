-- Down: 000051_standardize_event_outbox

ALTER TABLE `event_outbox`
  DROP KEY `idx_outbox_status_created`,
  DROP KEY `idx_outbox_dead_lettered_at`,
  DROP KEY `idx_outbox_published_at`,
  DROP KEY `idx_outbox_idempotency_key`;

ALTER TABLE `event_outbox`
  DROP COLUMN `dead_lettered_at`,
  DROP COLUMN `published_at`,
  DROP COLUMN `metadata`,
  DROP COLUMN `trace_id`,
  DROP COLUMN `causation_id`,
  DROP COLUMN `correlation_id`,
  DROP COLUMN `idempotency_key`,
  DROP COLUMN `producer`,
  DROP COLUMN `schema_version`;
