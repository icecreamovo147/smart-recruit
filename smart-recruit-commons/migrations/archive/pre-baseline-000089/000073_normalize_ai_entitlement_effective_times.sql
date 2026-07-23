-- Migration 000069 seeded immediately-effective plan and candidate price
-- records with the database session's local NOW(), while runtime entitlement
-- queries compare them against UTC_TIMESTAMP(). Normalize only those exact seed
-- records when they are still future-dated in UTC terms.
UPDATE `platform_plan_versions`
SET `effective_at` = UTC_TIMESTAMP(3)
WHERE `version` = 2
  AND `change_note` = 'AI billing shadow defaults; review before enforcement'
  AND `effective_at` > UTC_TIMESTAMP(3);

UPDATE `billing_price_versions` price
JOIN `billing_products` product
  ON product.id = price.product_id
 AND product.product_key = 'candidate_free'
SET price.effective_at = UTC_TIMESTAMP(3)
WHERE price.version = 1
  AND price.status = 'published'
  AND price.effective_at > UTC_TIMESTAMP(3);

-- Migration 000072 used the same local-time expression for its immediately
-- active compatibility subscriptions. Correct only rows created by that
-- migration and only while they are future-dated in UTC terms.
UPDATE `tenant_subscriptions`
SET `starts_at` = UTC_TIMESTAMP(3)
WHERE `reason` = 'legacy tenant starter plan bootstrap'
  AND `created_by` IS NULL
  AND `starts_at` > UTC_TIMESTAMP(3);
