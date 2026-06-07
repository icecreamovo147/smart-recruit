-- Down: 000017_add_status_key_and_transitions

DROP TABLE IF EXISTS `application_status_transitions`;

ALTER TABLE `applications`
  DROP KEY `uk_active_application`,
  DROP COLUMN `active_key`;

-- Restore the original active_key generated column (pre-migration-17 definition).
ALTER TABLE `applications`
  ADD COLUMN `active_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_current` = 1 AND `status` <> 3 THEN 1 ELSE NULL END
  ) STORED AFTER `updated_at`,
  ADD UNIQUE KEY `uk_active_application` (`job_id`, `user_id`, `active_key`);

ALTER TABLE `applications`
  DROP KEY `idx_status_key`,
  DROP COLUMN `status_key`;
