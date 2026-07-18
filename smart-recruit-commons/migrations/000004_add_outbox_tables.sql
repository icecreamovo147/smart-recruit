-- 000004_add_outbox_tables.sql
-- Transactional outbox table for reliable event publishing.

CREATE TABLE IF NOT EXISTS event_outbox (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  event_id VARCHAR(64) NOT NULL COMMENT '全局唯一事件ID',
  event_type VARCHAR(64) NOT NULL COMMENT 'notification.create / resume.parse',
  aggregate_type VARCHAR(64) NOT NULL COMMENT 'application / resume / notification',
  aggregate_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  routing_key VARCHAR(128) NOT NULL,
  payload JSON NOT NULL,
  status TINYINT NOT NULL DEFAULT 0 COMMENT '0=pending 1=published 2=dead 3=processing',
  retry_count INT NOT NULL DEFAULT 0,
  next_retry_at DATETIME NULL,
  last_error TEXT NULL,
  locked_at DATETIME NULL,
  locked_by VARCHAR(128) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_event_id (event_id),
  KEY idx_status_next_retry (status, next_retry_at, locked_at, id),
  KEY idx_aggregate (aggregate_type, aggregate_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事务消息 outbox 表';
