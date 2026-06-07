-- 000017: Add status_key and application_status_transitions (Phase 1 Pipeline State)
-- Uses IF NOT EXISTS / IF EXISTS for idempotency when db.sql is pre-imported.

ALTER TABLE `applications`
  ADD COLUMN IF NOT EXISTS `status_key` VARCHAR(64) NOT NULL DEFAULT 'applied' COMMENT '投递状态键（Phase 1 新状态机）' AFTER `status`;

ALTER TABLE `applications`
  DROP INDEX IF EXISTS `idx_status_key`,
  ADD INDEX `idx_status_key` (`status_key`);

-- Backfill status_key from legacy numeric status so existing rows are not
-- all mapped to 'applied'. This must run before the active_key rebuild so
-- historical terminal rows are correctly excluded from the unique index.
UPDATE `applications`
SET `status_key` = CASE `status`
  WHEN 0 THEN 'applied'
  WHEN 1 THEN 'viewed'
  WHEN 2 THEN 'screen_passed'
  WHEN 3 THEN 'rejected'
  ELSE 'applied'
END;

-- Rebuild active_key to reference status_key (instead of numeric status)
ALTER TABLE `applications`
  DROP INDEX IF EXISTS `uk_active_application`;

ALTER TABLE `applications`
  DROP COLUMN IF EXISTS `active_key`;

ALTER TABLE `applications`
  ADD COLUMN `active_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_current` = 1 AND `status_key` NOT IN ('rejected', 'withdrawn', 'offer_rejected', 'hired') THEN 1 ELSE NULL END
  ) STORED AFTER `updated_at`,
  ADD UNIQUE KEY `uk_active_application` (`job_id`, `user_id`, `active_key`);

CREATE TABLE IF NOT EXISTS `application_status_transitions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 applications.id',
  `from_status` VARCHAR(64) NOT NULL COMMENT '变更前状态 key',
  `to_status` VARCHAR(64) NOT NULL COMMENT '变更后状态 key',
  `actor_user_id` BIGINT UNSIGNED NOT NULL COMMENT '操作人用户ID',
  `actor_account_type` VARCHAR(32) NOT NULL COMMENT '操作人账号类型',
  `reason` VARCHAR(512) DEFAULT NULL COMMENT '变更原因',
  `metadata_json` TEXT DEFAULT NULL COMMENT '附加元数据 JSON',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_transition_app` (`application_id`),
  KEY `idx_transition_created` (`application_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='投递状态变更审计记录表';
