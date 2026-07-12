-- 000052_add_event_inbox.sql
-- Consumer-side Inbox table for idempotent event processing.

CREATE TABLE IF NOT EXISTS `event_inbox` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` VARCHAR(128) NOT NULL COMMENT '事件ID或无事件ID消息的稳定哈希',
  `event_type` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '事件类型',
  `consumer_name` VARCHAR(128) NOT NULL COMMENT '消费者名称',
  `idempotency_key` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '消费者幂等键',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0=processing 1=processed 2=failed 3=dead',
  `attempt_count` INT NOT NULL DEFAULT 0,
  `last_error` TEXT NULL,
  `received_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `processing_at` DATETIME NULL,
  `processed_at` DATETIME NULL,
  `dead_lettered_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_event_inbox_consumer_event` (`consumer_name`, `event_id`),
  KEY `idx_event_inbox_consumer_status` (`consumer_name`, `status`),
  KEY `idx_event_inbox_idempotency_key` (`idempotency_key`),
  KEY `idx_event_inbox_processed_at` (`status`, `processed_at`),
  KEY `idx_event_inbox_dead_lettered_at` (`status`, `dead_lettered_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事件消费者 Inbox 幂等表';
