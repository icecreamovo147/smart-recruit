-- Migration: 000014_add_notification_account_type
-- Add receiver_account_type to notifications, backfill from receiver_role,
-- and update the unique constraint to include the new column.
-- Uses IF NOT EXISTS / IF EXISTS for idempotency when db.sql is pre-imported.

ALTER TABLE `notifications`
  ADD COLUMN IF NOT EXISTS `receiver_account_type` VARCHAR(32) NOT NULL DEFAULT '' AFTER `receiver_role`;

-- Backfill: derive from legacy receiver_role
UPDATE `notifications` SET `receiver_account_type` = 'candidate' WHERE `receiver_role` = 1;
UPDATE `notifications` SET `receiver_account_type` = 'staff' WHERE `receiver_role` >= 2;

-- Update unique index: replace receiver_role with receiver_account_type
ALTER TABLE `notifications`
  DROP INDEX IF EXISTS `uk_notification_once`;

ALTER TABLE `notifications`
  ADD UNIQUE KEY `uk_notification_once` (`receiver_id`, `receiver_account_type`, `biz_type`, `biz_id`, `type`);
