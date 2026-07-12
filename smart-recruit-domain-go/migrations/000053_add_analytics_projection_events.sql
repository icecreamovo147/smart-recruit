-- 000053_add_analytics_projection_events.sql
-- Analytics-owned projection event ledger and checkpoint store.

CREATE TABLE IF NOT EXISTS `analytics_projection_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `projection_name` VARCHAR(64) NOT NULL COMMENT 'Analytics projection/read-model name',
  `source` VARCHAR(32) NOT NULL DEFAULT 'domain_event' COMMENT 'domain_event / replay / backfill',
  `event_id` VARCHAR(128) NOT NULL COMMENT 'Domain event id',
  `event_type` VARCHAR(128) NOT NULL COMMENT 'Source-domain event type',
  `aggregate_type` VARCHAR(64) NOT NULL COMMENT 'Source aggregate type',
  `aggregate_id` VARCHAR(128) NOT NULL COMMENT 'Source aggregate id',
  `producer` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'Event producer',
  `idempotency_key` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Projection idempotency key',
  `correlation_id` VARCHAR(128) NOT NULL DEFAULT '',
  `causation_id` VARCHAR(128) NOT NULL DEFAULT '',
  `trace_id` VARCHAR(128) NOT NULL DEFAULT '',
  `payload` JSON NOT NULL,
  `metadata` JSON NULL,
  `occurred_at` DATETIME NOT NULL COMMENT 'Source event occurrence time',
  `projected_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Projection write time',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_analytics_projection_event` (`event_id`),
  KEY `idx_analytics_projection_name_occurred` (`projection_name`, `occurred_at`),
  KEY `idx_analytics_projection_event_type` (`event_type`),
  KEY `idx_analytics_projection_aggregate` (`aggregate_type`, `aggregate_id`),
  KEY `idx_analytics_projection_idempotency_key` (`idempotency_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Analytics event-projection read model input';

CREATE TABLE IF NOT EXISTS `analytics_projection_checkpoints` (
  `projection_name` VARCHAR(64) NOT NULL,
  `cursor` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Last processed event cursor',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`projection_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Analytics projection checkpoint store';
