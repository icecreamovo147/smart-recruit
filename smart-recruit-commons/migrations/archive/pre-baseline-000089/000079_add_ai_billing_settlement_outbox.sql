-- Durable AI Agent delivery state for idempotent Billing settlement/cancellation.

CREATE TABLE IF NOT EXISTS `ai_billing_settlement_outbox` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `reservation_no` CHAR(36) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `capability` VARCHAR(96) NOT NULL,
  `operation` VARCHAR(64) NOT NULL,
  `provider_key` VARCHAR(64) NOT NULL,
  `model_key` VARCHAR(128) NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'reserved',
  `request_payload` JSON DEFAULT NULL,
  `idempotency_key` VARCHAR(191) DEFAULT NULL,
  `retry_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `next_attempt_at` DATETIME(3) DEFAULT NULL,
  `locked_at` DATETIME(3) DEFAULT NULL,
  `last_error` VARCHAR(500) DEFAULT NULL,
  `completed_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_billing_settlement_reservation` (`reservation_no`),
  KEY `idx_ai_billing_settlement_due` (`status`, `next_attempt_at`),
  KEY `idx_ai_billing_settlement_owner` (`owner_type`, `owner_id`, `created_at`),
  CONSTRAINT `fk_ai_billing_settlement_reservation`
    FOREIGN KEY (`reservation_no`) REFERENCES `ai_credit_reservations` (`reservation_no`),
  CONSTRAINT `chk_ai_billing_settlement_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_billing_settlement_status` CHECK (`status` IN (
    'reserved', 'pending_settle', 'processing_settle', 'pending_cancel',
    'processing_cancel', 'settled', 'cancelled', 'dead'
  ))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI Agent durable Billing settlement delivery outbox';
