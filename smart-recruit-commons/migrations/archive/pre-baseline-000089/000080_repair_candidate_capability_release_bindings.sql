-- Migration 000078 repaired tenant-HR capability releases but its billing price
-- update also matched user-owned candidate snapshots that shared entitlement keys.
-- Restore every published/draft candidate subscription snapshot to the immutable
-- candidate release owned by the candidate audience.

UPDATE `billing_price_versions` price
JOIN `billing_products` product
  ON product.id = price.product_id
 AND product.owner_type = 'user'
 AND product.product_type = 'subscription'
JOIN `platform_ai_capabilities` capability
  ON capability.capability_key = 'ai.chat'
 AND capability.audience = 'candidate'
 AND capability.status = 'active'
 AND capability.current_published_version_id IS NOT NULL
SET price.entitlement_snapshot = JSON_SET(
      COALESCE(price.entitlement_snapshot, JSON_OBJECT()),
      '$."ai.chat.release_version_id"',
      capability.current_published_version_id
    ),
    price.updated_at = UTC_TIMESTAMP(3)
WHERE JSON_CONTAINS_PATH(
  COALESCE(price.entitlement_snapshot, JSON_OBJECT()),
  'one',
  '$."ai.chat.enabled"'
);
