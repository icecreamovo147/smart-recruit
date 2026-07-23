UPDATE platform_plan_entitlements entitlement
JOIN platform_ai_capabilities capability
  ON capability.audience = 'tenant_hr'
 AND entitlement.entitlement_key = CONCAT(capability.capability_key, '.release_version_id')
JOIN platform_ai_capability_versions repaired
  ON repaired.capability_id = capability.id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
JOIN platform_ai_capability_versions previous
  ON previous.capability_id = repaired.capability_id
 AND previous.version = repaired.version - 1
SET entitlement.value_json = CAST(previous.id AS JSON),
    entitlement.updated_at = NOW(3)
WHERE CAST(JSON_UNQUOTE(entitlement.value_json) AS UNSIGNED) = repaired.id;

UPDATE tenant_entitlement_overrides entitlement_override
JOIN platform_ai_capabilities capability
  ON capability.audience = 'tenant_hr'
 AND entitlement_override.entitlement_key = CONCAT(capability.capability_key, '.release_version_id')
JOIN platform_ai_capability_versions repaired
  ON repaired.capability_id = capability.id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
JOIN platform_ai_capability_versions previous
  ON previous.capability_id = repaired.capability_id
 AND previous.version = repaired.version - 1
SET entitlement_override.value_json = CAST(previous.id AS JSON),
    entitlement_override.updated_at = NOW(3)
WHERE CAST(JSON_UNQUOTE(entitlement_override.value_json) AS UNSIGNED) = repaired.id;

UPDATE billing_price_versions price
JOIN platform_ai_capabilities capability ON capability.audience = 'tenant_hr'
JOIN platform_ai_capability_versions repaired
  ON repaired.capability_id = capability.id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
JOIN platform_ai_capability_versions previous
  ON previous.capability_id = repaired.capability_id
 AND previous.version = repaired.version - 1
SET price.entitlement_snapshot = JSON_SET(
      price.entitlement_snapshot,
      CONCAT('$."', capability.capability_key, '.release_version_id"'),
      previous.id
    ),
    price.updated_at = NOW(3)
WHERE CAST(JSON_UNQUOTE(JSON_EXTRACT(
  price.entitlement_snapshot,
  CONCAT('$."', capability.capability_key, '.release_version_id"')
)) AS UNSIGNED) = repaired.id;

UPDATE platform_ai_capabilities capability
JOIN platform_ai_capability_versions repaired
  ON repaired.id = capability.current_published_version_id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
JOIN platform_ai_capability_versions previous
  ON previous.capability_id = repaired.capability_id
 AND previous.version = repaired.version - 1
SET capability.current_published_version_id = previous.id,
    capability.updated_at = NOW(3);

DELETE audit
FROM platform_ai_config_audit_logs audit
JOIN platform_ai_capability_versions repaired
  ON repaired.id = audit.capability_version_id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
WHERE audit.action = 'capability.release.migration.repair';

DELETE FROM platform_ai_capability_versions
WHERE change_note = 'migration 000078: repair Agent-bound Prompt release references';
