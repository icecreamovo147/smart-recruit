-- Down: 000014_notification_account_type

ALTER TABLE `notifications`
  DROP KEY `uk_notification_once`,
  ADD UNIQUE KEY `uk_notification_once` (`receiver_id`, `receiver_role`, `biz_type`, `biz_id`, `type`),
  DROP COLUMN `receiver_account_type`;
