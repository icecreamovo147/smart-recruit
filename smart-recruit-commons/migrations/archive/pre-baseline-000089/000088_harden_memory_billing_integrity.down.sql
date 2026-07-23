UPDATE `billing_refunds`
SET `status` = 'reviewing',
    `review_mode` = 'manual',
    `next_reconcile_at` = NULL,
    `updated_at` = NOW(3)
WHERE `status` = 'waiting_usage';

UPDATE `ai_credit_grants`
SET `status` = CASE
      WHEN `remaining_credits` = 0 THEN 'exhausted'
      WHEN `expires_at` IS NOT NULL AND `expires_at` <= NOW(3) THEN 'expired'
      ELSE 'active'
    END,
    `remaining_credits` = CASE
      WHEN `expires_at` IS NOT NULL AND `expires_at` <= NOW(3) THEN 0
      ELSE `remaining_credits`
    END,
    `updated_at` = NOW(3)
WHERE `status` = 'refund_frozen';

DROP TABLE IF EXISTS `ai_credit_reservation_allocations`;

ALTER TABLE `billing_refunds`
  DROP CHECK `chk_billing_refunds_status`,
  ADD CONSTRAINT `chk_billing_refunds_status`
  CHECK (`status` IN ('requested', 'reviewing', 'approved', 'processing', 'unknown', 'succeeded', 'failed', 'rejected'));

ALTER TABLE `ai_credit_grants`
  DROP CHECK `chk_ai_credit_grants_status`,
  ADD CONSTRAINT `chk_ai_credit_grants_status`
  CHECK (`status` IN ('active', 'exhausted', 'expired', 'revoked'));

ALTER TABLE `ai_memories`
  DROP CHECK `chk_ai_memories_owner_tenant`,
  DROP KEY `idx_ai_memories_content_hash_owner`,
  ADD KEY `idx_ai_memories_content_hash_owner` (`content_hash`, `owner_role`, `owner_id`);
