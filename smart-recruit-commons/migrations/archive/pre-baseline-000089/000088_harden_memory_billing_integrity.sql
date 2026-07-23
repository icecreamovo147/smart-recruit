-- Add tenant-safe Memory ownership and grant-backed AI credit reservations
-- without rewriting the already-published 000069, 000076, or 000083 files.

ALTER TABLE `ai_memories`
  DROP KEY `idx_ai_memories_content_hash_owner`,
  ADD KEY `idx_ai_memories_content_hash_owner` (`tenant_id`, `owner_role`, `owner_id`, `content_hash`),
  ADD CONSTRAINT `chk_ai_memories_owner_tenant`
  CHECK (
    (`owner_role` = 2 AND `tenant_id` IS NOT NULL)
    OR (`owner_role` = 1 AND `tenant_id` IS NULL)
  );

ALTER TABLE `ai_credit_grants`
  DROP CHECK `chk_ai_credit_grants_status`,
  ADD CONSTRAINT `chk_ai_credit_grants_status`
  CHECK (`status` IN ('active', 'refund_frozen', 'exhausted', 'expired', 'revoked'));

ALTER TABLE `billing_refunds`
  DROP CHECK `chk_billing_refunds_status`,
  ADD CONSTRAINT `chk_billing_refunds_status`
  CHECK (`status` IN ('requested', 'waiting_usage', 'reviewing', 'approved', 'processing', 'unknown', 'succeeded', 'failed', 'rejected'));

CREATE TABLE IF NOT EXISTS `ai_credit_reservation_allocations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `reservation_id` BIGINT UNSIGNED NOT NULL,
  `grant_id` BIGINT UNSIGNED NOT NULL,
  `reserved_credits` BIGINT UNSIGNED NOT NULL,
  `consumed_credits` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `released_credits` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_credit_allocation_reservation_grant` (`reservation_id`, `grant_id`),
  KEY `idx_ai_credit_allocation_grant_open` (`grant_id`, `reservation_id`),
  CONSTRAINT `fk_ai_credit_allocation_reservation`
    FOREIGN KEY (`reservation_id`) REFERENCES `ai_credit_reservations` (`id`),
  CONSTRAINT `fk_ai_credit_allocation_grant`
    FOREIGN KEY (`grant_id`) REFERENCES `ai_credit_grants` (`id`),
  CONSTRAINT `chk_ai_credit_allocation_amounts`
    CHECK (
      `reserved_credits` > 0
      AND `consumed_credits` + `released_credits` <= `reserved_credits`
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
  COMMENT='Exact grant backing for enforce-mode AI credit reservations';

-- Reservations created by the pre-allocation implementation cannot be proven
-- to own a specific grant. Release their logical holds and expire them rather
-- than allowing an unbacked settlement after this migration.
INSERT INTO `ai_credit_ledger` (
  `entry_id`,
  `owner_type`,
  `owner_id`,
  `reservation_id`,
  `entry_type`,
  `credits_delta`,
  `balance_after`,
  `idempotency_key`,
  `description`,
  `created_at`
)
SELECT
  UUID(),
  reservation.`owner_type`,
  reservation.`owner_id`,
  reservation.`id`,
  'release',
  reservation.`reserved_credits`,
  COALESCE((
    SELECT SUM(grant_row.`remaining_credits`)
    FROM `ai_credit_grants` grant_row
    WHERE grant_row.`owner_type` = reservation.`owner_type`
      AND grant_row.`owner_id` = reservation.`owner_id`
      AND grant_row.`status` = 'active'
      AND grant_row.`valid_from` <= NOW(3)
      AND (grant_row.`expires_at` IS NULL OR grant_row.`expires_at` > NOW(3))
  ), 0),
  CONCAT('migration-88-release:', reservation.`reservation_no`),
  'release pre-allocation reservation during allocation migration',
  NOW(3)
FROM `ai_credit_reservations` reservation
WHERE reservation.`status` = 'active'
  AND reservation.`enforcement_mode` = 'enforce'
  AND reservation.`reserved_credits` > 0;

UPDATE `ai_credit_reservations`
SET `status` = 'expired',
    `settled_at` = NOW(3),
    `updated_at` = NOW(3)
WHERE `status` = 'active'
  AND `enforcement_mode` = 'enforce';
