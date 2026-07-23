-- Publish the seeded candidate Pro offer for the development-only Alipay sandbox flow.
-- Existing operator-configured draft price values are preserved; only the required
-- AI entitlement release binding and publication state are completed here.

INSERT IGNORE INTO `billing_price_versions`
  (`product_id`, `version`, `billing_term`, `amount_fen`, `currency`, `included_credits`,
   `entitlement_snapshot`, `status`, `effective_at`, `created_at`, `updated_at`)
SELECT
  product.id,
  1,
  'monthly',
  990,
  'CNY',
  200,
  JSON_OBJECT(
    'ai.chat.enabled', true,
    'ai.chat.release_version_id', capability.current_published_version_id,
    'ai.credits.monthly', 200
  ),
  'draft',
  NULL,
  UTC_TIMESTAMP(3),
  UTC_TIMESTAMP(3)
FROM `billing_products` product
JOIN `platform_ai_capabilities` capability
  ON capability.capability_key = 'ai.chat'
 AND capability.audience = 'candidate'
 AND capability.current_published_version_id IS NOT NULL
WHERE product.product_key = 'candidate_pro'
  AND product.owner_type = 'user'
  AND product.product_type = 'subscription';

UPDATE `billing_price_versions` price
JOIN `billing_products` product
  ON product.id = price.product_id
 AND product.product_key = 'candidate_pro'
 AND product.owner_type = 'user'
JOIN `platform_ai_capabilities` capability
  ON capability.capability_key = 'ai.chat'
 AND capability.audience = 'candidate'
 AND capability.current_published_version_id IS NOT NULL
SET price.entitlement_snapshot = JSON_SET(
      COALESCE(price.entitlement_snapshot, JSON_OBJECT()),
      '$."ai.chat.enabled"',
      true,
      '$."ai.chat.release_version_id"',
      capability.current_published_version_id,
      '$."ai.credits.monthly"',
      price.included_credits
    ),
    price.effective_at = UTC_TIMESTAMP(3),
    price.status = 'published',
    price.updated_at = UTC_TIMESTAMP(3)
WHERE price.version = 1
  AND price.status = 'draft';

UPDATE `billing_products`
SET `status` = 'active',
    `updated_at` = UTC_TIMESTAMP(3)
WHERE `product_key` = 'candidate_pro'
  AND `owner_type` = 'user'
  AND `product_type` = 'subscription'
  AND `status` = 'draft';
